package acsengine

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/Azure/acs-engine/pkg/acsengine"
	"github.com/Azure/acs-engine/pkg/acsengine/transform"
	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/armhelpers/utils"
	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/acs-engine/pkg/operations"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client"
	"github.com/hashicorp/terraform/helper/schema"
)

func scaleCluster(d *schema.ResourceData, agentIndex, agentCount int) error {
	sc, err := setScaleClient(d, agentIndex, agentCount)
	if err != nil {
		return fmt.Errorf("failed to initialize scale client: %+v", err)
	}

	var currentNodeCount, highestUsedIndex, windowsIndex int
	var indexToVM []string
	if sc.AgentPool.IsAvailabilitySets() {
		if highestUsedIndex, currentNodeCount, windowsIndex, indexToVM, err = scaleVMAS(sc, d); err != nil {
			return fmt.Errorf("failed to scale availability set: %+v", err)
		}

		if currentNodeCount == sc.DesiredAgentCount {
			log.Printf("Cluster is currently at the desired agent count")
			return nil
		}
		if currentNodeCount > sc.DesiredAgentCount {
			return scaleDownCluster(sc, currentNodeCount, indexToVM, d)
		}
	} else {
		if highestUsedIndex, currentNodeCount, windowsIndex, err = scaleVMSS(sc); err != nil {
			return fmt.Errorf("failed to scale scale set: %+v", err)
		}
	}

	return scaleUpCluster(sc, highestUsedIndex, currentNodeCount, windowsIndex, d)
}

func setScaleClient(d *schema.ResourceData, agentIndex int, agentCount int) (*client.ScaleClient, error) {
	sc := client.NewScaleClient()
	var err error

	err = setACSEngineClient(d, &sc.ACSEngineClient)
	if err != nil {
		return sc, fmt.Errorf("failed to initialize ACSEngineClient: %+v", err)
	}

	sc.DesiredAgentCount = agentCount
	if v, ok := d.GetOk("master_profile.0.fqdn"); ok {
		sc.MasterFQDN = v.(string)
	}
	sc.AgentPoolIndex = agentIndex
	v, ok := d.GetOk(fmt.Sprintf("agent_pool_profiles.%d.name", agentIndex))
	if !ok {
		return sc, fmt.Errorf("agent pool profile name not found")
	}
	sc.AgentPoolToScale = v.(string)
	if err := sc.Validate(); err != nil {
		return sc, fmt.Errorf("error validating scale client: %+v", err)
	}

	sc.AgentPool = sc.Cluster.Properties.AgentPoolProfiles[sc.AgentPoolIndex]

	return sc, nil
}

