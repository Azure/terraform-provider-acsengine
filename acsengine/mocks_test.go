package acsengine

import (
	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/api/common"
	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/hashicorp/terraform/helper/schema"
)

func mockClusterResourceData(name string, location string, resourceGroup string, dnsPrefix string) *schema.ResourceData {
	r := resourceArmACSEngineKubernetesCluster()
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

func mockContainerService(name string, location string, dnsPrefix string) *api.ContainerService {
	linuxProfile := testExpandLinuxProfile("azureuser", "public key")
	servicePrincipal := testExpandServicePrincipal("client id", "client secret")
	masterProfile := testExpandMasterProfile(1, dnsPrefix, "Standard_D2_v2", "fqdn", 0)

	agentPoolProfile1 := testExpandAgentPoolProfile("agentpool1", 1, "Standard_D2_v2", 0, false)
	agentPoolProfile2 := testExpandAgentPoolProfile("agentpool2", 2, "Standard_D2_v2", 30, false)
	agentPoolProfiles := []*api.AgentPoolProfile{agentPoolProfile1, agentPoolProfile2}

	orchestratorProfile := api.OrchestratorProfile{
		OrchestratorType:    "Kubernetes",
		OrchestratorVersion: common.GetDefaultKubernetesVersion(),
	}

	certificateProfile := testExpandCertificateProfile()

	properties := api.Properties{
		LinuxProfile:            &linuxProfile,
		ServicePrincipalProfile: &servicePrincipal,
		MasterProfile:           &masterProfile,
		AgentPoolProfiles:       agentPoolProfiles,
		OrchestratorProfile:     &orchestratorProfile,
		CertificateProfile:      &certificateProfile,
	}

	cluster := &api.ContainerService{
		Name:       name,
		Location:   location,
		Properties: &properties,
		Tags:       map[string]string{},
	}

	return cluster
}
