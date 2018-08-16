package acsengine

import (
	"github.com/Azure/terraform-provider-acsengine/internal/resource"
	"github.com/Azure/terraform-provider-acsengine/internal/tester"
)

func mockClusterResourceData(name, location, resourceGroup, dnsPrefix string) *resourceData {
	r := resourceArmACSEngineKubernetesCluster()
	data := r.TestResourceData()
	d := newResourceData(data)

	d.Set("name", name)
	d.Set("location", location)
	d.Set("resource_group", resourceGroup)
	d.Set("kubernetes_version", "1.10.0")

	adminUsername := "azureuser"
	linuxProfiles := tester.MockFlattenLinuxProfile(adminUsername)
	d.Set("linux_profile", &linuxProfiles)

	servicePrincipals := tester.MockFlattenServicePrincipal()
	d.Set("service_principal", servicePrincipals)

	vmSize := "Standard_D2_v2"
	masterProfiles := tester.MockFlattenMasterProfile(1, dnsPrefix, vmSize)
	d.Set("master_profile", &masterProfiles)

	agentPool1Name := "agentpool1"
	agentPool1Count := 1
	agentPool2Name := "agentpool2"
	agentPool2Count := 2
	agentPool2osDiskSize := 30

	agentPoolProfiles := []interface{}{}
	agentPoolProfile0 := tester.MockFlattenAgentPoolProfiles(agentPool1Name, agentPool1Count, "Standard_D2_v2", 0, false)
	agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile0)
	agentPoolProfile1 := tester.MockFlattenAgentPoolProfiles(agentPool2Name, agentPool2Count, "Standard_D2_v2", agentPool2osDiskSize, true)
	agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile1)
	d.Set("agent_pool_profiles", &agentPoolProfiles)

	d.Set("tags", map[string]interface{}{})

	apimodel := resource.ACSEngineK8sClusterAPIModel(name, location, dnsPrefix)
	d.Set("api_model", base64Encode(apimodel))

	return d
}

func mockCluster(name, location, dnsPrefix string) *Cluster {
	cluster := tester.MockContainerService(name, location, dnsPrefix)
	return &Cluster{
		ContainerService: cluster,
	}
}
