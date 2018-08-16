package acsengine

import (
	"encoding/base64"
	"fmt"
	"log"
	"testing"

	"github.com/Azure/terraform-provider-acsengine/internal/resource"
	"github.com/Azure/terraform-provider-acsengine/internal/utils"
	"github.com/hashicorp/terraform/terraform"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
)

func TestValidateKubernetesVersion(t *testing.T) {
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

func TestGetKubeConfig(t *testing.T) {
	name := "cluster"
	location := "southcentralus"
	prefix := "masterDNSPrefix"
	cluster := mockCluster(name, location, prefix)

	kubeconfig, err := cluster.getKubeConfig(nil, false)
	if err != nil {
		t.Fatalf("failed to get kube config: %+v", err)
	}

	assert.Contains(t, kubeconfig, fmt.Sprintf(`"cluster": "%s"`, prefix), "kubeconfig was not set correctly")
}

func TestFlattenKubeConfig(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenKubeConfig failed")
		}
	}()

	kubeConfigFile := resource.ACSEngineK8sClusterKubeConfig("masterfqdn", "southcentralus")

	_, kubeConfigs, err := flattenKubeConfig(kubeConfigFile)
	if err != nil {
		t.Fatalf("flattenKubeConfig failed: %+v", err)
	}
	assert.Equal(t, 1, len(kubeConfigs), "incorrect number of kube configs")
	kubeConfig := kubeConfigs[0].(map[string]interface{})
	v, ok := kubeConfig["cluster_ca_certificate"]
	assert.True(t, ok, "'cluster_ca_certificate' not found")
	caCert := v.(string)
	assert.Equal(t, base64Encode("0123"), caCert, "'cluster_ca_certificate' not set correctly")
	if v, ok := kubeConfig["host"]; ok {
		server := v.(string)
		expected := fmt.Sprintf("https://%s.%s.cloudapp.azure.com", "masterfqdn", "southcentralus")
		assert.Equal(t, expected, server, "master fqdn is not set correctly")
	}
	if v, ok := kubeConfig["username"]; ok {
		user := v.(string)
		expected := fmt.Sprintf("%s-admin", "masterfqdn")
		assert.Equal(t, expected, user, "username is not set correctly")
	}
}

func TestFlattenInvalidKubeConfig(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenKubeConfig failed")
		}
	}()

	kubeConfigFile := ""

	_, _, err := flattenKubeConfig(kubeConfigFile)
	if err == nil {
		t.Fatalf("flattenKubeConfig should have failed")
	}
}

func TestSetKubeConfig(t *testing.T) {
	d := mockClusterResourceData("cluster", "southcentralus", "rg", "prefix")
	// I need a mock container service
	cluster, err := d.loadContainerServiceFromApimodel(true, false)
	if err != nil {
		t.Fatalf("failed to load cluster: %+v", err)
	}

	if err = d.setKubeConfig(nil, &cluster, false); err != nil {
		t.Fatalf("failed to set kube config: %+v", err)
	}
}

// clusterIsRunning is a helper function for testCheckACSEngineClusterExists
func clusterIsRunning(is *terraform.InstanceState, name string) error {
	key := "kube_config_raw"
	var config []byte
	var err error
	v, ok := is.Attributes[key]
	if !ok {
		return fmt.Errorf("%s: Attribute '%s' not found", name, key)
	}
	config, err = base64.StdEncoding.DecodeString(v)
	if err != nil {
		return fmt.Errorf("kube config could not be decoded from base64: %+v", err)
	}

	kubeConfig, _ /*namespace*/, err := newClientConfigFromBytes(config)
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("Could not get Kubernetes client: %+v", err)
	}

	api := clientset.CoreV1()

	if err := checkNodes(api); err != nil {
		return fmt.Errorf("checking nodes failed: %+v", err)
	}

	return nil
}

func checkNodes(api corev1.CoreV1Interface) error {
	retryErr := utils.RetryOnFailure(retry.DefaultRetry, func() error {
		log.Println("[INFO] trying to get nodes...") // log
		nodes, err := api.Nodes().List(metav1.ListOptions{})
		if err != nil {
			log.Printf("[INFO] Reason for error: %+v\n", errors.ReasonForError(err))
			return fmt.Errorf("failed to get nodes: %+v", err)
		}
		if len(nodes.Items) < 2 {
			return fmt.Errorf("not enough nodes found (there should be a at least one master and agent pool): only %d found", len(nodes.Items))
		}
		for _, node := range nodes.Items {
			log.Printf("[INFO] Node: %s\n", node.Name) // log
		}
		return nil
	})
	if retryErr != nil {
		return fmt.Errorf("Failed to get nodes: %+v", retryErr)
	}

	return nil
}

func checkVersion(api corev1.CoreV1Interface) error {
	// can I use apimachinery package to get kubernetes version?
	return nil
}

// Gets a client config and namespace, based on function in aks e2e tests
func newClientConfigFromBytes(configBytes []byte) (*rest.Config, string, error) { // I need kubeconfig
	config, err := clientcmd.Load(configBytes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load kube config from bytes: %+v", err)
	}

	conf := clientcmd.NewNonInteractiveClientConfig(*config, "", &clientcmd.ConfigOverrides{}, nil)

	namespace, _, err := conf.Namespace()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get cluster namespace: %+v", err)
	}

	cc, err := conf.ClientConfig()
	if err != nil {
		return nil, "", fmt.Errorf("failed to : %+v", err)
	}

	return cc, namespace, nil
}
