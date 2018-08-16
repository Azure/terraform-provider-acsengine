package operations

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/armhelpers/utils"
	"github.com/Azure/acs-engine/pkg/operations"
	"github.com/Azure/terraform-provider-acsengine/internal/resource"
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
	DeploymentName    string
}

// NewScaleClient returns a new ScaleClient
func NewScaleClient(secret string) *ScaleClient {
	acsengineClient := NewACSEngineClient(secret)
	return &ScaleClient{
		ACSEngineClient: *acsengineClient,
	}
}

// SetScaleClient sets values in acsengine scale client
func (sc *ScaleClient) SetScaleClient(cluster *api.ContainerService, azureID string, agentIndex, agentCount int) error {
	var err error

	err = sc.ACSEngineClient.SetACSEngineClient(cluster, azureID)
	if err != nil {
		return fmt.Errorf("failed to initialize ACSEngineClient: %+v", err)
	}

	id, err := resource.ParseAzureResourceID(azureID)
	if err != nil {
		return fmt.Errorf("error parsing resource ID: %+v", err)
	}

	sc.DesiredAgentCount = agentCount

	// sc.MasterFQDN = cluster.Properties.MasterProfile.FQDN // is this not being set??
	endpointSuffix := "cloudapp.azure.com"
	sc.MasterFQDN = cluster.Properties.MasterProfile.DNSPrefix + "." + cluster.Location + "." + endpointSuffix

	sc.AgentPoolIndex = agentIndex
	sc.AgentPoolToScale = cluster.Properties.AgentPoolProfiles[agentIndex].Name
	sc.DeploymentName = id.Path["deployments"]
	if sc.DeploymentName == "" {
		sc.DeploymentName = id.Path["Deployments"]
	}
	if err := sc.Validate(); err != nil {
		return fmt.Errorf("error validating scale client: %+v", err)
	}

	sc.AgentPool = sc.Cluster.Properties.AgentPoolProfiles[sc.AgentPoolIndex]

	return nil
}

// ScaleVMAS gets information for scaling Virtual Machine availability sets
func (sc *ScaleClient) ScaleVMAS(ctx context.Context) (int, int, int, []string, error) {
	var currentNodeCount, highestUsedIndex, vmNum int
	windowsIndex := -1
	highestUsedIndex = 0
	indexToVM := make([]string, 0)
	vms, err := sc.Client.ListVirtualMachines(ctx, sc.ResourceGroupName)
	if err != nil {
		return highestUsedIndex, currentNodeCount, windowsIndex, indexToVM, fmt.Errorf("failed to get vms in the resource group. Error: %s", err.Error())
	} else if len(vms.Values()) < 1 {
		return highestUsedIndex, currentNodeCount, windowsIndex, indexToVM, fmt.Errorf("The provided resource group does not contain any vms")
	}
	for _, vm := range vms.Values() {
		vmTags := vm.Tags
		poolName := *vmTags["poolName"]
		nameSuf := *vmTags["resourceNameSuffix"]

		if err != nil || !strings.EqualFold(poolName, sc.AgentPoolToScale) || !strings.Contains(sc.NameSuffix, nameSuf) {
			continue
		}

		osPublisher := vm.StorageProfile.ImageReference.Publisher
		if osPublisher != nil && strings.EqualFold(*osPublisher, "MicrosoftWindowsServer") {
			_, _, windowsIndex, vmNum, err = utils.WindowsVMNameParts(*vm.Name)
		} else {
			_, _, vmNum, err = utils.K8sLinuxVMNameParts(*vm.Name) // this needs to be tested
		}
		if err != nil {
			return highestUsedIndex, currentNodeCount, windowsIndex, indexToVM, fmt.Errorf("error getting VM parts: %+v", err)
		}
		if vmNum > highestUsedIndex {
			highestUsedIndex = vmNum
		}

		indexToVM = append(indexToVM, *vm.Name)
	}
	currentNodeCount = len(indexToVM)

	return highestUsedIndex, currentNodeCount, windowsIndex, indexToVM, nil
}

// ScaleVMSS gets information for scaling Virtual Machine scale sets
func (sc *ScaleClient) ScaleVMSS(ctx context.Context) (int, int, int, error) {
	var currentNodeCount, highestUsedIndex int
	windowsIndex := -1
	highestUsedIndex = 0
	vmssList, err := sc.Client.ListVirtualMachineScaleSets(ctx, sc.ResourceGroupName)
	if err != nil {
		return highestUsedIndex, currentNodeCount, windowsIndex, fmt.Errorf("failed to get vmss list in the resource group: %+v", err)
	}
	for _, vmss := range vmssList.Values() {
		vmTags := vmss.Tags
		poolName := *vmTags["poolName"]
		nameSuffix := *vmTags["resourceNameSuffix"]

		if err != nil || !strings.EqualFold(poolName, sc.AgentPoolToScale) || !strings.Contains(sc.NameSuffix, nameSuffix) {
			continue
		}

		osPublisher := *vmss.VirtualMachineProfile.StorageProfile.ImageReference.Publisher
		if strings.EqualFold(osPublisher, "MicrosoftWindowsServer") {
			_, _, windowsIndex, err = utils.WindowsVMSSNameParts(*vmss.Name)
			// log error here?
		}

		currentNodeCount = int(*vmss.Sku.Capacity)
		highestUsedIndex = 0
	}

	return highestUsedIndex, currentNodeCount, windowsIndex, nil
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

// Validate checks that required client fields are set
func (sc *ScaleClient) Validate() error {
	sc.Logger = log.New().WithField("source", "scaling update")

	if err := sc.ACSEngineClient.Validate(); err != nil {
		return fmt.Errorf("ACSEngineClient validation failed: %+v", err)
	}

	if sc.DesiredAgentCount < 1 {
		return fmt.Errorf("Desired agent count must be specified")
	}

	if sc.DeploymentName == "" {
		return fmt.Errorf("Deployment name must be specified")
	}

	return nil
}
