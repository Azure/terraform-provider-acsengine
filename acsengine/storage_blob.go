package acsengine

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"runtime"
	"sync"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform/helper/schema"
)

func createBlob(d *schema.ResourceData, m interface{}, deploymentDirectory string, filename string) error {
	client := m.(*ArmClient)
	ctx := client.StopContext

	// get resource group and storage account
	var resourceGroup, storageAccount string
	if v, ok := d.GetOk("resource_group"); ok {
		resourceGroup = v.(string)
		storageAccount = storageAccountName(resourceGroup)
	} else {
		return fmt.Errorf("cluster 'resource_group' not found")
	}
	blobClient, accountExists, err := client.getBlobStorageClientForStorageAccount(ctx, resourceGroup, storageAccount)
	if err != nil {
		return err
	}
	if !accountExists {
		return fmt.Errorf("Storage Account %q Not Found", storageAccount)
	}

	// get container
	containerName := fmt.Sprintf("%s-container", storageAccount)
	container := blobClient.GetContainerReference(containerName)
	containerExists, err := container.Exists()
	if err != nil {
		return err
	}
	if !containerExists {
		return fmt.Errorf("Container %s does not exist", containerName)
	}

	// create blob
	source := path.Join(deploymentDirectory, filename)
	contentType := "application/octet-stream"
	options := &storage.PutBlobOptions{}
	blob := container.GetBlobReference(filename)
	err = blob.CreateBlockBlob(options)
	if err != nil {
		return fmt.Errorf("Error creating storage blob on Azure: %s", err)
	}
	// upload blob to container
	// function is from resource_arm_storage_blob.go
	if err := resourceArmStorageBlobBlockUploadFromSource(containerName, filename, source, contentType, blobClient, 8, 1); err != nil {
		return fmt.Errorf("Error creating storage blob on Azure: %s", err)
	}

	return nil
}

func getBlob(d *schema.ResourceData, m interface{}, filename string) (string, error) {
	client := m.(*ArmClient)
	ctx := client.StopContext

	// get resource group and storage account
	var resourceGroup, storageAccount string
	if v, ok := d.GetOk("resource_group"); ok {
		resourceGroup = v.(string)
		storageAccount = storageAccountName(resourceGroup)
	} else {
		return "", fmt.Errorf("cluster 'resource_group' not found")
	}
	blobClient, accountExists, err := client.getBlobStorageClientForStorageAccount(ctx, resourceGroup, storageAccount)
	if err != nil {
		return "", err
	}
	if !accountExists {
		return "", fmt.Errorf("Storage Account %q Not Found", storageAccount)
	}

	// get container
	containerName := fmt.Sprintf("%s-container", storageAccount)
	container := blobClient.GetContainerReference(containerName)
	containerExists, err := container.Exists()
	if err != nil {
		return "", err
	}
	if !containerExists {
		return "", fmt.Errorf("Container %s does not exist", containerName)
	}

	// get blob
	options := &storage.GetBlobOptions{}
	blob := container.GetBlobReference(filename)
	blobExists, err := blob.Exists()
	if err != nil {
		return "", err
	}
	if !blobExists {
		return "", fmt.Errorf("Blob %s does not exist", filename)
	}
	// read from blob
	reader, err := blob.Get(options)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	output := buf.String()

	return output, nil
}

// from resource_arm_storage_blob.go

type resourceArmStorageBlobBlock struct {
	section *io.SectionReader
	id      string
}

type resourceArmStorageBlobBlockUploadContext struct {
	client    *storage.BlobStorageClient
	container string
	name      string
	source    string
	attempts  int
	blocks    chan resourceArmStorageBlobBlock
	errors    chan error
	wg        *sync.WaitGroup
}

func resourceArmStorageBlobBlockSplit(file *os.File) ([]storage.Block, []resourceArmStorageBlobBlock, error) {
	const (
		idSize          = 64
		blockSize int64 = 4 * 1024 * 1024
	)
	var parts []resourceArmStorageBlobBlock
	var blockList []storage.Block

	info, err := file.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("Error stating source file %q: %s", file.Name(), err)
	}

	for i := int64(0); i < info.Size(); i = i + blockSize {
		entropy := make([]byte, idSize)
		_, err = rand.Read(entropy)
		if err != nil {
			return nil, nil, fmt.Errorf("Error generating a random block ID for source file %q: %s", file.Name(), err)
		}

		sectionSize := blockSize
		remainder := info.Size() - i
		if remainder < blockSize {
			sectionSize = remainder
		}

		block := storage.Block{
			ID:     base64.StdEncoding.EncodeToString(entropy),
			Status: storage.BlockStatusUncommitted,
		}

		blockList = append(blockList, block)

		parts = append(parts, resourceArmStorageBlobBlock{
			id:      block.ID,
			section: io.NewSectionReader(file, i, sectionSize),
		})
	}

	return blockList, parts, nil
}

func resourceArmStorageBlobBlockUploadWorker(ctx resourceArmStorageBlobBlockUploadContext) {
	for block := range ctx.blocks {
		buffer := make([]byte, block.section.Size())

		_, err := block.section.Read(buffer)
		if err != nil {
			ctx.errors <- fmt.Errorf("Error reading source file %q: %s", ctx.source, err)
			ctx.wg.Done()
			continue
		}

		for i := 0; i < ctx.attempts; i++ {
			container := ctx.client.GetContainerReference(ctx.container)
			blob := container.GetBlobReference(ctx.name)
			options := &storage.PutBlockOptions{}
			err = blob.PutBlock(block.id, buffer, options)
			if err == nil {
				break
			}
		}
		if err != nil {
			ctx.errors <- fmt.Errorf("Error uploading block %q for source file %q: %s", block.id, ctx.source, err)
			ctx.wg.Done()
			continue
		}

		ctx.wg.Done()
	}
}

func resourceArmStorageBlobBlockUploadFromSource(container, name, source, contentType string, client *storage.BlobStorageClient, parallelism, attempts int) error {
	workerCount := parallelism * runtime.NumCPU()

	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("Error opening source file for upload %q: %s", source, err)
	}
	defer file.Close()

	blockList, parts, err := resourceArmStorageBlobBlockSplit(file)
	if err != nil {
		return fmt.Errorf("Error reading and splitting source file for upload %q: %s", source, err)
	}

	wg := &sync.WaitGroup{}
	blocks := make(chan resourceArmStorageBlobBlock, len(parts))
	errors := make(chan error, len(parts))

	wg.Add(len(parts))
	for _, p := range parts {
		blocks <- p
	}
	close(blocks)

	for i := 0; i < workerCount; i++ {
		go resourceArmStorageBlobBlockUploadWorker(resourceArmStorageBlobBlockUploadContext{
			client:    client,
			source:    source,
			container: container,
			name:      name,
			blocks:    blocks,
			errors:    errors,
			wg:        wg,
			attempts:  attempts,
		})
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("Error while uploading source file %q: %s", source, <-errors)
	}

	containerReference := client.GetContainerReference(container)
	blobReference := containerReference.GetBlobReference(name)
	blobReference.Properties.ContentType = contentType
	options := &storage.PutBlockListOptions{}
	err = blobReference.PutBlockList(blockList, options)
	if err != nil {
		return fmt.Errorf("Error updating block list for source file %q: %s", source, err)
	}

	return nil
}
