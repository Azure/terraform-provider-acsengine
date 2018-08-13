package acsengine

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/mgmt/keyvault"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/satori/go.uuid"
)

// I don't actually need this...

// I ought to check if something like this exists in keyvault package already
type keyVault struct {
	name        string
	rawTenantID string
	tenantID    uuid.UUID
	sku         *keyvault.Sku
}

func accessPolicySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		MaxItems: 16,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"tenant_id":               {},
				"object_id":               {},
				"application_id":          {},
				"certificate_permissions": {},
				"key_permissions":         {},
				"secret_permissions":      {},
			},
		},
	}
}

// should this be something to configure?
func expandKeyVaultSku(d *ResourceData) *keyvault.Sku {
	skuFamily := "A" // what should this be?
	skuName := ""

	return &keyvault.Sku{
		Family: &skuFamily,
		Name:   keyvault.SkuName(skuName),
	}
}

func expandKeyVaultAccessPolicies(policies []interface{}) (*[]keyvault.AccessPolicyEntry, error) {
	return nil, nil
}

func keyVaultRefreshFunc(vaultURI string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Checking to see if key vault %q is available...", vaultURI)

		var PTransport = &http.Transport{Proxy: http.ProxyFromEnvironment}

		client := &http.Client{
			Transport: PTransport,
		}

		conn, err := client.Get(vaultURI)
		if err != nil {
			log.Printf("[DEBUG] Didn't find KeyVault at %q", vaultURI)
			return nil, "pending", fmt.Errorf("Error connecting to %q: %s", vaultURI, err)
		}

		defer conn.Body.Close()

		log.Printf("[DEBUG] Found KeyVault at %q", vaultURI)
		return "available", "available", nil

	}
}

func createClusterKeyVault(d *ResourceData, client *ArmClient) error {
	keyVaultClient := client.keyVaultClient
	log.Printf("[INFO] preparing arugments for Azure key vault creation.")

	var v interface{}
	var ok bool
	var name, resourceGroup, location, rawTenantID string

	// should I make my own key vault struct?
	v, ok = d.GetOk("key_vault.0.name")
	if !ok {
		return fmt.Errorf("cluster 'key_vault.0.name' not found")
	}
	name = v.(string)

	v, ok = d.GetOk("key_vault.0.tenant_id")
	if !ok {
		return fmt.Errorf("cluster 'key_vault.0.name' not found")
	}
	rawTenantID = v.(string)
	//

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
	tenantID, err := uuid.FromString(rawTenantID)
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

	future, err := keyVaultClient.CreateOrUpdate(client.StopContext, resourceGroup, name, parameters)
	if err != nil {
		return fmt.Errorf("error creating key vault %q (resource group %q): %+v", name, resourceGroup, err)
	}
	fmt.Println(future)

	read, err := keyVaultClient.Get(client.StopContext, resourceGroup, name)
	if err != nil {
		return fmt.Errorf("error retrieving key vault %q (resource group %q): %+v", name, resourceGroup, err)
	}
	if read.ID == nil {
		return fmt.Errorf("cannot read key vault %q (resource group %q) ID", name, resourceGroup)
	}

	// what's the reasoning behind this vs future.WaitForCompletion?
	if d.IsNewResource() { // will I ever be updating? does it make sense to have this?
		if props := read.Properties; props != nil {
			if vault := props.VaultURI; vault != nil {
				log.Printf("[DEBUG] Waiting for key vault %q (resource group %q) to become available", name, resourceGroup)
				stateConf := &resource.StateChangeConf{
					Pending:                   []string{"pending"},
					Target:                    []string{"target"},
					Refresh:                   keyVaultRefreshFunc(*vault),
					Timeout:                   30 * time.Minute,
					Delay:                     30 * time.Second,
					PollInterval:              10 * time.Second,
					ContinuousTargetOccurence: 10,
				}

				if _, err := stateConf.WaitForState(); err != nil {
					return fmt.Errorf("error waiting for key vault %q (resource group %q) to become available: %+v", name, resourceGroup, err)
				}
			}
		}
	}

	return nil
}
