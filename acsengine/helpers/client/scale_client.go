package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/operations"
	log "github.com/sirupsen/logrus"
)

// ScaleClient includes arguments needed to scale a Kubernetes cluster
type ScaleClient struct {
	ACSEngineClient

	DesiredAgentCount int
	AgentPoolToScale  string
	MasterFQDN        string
	AgentPool         *api.AgentPoolProfile
	AgentPoolIndex    int
}

// Validate checks that required client fields are set
func (sc *ScaleClient) Validate() error {
	// client.Logger = log.NewEntry(log.New())
	sc.Logger = log.New().WithField("source", "scaling update")

	if err := sc.ACSEngineClient.Validate(); err != nil {
		return fmt.Errorf("ACSEngineClient validation failed: %+v", err)
	}

	if sc.DesiredAgentCount < 1 {
		return fmt.Errorf("Desired agent count must be specified")
	}

	return nil
}

// DrainNodes drains and deletes all nodes in array provided
func (sc *ScaleClient) DrainNodes(kubeConfig string, vmsToDelete []string) error {
	masterURL := sc.MasterFQDN
	if !strings.HasPrefix(masterURL, "https://") {
		masterURL = fmt.Sprintf("https://%s", masterURL)
	}
	numVmsToDrain := len(vmsToDelete)
	errChan := make(chan *operations.VMScalingErrorDetails, numVmsToDrain)
	defer close(errChan)
	for _, vmName := range vmsToDelete {
		go func(vmName string) {
			err := operations.SafelyDrainNode(sc.Client, sc.Logger,
				masterURL, kubeConfig, vmName, time.Duration(60)*time.Minute) // is the vmName the node name?
			if err != nil {
				log.Errorf("Failed to drain node %s, got error %v", vmName, err)
				errChan <- &operations.VMScalingErrorDetails{Error: err, Name: vmName}
				return
			}
			errChan <- nil
		}(vmName)
	}

	for i := 0; i < numVmsToDrain; i++ {
		errDetails := <-errChan
		if errDetails != nil {
			return fmt.Errorf("Node %q failed to drain with error: %v", errDetails.Name, errDetails.Error) // failing here w/ upgrade then scale down
		}
	}

	return nil
}
