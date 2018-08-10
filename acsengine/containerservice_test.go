package acsengine

import (
	"testing"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/stretchr/testify/assert"
)

func TestFlattenLinuxProfile(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenLinuxProfile failed")
		}
	}()

	adminUsername := "adminUser"
	keyData := "public key data"
	profile := utils.ExpandLinuxProfile(adminUsername, keyData)

	linuxProfile, err := flattenLinuxProfile(profile)
	if err != nil {
		t.Fatalf("flattenLinuxProfile failed: %v", err)
	}

	assert.Equal(t, len(linuxProfile), 1, "did not find linux profile")
	linuxPf := linuxProfile[0].(map[string]interface{})
	val, ok := linuxPf["admin_username"]
	assert.True(t, ok, "flattenLinuxProfile failed: Master count does not exist")
	assert.Equal(t, val, adminUsername)
}

func TestFlattenUnsetLinuxProfile(t *testing.T) {
	profile := api.LinuxProfile{
		AdminUsername: "",
		SSH: struct {
			PublicKeys []api.PublicKey `json:"publicKeys"`
		}{
			PublicKeys: []api.PublicKey{
				{KeyData: ""},
			},
		},
	}
	_, err := flattenLinuxProfile(profile)

	if err == nil {
		t.Fatalf("flattenLinuxProfile should have failed with unset values")
	}
}

func TestFlattenWindowsProfile(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenLinuxProfile failed")
		}
	}()

	adminUsername := "adminUser"
	adminPassword := "password"
	profile := utils.ExpandWindowsProfile(adminUsername, adminPassword)

	windowsProfile, err := flattenWindowsProfile(&profile)
	if err != nil {
		t.Fatalf("flattenWindowsProfile failed: %v", err)
	}

	assert.Equal(t, len(windowsProfile), 1, "did not find windows profile")
	windowsPf := windowsProfile[0].(map[string]interface{})
	val, ok := windowsPf["admin_username"]
	assert.True(t, ok, "flattenWindowsProfile failed: admin username does not exist")
	assert.Equal(t, val, adminUsername)
}

func TestFlattenUnsetWindowsProfile(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenLinuxProfile failed")
		}
	}()

	var profile *api.WindowsProfile
	profile = nil

	windowsProfile, err := flattenWindowsProfile(profile)
	if err != nil {
		t.Fatalf("flattenWindowsProfile failed: %v", err)
	}

	if len(windowsProfile) != 0 {
		t.Fatalf("flattenWindowsProfile failed: did not find zero Windows profiles")
	}
	assert.Equal(t, len(windowsProfile), 0, "did not find zero Windows profiles")
}

func TestFlattenServicePrincipal(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenServicePrincipal failed")
		}
	}()

	clientID := "client id"
	clientSecret := "secret"
	profile := utils.ExpandServicePrincipal(clientID, clientSecret)

	servicePrincipal, err := flattenServicePrincipal(profile)
	if err != nil {
		t.Fatalf("flattenServicePrincipal failed: %v", err)
	}

	assert.Equal(t, len(servicePrincipal), 1, "did not find one service principal")
	spPf := servicePrincipal[0].(map[string]interface{})
	val, ok := spPf["client_id"]
	assert.True(t, ok, "flattenServicePrincipal failed: Master count does not exist")
	assert.Equal(t, val, clientID)
}

func TestFlattenUnsetServicePrincipal(t *testing.T) {
	profile := api.ServicePrincipalProfile{}
	_, err := flattenServicePrincipal(profile)

	if err == nil {
		t.Fatalf("flattenServicePrincipal should have failed with unset values")
	}
}

func TestFlattenDataSourceServicePrincipal(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenServicePrincipal failed")
		}
	}()

	clientID := "client id"
	clientSecret := "secret"
	profile := utils.ExpandServicePrincipal(clientID, clientSecret)

	servicePrincipal, err := flattenDataSourceServicePrincipal(profile)
	if err != nil {
		t.Fatalf("flattenDataSourceServicePrincipal failed: %v", err)
	}

	assert.Equal(t, len(servicePrincipal), 1, "did not find one master profile")
	spPf := servicePrincipal[0].(map[string]interface{})
	val, ok := spPf["client_id"]
	assert.True(t, ok, "flattenDataSourceServicePrincipal failed: Master count does not exist")
	assert.Equal(t, val, clientID)
}

func TestFlattenUnsetDataSourceServicePrincipal(t *testing.T) {
	profile := api.ServicePrincipalProfile{}
	_, err := flattenDataSourceServicePrincipal(profile)

	if err == nil {
		t.Fatalf("flattenServicePrincipal should have failed with unset values")
	}
}

