package acsengine

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/hashicorp/terraform/terraform"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	// nodeutil "k8s.io/kubernetes/pkg/api/v1/node"
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

// clusterIsRunning is a helper function for testCheckACSEngineClusterExists
func clusterIsRunning(is *terraform.InstanceState, name string) error {
	// get kube config
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

	// get kubernetes client
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
		fmt.Println("trying to get nodes...")
		nodes, err := api.Nodes().List(metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Reason for error: %+v\n", errors.ReasonForError(err))
			return fmt.Errorf("failed to get nodes: %+v", err)
		}
		if len(nodes.Items) < 2 {
			return fmt.Errorf("not enough nodes found (there should be a at least one master and agent pool): only %d found", len(nodes.Items))
		}
		for _, node := range nodes.Items {
			fmt.Printf("Node: %s\n", node.Name)
			// can I use apimachinery package to get kubernetes version?
			// if !nodeutil.IsNodeReady(&node) { // default eviction time is 5m, so it would probably need to be 5m timeout?
			// 	return fmt.Errorf("node is not ready: %+v", node) // do I need to not return here? continue instead?
			// }
			// maybe I can check the node condition and at least see that it's running? That's not a condition that can be checked...
			// fmt.Println("node condition: %+v", nodeutil.GetNodeCondition(&node))
		}
		return nil
	})
	if retryErr != nil {
		return fmt.Errorf("Failed to get nodes: %+v", retryErr)
	}

	return nil
}

func checkVersion(api corev1.CoreV1Interface) error {
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
