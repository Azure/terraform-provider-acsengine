package acsengine

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/acs-engine/pkg/operations/kubernetesupgrade"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client"
	"github.com/hashicorp/terraform/helper/schema"
)

func upgradeCluster(d *schema.ResourceData, upgradeVersion string) error {
	cluster, err := loadContainerServiceFromApimodel(d, true, true)
	if err != nil {
		return fmt.Errorf("error parsing the api model: %+v", err)
	}

	uc := client.NewUpgradeClient()
	if err := uc.SetUpgradeClient(cluster, d.Id(), upgradeVersion); err != nil {
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
