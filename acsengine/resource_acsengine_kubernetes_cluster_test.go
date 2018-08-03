package acsengine

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
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

/* TESTS */

/* UNIT TESTS */

func TestACSEngineK8sCluster_initializeContainerService(t *testing.T) {
	name := "testcluster"
	location := "southcentralus"
	resourceGroup := "testrg"
	masterDNSPrefix := "creativeMasterDNSPrefix"

	d := mockClusterResourceData(name, location, resourceGroup, masterDNSPrefix)

	cluster, err := initializeContainerService(d)
	if err != nil {
		t.Fatalf("initializeContainerService failed: %+v", err)
	}

	if cluster.Name != "testcluster" {
		t.Fatalf("cluster name was not set correctly: was %s but should be testcluster", cluster.Name)
	}
	version := cluster.Properties.OrchestratorProfile.OrchestratorVersion
	if version != "1.10.0" {
		t.Fatalf("cluster Kubernetes version was not set correctly: was '%s' but it should be '1.10.0'", version)
	}
	dnsPrefix := cluster.Properties.MasterProfile.DNSPrefix
	if dnsPrefix != masterDNSPrefix {
		t.Fatalf("master DNS prefix was not set correctly: was %s but it should be 'masterDNSPrefix'", dnsPrefix)
	}
	if cluster.Properties.AgentPoolProfiles[0].Count != 1 {
		t.Fatalf("agent pool profile is not set correctly")
	}
}

func TestACSEngineK8sCluster_loadContainerServiceFromApimodel(t *testing.T) {
	name := "testcluster"
	location := "southcentralus"

	d := mockClusterResourceData(name, location, "testrg", "creativeMasterDNSPrefix") // I need to add a test apimodel in here

	apimodel, err := loadContainerServiceFromApimodel(d, true, false)
	if err != nil {
		t.Fatalf("failed to load container service from api model: %+v", err)
	}

	if apimodel.Name != name {
		t.Fatalf("cluster name '%s' not found", name)
	}
	if apimodel.Location != location {
		t.Fatalf("cluster location '%s' not found", location)
	}
}

func TestACSEngineK8sCluster_parseImportID(t *testing.T) {
	azureIDInput := "/subscriptions/1234/resourceGroups/testrg/providers/Microsoft.Resources/deployments/deploymentName"
	deploymentDirectoryInput := "_output/dnsPrefix"
	id := fmt.Sprintf("%s %s", azureIDInput, deploymentDirectoryInput)

	azureID, deploymentDirectory, err := parseImportID(id)
	if err != nil {
		t.Fatalf("parseImportID failed: %+v", err)
	}

	if azureID != azureIDInput {
		t.Fatalf("parseImportID failed: azureID was %s but expected %s", azureID, azureIDInput)
	}
	if deploymentDirectory != deploymentDirectoryInput {
		t.Fatalf("parseImportID failed: deploymentDirectory was %s but expected %s", deploymentDirectory, deploymentDirectoryInput)
	}

	_, err = parseAzureResourceID(azureID)
	if err != nil {
		t.Fatalf("failed to parse azureID: %+v", err)
	}
}

/* ACCEPTANCE TESTS */

