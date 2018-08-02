package acsengine

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/acsengine"
	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/acs-engine/pkg/operations/kubernetesupgrade"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client"
	"github.com/hashicorp/terraform/helper/schema"
)

// Upgrades a cluster to a higher Kubernetes version
func upgradeCluster(d *schema.ResourceData, m interface{}, upgradeVersion string) error {
	uc, err := initializeUpgradeClient(d, m, upgradeVersion)
	if err != nil {
		return fmt.Errorf("error initializing upgrade client: %+v", err)
	}

	// I already validated elsewhere, consider deleting
	kubernetesInfo, err := api.GetOrchestratorVersionProfile(uc.Cluster.Properties.OrchestratorProfile)
	if err != nil {
		return fmt.Errorf("error getting a list of the available upgrades: %+v", err)
	}
	found := false
	for _, up := range kubernetesInfo.Upgrades { // checking that version I want is within the allowed versions
		if up.OrchestratorVersion == uc.UpgradeVersion {
			uc.Cluster.Properties.OrchestratorProfile.OrchestratorVersion = uc.UpgradeVersion
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("version %s is not supported", uc.UpgradeVersion)
	}

	uc.AgentPoolsToUpgrade = []string{}
	for _, agentPool := range uc.Cluster.Properties.AgentPoolProfiles {
		uc.AgentPoolsToUpgrade = append(uc.AgentPoolsToUpgrade, agentPool.Name)
	}

	upgradeCluster := kubernetesupgrade.UpgradeCluster{
		Translator: &i18n.Translator{
			Locale: uc.Locale,
		},
		Logger:      uc.Logger,
		Client:      uc.Client,
		StepTimeout: uc.Timeout,
	}

	kubeconfig, err := acsengine.GenerateKubeConfig(uc.Cluster.Properties, uc.Location)
	if err != nil {
		return fmt.Errorf("failed to generate kube config: %+v", err)
	}

	err = upgradeCluster.UpgradeCluster(
		uc.SubscriptionID,
		kubeconfig,
		uc.ResourceGroupName,
		uc.Cluster,
		uc.NameSuffix,
		uc.AgentPoolsToUpgrade,
		acsEngineVersion)
	if err != nil {
		return fmt.Errorf("failed to deploy upgraded cluster: %+v", err)
	}

	return saveUpgradedApimodel(&uc, d)
}

func initializeUpgradeClient(d *schema.ResourceData, m interface{}, upgradeVersion string) (client.UpgradeClient, error) {
	uc := client.UpgradeClient{}

	err := initializeACSEngineClient(d, m, &uc.ACSEngineClient)
	if err != nil {
		return uc, fmt.Errorf("failed to initialize ACSEngineClient: %+v", err)
	}

	uc.UpgradeVersion = upgradeVersion
	uc.TimeoutInMinutes = -1
	err = uc.Validate()
	if err != nil {
		return uc, fmt.Errorf(": %+v", err)
	}

	return uc, nil
}

func saveUpgradedApimodel(uc *client.UpgradeClient, d *schema.ResourceData) error {
	return saveTemplates(d, uc.Cluster, uc.DeploymentDirectory)
}
