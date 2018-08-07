package client

import (
	"fmt"
	"time"

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