// made it an acceptance test more because of the time it takes
func TestAccACSEngineK8sCluster_generateTemplateBasic(t *testing.T) {
	cases := []struct {
		Name           string
		ResourceGroup  string
		Location       string
		AdminUsername  string
		MasterCount    int
		DNSPrefix      string
		AgentPoolCount int
		ExpectError    bool
	}{
		{Name: "cluster1", ResourceGroup: "RG1", Location: "southcentralus", AdminUsername: "azureuser1",
			MasterCount: 1, DNSPrefix: "laughingAlligator", AgentPoolCount: 1, ExpectError: false},
		{Name: "cluster2", ResourceGroup: "RG2", Location: "eastus", AdminUsername: "azureuser2",
			MasterCount: 1, DNSPrefix: "dancingEmu", AgentPoolCount: 2, ExpectError: false},
		{Name: "cluster2", ResourceGroup: "RG3", Location: "westeurope", AdminUsername: "azureuser3",
			MasterCount: 1, DNSPrefix: "jumpingJabberwock", AgentPoolCount: 10, ExpectError: false},
	}

	r := resourceArmAcsEngineKubernetesCluster()
	d := r.TestResourceData()

	for _, tc := range cases {
		d.Set("name", tc.Name)
		d.Set("location", tc.Location)
		d.Set("resource_group", tc.ResourceGroup)

		linuxProfiles := testFlattenLinuxProfile(tc.AdminUsername)
		d.Set("linux_profile", &linuxProfiles)

		servicePrincipals := testFlattenServicePrincipal()
		d.Set("service_principal", servicePrincipals)

		vmSize := "Standard_D2_v2"
		masterProfiles := testFlattenMasterProfile(tc.MasterCount, tc.DNSPrefix, vmSize)
		d.Set("master_profile", &masterProfiles)

		agentPoolProfiles := []interface{}{}
		agentPoolName := "agentpool0"
		agentPoolProfile0 := testFlattenAgentPoolProfiles(agentPoolName, tc.AgentPoolCount, vmSize, 0, false)
		agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile0)
		agentPoolName = "agentpool1"
		agentPoolProfile1 := testFlattenAgentPoolProfiles(agentPoolName, tc.AgentPoolCount+1, vmSize, 0, false)
		agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile1)
		d.Set("agent_pool_profiles", &agentPoolProfiles)

		template, parameters, err := generateACSEngineTemplate(d, false) // don't write files
		if err != nil {
			t.Fatalf("Template generation failed: %v", err)
		}

		// now I can test that the template and parameters look okay I guess...
		if !strings.Contains(parameters, tc.AdminUsername) {
			t.Fatalf("Expected the Azure RM Kubernetes cluster to have parameter '%s'", tc.AdminUsername)
		}
		if !strings.Contains(parameters, testClientID()) {
			t.Fatalf("Expected the Azure RM Kubernetes cluster to have parameter '%s'", testClientID())
		}
		if !strings.Contains(parameters, vmSize) {
			t.Fatalf("Expected the Azure RM Kubernetes cluster to have parameter '%s'", vmSize)
		}
		if !strings.Contains(parameters, strconv.Itoa(tc.AgentPoolCount)) {
			t.Fatalf("Expected the Azure RM Kubernetes cluster to have parameter '%d'", tc.AgentPoolCount)
		}

		if !strings.Contains(template, agentPoolName+"Count") {
			t.Fatalf("Expected the Azure RM Kubernetes cluster to have field '%s'", agentPoolName+"Count")
		}

		if tc.ExpectError {
			t.Fatalf("Expected the Kubernetes Cluster Agent Pool Name to trigger an error for '%s'", tc.Name)
		}
	}
}

func TestAccACSEngineK8sCluster_generateTemplateCustomized(t *testing.T) {
	cases := []struct {
		Name           string
		ResourceGroup  string
		Location       string
		Version        string
		AdminUsername  string
		MasterCount    int
		MasterVMSize   string
		DNSPrefix      string
		AgentPoolCount int
		AgentVMSize    string
		ExpectError    bool
	}{
		{Name: "cluster1", ResourceGroup: "RG1", Location: "southcentralus", Version: "", AdminUsername: "azureuser1", MasterCount: 1,
			MasterVMSize: "", DNSPrefix: "laughingAlligator", AgentPoolCount: 1, AgentVMSize: "", ExpectError: false},
		{Name: "cluster2", ResourceGroup: "RG2", Location: "eastus", Version: "", AdminUsername: "azureuser2", MasterCount: 3,
			MasterVMSize: "", DNSPrefix: "dancingEmu", AgentPoolCount: 14, AgentVMSize: "", ExpectError: false},
		{Name: "cluster2", ResourceGroup: "RG3", Location: "westeurope", Version: "", AdminUsername: "azureuser3", MasterCount: 5,
			MasterVMSize: "", DNSPrefix: "jumpingJabberwock", AgentPoolCount: 50, AgentVMSize: "", ExpectError: false},
	}

	r := resourceArmAcsEngineKubernetesCluster()
	d := r.TestResourceData()

	for _, tc := range cases {
		d.Set("name", tc.Name)
		d.Set("location", tc.Location)
		d.Set("resource_group", tc.ResourceGroup)
		d.Set("kubernetes_version", tc.Version)

		linuxProfiles := testFlattenLinuxProfile(tc.AdminUsername)
		d.Set("linux_profile", &linuxProfiles)

		servicePrincipals := testFlattenServicePrincipal()
		d.Set("service_principal", servicePrincipals)

		masterProfiles := testFlattenMasterProfile(tc.MasterCount, tc.DNSPrefix, tc.MasterVMSize)
		d.Set("master_profile", &masterProfiles)

		agentPoolProfiles := []interface{}{}
		agentPoolName := "agentpool0"
		agentPoolProfile0 := testFlattenAgentPoolProfiles(agentPoolName, tc.AgentPoolCount, tc.AgentVMSize, 0, false)
		agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile0)
		agentPoolName = "agentpool1"
		agentPoolProfile1 := testFlattenAgentPoolProfiles(agentPoolName, tc.AgentPoolCount+1, tc.AgentVMSize, 0, false)
		agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile1)
		d.Set("agent_pool_profiles", &agentPoolProfiles)

		template, parameters, err := generateACSEngineTemplate(d, false) // don't write files
		if err != nil {
			t.Fatalf("Template generation failed: %v", err)
		}

		if !strings.Contains(parameters, tc.AdminUsername) {
			t.Fatalf("Expected the Azure RM Kubernetes cluster to have parameter '%s'", tc.AdminUsername)
		}
		if !strings.Contains(parameters, testClientID()) {
			t.Fatalf("Expected the Azure RM Kubernetes cluster to have parameter '%s'", testClientID())
		}
		if !strings.Contains(parameters, tc.MasterVMSize) {
			t.Fatalf("Expected the Azure RM Kubernetes cluster to have parameter '%s'", tc.MasterVMSize)
		}
		if !strings.Contains(parameters, tc.AgentVMSize) {
			t.Fatalf("Expected the Azure RM Kubernetes cluster to have parameter '%s'", tc.AgentVMSize)
		}
		if !strings.Contains(parameters, strconv.Itoa(tc.AgentPoolCount)) {
			t.Fatalf("Expected the Azure RM Kubernetes cluster to have parameter '%d'", tc.AgentPoolCount)
		}

		if !strings.Contains(template, agentPoolName+"Count") {
			t.Fatalf("Expected the Azure RM Kubernetes cluster to have field '%s'", agentPoolName+"Count")
		}

		if tc.ExpectError {
			t.Fatalf("Expected the Kubernetes Cluster Agent Pool Name to trigger an error for '%s'", tc.Name)
		}
	}
}

