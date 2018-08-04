package acsengine

import (
	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/api/common"
	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/hashicorp/terraform/helper/schema"
)

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

// oops, I forgot this existed and started writing another one
func mockACSEngineContainerService() *api.ContainerService {
	// do I need test expandLinuxProfile and so on?
	linuxProfile := testExpandLinuxProfile("azureuser", "public key data")
	servicePrincipal := testExpandServicePrincipal("clientid", "clientsecret")
	masterProfile := testExpandMasterProfile(3, "creativeDNSPrefix", "Standard_D2_v2", "fqdn.com")

	agentPoolProfile1 := testExpandAgentPoolProfile("agentpool1", 5, "Standard_DS_v2", 40)
	agentPoolProfile2 := testExpandAgentPoolProfile("agentpool1", 2, "Standard_DS_v2", 40)
	agentPoolProfiles := []*api.AgentPoolProfile{
		agentPoolProfile1,
		agentPoolProfile2,
	}

	// certificates?

	properties := &api.Properties{
		LinuxProfile:            &linuxProfile,
		ServicePrincipalProfile: &servicePrincipal,
		MasterProfile:           &masterProfile,
		AgentPoolProfiles:       agentPoolProfiles,
	}
	cluster := &api.ContainerService{
		Properties: properties,
	}

	return cluster
}

// this is the other one I started writing
// not using yet
func mockContainerService(name string, location string, dnsPrefix string) *api.ContainerService {
	linuxProfile := testExpandLinuxProfile("azureuser", "public key")
	servicePrincipal := testExpandServicePrincipal("client id", "client secret")
	masterProfile := testExpandMasterProfile(1, "dnsPrefix", "Standard_D2_v2", "fqdn")

	agentPoolProfile1 := testExpandAgentPoolProfile("agentpool1", 1, "Standard_D2_v2", 0)
	agentPoolProfile2 := testExpandAgentPoolProfile("agentpool2", 2, "Standard_D2_v2", 30)
	agentPoolProfiles := []*api.AgentPoolProfile{agentPoolProfile1, agentPoolProfile2}

	orchestratorProfile := api.OrchestratorProfile{
		OrchestratorType:    "Kubernetes",
		OrchestratorVersion: common.GetDefaultKubernetesVersion(),
	}

	properties := api.Properties{
		LinuxProfile:            &linuxProfile,
		ServicePrincipalProfile: &servicePrincipal,
		MasterProfile:           &masterProfile,
		AgentPoolProfiles:       agentPoolProfiles,
		OrchestratorProfile:     &orchestratorProfile,
		// CertificateProfile:      nil,
	}

	cluster := &api.ContainerService{
		Name:       name,
		Location:   location,
		Properties: &properties,
		Tags:       map[string]string{},
	}

	return cluster
}
