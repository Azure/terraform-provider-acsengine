package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/armhelpers"
	"github.com/Azure/acs-engine/pkg/helpers"
	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/acs-engine/pkg/operations"
	"github.com/leonelquinteros/gotext"
	log "github.com/sirupsen/logrus"
)

// ScaleClient includes arguments needed to scale a Kubernetes cluster
type ScaleClient struct {
	AuthArgs AuthArgs
	// user input
	ResourceGroupName   string
	DeploymentDirectory string
	DesiredAgentCount   int
	Location            string
	AgentPoolToScale    string
	MasterFQDN          string

	K8sCluster     *api.ContainerService
	APIVersion     string
	APIModelPath   string // Do I really need this and DeploymentDirectory?
	AgentPool      *api.AgentPoolProfile
	Client         armhelpers.ACSEngineClient
	Locale         *gotext.Locale
	NameSuffix     string
	AgentPoolIndex int
	Logger         *log.Entry
}

// Validate checks that required client fields are set
func (client *ScaleClient) Validate() error {
	// client.Logger = log.NewEntry(log.New())
	client.Logger = log.New().WithField("source", "scaling update")
	var err error

	client.Locale, err = i18n.LoadTranslations()
	if err != nil {
		return fmt.Errorf("error loading translation files: %s", err.Error())
	}

	if client.ResourceGroupName == "" {
		return fmt.Errorf("Resource group must be specified")
	}

	if client.Location == "" {
		return fmt.Errorf("Location must be specified")
	}
	client.Location = helpers.NormalizeAzureRegion(client.Location)

	if client.DesiredAgentCount < 1 {
		return fmt.Errorf("Desired agent count must be specified")
	}

	if client.DeploymentDirectory == "" {
		return fmt.Errorf("Deployment directory must be specified")
	}

	return nil
}

// DrainNodes drains and deletes all nodes in array provided
func (client *ScaleClient) DrainNodes(kubeConfig string, vmsToDelete []string) error {
	masterURL := client.MasterFQDN
	if !strings.HasPrefix(masterURL, "https://") {
		masterURL = fmt.Sprintf("https://%s", masterURL)
	}
	numVmsToDrain := len(vmsToDelete)
	errChan := make(chan *operations.VMScalingErrorDetails, numVmsToDrain)
	defer close(errChan)
	for _, vmName := range vmsToDelete {
		go func(vmName string) {
			err := operations.SafelyDrainNode(client.Client, client.Logger,
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