func TestAccACSEngineK8sCluster_initializeScaleClient(t *testing.T) {
	r := resourceArmAcsEngineKubernetesCluster()
	d := r.TestResourceData()

	resourceGroupName := "clusterResourceGroup"
	masterDNSPrefix := "masterDNSPrefix"
	d.Set("name", "clusterName")
	d.Set("location", "southcentralus")
	d.Set("resource_group", resourceGroupName)
	id := "/subscriptions/" + os.Getenv("ARM_SUBSCRIPTION_ID") + "/resourceGroups/" + resourceGroupName + "/providers/Microsoft.Resources/deployments/" + masterDNSPrefix
	d.SetId(id)

	linuxProfiles := testFlattenLinuxProfile("azureuser")
	d.Set("linux_profile", &linuxProfiles)

	servicePrincipals := testFlattenServicePrincipal()
	d.Set("service_principal", servicePrincipals)

	masterProfiles := testFlattenMasterProfile(1, "masterDNSPrefix", "Standard_D2_v2")
	d.Set("master_profile", &masterProfiles)

	agentPoolProfiles := []interface{}{}
	agentPoolProfile0 := testFlattenAgentPoolProfiles("agentpool1", 1, "Standard_D2_v2", 0, false)
	agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile0)
	agentPoolProfile1 := testFlattenAgentPoolProfiles("agentpool2", 2, "Standard_D2_v2", 0, false)
	agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile1)
	d.Set("agent_pool_profiles", &agentPoolProfiles)

	// create and delete file for testing
	apimodelPath := "_output/k8scluster" // this isn't accurate anymore
	_, _, err := generateACSEngineTemplate(d, true)
	if err != nil {
		t.Fatalf("GenerateACSEngineTemplate failed: %+v", err)
	}
	defer func() {
		err = os.RemoveAll(apimodelPath)
		if err != nil {
			t.Fatalf("Could not remove apimodel.json")
		}
	}()

	agentIndex := 0
	desiredAgentCount := 2
	sc, err := initializeScaleClient(d, nil, agentIndex, desiredAgentCount)
	if err != nil {
		t.Fatalf("initializeScaleClient failed: %+v", err)
	}

	if sc.ResourceGroupName != resourceGroupName {
		t.Fatalf("Resource group is not named correctly")
	}
	if sc.DesiredAgentCount != desiredAgentCount {
		t.Fatalf("Desired agent count is not set correctly")
	}
	profile := servicePrincipals[0].(map[string]interface{})
	if sc.AuthArgs.RawClientID != profile["client_id"] {
		t.Fatalf("Client ID not set correctly")
	}
	if sc.AuthArgs.SubscriptionID.String() != os.Getenv("ARM_SUBSCRIPTION_ID") {
		t.Fatalf("Subscription ID is not set correctly")
	}
}

