package acsengine

import (
	"context"
	"fmt"
	"path"

	"github.com/Azure/acs-engine/pkg/acsengine"
	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client"
	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/hashicorp/terraform/helper/schema"
)

func addAuthArgs(d *schema.ResourceData, a *client.AuthArgs) error {
	client.AddAuthArgs(a)
	id, err := utils.ParseAzureResourceID(d.Id())
	if err != nil {
		return fmt.Errorf("error parsing resource ID: %+v", err)
	}
	a.RawSubscriptionID = id.SubscriptionID
	a.AuthMethod = "client_secret"
	if v, ok := d.GetOk("service_principal.0.client_id"); ok {
		a.RawClientID = v.(string)
	}
	if v, ok := d.GetOk("service_principal.0.client_secret"); ok {
		a.ClientSecret = v.(string)
	}
	if err = a.ValidateAuthArgs(); err != nil {
		return fmt.Errorf("error validating auth args: %+v", err)
	}
	return nil
}

func addACSEngineClientAuthArgs(d *schema.ResourceData, c *client.ACSEngineClient) error {
	err := addAuthArgs(d, &c.AuthArgs)
	if err != nil {
		return fmt.Errorf("failed to add auth args: %+v", err)
	}

	if c.Client, err = c.GetClient(); err != nil {
		return fmt.Errorf("failed to get client: %+v", err)
	}
	if _, err = c.Client.EnsureResourceGroup(context.Background(), c.ResourceGroupName, c.Location, nil); err != nil {
		return fmt.Errorf("failed to get client: %+v", err)
	}

	return nil
}

// If I pass an api.ContainerService struct to this, it's easier to move this into separate package
func setACSEngineClient(d *schema.ResourceData, c *client.ACSEngineClient) error {
	var err error
	if v, ok := d.GetOk("resource_group"); ok {
		c.ResourceGroupName = v.(string)
	}
	if v, ok := d.GetOk("master_profile.0.dns_name_prefix"); ok {
		c.DeploymentDirectory = path.Join("_output", v.(string))
	}
	if v, ok := d.GetOk("location"); ok {
		c.Location = azureRMNormalizeLocation(v.(string))
	}
	if c.Locale, err = i18n.LoadTranslations(); err != nil {
		return fmt.Errorf("error loading translation files: %+v", err)
	}

	if err = addACSEngineClientAuthArgs(d, c); err != nil {
		return fmt.Errorf("failed to add ACSEngineClient auth args: %+v", err)
	}

	c.Cluster, err = loadContainerServiceFromApimodel(d, true, true)
	if err != nil {
		return fmt.Errorf("error parsing the api model: %+v", err)
	}
	if c.Cluster.Location != c.Location {
		return fmt.Errorf("location does not match api model location") // this should probably never happen?
	}

	c.NameSuffix = acsengine.GenerateClusterID(c.Cluster.Properties)

	return nil
}
