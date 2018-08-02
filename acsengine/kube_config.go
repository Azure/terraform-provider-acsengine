package acsengine

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/acsengine"
	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/kubernetes"
)

func getKubeConfig(cluster *api.ContainerService) (string, error) {
	kubeConfig, err := acsengine.GenerateKubeConfig(cluster.Properties, cluster.Location)
	if err != nil {
		return "", fmt.Errorf("failed to generate kube config: %+v", err)
	}
	return kubeConfig, nil
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
