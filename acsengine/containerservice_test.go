package acsengine

import (
	"fmt"
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

	assert.Equal(t, 1, len(linuxProfile), "did not find linux profile")
	linuxPf := linuxProfile[0].(map[string]interface{})
	val, ok := linuxPf["admin_username"]
	assert.True(t, ok, "flattenLinuxProfile failed: Master count does not exist")
	assert.Equal(t, adminUsername, val)
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
	if _, err := flattenLinuxProfile(profile); err == nil {
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

	assert.Equal(t, 1, len(windowsProfile), "did not find windows profile")
	windowsPf := windowsProfile[0].(map[string]interface{})
	val, ok := windowsPf["admin_username"]
	assert.True(t, ok, "flattenWindowsProfile failed: admin username does not exist")
	assert.Equal(t, adminUsername, val)
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

	assert.Equal(t, 0, len(windowsProfile), "did not find zero Windows profiles")
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

	assert.Equal(t, 1, len(servicePrincipal), "did not find one service principal")
	spPf := servicePrincipal[0].(map[string]interface{})
	val, ok := spPf["client_id"]
	assert.True(t, ok, "flattenServicePrincipal failed: Master count does not exist")
	assert.Equal(t, clientID, val)
}

func TestFlattenUnsetServicePrincipal(t *testing.T) {
	profile := api.ServicePrincipalProfile{}
	if _, err := flattenServicePrincipal(profile); err == nil {
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

	assert.Equal(t, 1, len(servicePrincipal), "did not find one master profile")
	spPf := servicePrincipal[0].(map[string]interface{})
	val, ok := spPf["client_id"]
	assert.True(t, ok, "flattenDataSourceServicePrincipal failed: Master count does not exist")
	assert.Equal(t, clientID, val)
}

func TestFlattenUnsetDataSourceServicePrincipal(t *testing.T) {
	profile := api.ServicePrincipalProfile{}
	if _, err := flattenDataSourceServicePrincipal(profile); err == nil {
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
	assert.Equal(t, int(count), val)
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

	assert.Equal(t, 1, len(masterProfile), "did not find one master profile")
	masterPf := masterProfile[0].(map[string]interface{})
	val, ok := masterPf["count"]
	assert.True(t, ok, "flattenMasterProfile failed: Master count does not exist")
	assert.Equal(t, int(count), val)
	val, ok = masterPf["os_disk_size"]
	assert.True(t, ok, "OS disk size should was not set correctly")
	assert.Equal(t, osDiskSize, val.(int))
}

func TestFlattenUnsetMasterProfile(t *testing.T) {
	profile := api.MasterProfile{}
	if _, err := flattenMasterProfile(profile, ""); err == nil {
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
	if _, err := flattenAgentPoolProfiles(profiles); err == nil {
		t.Fatalf("flattenAgentPoolProfiles should have failed with unset values")
	}
}

func TestExpandLinuxProfile(t *testing.T) {
	d := mockClusterResourceData("name", "southcentralus", "rg", "prefix")

	adminUsername := "azureuser"
	linuxProfiles := utils.FlattenLinuxProfile(adminUsername)
	d.Set("linux_profile", &linuxProfiles)

	linuxProfile, err := expandLinuxProfile(d)
	if err != nil {
		t.Fatalf("expand linux profile failed: %v", err)
	}

	assert.Equal(t, "azureuser", linuxProfile.AdminUsername)
}

func TestExpandWindowsProfile(t *testing.T) {
	d := mockClusterResourceData("name", "southcentralus", "rg", "prefix")

	adminUsername := "azureuser"
	adminPassword := "password"
	windowsProfiles := utils.FlattenWindowsProfile(adminUsername, adminPassword)
	d.Set("windows_profile", &windowsProfiles)

	windowsProfile, err := expandWindowsProfile(d)
	if err != nil {
		t.Fatalf("expand Windows profile failed: %v", err)
	}

	assert.Equal(t, adminUsername, windowsProfile.AdminUsername)
	assert.Equal(t, adminPassword, windowsProfile.AdminPassword)
}

func TestExpandServicePrincipal(t *testing.T) {
	d := mockClusterResourceData("name", "southcentralus", "rg", "prefix")

	clientID := testClientID()
	servicePrincipals := utils.FlattenServicePrincipal()
	d.Set("service_principal", servicePrincipals)

	servicePrincipal, err := expandServicePrincipal(d)
	if err != nil {
		t.Fatalf("expand service principal failed: %v", err)
	}

	assert.Equal(t, clientID, servicePrincipal.ClientID)
}

func TestExpandMasterProfile(t *testing.T) {
	d := mockClusterResourceData("name", "southcentralus", "rg", "prefix")

	dnsPrefix := "masterDNSPrefix"
	vmSize := "Standard_D2_v2"
	masterProfiles := utils.FlattenMasterProfile(1, dnsPrefix, vmSize)
	d.Set("master_profile", &masterProfiles)

	masterProfile, err := expandMasterProfile(d)
	if err != nil {
		t.Fatalf("expand master profile failed: %v", err)
	}

	assert.Equal(t, dnsPrefix, masterProfile.DNSPrefix)
	assert.Equal(t, vmSize, masterProfile.VMSize)
}

func TestExpandAgentPoolProfiles(t *testing.T) {
	d := mockClusterResourceData("name", "southcentralus", "rg", "prefix")

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
	assert.Equal(t, agentPool1Name, profiles[0].Name)
	assert.Equal(t, agentPool1Count, profiles[0].Count)
	assert.Equal(t, 0, profiles[0].OSDiskSizeGB)
	assert.Equal(t, api.Linux, profiles[0].OSType, "first agent pool OS type is incorrect")
	assert.Equal(t, agentPool2Count, profiles[1].Count)
	assert.Equal(t, agentPool2osDiskSize, profiles[1].OSDiskSizeGB)
	assert.Equal(t, api.Windows, profiles[1].OSType, "second agent pool OS type is incorrect")
}

func TestSetContainerService(t *testing.T) {
	name := "testcluster"
	location := "southcentralus"
	resourceGroup := "testrg"
	masterDNSPrefix := "creativeMasterDNSPrefix"

	d := mockClusterResourceData(name, location, resourceGroup, masterDNSPrefix)

	cluster, err := d.setContainerService()
	if err != nil {
		t.Fatalf("setContainerService failed: %+v", err)
	}

	assert.Equal(t, "testcluster", cluster.Name, "cluster name was not set correctly")
	version := cluster.Properties.OrchestratorProfile.OrchestratorVersion
	assert.Equal(t, "1.10.0", version, "cluster Kubernetes version was not set correctly")
	dnsPrefix := cluster.Properties.MasterProfile.DNSPrefix
	assert.Equal(t, masterDNSPrefix, dnsPrefix, "master DNS prefix was not set correctly")
	assert.Equal(t, 1, cluster.Properties.AgentPoolProfiles[0].Count, "agent pool profile is not set correctly")
}

func TestLoadContainerServiceFromApimodel(t *testing.T) {
	name := "testcluster"
	location := "southcentralus"

	d := mockClusterResourceData(name, location, "testrg", "creativeMasterDNSPrefix") // I need to add a test apimodel in here

	apimodel, err := d.loadContainerServiceFromApimodel(true, false)
	if err != nil {
		t.Fatalf("failed to load container service from api model: %+v", err)
	}

	assert.Equal(t, name, apimodel.Name, "cluster name '%s' not found", name)
	assert.Equal(t, location, apimodel.Location, "cluster location '%s' not found", location)
}

func TestSetProfiles(t *testing.T) {
	dnsPrefix := "lessCreativeMasterDNSPrefix"
	d := mockClusterResourceData("name1", "westus", "testrg", "creativeMasterDNSPrefix")
	cluster := mockCluster("name2", "southcentralus", dnsPrefix)

	if err := d.setStateProfiles(cluster); err != nil {
		t.Fatalf("setProfiles failed: %+v", err)
	}
	v, ok := d.GetOk("master_profile.0.dns_name_prefix")
	assert.True(t, ok, "failed to get 'master_profile.0.dns_name_prefix'")
	assert.Equal(t, dnsPrefix, v.(string), "'master_profile.0.dns_name_prefix' is not set correctly")
}

// These need to test linux profile...
func TestSetResourceProfiles(t *testing.T) {
	dnsPrefix := "lessCreativeMasterDNSPrefix"
	d := mockClusterResourceData("name1", "westus", "testrg", "creativeMasterDNSPrefix")
	cluster := mockCluster("name2", "southcentralus", dnsPrefix)

	if err := d.setResourceStateProfiles(cluster); err != nil {
		t.Fatalf("setProfiles failed: %+v", err)
	}
	v, ok := d.GetOk("master_profile.0.dns_name_prefix")
	assert.True(t, ok, "failed to get 'master_profile.0.dns_name_prefix'")
	assert.Equal(t, dnsPrefix, v.(string), "'master_profile.0.dns_name_prefix' is not set correctly")
}

func TestSetDataSourceProfiles(t *testing.T) {
	dnsPrefix := "lessCreativeMasterDNSPrefix"
	d := mockClusterResourceData("name1", "westus", "testrg", "creativeMasterDNSPrefix")
	cluster := mockCluster("name2", "southcentralus", dnsPrefix)

	if err := d.setDataSourceStateProfiles(cluster); err != nil {
		t.Fatalf("setProfiles failed: %+v", err)
	}
	v, ok := d.GetOk("master_profile.0.dns_name_prefix")
	assert.True(t, ok, "failed to get 'master_profile.0.dns_name_prefix'")
	assert.Equal(t, dnsPrefix, v.(string), "'master_profile.0.dns_name_prefix' is not set correctly")
}

func TestAddValue(t *testing.T) {
	parameters := map[string]interface{}{}

	addValue(parameters, "key", "data")

	v, ok := parameters["key"]
	assert.True(t, ok, "could not find key")
	val := v.(map[string]interface{})
	assert.Equal(t, val["value"], "data", "value not set correctly")
}

func TestExpandTemplateBodies(t *testing.T) {
	body := `{
		"groceries": {
			"quinoa": "5",
			"pasta": "2"
		}
	}`

	template, parameters, err := expandTemplates(body, body)
	if err != nil {
		t.Fatalf("expand templates failed: %+v", err)
	}

	v, ok := parameters["groceries"]
	assert.True(t, ok, "could not find `groceries`")
	paramsGroceries := v.(map[string]interface{})
	assert.Equal(t, len(paramsGroceries), 2, fmt.Sprintf("length of grocery list is not correct: expected 2 and found %d", len(paramsGroceries)))
	v, ok = paramsGroceries["quinoa"]
	assert.True(t, ok, "could not find `quinoa`")
	assert.Equal(t, v.(string), "5")

	v, ok = template["groceries"]
	assert.True(t, ok, "could not find `groceries`")
	templateGroceries := v.(map[string]interface{})
	if len(templateGroceries) != 2 {
		t.Fatalf("length of grocery list is not correct: expected 2 and found %d", len(templateGroceries))
	}
	assert.Equal(t, len(templateGroceries), 2)
	v, ok = templateGroceries["pasta"]
	assert.True(t, ok, "could not find `pasta`")
	assert.Equal(t, v.(string), "2")
}

func TestExpandBody(t *testing.T) {
	body := `{
		"groceries": {
			"bananas": "5",
			"pasta": "2"
		}
	}`

	expandedBody, err := expandBody(body)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	v, ok := expandedBody["groceries"]
	assert.True(t, ok, "could not find `groceries`")
	groceries := v.(map[string]interface{})
	assert.Equal(t, len(groceries), 2)
	v, ok = groceries["bananas"]
	assert.True(t, ok, "could not find `bananas`")
	assert.Equal(t, v.(string), "5")
}

func testCertificateProfile() *api.CertificateProfile {
	profile := &api.CertificateProfile{}

	return profile
}