func TestFlattenMasterProfile(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenMasterProfile failed")
		}
	}()

	count := 1
	dnsNamePrefix := "testPrefix"
	vmSize := "Standard_D2_v2"
	fqdn := "abcdefg"
	profile := utils.ExpandMasterProfile(count, dnsNamePrefix, vmSize, fqdn, 0)

	masterProfile, err := flattenMasterProfile(profile, "southcentralus")
	if err != nil {
		t.Fatalf("flattenServicePrincipal failed: %v", err)
	}

	assert.Equal(t, len(masterProfile), 1, "did not find one master profile")
	masterPf := masterProfile[0].(map[string]interface{})
	val, ok := masterPf["count"]
	assert.True(t, ok, "flattenMasterProfile failed: Master count does not exist")
	assert.Equal(t, val, int(count))
	if val, ok := masterPf["os_disk_size"]; ok {
		t.Fatalf("OS disk size should not be set but value is %d", val.(int))
	}
}

func TestFlattenMasterProfileWithOSDiskSize(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenMasterProfile failed")
		}
	}()

	count := 1
	dnsNamePrefix := "testPrefix"
	vmSize := "Standard_D2_v2"
	fqdn := "abcdefg"
	osDiskSize := 30
	profile := utils.ExpandMasterProfile(count, dnsNamePrefix, vmSize, fqdn, osDiskSize)

	masterProfile, err := flattenMasterProfile(profile, "southcentralus")
	if err != nil {
		t.Fatalf("flattenServicePrincipal failed: %v", err)
	}

	assert.Equal(t, len(masterProfile), 1, "did not find one master profile")
	masterPf := masterProfile[0].(map[string]interface{})
	val, ok := masterPf["count"]
	assert.True(t, ok, "flattenMasterProfile failed: Master count does not exist")
	assert.Equal(t, val, int(count))
	val, ok = masterPf["os_disk_size"]
	assert.True(t, ok, "OS disk size should was not set correctly")
	assert.Equal(t, val.(int), osDiskSize)
}

func TestFlattenUnsetMasterProfile(t *testing.T) {
	profile := api.MasterProfile{}
	_, err := flattenMasterProfile(profile, "")

	if err == nil {
		t.Fatalf("flattenMasterProfile should have failed with unset values")
	}
}

func TestFlattenAgentPoolProfiles(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenAgentPoolProfiles failed")
		}
	}()

	name := "agentpool1"
	count := 1
	vmSize := "Standard_D2_v2"
	osDiskSize := 200

	profile1 := utils.ExpandAgentPoolProfile(name, count, vmSize, 0, false)

	name = "agentpool2"
	profile2 := utils.ExpandAgentPoolProfile(name, count, vmSize, osDiskSize, false)

	profiles := []*api.AgentPoolProfile{profile1, profile2}
	agentPoolProfiles, err := flattenAgentPoolProfiles(profiles)
	if err != nil {
		t.Fatalf("flattenAgentPoolProfiles failed: %v", err)
	}

	assert.Equal(t, 2, len(agentPoolProfiles), "did not find correct number of agent pool profiles")
	agentPf0 := agentPoolProfiles[0].(map[string]interface{})
	val, ok := agentPf0["count"]
	assert.True(t, ok, "agent pool count does not exist")
	assert.Equal(t, count, val.(int))
	if val, ok := agentPf0["os_disk_size"]; ok {
		t.Fatalf("agent pool OS disk size should not be set, but is %d", val.(int))
	}
	agentPf1 := agentPoolProfiles[1].(map[string]interface{})
	val, ok = agentPf1["name"]
	assert.True(t, ok, "flattenAgentPoolProfile failed: agent pool count does not exist")
	assert.Equal(t, name, val.(string))
	val, ok = agentPf1["os_disk_size"]
	assert.True(t, ok, "agent pool os disk size is not set when it should be")
	assert.Equal(t, osDiskSize, val.(int))
}

