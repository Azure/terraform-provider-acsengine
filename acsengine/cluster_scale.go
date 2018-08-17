package acsengine

import (
	"fmt"
	"log"

	"github.com/Azure/acs-engine/pkg/acsengine/transform"
	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/i18n"
	ops "github.com/Azure/acs-engine/pkg/operations"
	"github.com/Azure/terraform-provider-acsengine/internal/operations"
)

func scaleCluster(d *resourceData, c *ArmClient, agentIndex, agentCount int) error {
	cluster, err := d.loadContainerServiceFromApimodel(true, true)
	if err != nil {
		return fmt.Errorf("error parsing the api model: %+v", err)
	}

	keyVaultSecretRef := cluster.Properties.ServicePrincipalProfile.KeyvaultSecretRef
	clientSecret, err := getSecretFromKeyVault(c, keyVaultSecretRef.VaultID, keyVaultSecretRef.SecretName, "")
	if err != nil {
		return fmt.Errorf("error getting service principal key: %+v", err)
	}

	sc := operations.NewScaleClient(clientSecret)
	if err = sc.SetScaleClient(cluster.ContainerService, d.Id(), agentIndex, agentCount); err != nil {
		return fmt.Errorf("failed to initialize scale client: %+v", err)
	}

	var currentNodeCount, highestUsedIndex, windowsIndex int
	var vms []string
	if sc.AgentPool.IsAvailabilitySets() {
		if highestUsedIndex, currentNodeCount, windowsIndex, vms, err = sc.ScaleVMAS(c.StopContext); err != nil {
			return fmt.Errorf("failed to scale availability set: %+v", err)
		}

		if currentNodeCount == sc.DesiredAgentCount {
			log.Printf("[INFO] Cluster is currently at the desired agent count")
			return nil
		}
		if currentNodeCount > sc.DesiredAgentCount {
			if err = scaleDownCluster(c, sc, currentNodeCount, vms); err != nil {
				return fmt.Errorf("scaling down cluster failed: %+v", err)
			}
			return saveScaledApimodel(d, sc)
		}
	} else {
		if highestUsedIndex, currentNodeCount, windowsIndex, err = sc.ScaleVMSS(c.StopContext); err != nil {
			return fmt.Errorf("failed to scale scale set: %+v", err)
		}
	}

	if err = scaleUpCluster(c, sc, highestUsedIndex, currentNodeCount, windowsIndex); err != nil {
		return fmt.Errorf("scaling cluster failed: %+v", err)
	}
	return saveScaledApimodel(d, sc)
}

func scaleDownCluster(c *ArmClient, sc *operations.ScaleClient, currentNodeCount int, vms []string) error {
	if sc.MasterFQDN == "" {
		return fmt.Errorf("Master FQDN is required to scale down a Kubernetes cluster's agent pool")
	}

	vmsToDelete := vmsToDeleteList(vms, currentNodeCount, sc.DesiredAgentCount)

	cluster := newContainerService(sc.Cluster)
	kubeconfig, err := cluster.getKubeConfig(c, true)
	if err != nil {
		return fmt.Errorf("error getting kube config: %+v", err)
	}
	if err = sc.DrainNodes(kubeconfig, vmsToDelete); err != nil {
		return fmt.Errorf("Got error while draining the nodes to be deleted: %+v", err)
	}

	errList := ops.ScaleDownVMs(
		sc.Client,
		sc.Logger,
		sc.SubscriptionID.String(),
		sc.ResourceGroupName,
		vmsToDelete...)
	if errList != nil {
		errorMessage := ""
		for element := errList.Front(); element != nil; element = element.Next() {
			vmError, ok := element.Value.(*ops.VMScalingErrorDetails)
			if ok {
				error := fmt.Sprintf("Node '%s' failed to delete with error: '%s'", vmError.Name, vmError.Error.Error())
				errorMessage = errorMessage + error
			}
		}
		return fmt.Errorf(errorMessage)
	}

	return nil
}

func scaleUpCluster(c *ArmClient, sc *operations.ScaleClient, highestUsedIndex, currentNodeCount, windowsIndex int) error {
	sc.Cluster.Properties.AgentPoolProfiles = []*api.AgentPoolProfile{sc.AgentPool}

	// don't format parameters! It messes things up
	cluster := newContainerService(sc.Cluster)
	template, parameters, _, err := cluster.formatTemplates(false)
	if err != nil {
		return fmt.Errorf("failed to format templates: %+v", err)
	}

	templateJSON, parametersJSON, err := expandTemplates(template, parameters)
	if err != nil {
		return fmt.Errorf("failed to expand template and parameters: %+v", err)
	}

	transformer := transform.Transformer{
		Translator: &i18n.Translator{
			Locale: sc.Locale,
		},
	}

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
		c.StopContext,
		sc.ResourceGroupName,
		sc.DeploymentName,
		templateJSON,
		parametersJSON)
	if err != nil {
		return fmt.Errorf("error deploying scaled template: %+v", err)
	}
	log.Printf("[INFO] Deployment '%s' successful", sc.DeploymentName)

	return nil
}

func saveScaledApimodel(d *resourceData, sc *operations.ScaleClient) error {
	sc.Cluster.Properties.AgentPoolProfiles[sc.AgentPoolIndex].Count = sc.DesiredAgentCount
	cluster := newContainerService(sc.Cluster)
	return cluster.saveTemplates(d, sc.DeploymentDirectory)
}

func setCountForTemplate(sc *operations.ScaleClient, highestUsedIndex, currentNodeCount int) int {
	countForTemplate := sc.DesiredAgentCount
	if highestUsedIndex != 0 { // if not scale set
		countForTemplate += highestUsedIndex + 1 - currentNodeCount
	}
	return countForTemplate
}

func setWindowsIndex(sc *operations.ScaleClient, windowsIndex int, templateJSON map[string]interface{}) {
	if windowsIndex != -1 {
		templateJSON["variables"].(map[string]interface{})[sc.AgentPool.Name+"Index"] = windowsIndex
	}
}

func vmsToDeleteList(vms []string, currentNodeCount, desiredNodeCount int) []string {
	vmsToDelete := make([]string, 0)
	for i := currentNodeCount - 1; i >= desiredNodeCount; i-- {
		vmsToDelete = append(vmsToDelete, vms[i])
	}
	return vmsToDelete
}