// very similar to initializeScaleClient test, get rid of duplicate code (with mock ResourceData function?)
func TestAccACSEngineK8sCluster_initializeUpgradeClient(t *testing.T) {
	r := resourceArmAcsEngineKubernetesCluster()
	d := r.TestResourceData()

	resourceGroupName := "clusterResourceGroup"
	masterDNSPrefix := "masterDNSPrefix"
	d.Set("name", "clusterName")
	d.Set("location", "southcentralus")
	d.Set("resource_group", resourceGroupName)
	id := "/subscriptions/" + os.Getenv("ARM_SUBSCRIPTION_ID") + "/resourceGroups/" + resourceGroupName + "/providers/Microsoft.Resources/deployments/" + masterDNSPrefix
	d.SetId(id)

	linuxProfiles := testFlattenLinuxProfile("azureuser")
	d.Set("linux_profile", &linuxProfiles)

	servicePrincipals := testFlattenServicePrincipal()
	d.Set("service_principal", servicePrincipals)

	masterProfiles := testFlattenMasterProfile(1, "masterDNSPrefix", "Standard_D2_v2")
	d.Set("master_profile", &masterProfiles)

	agentPoolProfiles := []interface{}{}
	agentPoolProfile0 := testFlattenAgentPoolProfiles("agentpool1", 1, "Standard_D2_v2", 0, false)
	agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile0)
	agentPoolProfile1 := testFlattenAgentPoolProfiles("agentpool2", 2, "Standard_D2_v2", 0, false)
	agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile1)
	d.Set("agent_pool_profiles", &agentPoolProfiles)

	// create and delete file for testing
	apimodelPath := "_output/k8scluster" // this isn't accurate anymore
	_, _, err := generateACSEngineTemplate(d, true)
	if err != nil {
		t.Fatalf("GenerateACSEngineTemplate failed: %+v", err)
	}
	defer func() {
		err = os.RemoveAll(apimodelPath)
		if err != nil {
			t.Fatalf("Could not remove apimodel.json")
		}
	}()

	upgradeVersion := "1.9.8"
	uc, err := initializeUpgradeClient(d, nil, upgradeVersion)
	if err != nil {
		t.Fatalf("initializeScaleClient failed: %+v", err)
	}

	if uc.ResourceGroupName != resourceGroupName {
		t.Fatalf("Resource group is not named correctly")
	}
	if uc.UpgradeVersion != upgradeVersion {
		t.Fatalf("Desired agent count is not set correctly")
	}
	profile := servicePrincipals[0].(map[string]interface{})
	if uc.AuthArgs.RawClientID != profile["client_id"] {
		t.Fatalf("Client ID not set correctly")
	}
	if uc.AuthArgs.SubscriptionID.String() != os.Getenv("ARM_SUBSCRIPTION_ID") {
		t.Fatalf("Subscription ID is not set correctly")
	}
}

func TestAccACSEngineK8sCluster_createBasic(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	config := testAccACSEngineK8sClusterBasic(ri, clientID, clientSecret, location, keyData)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "resource_group", "acctestRG-"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "location", location),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_id", clientID),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_secret", clientSecret),
					resource.TestCheckResourceAttr(tfResourceName, "linux_profile.0.admin_username", "acctestuser"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.dns_name_prefix", "acctestmaster"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.fqdn", "acctestmaster"+strconv.Itoa(ri)+"."+location+".cloudapp.azure.com"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.name", "agentpool1"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_createMultipleAgentPools(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	config := testAccACSEngineK8sClusterMultipleAgentPools(ri, clientID, clientSecret, location, keyData)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "resource_group", "acctestRG-"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "location", location),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_id", clientID),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_secret", clientSecret),
					resource.TestCheckResourceAttr(tfResourceName, "linux_profile.0.admin_username", "acctestuser"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.dns_name_prefix", "acctestmaster"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.fqdn", "acctestmaster"+strconv.Itoa(ri)+"."+location+".cloudapp.azure.com"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.name", "agentpool1"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.1.name", "agentpool2"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_createCustomized(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	version := "1.9.8"
	vmSize := "Standard_D2_v2" // change
	agentCount := 1
	osDiskSizeGB := 40
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, version, agentCount, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "resource_group", "acctestRG-"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "location", location),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_id", clientID),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_secret", clientSecret),
					resource.TestCheckResourceAttr(tfResourceName, "linux_profile.0.admin_username", "acctestuser"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.dns_name_prefix", "acctestmaster"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.vm_size", vmSize),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", version),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.name", "agentpool1"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.vm_size", vmSize),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_createVersion10AndAbove(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	version := "1.10.0"
	vmSize := "Standard_D2_v2"
	agentCount := 1
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, version, agentCount, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "resource_group", "acctestRG-"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "location", location),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_id", clientID),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_secret", clientSecret),
					resource.TestCheckResourceAttr(tfResourceName, "linux_profile.0.admin_username", "acctestuser"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.dns_name_prefix", "acctestmaster"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.vm_size", vmSize),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", version),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.name", "agentpool1"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.vm_size", vmSize),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_scaleUp(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	config := testAccACSEngineK8sClusterScale(ri, clientID, clientSecret, location, keyData, 1)
	updatedConfig := testAccACSEngineK8sClusterScale(ri, clientID, clientSecret, location, keyData, 2)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
		},
	})

}

