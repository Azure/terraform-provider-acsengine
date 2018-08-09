package acsengine

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/acs-engine/pkg/operations/kubernetesupgrade"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client"
	"github.com/hashicorp/terraform/helper/schema"
)

// Upgrades a cluster to a higher Kubernetes version
func upgradeCluster(d *schema.ResourceData, upgradeVersion string) error {
	uc, err := setUpgradeClient(d, upgradeVersion)
	if err != nil {
		return fmt.Errorf("error initializing upgrade client: %+v", err)
	}

	upgradeCluster := kubernetesupgrade.UpgradeCluster{
		Translator: &i18n.Translator{
			Locale: uc.Locale,
		},
		Logger:      uc.Logger,
		Client:      uc.Client,
		StepTimeout: uc.Timeout,
	}

	kubeconfig, err := getKubeConfig(uc.Cluster)
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

	return saveTemplates(d, uc.Cluster, uc.DeploymentDirectory)
}

func setUpgradeClient(d *schema.ResourceData, upgradeVersion string) (*client.UpgradeClient, error) {
	uc := client.NewUpgradeClient()

	if err := setACSEngineClient(d, &uc.ACSEngineClient); err != nil {
		return uc, fmt.Errorf("failed to initialize ACSEngineClient: %+v", err)
	}

	uc.UpgradeVersion = upgradeVersion
	uc.TimeoutInMinutes = -1
	if err := uc.Validate(); err != nil {
		return uc, fmt.Errorf(": %+v", err)
	}

	uc.Cluster.Properties.OrchestratorProfile.OrchestratorVersion = uc.UpgradeVersion

	uc.AgentPoolsToUpgrade = []string{}
	for _, agentPool := range uc.Cluster.Properties.AgentPoolProfiles {
		uc.AgentPoolsToUpgrade = append(uc.AgentPoolsToUpgrade, agentPool.Name)
	}

	return uc, nil
}