func TestFlattenAgentPoolProfilesWithOSType(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenAgentPoolProfiles failed")
		}
	}()

	name := "agentpool1"
	count := 1
	vmSize := "Standard_D2_v2"

	profile1 := utils.ExpandAgentPoolProfile(name, count, vmSize, 0, false)

	name = "agentpool2"
	profile2 := utils.ExpandAgentPoolProfile(name, count, vmSize, 0, true)

	profiles := []*api.AgentPoolProfile{profile1, profile2}
	agentPoolProfiles, err := flattenAgentPoolProfiles(profiles)
	if err != nil {
		t.Fatalf("flattenAgentPoolProfiles failed: %v", err)
	}

	assert.Equal(t, 2, len(agentPoolProfiles), "did not find correct number of agent pool profiles")
	agentPf0 := agentPoolProfiles[0].(map[string]interface{})
	val, ok := agentPf0["count"]
	assert.True(t, ok, "agent pool count does not exist")
	assert.Equal(t, count, val.(int))
	if val, ok := agentPf0["os_type"]; ok {
		t.Fatalf("agent pool OS type should not be set, but is %d", val.(int))
	}
	agentPf1 := agentPoolProfiles[1].(map[string]interface{})
	val, ok = agentPf1["name"]
	assert.True(t, ok, "flattenAgentPoolProfile failed: agent pool count does not exist")
	assert.Equal(t, name, val.(string))
	val, ok = agentPf1["os_type"]
	assert.True(t, ok, "'os_type' does not exist")
	assert.Equal(t, "Windows", val.(string))
}

func TestFlattenUnsetAgentPoolProfiles(t *testing.T) {
	profile := &api.AgentPoolProfile{}
	profiles := []*api.AgentPoolProfile{profile}
	_, err := flattenAgentPoolProfiles(profiles)

	if err == nil {
		t.Fatalf("flattenAgentPoolProfiles should have failed with unset values")
	}
}

func TestExpandLinuxProfile(t *testing.T) {
	r := resourceArmACSEngineKubernetesCluster()
	d := r.TestResourceData()

	adminUsername := "azureuser"
	linuxProfiles := utils.FlattenLinuxProfile(adminUsername)
	d.Set("linux_profile", &linuxProfiles)

	linuxProfile, err := expandLinuxProfile(d)
	if err != nil {
		t.Fatalf("expand linux profile failed: %v", err)
	}

	assert.Equal(t, linuxProfile.AdminUsername, "azureuser")
}

func TestExpandWindowsProfile(t *testing.T) {
	r := resourceArmACSEngineKubernetesCluster()
	d := r.TestResourceData()

	adminUsername := "azureuser"
	adminPassword := "password"
	windowsProfiles := utils.FlattenWindowsProfile(adminUsername, adminPassword)
	d.Set("windows_profile", &windowsProfiles)

	windowsProfile, err := expandWindowsProfile(d)
	if err != nil {
		t.Fatalf("expand Windows profile failed: %v", err)
	}

	assert.Equal(t, windowsProfile.AdminUsername, adminUsername)
	assert.Equal(t, windowsProfile.AdminPassword, adminPassword)
}

func TestExpandServicePrincipal(t *testing.T) {
	r := resourceArmACSEngineKubernetesCluster()
	d := r.TestResourceData()

	clientID := testClientID()
	servicePrincipals := utils.FlattenServicePrincipal()
	d.Set("service_principal", servicePrincipals)

	servicePrincipal, err := expandServicePrincipal(d)
	if err != nil {
		t.Fatalf("expand service principal failed: %v", err)
	}

	assert.Equal(t, servicePrincipal.ClientID, clientID)
}

func TestExpandMasterProfile(t *testing.T) {
	r := resourceArmACSEngineKubernetesCluster()
	d := r.TestResourceData()

	dnsPrefix := "masterDNSPrefix"
	vmSize := "Standard_D2_v2"
	masterProfiles := utils.FlattenMasterProfile(1, dnsPrefix, vmSize)
	d.Set("master_profile", &masterProfiles)

	masterProfile, err := expandMasterProfile(d)
	if err != nil {
		t.Fatalf("expand master profile failed: %v", err)
	}

	assert.Equal(t, masterProfile.DNSPrefix, dnsPrefix)
	assert.Equal(t, masterProfile.VMSize, vmSize)
}