func TestAccACSEngineK8sCluster_scaleDown(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	config := testAccACSEngineK8sClusterScale(ri, clientID, clientSecret, location, keyData, 2)
	updatedConfig := testAccACSEngineK8sClusterScale(ri, clientID, clientSecret, location, keyData, 1)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_scaleUpDown(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	config := testAccACSEngineK8sClusterScale(ri, clientID, clientSecret, location, keyData, 1)
	scaledUpConfig := testAccACSEngineK8sClusterScale(ri, clientID, clientSecret, location, keyData, 2)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: scaledUpConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_scaleDownUp(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	config := testAccACSEngineK8sClusterScale(ri, clientID, clientSecret, location, keyData, 2)
	scaledDownConfig := testAccACSEngineK8sClusterScale(ri, clientID, clientSecret, location, keyData, 1)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: scaledDownConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
		},
	})
}

// how can I test that cluster wasn't recreated instead of updated?
func TestAccACSEngineK8sCluster_upgradeOnce(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	upgradedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 1, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
				),
			},
			{
				Config: upgradedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_upgradeMultiple(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	upgradedConfig1 := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 1, vmSize, osDiskSizeGB)
	upgradedConfig2 := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.10.0", 1, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
				),
			},
			{
				Config: upgradedConfig1,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
				),
			},
			{
				Config: upgradedConfig2,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.10.0"),
				),
			},
		},
	})
}

// I need to look into what the expected behavior is, and if this is always a scale sets above a certain version
// also test below certain version upgraded to above, followed by scaling
func TestAccACSEngineK8sCluster_upgradeVersion10AndAbove(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.10.0", 1, vmSize, osDiskSizeGB)
	upgradedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.10.1", 1, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.10.0"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: upgradedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.10.1"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_updateUpgradeScaleUp(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	upgradedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 1, vmSize, osDiskSizeGB)
	scaledConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 2, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: upgradedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: scaledConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_updateScaleUpUpgrade(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	scaledConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 2, vmSize, osDiskSizeGB)
	upgradedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 2, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: scaledConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: upgradedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_updateUpgradeScaleDown(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 2, vmSize, osDiskSizeGB)
	upgradedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 2, vmSize, osDiskSizeGB)
	scaledConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 1, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: upgradedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: scaledConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_updateScaleDownUpgrade(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 2, vmSize, osDiskSizeGB)
	scaledConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	upgradedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 1, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: scaledConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: upgradedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_updateScaleUpgradeInOne(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	updatedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 2, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
		},
	})

}

// can I somehow check that az group show -g *rg* --query tags actually works
func TestAccACSEngineK8sCluster_updateTags(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	config := testAccACSEngineK8sClusterBasic(ri, clientID, clientSecret, location, keyData)
	newTagsConfig := testAccACSEngineK8sClusterTags(ri, clientID, clientSecret, location, keyData, "Prod", "IT")
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "tags.Environment", "Production"),
				),
			},
			{
				Config: newTagsConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "tags.Environment", "Prod"),
					resource.TestCheckResourceAttr(tfResourceName, "tags.Department", "IT"),
				),
			},
		},
	})
}

// failing because I haven't implemented yet
// func TestAccACSEngineK8sCluster_createWindowsAgentCluster(t *testing.T) {
// 	ri := acctest.RandInt()
// 	clientID := testClientID()
// 	clientSecret := testClientSecret()
// 	location := testLocation()
// 	keyData := testSSHPublicKey()
// 	kubernetesVersion := "1.8.13"
// 	count := 1
// 	config := testAccACSEngineK8sClusterOSType(ri, clientID, clientSecret, location, keyData, kubernetesVersion, count)

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:     func() { testAccPreCheck(t) },
// 		Providers:    testAccProviders,
// 		CheckDestroy: testCheckACSEngineClusterDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: config,
// 				Check: resource.ComposeTestCheckFunc(
// 					testCheckACSEngineClusterExists("acsengine_kubernetes_cluster.test"),
// 				),
// 			},
// 		},
// 	})
// }

