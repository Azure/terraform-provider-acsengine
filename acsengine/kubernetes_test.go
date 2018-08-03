package acsengine

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
)

func TestACSEngineK8sCluster_validateKubernetesVersion(t *testing.T) {
	cases := []struct {
		Version     string
		ExpectError bool
	}{
		{Version: "1.8.2", ExpectError: false},
		{Version: "3.0.0", ExpectError: true},
		{Version: "1.7.12", ExpectError: false},
		{Version: "181", ExpectError: true},
		{Version: "2.18.3", ExpectError: true},
	}

	for _, tc := range cases {
		_, errors := validateKubernetesVersion(tc.Version, "")
		if !tc.ExpectError && len(errors) > 0 {
			t.Fatalf("Version %s should not have failed", tc.Version)
		}
		if tc.ExpectError && len(errors) == 0 {
			t.Fatalf("Version %s should have failed", tc.Version)
		}
	}
}

func TestACSEngineK8sCluster_getKubeConfig(t *testing.T) {
	// I should have mockContainerService
	name := "cluster"
	location := "southcentralus"
	resourceGroup := "rg"
	prefix := "masterDNSPrefix"
	d := mockClusterResourceData(name, location, resourceGroup, prefix)
	cluster, err := loadContainerServiceFromApimodel(d, true, false)
	if err != nil {
		t.Fatalf("failed to load cluster: %+v", err)
	}

	kubeconfig, err := getKubeConfig(cluster)
	if err != nil {
		t.Fatalf("failed to get kube config: %+v", err)
	}

	if !strings.Contains(kubeconfig, fmt.Sprintf(`"cluster": "%s"`, prefix)) {
		t.Fatalf(fmt.Sprintf(`kubeconfig was not set correctly: does not contain string '"cluster": "%s"'`, prefix))
	}
}

func TestACSEngineK8sCluster_flattenKubeConfig(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenKubeConfig failed")
		}
	}()

	kubeConfigFile := utils.ACSEngineK8sClusterKubeConfig("masterfqdn", "southcentralus")

	_, kubeConfigs, err := flattenKubeConfig(kubeConfigFile)
	if err != nil {
		t.Fatalf("flattenKubeConfig failed: %+v", err)
	}
	if len(kubeConfigs) != 1 {
		t.Fatalf("Incorrect number of kube configs: there are %d kube configs", len(kubeConfigs))
	}
	kubeConfig := kubeConfigs[0].(map[string]interface{})
	v, ok := kubeConfig["cluster_ca_certificate"]
	if !ok {
		t.Fatalf("'cluster_ca_certificate' not found")
	}
	caCert := v.(string)
	if caCert != base64Encode("0123") {
		t.Fatalf("'cluster_ca_certificate' not set correctly: set to %s", caCert)
	}
	if v, ok := kubeConfig["host"]; ok {
		server := v.(string)
		expected := fmt.Sprintf("https://%s.%s.cloudapp.azure.com", "masterfqdn", "southcentralus")
		if server != expected {
			t.Fatalf("Master fqdn is not set correctly: %s != %s", server, expected)
		}
	}
	if v, ok := kubeConfig["username"]; ok {
		user := v.(string)
		expected := fmt.Sprintf("%s-admin", "masterfqdn")
		if user != expected {
			t.Fatalf("Username is not set correctly: %s != %s", user, expected)
		}
	}
}

func TestACSEngineK8sCluster_setKubeConfig(t *testing.T) {
	d := mockClusterResourceData("cluster", "southcentralus", "rg", "prefix")
	// I need a mock container service
	cluster, err := loadContainerServiceFromApimodel(d, true, false)
	if err != nil {
		t.Fatalf("failed to load cluster: %+v", err)
	}

	err = setKubeConfig(d, cluster)
	if err != nil {
		t.Fatalf("failed to set kube config: %+v", err)
	}
}
