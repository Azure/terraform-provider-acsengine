package acsengine

import (
	"strconv"
	"testing"

	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/stretchr/testify/assert"
)

func TestGenerateTemplateBasic(t *testing.T) {
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

	r := resourceArmACSEngineKubernetesCluster()
	data := r.TestResourceData()
	d := newResourceData(data)

	for _, tc := range cases {
		d.Set("name", tc.Name)
		d.Set("location", tc.Location)
		d.Set("resource_group", tc.ResourceGroup)

		linuxProfiles := utils.MockFlattenLinuxProfile(tc.AdminUsername)
		d.Set("linux_profile", &linuxProfiles)

		servicePrincipals := utils.MockFlattenServicePrincipal()
		d.Set("service_principal", servicePrincipals)

		vmSize := "Standard_D2_v2"
		masterProfiles := utils.MockFlattenMasterProfile(tc.MasterCount, tc.DNSPrefix, vmSize)
		d.Set("master_profile", &masterProfiles)

		agentPoolProfiles := []interface{}{}
		agentPoolName := "agentpool0"
		agentPoolProfile0 := utils.MockFlattenAgentPoolProfiles(agentPoolName, tc.AgentPoolCount, vmSize, 0, false)
		agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile0)
		agentPoolName = "agentpool1"
		agentPoolProfile1 := utils.MockFlattenAgentPoolProfiles(agentPoolName, tc.AgentPoolCount+1, vmSize, 0, false)
		agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile1)
		d.Set("agent_pool_profiles", &agentPoolProfiles)

		cluster, err := d.setContainerService()
		if err != nil {
			t.Fatalf("failed to set cluster")
		}

		template, parameters, err := generateACSEngineTemplate(cluster, false) // don't write files
		if err != nil {
			t.Fatalf("Template generation failed: %v", err)
		}

		// now I can test that the template and parameters look okay I guess...
		assert.Contains(t, parameters, tc.AdminUsername, "cluster admin username set incorrectly in parameters")
		assert.Contains(t, parameters, testClientID(), "cluster client ID set incorrectly in parameters")
		assert.Contains(t, parameters, vmSize, "cluster VM size set incorrectly in parameters")
		assert.Contains(t, parameters, strconv.Itoa(tc.AgentPoolCount), "cluster agent pool count set incorrectly in parameters")

		assert.Contains(t, template, agentPoolName+"Count", "cluster count set incorrectly in template")

		assert.False(t, tc.ExpectError, "Expected the Kubernetes Cluster Agent Pool Name to trigger an error for '%s'", tc.Name)
	}
}

func TestGenerateTemplateCustomized(t *testing.T) {
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

	r := resourceArmACSEngineKubernetesCluster()
	data := r.TestResourceData()
	d := newResourceData(data)

	for _, tc := range cases {
		d.Set("name", tc.Name)
		d.Set("location", tc.Location)
		d.Set("resource_group", tc.ResourceGroup)
		d.Set("kubernetes_version", tc.Version)

		linuxProfiles := utils.MockFlattenLinuxProfile(tc.AdminUsername)
		d.Set("linux_profile", &linuxProfiles)

		servicePrincipals := utils.MockFlattenServicePrincipal()
		d.Set("service_principal", servicePrincipals)

		masterProfiles := utils.MockFlattenMasterProfile(tc.MasterCount, tc.DNSPrefix, tc.MasterVMSize)
		d.Set("master_profile", &masterProfiles)

		agentPoolProfiles := []interface{}{}
		agentPoolName := "agentpool0"
		agentPoolProfile0 := utils.MockFlattenAgentPoolProfiles(agentPoolName, tc.AgentPoolCount, tc.AgentVMSize, 0, false)
		agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile0)
		agentPoolName = "agentpool1"
		agentPoolProfile1 := utils.MockFlattenAgentPoolProfiles(agentPoolName, tc.AgentPoolCount+1, tc.AgentVMSize, 0, false)
		agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile1)
		d.Set("agent_pool_profiles", &agentPoolProfiles)

		cluster, err := d.setContainerService()
		if err != nil {
			t.Fatalf("failed to set cluster")
		}

		template, parameters, err := generateACSEngineTemplate(cluster, false) // don't write files
		if err != nil {
			t.Fatalf("Template generation failed: %v", err)
		}

		assert.Contains(t, parameters, tc.AdminUsername, "cluster admin username set incorrectly in parameters")
		assert.Contains(t, parameters, testClientID(), "cluster client ID set incorrectly in parameters")
		assert.Contains(t, parameters, tc.MasterVMSize, "Expected the Azure RM Kubernetes cluster to have parameter '%s'", tc.MasterVMSize)
		assert.Contains(t, parameters, tc.AgentVMSize, "Expected the Azure RM Kubernetes cluster to have parameter '%s'", tc.AgentVMSize)
		assert.Contains(t, parameters, strconv.Itoa(tc.AgentPoolCount), "Expected the Azure RM Kubernetes cluster to have parameter '%d'", tc.AgentPoolCount)

		assert.Contains(t, template, agentPoolName+"Count", "Expected the Azure RM Kubernetes cluster to have field '%s'", agentPoolName+"Count")

		assert.False(t, tc.ExpectError, "Expected the Kubernetes Cluster Agent Pool Name to trigger an error for '%s'", tc.Name)
	}
}
