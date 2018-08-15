package acsengine

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/acs-engine/pkg/operations/kubernetesupgrade"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client"
)

func upgradeCluster(d *ResourceData, c *ArmClient, upgradeVersion string) error {
	cluster, err := d.loadContainerServiceFromApimodel(true, true)
	if err != nil {
		return fmt.Errorf("error parsing the api model: %+v", err)
	}

	keyvaultSecretRef := cluster.Properties.ServicePrincipalProfile.KeyvaultSecretRef
	clientSecret, err := getSecret(c, keyvaultSecretRef.VaultID, keyvaultSecretRef.SecretName, "")
	if err != nil {
		return fmt.Errorf("error getting service principal key: %+v", err)
	}

	uc := client.NewUpgradeClient(clientSecret)
	if err := uc.SetUpgradeClient(cluster.ContainerService, d.Id(), upgradeVersion); err != nil {
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

	cluster.ContainerService = uc.Cluster
	kubeconfig, err := cluster.getKubeConfig()
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

	cluster.ContainerService = uc.Cluster
	return cluster.saveTemplates(d, uc.DeploymentDirectory)
}
