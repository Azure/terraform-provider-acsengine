package client

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/armhelpers"
	"github.com/Azure/acs-engine/pkg/helpers"
	"github.com/Azure/acs-engine/pkg/i18n"
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
