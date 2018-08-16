package operations

import (
	"context"
	"fmt"
	"path"

	"github.com/Azure/acs-engine/pkg/acsengine"
	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/armhelpers"
	"github.com/Azure/acs-engine/pkg/helpers"
	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/terraform-provider-acsengine/internal/resource"
	"github.com/leonelquinteros/gotext"
	log "github.com/sirupsen/logrus"
)

// ACSEngineClient contains fields needed for acs-engine cluster operations
type ACSEngineClient struct {
	AuthArgs

	ResourceGroupName   string
	Location            string
	Cluster             *api.ContainerService
	Client              armhelpers.ACSEngineClient
	DeploymentDirectory string
	APIVersion          string
	APIModelPath        string // Do I really need this and DeploymentDirectory?
	Locale              *gotext.Locale
	NameSuffix          string
	Logger              *log.Entry
}

// NewACSEngineClient returns a new acs-engine cluster client
func NewACSEngineClient(secret string) *ACSEngineClient {
	authArgs := NewAuthArgs(secret)
	return &ACSEngineClient{
		AuthArgs: *authArgs,
	}
}

// AddACSEngineClientAuthArgs adds auth args and sets up client
func (c *ACSEngineClient) AddACSEngineClientAuthArgs(cluster *api.ContainerService, azureID string) error {
	var err error
	if err = c.AddAuthArgs(cluster, azureID); err != nil {
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

// SetACSEngineClient sets all necessary client fields for cluster
func (c *ACSEngineClient) SetACSEngineClient(cluster *api.ContainerService, azureID string) error {
	var err error

	id, err := resource.ParseAzureResourceID(azureID)
	if err != nil {
		return fmt.Errorf("error parsing resource ID: %+v", err)
	}
	c.ResourceGroupName = id.ResourceGroup

	c.DeploymentDirectory = path.Join("_output", cluster.Properties.MasterProfile.DNSPrefix)
	c.Location = cluster.Location
	if c.Locale, err = i18n.LoadTranslations(); err != nil {
		return fmt.Errorf("error loading translation files: %+v", err)
	}

	if err = c.AddACSEngineClientAuthArgs(cluster, azureID); err != nil {
		return fmt.Errorf("failed to add ACSEngineClient auth args: %+v", err)
	}

	c.Cluster = cluster
	if c.Cluster.Location != c.Location {
		return fmt.Errorf("location does not match api model location") // this should probably never happen?
	}

	c.NameSuffix = acsengine.GenerateClusterID(c.Cluster.Properties)

	return nil
}

// Validate validates general parameters needed for acs-engine cluster operations
func (c *ACSEngineClient) Validate() error {
	var err error

	c.Locale, err = i18n.LoadTranslations()
	if err != nil {
		return fmt.Errorf("error loading translation files: %s", err.Error())
	}

	if c.ResourceGroupName == "" {
		return fmt.Errorf("Resource group must be specified")
	}

	if c.Location == "" {
		return fmt.Errorf("Location must be specified")
	}
	c.Location = helpers.NormalizeAzureRegion(c.Location)

	if c.DeploymentDirectory == "" {
		return fmt.Errorf("Deployment directory must be specified")
	}

	return nil
}
