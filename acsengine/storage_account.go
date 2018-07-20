package acsengine

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/hashicorp/terraform/helper/schema"
)

// Creates a storage account for storing ACS Engine output in a storage blob
func createClusterStorageAccount(d *schema.ResourceData, m interface{}) error {
	client := m.(*ArmClient)
	storageClient := client.storageServiceClient

	/* Initialize storage account parameters */
	var resourceGroup, storageAccount string
	if v, ok := d.GetOk("resource_group"); ok {
		resourceGroup = v.(string)
		storageAccount = storageAccountName(resourceGroup)
	} else {
		return fmt.Errorf("cluster 'resource_group' not found")
	}

	var location string
	if v, ok := d.GetOk("location"); ok {
		location = azureRMNormalizeLocation(v.(string))
	} else {
		return fmt.Errorf("cluster 'location' not found")
	}
	var tags map[string]interface{}
	if v, ok := d.GetOk("tags"); ok {
		tags = v.(map[string]interface{})
	} else {
		tags = map[string]interface{}{}
	}

	accountKind := "BlobStorage"
	accountTier := "Standard"
	accessTier := "Hot"
	replicationType := "RAGRS"
	storageType := fmt.Sprintf("%s_%s", accountTier, replicationType)
	storageAccountEncryptionSource := "Microsoft.Storage"

	enableBlobEncryption := true
	enableFileEncryption := true
	enableHTTPSTrafficOnly := false

	networkRules := &storage.NetworkRuleSet{DefaultAction: storage.DefaultActionAllow}

	parameters := storage.AccountCreateParameters{
		Location: &location,
		Sku: &storage.Sku{
			Name: storage.SkuName(storageType),
		},
		Tags: expandTags(tags),
		Kind: storage.Kind(accountKind),
		AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{
			Encryption: &storage.Encryption{
				Services: &storage.EncryptionServices{
					Blob: &storage.EncryptionService{
						Enabled: utils.Bool(enableBlobEncryption),
					},
					File: &storage.EncryptionService{
						Enabled: utils.Bool(enableFileEncryption),
					}},
				KeySource: storage.KeySource(storageAccountEncryptionSource),
			},
			EnableHTTPSTrafficOnly: &enableHTTPSTrafficOnly,
			NetworkRuleSet:         networkRules,
			AccessTier:             storage.AccessTier(accessTier),
		},
	}

	/* Create account */
	ctx := client.StopContext
	future, err := storageClient.Create(ctx, resourceGroup, storageAccount, parameters)
	if err != nil {
		return fmt.Errorf("Error creating Azure Storage Account %q: %+v", storageAccount, err)
	}
	err = future.WaitForCompletion(ctx, storageClient.Client)
	if err != nil {
		return fmt.Errorf("Error waiting for Azure Storage Account %q to be created: %+v", storageAccount, err)
	}

	/* Check that account exists */
	account, err := storageClient.GetProperties(ctx, resourceGroup, storageAccount)
	if err != nil {
		return fmt.Errorf("Error retrieving Azure Storage Account %q: %+v", storageAccount, err)
	}
	if account.ID == nil {
		return fmt.Errorf("Cannot read Storage Account %q (resource group %q) ID", storageAccount, resourceGroup)
	}
	log.Printf("[INFO] storage account %q ID: %q", storageAccount, *account.ID)

	return nil
}

// This needs to be more unique, make this a random value that I can still access
// Returns a storage account name which will meet the naming conventions, must be deterministic
func storageAccountName(str string) string {
	// I need to make sure the string is unique somehow, maybe make a hash value
	rgx, err := regexp.Compile("[^a-zA-Z0-9]")
	if err != nil {
		log.Fatalf("%+v", err)
	}
	str = strings.ToLower(rgx.ReplaceAllString(str, ""))
	var prefix string
	if len(str) > 20 {
		prefix = str[:20]
	} else {
		prefix = str
	}
	accountName := fmt.Sprintf("%sacc", prefix)
	return accountName
}

// this could still be useful (except I dont need access key anymore)
func storageAccountInfo(d *schema.ResourceData, m interface{}) (string, string, error) {
	client := m.(*ArmClient)
	ctx := client.StopContext
	storageClient := client.storageServiceClient

	var resourceGroup string
	if v, ok := d.GetOk("resource_group"); ok {
		resourceGroup = v.(string)
	} else {
		return "", "", fmt.Errorf("cluster 'resource_group' not found")
	}

	accountName := storageAccountName(resourceGroup)

	_, err := storageClient.GetProperties(ctx, resourceGroup, accountName)
	if err != nil {
		return "", "", fmt.Errorf("Error reading the state of AzureRM Storage Account %q: %+v", accountName, err)
	}

	keys, err := storageClient.ListKeys(ctx, resourceGroup, accountName)
	if err != nil {
		return "", "", err
	}
	accessKeys := *keys.Keys
	if len(accessKeys) < 1 {
		return "", "", fmt.Errorf("could not find storage account access keys")
	}

	accountKey := *accessKeys[0].Value

	return accountName, accountKey, nil
}
