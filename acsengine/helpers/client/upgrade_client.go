package client

import (
	"fmt"
	"time"

	"github.com/Azure/acs-engine/pkg/api"
	log "github.com/sirupsen/logrus"
)

// UpgradeClient includes arguments needed to upgrade a Kubernetes cluster
type UpgradeClient struct {
	ACSEngineClient

	UpgradeVersion      string
	TimeoutInMinutes    int
	AgentPoolsToUpgrade []string
	Timeout             *time.Duration
}

// NewUpgradeClient returns a new UpgradeClient
func NewUpgradeClient(secret string) *UpgradeClient {
	acsengineClient := NewACSEngineClient(secret)
	return &UpgradeClient{
		ACSEngineClient: *acsengineClient,
	}
}

// SetUpgradeClient sets acs-engine upgrade client fields
func (uc *UpgradeClient) SetUpgradeClient(cluster *api.ContainerService, azureID, upgradeVersion string) error {
	if err := uc.ACSEngineClient.SetACSEngineClient(cluster, azureID); err != nil {
		return fmt.Errorf("failed to initialize ACSEngineClient: %+v", err)
	}

	uc.UpgradeVersion = upgradeVersion
	uc.TimeoutInMinutes = -1
	if err := uc.Validate(); err != nil {
		return fmt.Errorf(": %+v", err)
	}

	uc.Cluster.Properties.OrchestratorProfile.OrchestratorVersion = uc.UpgradeVersion

	uc.AgentPoolsToUpgrade = []string{}
	for _, agentPool := range uc.Cluster.Properties.AgentPoolProfiles {
		uc.AgentPoolsToUpgrade = append(uc.AgentPoolsToUpgrade, agentPool.Name)
	}

	return nil
}

// Validate checks that required client fields are set
func (uc *UpgradeClient) Validate() error {
	uc.Logger = log.New().WithField("source", "upgrade update")

	if err := uc.ACSEngineClient.Validate(); err != nil {
		return fmt.Errorf("ACSEngineClient validation failed: %+v", err)
	}

	if uc.TimeoutInMinutes != -1 {
		timeout := time.Duration(uc.TimeoutInMinutes) * time.Minute
		uc.Timeout = &timeout
	}

	if uc.UpgradeVersion == "" {
		return fmt.Errorf("Upgrade version must be specified")
	}

	return nil
}