func scaleVMAS(sc *client.ScaleClient, d *schema.ResourceData) (int, int, int, []string, error) {
	var currentNodeCount, highestUsedIndex, vmNum int
	windowsIndex := -1
	highestUsedIndex = 0
	indexToVM := make([]string, 0)
	ctx := context.Background() // StopContext
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

func scaleVMSS(sc *client.ScaleClient) (int, int, int, error) {
	var currentNodeCount, highestUsedIndex int
	windowsIndex := -1
	highestUsedIndex = 0
	ctx := context.Background() // StopContext
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

// Scales down a cluster by draining and deleting the nodes given as input
func scaleDownCluster(sc *client.ScaleClient, currentNodeCount int, indexToVM []string, d *schema.ResourceData) error {
	if sc.MasterFQDN == "" {
		return fmt.Errorf("Master FQDN is required to scale down a Kubernetes cluster's agent pool")
	}

	vmsToDelete := make([]string, 0)
	for i := currentNodeCount - 1; i >= sc.DesiredAgentCount; i-- {
		vmsToDelete = append(vmsToDelete, indexToVM[i])
	}

	kubeconfig, err := acsengine.GenerateKubeConfig(sc.Cluster.Properties, sc.Location)
	if err != nil {
		return fmt.Errorf("failed to generate kube config: %+v", err)
	}
	if err = sc.DrainNodes(kubeconfig, vmsToDelete); err != nil {
		return fmt.Errorf("Got error while draining the nodes to be deleted: %+v", err)
	}

	errList := operations.ScaleDownVMs(
		sc.Client,
		sc.Logger,
		sc.SubscriptionID.String(),
		sc.ResourceGroupName,
		vmsToDelete...)
	if errList != nil {
		errorMessage := ""
		for element := errList.Front(); element != nil; element = element.Next() {
			vmError, ok := element.Value.(*operations.VMScalingErrorDetails)
			if ok {
				error := fmt.Sprintf("Node '%s' failed to delete with error: '%s'", vmError.Name, vmError.Error.Error())
				errorMessage = errorMessage + error
			}
		}
		return fmt.Errorf(errorMessage)
	}

	return saveScaledApimodel(sc, d)
}

func scaleUpCluster(sc *client.ScaleClient, highestUsedIndex, currentNodeCount, windowsIndex int, d *schema.ResourceData) error {
	sc.Cluster.Properties.AgentPoolProfiles = []*api.AgentPoolProfile{sc.AgentPool} // how does this work when there's multiple agent pools?

	ctx := acsengine.Context{
		Translator: &i18n.Translator{
			Locale: sc.Locale,
		},
	}
	// don't format parameters! It messes things up
	template, parameters, _, err := formatTemplates(sc.Cluster, false)
	if err != nil {
		return fmt.Errorf("failed to format templates: %+v", err)
	}

	templateJSON, parametersJSON, err := expandTemplates(template, parameters)
	if err != nil {
		return fmt.Errorf("failed to expand template and parameters: %+v", err)
	}

	transformer := transform.Transformer{Translator: ctx.Translator}

	countForTemplate := setCountForTemplate(sc, highestUsedIndex, currentNodeCount)
	addValue(parametersJSON, sc.AgentPoolToScale+"Count", countForTemplate)

	setWindowsIndex(sc, windowsIndex, templateJSON)

	if err = transformer.NormalizeForK8sVMASScalingUp(sc.Logger, templateJSON); err != nil {
		return fmt.Errorf("error transforming the template for scaling template: %+v", err)
	}
	if sc.AgentPool.IsAvailabilitySets() {
		addValue(parametersJSON, fmt.Sprintf("%sOffset", sc.AgentPoolToScale), highestUsedIndex+1)
	}

	_, err = sc.Client.DeployTemplate(
		context.Background(),
		sc.ResourceGroupName,
		fmt.Sprintf("%s-%d", sc.ResourceGroupName, randomDeploymentSuffix()),
		templateJSON,
		parametersJSON)
	if err != nil {
		return fmt.Errorf("error deploying scaled template: %+v", err)
	}

	return saveScaledApimodel(sc, d)
}

func saveScaledApimodel(sc *client.ScaleClient, d *schema.ResourceData) error {
	var err error
	sc.Cluster, err = loadContainerServiceFromApimodel(d, false, true)
	if err != nil {
		return fmt.Errorf("failed to load container service from apimodel: %+v", err)
	}
	sc.Cluster.Properties.AgentPoolProfiles[sc.AgentPoolIndex].Count = sc.DesiredAgentCount

	return saveTemplates(d, sc.Cluster, sc.DeploymentDirectory)
}

func randomDeploymentSuffix() int32 {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	return random.Int31()
}

func setCountForTemplate(sc *client.ScaleClient, highestUsedIndex, currentNodeCount int) int {
	countForTemplate := sc.DesiredAgentCount
	if highestUsedIndex != 0 { // if not scale set
		countForTemplate += highestUsedIndex + 1 - currentNodeCount
	}
	return countForTemplate
}

func setWindowsIndex(sc *client.ScaleClient, windowsIndex int, templateJSON map[string]interface{}) {
	if windowsIndex != -1 {
		templateJSON["variables"].(map[string]interface{})[sc.AgentPool.Name+"Index"] = windowsIndex
	}
}
