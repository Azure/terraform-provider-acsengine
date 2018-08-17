package acsengine

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/acs-engine/pkg/operations/kubernetesupgrade"
	"github.com/Azure/terraform-provider-acsengine/internal/operations"
)

func upgradeCluster(d *resourceData, c *ArmClient, upgradeVersion string) error {
	cluster, err := d.loadContainerServiceFromApimodel(true, true)
	if err != nil {
		return fmt.Errorf("error parsing the api model: %+v", err)
	}

	keyvaultSecretRef := cluster.Properties.ServicePrincipalProfile.KeyvaultSecretRef
	clientSecret, err := getSecretFromKeyVault(c, keyvaultSecretRef.VaultID, keyvaultSecretRef.SecretName, "")
	if err != nil {
		return fmt.Errorf("error getting service principal key: %+v", err)
	}

	uc := operations.NewUpgradeClient(clientSecret)
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

	// cluster.ContainerService = uc.Cluster // I think it's okay to delete this
	kubeconfig, err := cluster.getKubeConfig(c, true)
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

	return cluster.saveTemplates(d, uc.DeploymentDirectory)
}
