package acsengine

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/mgmt/keyvault"
	"github.com/satori/go.uuid"
)

func createClusterKeyVault(d *ResourceData, client *ArmClient) error {
	// get key vault client, which I have to add back in
	// client := client.keyVaultClient
	// ctx := client.StopContext
	log.Printf("[INFO] preparing arugments for Azure key vault creation.")

	var v interface{}
	var ok bool
	var name, resourceGroup, location string

	v, ok = d.GetOk("key_vault.0.name")
	if !ok {
		return fmt.Errorf("cluster 'key_vault.0.name' not found")
	}
	name = v.(string)

	v, ok = d.GetOk("resource_group")
	if !ok {
		return fmt.Errorf("cluster 'resource_group' not found")
	}
	resourceGroup = v.(string)

	v, ok = d.GetOk("location")
	if !ok {
		return fmt.Errorf("cluster 'location' not found")
	}
	location = azureRMNormalizeLocation(v.(string))

	fmt.Println(name, resourceGroup, location)

	// sku: standard or premium
	// sku := ""
	// tenantID, I'm pretty sure this is associated with subscription
	tenantID, err := uuid.FromString("")
	if err != nil {
		return err
	}
	enabledForDeployment := true
	enabledForDiskEncryption := true
	enabledForTemplateDeployment := true
	tags := d.getTags()

	parameters := keyvault.VaultCreateOrUpdateParameters{
		Location: &location,
		Properties: &keyvault.VaultProperties{
			TenantID: &tenantID,
			// Sku:      sku,
			// AccessPolicies:
			EnabledForDeployment:         &enabledForDeployment,
			EnabledForDiskEncryption:     &enabledForDiskEncryption,
			EnabledForTemplateDeployment: &enabledForTemplateDeployment,
		},
		Tags: expandTags(tags),
	}

	fmt.Println(parameters)

	return nil
}

func expandKeyVaultSku(d *ResourceData) keyvault.Sku {
	return keyvault.Sku{}
}

func expandKeyVaultAccessPolicies(policies []interface{}) (*[]keyvault.AccessPolicyEntry, error) {
	return nil, nil
}
