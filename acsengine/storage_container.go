package acsengine

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func createStorageContainer(d *schema.ResourceData, m interface{}) error {
	armClient := m.(*ArmClient)
	ctx := armClient.StopContext

	// get resource group and storage account
	var resourceGroup, storageAccount, container string
	if v, ok := d.GetOk("resource_group"); ok {
		resourceGroup = v.(string)
		storageAccount = storageAccountName(resourceGroup)
		container = fmt.Sprintf("%s-container", storageAccount)
	} else {
		return fmt.Errorf("cluster 'resource_group' not found")
	}
	blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(ctx, resourceGroup, storageAccount)
	if err != nil {
		return err
	}
	if !accountExists {
		return fmt.Errorf("Storage Account %q Not Found", storageAccount)
	}

	// create container
	reference := blobClient.GetContainerReference(container)
	err = resource.Retry(120*time.Second, checkContainerIsCreated(reference))
	if err != nil {
		return fmt.Errorf("Error creating container %q in storage account %q: %s", container, storageAccount, err)
	}

	// set container permissions
	accessType := storage.ContainerAccessType("") // private
	permissions := storage.ContainerPermissions{
		AccessType: accessType,
	}
	permissionOptions := &storage.SetContainerPermissionOptions{}
	err = reference.SetPermissions(permissions, permissionOptions)
	if err != nil {
		return fmt.Errorf("Error setting permissions for container %s in storage account %s: %+v", container, storageAccount, err)
	}

	return nil
}

// from resource_arm_storage_container.go

func checkContainerIsCreated(reference *storage.Container) func() *resource.RetryError {
	return func() *resource.RetryError {
		createOptions := &storage.CreateContainerOptions{}
		_, err := reference.CreateIfNotExists(createOptions)
		if err != nil {
			return resource.RetryableError(err)
		}

		return nil
	}
}
