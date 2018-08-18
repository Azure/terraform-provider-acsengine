package acsengine

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/acsengine"
	"github.com/Azure/acs-engine/pkg/api/common"
	"github.com/Azure/terraform-provider-acsengine/internal/kubernetes"
	"github.com/hashicorp/terraform/helper/schema"
)

func kubernetesVersionSchema() *schema.Schema {
	return &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Default:      common.GetDefaultKubernetesVersion(), // default is 1.8.13
		ValidateFunc: validateKubernetesVersion,
	}
}

func kubernetesVersionForDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
}

func kubeConfigRawSchema() *schema.Schema {
	return &schema.Schema{
		Type:      schema.TypeString,
		Computed:  true,
		Sensitive: true,
	}
}

func validateKubernetesVersion(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	capacities := common.AllKubernetesSupportedVersions

	if !capacities[value] {
		errors = append(errors, fmt.Errorf("ACS Engine Kubernetes Cluster: Kubernetes version %s is invalid or not supported", value))
	}
	return
}

func flattenKubeConfig(kubeConfigFile string) (string, []interface{}, error) {
	rawKubeConfig := base64Encode(kubeConfigFile)

	config, err := kubernetes.ParseKubeConfig(kubeConfigFile)
	if err != nil {
		return "", nil, fmt.Errorf("error parsing kube config: %+v", err)
	}

	kubeConfig := []interface{}{}
	cluster := config.Clusters[0].Cluster
	user := config.Users[0].User
	name := config.Users[0].Name

	values := map[string]interface{}{}
	values["host"] = cluster.Server
	values["username"] = name
	values["password"] = user.Token
	values["client_certificate"] = base64Encode(user.ClientCertificteData)
	values["client_key"] = base64Encode(user.ClientKeyData)
	values["cluster_ca_certificate"] = base64Encode(cluster.ClusterAuthorityData)

	kubeConfig = append(kubeConfig, values)

	return rawKubeConfig, kubeConfig, nil
}

func (cluster *containerService) getKubeConfig(c *ArmClient, keyVault bool) (string, error) {
	if keyVault {
		if err := getCertificateProfileSecretsFromKeyVault(c, cluster); err != nil {
			return "", fmt.Errorf("failed to get secrets from key vault for kube config: %+v", err)
		}
	}
	kubeConfig, err := acsengine.GenerateKubeConfig(cluster.Properties, cluster.Location)
	if err != nil {
		return "", fmt.Errorf("failed to generate kube config: %+v", err)
	}
	if keyVault {
		if err := cluster.setCertificateProfileSecretsAPIModel(); err != nil {
			return "", fmt.Errorf("failed to set certificates back to key vault reference: %+v", err)
		}
	}
	return kubeConfig, nil
}

func (d *resourceData) setKubeConfig(c *ArmClient, cluster *containerService, keyVault bool) error {
	kubeConfigFile, err := cluster.getKubeConfig(c, keyVault)
	if err != nil {
		return fmt.Errorf("Error getting kube config: %+v", err)
	}
	kubeConfigRaw, kubeConfig, err := flattenKubeConfig(kubeConfigFile)
	if err != nil {
		return fmt.Errorf("Error flattening kube config: %+v", err)
	}
	if err = d.Set("kube_config_raw", kubeConfigRaw); err != nil {
		return fmt.Errorf("Error setting `kube_config_raw`: %+v", err)
	}
	if err = d.Set("kube_config", kubeConfig); err != nil {
		return fmt.Errorf("Error setting `kube_config`: %+v", err)
	}

	return nil
}
