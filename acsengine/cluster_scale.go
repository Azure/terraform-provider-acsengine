package acsengine

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/acs-engine/pkg/acsengine"
	"github.com/Azure/acs-engine/pkg/acsengine/transform"
	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/acs-engine/pkg/operations"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client"
	"github.com/hashicorp/terraform/helper/schema"
)

func scaleCluster(d *schema.ResourceData, agentIndex, agentCount int) error {
	cluster, err := loadContainerServiceFromApimodel(d, true, true)
	if err != nil {
		return fmt.Errorf("error parsing the api model: %+v", err)
	}

	sc := client.NewScaleClient()
	if err = sc.SetScaleClient(cluster, d.Id(), agentIndex, agentCount); err != nil {
		return fmt.Errorf("failed to initialize scale client: %+v", err)
	}

	var currentNodeCount, highestUsedIndex, windowsIndex int
	var indexToVM []string
	if sc.AgentPool.IsAvailabilitySets() {
		if highestUsedIndex, currentNodeCount, windowsIndex, indexToVM, err = sc.ScaleVMAS(); err != nil {
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
		if highestUsedIndex, currentNodeCount, windowsIndex, err = sc.ScaleVMSS(); err != nil {
			return fmt.Errorf("failed to scale scale set: %+v", err)
		}
	}

	return scaleUpCluster(sc, highestUsedIndex, currentNodeCount, windowsIndex, d)
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
		sc.DeploymentName,
		templateJSON,
		parametersJSON)
	if err != nil {
		return fmt.Errorf("error deploying scaled template: %+v", err)
	}
	// I could get deployment here, check for existence and get new ID

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