// failing because I haven't implemented yet
// func TestAccACSEngineK8sCluster_scaleUpDownWindowsAgentCluster(t *testing.T) {
// 	ri := acctest.RandInt()
// 	clientID := testClientID()
// 	clientSecret := testClientSecret()
// 	location := testLocation()
// 	keyData := testSSHPublicKey()
// 	kubernetesVersion := "1.8.13"
// 	config := testAccACSEngineK8sClusterOSType(ri, clientID, clientSecret, location, keyData, kubernetesVersion, 1)
// 	scaledUpConfig := testAccACSEngineK8sClusterOSType(ri, clientID, clientSecret, location, keyData, kubernetesVersion, 2)
// 	scaledDownConfig := testAccACSEngineK8sClusterOSType(ri, clientID, clientSecret, location, keyData, kubernetesVersion, 1)

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:     func() { testAccPreCheck(t) },
// 		Providers:    testAccProviders,
// 		CheckDestroy: testCheckACSEngineClusterDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: config,
// 				Check: resource.ComposeTestCheckFunc(
// 					testCheckACSEngineClusterExists("acsengine_kubernetes_cluster.test"),
// 				),
// 			},
// 			{
// 				Config: scaledUpConfig,
// 				Check: resource.ComposeTestCheckFunc(
// 					testCheckACSEngineClusterExists("acsengine_kubernetes_cluster.test"),
// 				),
// 			},
// 			{
// 				Config: scaledDownConfig,
// 				Check: resource.ComposeTestCheckFunc(
// 					testCheckACSEngineClusterExists("acsengine_kubernetes_cluster.test"),
// 				),
// 			},
// 		},
// 	})
// }

// scaleDownUpWindowsAgentCluster
// updateUpgradeScaleUpWindowsAgentCluster
// updateUpgradeScaleDownWindowsAgentCluster
// createHybridAgentCluster

// test validation (incorrect commands should not let you do 'apply')

/* HELPER FUNCTIONS */

// can I get rid of some of these? There's so many

func testAccACSEngineK8sClusterBasic(rInt int, clientID string, clientSecret string, location string, keyData string) string {
	return fmt.Sprintf(`resource "acsengine_kubernetes_cluster" "test%d" {
		name               = "acctest"
		resource_group     = "acctestRG-%d"
		location           = "%s"

		master_profile {
			count           = 1
			dns_name_prefix = "acctestmaster%d"
			vm_size         = "Standard_D2_v2"
		}
	
		agent_pool_profiles {
			name    = "agentpool1"
			count   = 1
			vm_size = "Standard_D2_v2"
		}
	
		linux_profile {
			admin_username = "acctestuser%d"
			ssh {
				key_data = "%s"
			}
		}

		service_principal {
			client_id     = "%s"
			client_secret = "%s"
		}

		tags {
			Environment = "Production"
		}
	}`, rInt, rInt, location, rInt, rInt, keyData, clientID, clientSecret)
}

func testAccACSEngineK8sClusterMultipleAgentPools(rInt int, clientID string, clientSecret string, location string, keyData string) string {
	return fmt.Sprintf(`resource "acsengine_kubernetes_cluster" "test%d" {
		name               = "acctest"
		resource_group     = "acctestRG-%d"
		location           = "%s"
	
		master_profile {
			count           = 1
			dns_name_prefix = "acctestmaster%d"
			vm_size         = "Standard_D2_v2"
		}
	
		agent_pool_profiles {
			name    = "agentpool1"
			count   = 2
			vm_size = "Standard_D2_v2"
		}
	
		agent_pool_profiles {
			name    = "agentpool2"
			count   = 1
			vm_size = "Standard_D2_v2"
		}
	
		linux_profile {
			admin_username = "acctestuser%d"
			ssh {
				key_data = "%s"
			}
		}

		service_principal {
			client_id     = "%s"
			client_secret = "%s"
		}
	}`, rInt, rInt, location, rInt, rInt, keyData, clientID, clientSecret)
}

