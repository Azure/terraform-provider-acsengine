package client

import (
	"fmt"
	"time"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/armhelpers"
	"github.com/Azure/acs-engine/pkg/helpers"
	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/leonelquinteros/gotext"
	log "github.com/sirupsen/logrus"
)

// UpgradeClient includes arguments needed to upgrade a Kubernetes cluster
type UpgradeClient struct {
	AuthArgs

	ResourceGroupName   string
	DeploymentDirectory string
	UpgradeVersion      string
	Location            string
	TimeoutInMinutes    int

	Cluster             *api.ContainerService
	APIVersion          string
	APIModelPath        string // Do I really need this and DeploymentDirectory?
	Client              armhelpers.ACSEngineClient
	Locale              *gotext.Locale
	NameSuffix          string
	AgentPoolsToUpgrade []string
	Timeout             *time.Duration
	Logger              *log.Entry
}

// Validate checks that required client fields are set
func (uc *UpgradeClient) Validate() error {
	uc.Logger = log.New().WithField("source", "upgrade update")
	var err error

	uc.Locale, err = i18n.LoadTranslations()
	if err != nil {
		return fmt.Errorf("error loading translation files: %s", err.Error())
	}

	if uc.ResourceGroupName == "" {
		return fmt.Errorf("Resource group must be specified")
	}

	if uc.Location == "" {
		return fmt.Errorf("Location must be specified")
	}
	uc.Location = helpers.NormalizeAzureRegion(uc.Location)

	if uc.TimeoutInMinutes != -1 {
		timeout := time.Duration(uc.TimeoutInMinutes) * time.Minute
		uc.Timeout = &timeout
	}

	if uc.UpgradeVersion == "" {
		return fmt.Errorf("Upgrade version must be specified")
	}

	if uc.DeploymentDirectory == "" {
		return fmt.Errorf("Deployment directory must be specified")
	}

	return nil
}