func TestExpandAgentPoolProfiles(t *testing.T) {
	r := resourceArmACSEngineKubernetesCluster()
	d := r.TestResourceData()

	agentPool1Name := "agentpool1"
	agentPool1Count := 1
	agentPool2Name := "agentpool2"
	agentPool2Count := 2
	agentPool2osDiskSize := 30

	agentPoolProfiles := []interface{}{}
	agentPoolProfile0 := utils.FlattenAgentPoolProfiles(agentPool1Name, agentPool1Count, "Standard_D2_v2", 0, false)
	agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile0)
	agentPoolProfile1 := utils.FlattenAgentPoolProfiles(agentPool2Name, agentPool2Count, "Standard_D2_v2", agentPool2osDiskSize, true)
	agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile1)
	d.Set("agent_pool_profiles", &agentPoolProfiles)

	profiles, err := expandAgentPoolProfiles(d)
	if err != nil {
		t.Fatalf("expand agent pool profiles failed: %v", err)
	}

	assert.Equal(t, len(profiles), 2)
	assert.Equal(t, profiles[0].Name, agentPool1Name)
	assert.Equal(t, profiles[0].Count, agentPool1Count)
	assert.Equal(t, profiles[0].OSDiskSizeGB, 0)
	if profiles[0].OSType != api.Linux {
		t.Fatalf("The first agent pool profile has OS type %s when it should be %s", profiles[0].OSType, api.Linux)
	}
	assert.Equal(t, profiles[0].OSType, api.Linux, "wrong OS type")
	assert.Equal(t, profiles[1].Count, agentPool2Count)
	assert.Equal(t, profiles[1].OSDiskSizeGB, agentPool2osDiskSize)
	if profiles[1].OSType != api.Windows {
		t.Fatalf("The first agent pool profile has OS type %s when it should be %s", profiles[0].OSType, api.Windows)
	}
}

func TestSetContainerService(t *testing.T) {
	name := "testcluster"
	location := "southcentralus"
	resourceGroup := "testrg"
	masterDNSPrefix := "creativeMasterDNSPrefix"

	d := mockClusterResourceData(name, location, resourceGroup, masterDNSPrefix)

	cluster, err := setContainerService(d)
	if err != nil {
		t.Fatalf("setContainerService failed: %+v", err)
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

func TestLoadContainerServiceFromApimodel(t *testing.T) {
	name := "testcluster"
	location := "southcentralus"

	d := mockClusterResourceData(name, location, "testrg", "creativeMasterDNSPrefix") // I need to add a test apimodel in here

	apimodel, err := loadContainerServiceFromApimodel(d, true, false)
	if err != nil {
		t.Fatalf("failed to load container service from api model: %+v", err)
	}

	assert.Equal(t, apimodel.Name, name, "cluster name '%s' not found", name)
	assert.Equal(t, apimodel.Location, location, "cluster location '%s' not found", location)
}

func TestSetProfiles(t *testing.T) {
	dnsPrefix := "lessCreativeMasterDNSPrefix"
	d := mockClusterResourceData("name1", "westus", "testrg", "creativeMasterDNSPrefix")
	cluster := utils.MockContainerService("name2", "southcentralus", dnsPrefix)

	if err := setProfiles(d, cluster); err != nil {
		t.Fatalf("setProfiles failed: %+v", err)
	}
	v, ok := d.GetOk("master_profile.0.dns_name_prefix")
	assert.True(t, ok, "failed to get 'master_profile.0.dns_name_prefix'")
	assert.Equal(t, v.(string), dnsPrefix, "'master_profile.0.dns_name_prefix' is not set correctly")
}

// These need to test linux profile...
func TestSetResourceProfiles(t *testing.T) {
	dnsPrefix := "lessCreativeMasterDNSPrefix"
	d := mockClusterResourceData("name1", "westus", "testrg", "creativeMasterDNSPrefix")
	cluster := utils.MockContainerService("name2", "southcentralus", dnsPrefix)

	if err := setResourceProfiles(d, cluster); err != nil {
		t.Fatalf("setProfiles failed: %+v", err)
	}
	v, ok := d.GetOk("master_profile.0.dns_name_prefix")
	assert.True(t, ok, "failed to get 'master_profile.0.dns_name_prefix'")
	assert.Equal(t, v.(string), dnsPrefix, "'master_profile.0.dns_name_prefix' is not set correctly")
}

func TestSetDataSourceProfiles(t *testing.T) {
	dnsPrefix := "lessCreativeMasterDNSPrefix"
	d := mockClusterResourceData("name1", "westus", "testrg", "creativeMasterDNSPrefix")
	cluster := utils.MockContainerService("name2", "southcentralus", dnsPrefix)

	if err := setDataSourceProfiles(d, cluster); err != nil {
		t.Fatalf("setProfiles failed: %+v", err)
	}
	v, ok := d.GetOk("master_profile.0.dns_name_prefix")
	assert.True(t, ok, "failed to get 'master_profile.0.dns_name_prefix'")
	assert.Equal(t, v.(string), dnsPrefix, "'master_profile.0.dns_name_prefix' is not set correctly")
}

func testCertificateProfile() *api.CertificateProfile {
	profile := &api.CertificateProfile{}

	return profile
}