// add more customization like os_type
func testAccACSEngineK8sClusterCustomized(rInt int, clientID string, clientSecret string, location string, keyData string, k8sVersion string, agentCount int, vmSize string, osDiskSize int) string {
	return fmt.Sprintf(`resource "acsengine_kubernetes_cluster" "test%d" {
		name               = "acctest"
		resource_group     = "acctestRG-%d"
		location           = "%s"
		kubernetes_version = "%s"
	
		master_profile {
			count           = 1
			dns_name_prefix = "acctestmaster%d"
			vm_size         = "%s"
			os_disk_size    = "%d"
		}
	
		agent_pool_profiles {
			name         = "agentpool1"
			count        = "%d"
			vm_size      = "%s"
			os_disk_size = "%d"
		}
	
		linux_profile {
			admin_username = "acctestuser%d"
			ssh {
				key_data = "%s"
			}
		}

		service_principal {
			client_id     = "%s"
			client_secret = "%s"
		}
	
		tags {
			Environment = "Production"
		}
	}`, rInt, rInt, location, k8sVersion, rInt, vmSize, osDiskSize, agentCount, vmSize, osDiskSize, rInt, keyData, clientID, clientSecret)
}

func testAccACSEngineK8sClusterScale(rInt int, clientID string, clientSecret string, location string, keyData string, agentCount int) string {
	return fmt.Sprintf(`resource "acsengine_kubernetes_cluster" "test%d" {
		name               = "acctest"
		resource_group     = "acctestRG-%d"
		location           = "%s"

		master_profile {
			count           = 1
			dns_name_prefix = "acctestmaster%d"
			vm_size         = "Standard_D2_v2"
		}
	
		agent_pool_profiles {
			name    = "agentpool1"
			count   = "%d"
			vm_size = "Standard_D2_v2"
		}
	
		linux_profile {
			admin_username = "acctestuser%d"
			ssh {
				key_data = "%s"
			}
		}

		service_principal {
			client_id     = "%s"
			client_secret = "%s"
		}

		tags {
			Environment = "Production"
		}
	}`, rInt, rInt, location, rInt, agentCount, rInt, keyData, clientID, clientSecret)
}

func testAccACSEngineK8sClusterTags(rInt int, clientID string, clientSecret string, location string, keyData string, tag1 string, tag2 string) string {
	return fmt.Sprintf(`resource "acsengine_kubernetes_cluster" "test%d" {
		name               = "acctest"
		resource_group     = "acctestRG-%d"
		location           = "%s"

		master_profile {
			count           = 1
			dns_name_prefix = "acctestmaster%d"
			vm_size         = "Standard_D2_v2"
		}
	
		agent_pool_profiles {
			name    = "agentpool1"
			count   = 1
			vm_size = "Standard_D2_v2"
		}
	
		linux_profile {
			admin_username = "acctestuser%d"
			ssh {
				key_data = "%s"
			}
		}

		service_principal {
			client_id     = "%s"
			client_secret = "%s"
		}

		tags {
			Environment = "%s"
			Department  = "%s"
		}
	}`, rInt, rInt, location, rInt, rInt, keyData, clientID, clientSecret, tag1, tag2)
}

func testAccACSEngineK8sClusterOSType(rInt int, clientID string, clientSecret string, location string, keyData string, kubernetesVersion string, agentCount int) string {
	return fmt.Sprintf(`resource "acsengine_kubernetes_cluster" "test%d" {
		name               = "acctest"
		resource_group     = "acctestRG-%d"
		location           = "%s"
		kubernetes_version = "%s"

		master_profile {
			count           = 1
			dns_name_prefix = "acctestmaster%d"
			vm_size         = "Standard_D2_v2"
		}
	
		agent_pool_profiles {
			name    = "windowspool1"
			count   = "%d"
			vm_size = "Standard_D2_v2"
			os_type = "Windows"
		}
	
		linux_profile {
			admin_username = "acctestuser%d"
			ssh {
				key_data = "%s"
			}
		}

		service_principal {
			client_id     = "%s"
			client_secret = "%s"
		}

		tags {
			Environment = "Production"
		}
	}`, rInt, rInt, location, kubernetesVersion, rInt, agentCount, rInt, keyData, clientID, clientSecret)
}

func testCheckACSEngineClusterExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		is := rs.Primary // primary instance state
		if is == nil {
			return fmt.Errorf("Bad: could not get primary instance state: %s in %s", name, ms.Path)
		}

		name := is.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for Kubernetes cluster: %s", name)
		}

		client := testAccProvider.Meta().(*ArmClient)
		deployClient := client.deploymentsClient
		ctx := client.StopContext

		resp, err := deployClient.Get(ctx, resourceGroup, name) // is this the best way to test for cluster existence?
		if err != nil {
			return fmt.Errorf("Bad: Get on deploymentsClient: %+v", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Kubernetes cluster %q (resource group: %q) does not exist", name, resourceGroup)
		}

		// check if cluster is actually running (not just that Terraform resource exists and deployment exists)
		if err = clusterIsRunning(is, name); err != nil {
			return fmt.Errorf("Bad: cluster not found to be running: %+v", err)
		}

		return nil
	}
}

func testCheckACSEngineClusterDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ArmClient)
	deployClient := client.deploymentsClient
	ctx := client.StopContext

	for _, rs := range s.RootModule().Resources { // for each resource
		if rs.Type != "acsengine_kubernetes_cluster" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group"]

		resp, err := deployClient.Get(ctx, resourceGroup, name)
		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Kubernetes cluster still exists:\n%#v", resp)
		}
	}

	return nil
}

// clusterIsRunning is a helper function for testCheckACSEngineClusterExists
func clusterIsRunning(is *terraform.InstanceState, name string) error {
	// get kube config
	key := "kube_config_raw"
	var config []byte
	var err error
	if v, ok := is.Attributes[key]; ok {
		config, err = base64.StdEncoding.DecodeString(v)
		if err != nil {
			return fmt.Errorf("kube config could not be decoded from base64: %+v", err)
		}
	} else {
		return fmt.Errorf("%s: Attribute '%s' not found", name, key)
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

func checkNodes(api corev1.CoreV1Interface) error {
	retryErr := utils.RetryOnFailedGet(retry.DefaultRetry, func() error {
		fmt.Println("trying to get nodes...")
		nodes, err := api.Nodes().List(metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Reason for error: %+v\n", errors.ReasonForError(err))
			return fmt.Errorf("failed to get nodes: %+v", err)
		}
		if len(nodes.Items) < 2 {
			return fmt.Errorf("not enough nodes found (there should be a at least one master and agent pool): only %d found", len(nodes.Items))
		}
		// do I need to wait some time to make sure nodes are ready?
		for _, node := range nodes.Items {
			fmt.Printf("Node: %s\n", node.Name)
			// if !nodeutil.IsNodeReady(&node) { // default eviction time is 5m, so it would probably need to be 5m timeout?
			// 	// return fmt.Errorf("node is not ready: %+v", node) // do I need to not return here? continue instead?
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

// not used yet
func checkTags(resourceGroup string, tags map[string]string) error {
	// try to run az group show -g *rg* --query tags and check string contains tags?
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "az", "group", "show", "-g", resourceGroup, "--query", "tags")
	by, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}
	results := string(by)
	for key, val := range tags {
		if !strings.Contains(results, key) || !strings.Contains(results, val) {
			return fmt.Errorf("key value pair (%s, %s) not found", key, val)
		}
	}

	return nil
}

func mockClusterResourceData(name string, location string, resourceGroup string, dnsPrefix string) *schema.ResourceData {
	r := resourceArmAcsEngineKubernetesCluster()
	d := r.TestResourceData()

	d.Set("name", name)
	d.Set("location", location)
	d.Set("resource_group", resourceGroup)
	d.Set("kubernetes_version", "1.10.0")

	adminUsername := "azureuser"
	linuxProfiles := testFlattenLinuxProfile(adminUsername)
	d.Set("linux_profile", &linuxProfiles)

	servicePrincipals := testFlattenServicePrincipal()
	d.Set("service_principal", servicePrincipals)

	vmSize := "Standard_D2_v2"
	masterProfiles := testFlattenMasterProfile(1, dnsPrefix, vmSize)
	d.Set("master_profile", &masterProfiles)

	agentPool1Name := "agentpool1"
	agentPool1Count := 1
	agentPool2Name := "agentpool2"
	agentPool2Count := 2
	agentPool2osDiskSize := 30

	agentPoolProfiles := []interface{}{}
	agentPoolProfile0 := testFlattenAgentPoolProfiles(agentPool1Name, agentPool1Count, "Standard_D2_v2", 0, false)
	agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile0)
	agentPoolProfile1 := testFlattenAgentPoolProfiles(agentPool2Name, agentPool2Count, "Standard_D2_v2", agentPool2osDiskSize, true)
	agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile1)
	d.Set("agent_pool_profiles", &agentPoolProfiles)

	d.Set("tags", map[string]interface{}{})

	apimodel := utils.ACSEngineK8sClusterAPIModel(name, location, dnsPrefix)
	d.Set("api_model", base64Encode(apimodel))

	return d
}

func resourceName(rInt int) string {
	return fmt.Sprintf("acsengine_kubernetes_cluster.test%d", rInt)
}
