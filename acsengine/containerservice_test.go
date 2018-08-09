package acsengine

import (
	"testing"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestACSEngineK8sCluster_flattenLinuxProfile(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenLinuxProfile failed")
		}
	}()

	adminUsername := "adminUser"
	keyData := "public key data"
	profile := testExpandLinuxProfile(adminUsername, keyData)

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

func TestACSEngineK8sCluster_flattenUnsetLinuxProfile(t *testing.T) {
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

func TestACSEngineK8sCluster_flattenWindowsProfile(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenLinuxProfile failed")
		}
	}()

	adminUsername := "adminUser"
	adminPassword := "password"
	profile := testExpandWindowsProfile(adminUsername, adminPassword)

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

func TestACSEngineK8sCluster_flattenUnsetWindowsProfile(t *testing.T) {
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
}

func TestACSEngineK8sCluster_flattenServicePrincipal(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenServicePrincipal failed")
		}
	}()

	clientID := "client id"
	clientSecret := "secret"
	profile := testExpandServicePrincipal(clientID, clientSecret)

	servicePrincipal, err := flattenServicePrincipal(profile)
	if err != nil {
		t.Fatalf("flattenServicePrincipal failed: %v", err)
	}

	if len(servicePrincipal) != 1 {
		t.Fatalf("flattenServicePrincipal failed: did not find one master profile")
	}
	spPf := servicePrincipal[0].(map[string]interface{})
	val, ok := spPf["client_id"]
	assert.True(t, ok, "flattenServicePrincipal failed: Master count does not exist")
	assert.Equal(t, val, clientID)
}

func TestACSEngineK8sCluster_flattenUnsetServicePrincipal(t *testing.T) {
	profile := api.ServicePrincipalProfile{}
	_, err := flattenServicePrincipal(profile)

	if err == nil {
		t.Fatalf("flattenServicePrincipal should have failed with unset values")
	}
}

func TestACSEngineK8sCluster_flattenDataSourceServicePrincipal(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenServicePrincipal failed")
		}
	}()

	clientID := "client id"
	clientSecret := "secret"
	profile := testExpandServicePrincipal(clientID, clientSecret)

	servicePrincipal, err := flattenDataSourceServicePrincipal(profile)
	if err != nil {
		t.Fatalf("flattenDataSourceServicePrincipal failed: %v", err)
	}

	if len(servicePrincipal) != 1 {
		t.Fatalf("flattenDataSourceServicePrincipal failed: did not find one master profile")
	}
	spPf := servicePrincipal[0].(map[string]interface{})
	val, ok := spPf["client_id"]
	assert.True(t, ok, "flattenDataSourceServicePrincipal failed: Master count does not exist")
	assert.Equal(t, val, clientID)
}

func TestACSEngineK8sCluster_flattenUnsetDataSourceServicePrincipal(t *testing.T) {
	profile := api.ServicePrincipalProfile{}
	_, err := flattenDataSourceServicePrincipal(profile)

	if err == nil {
		t.Fatalf("flattenServicePrincipal should have failed with unset values")
	}
}

func TestACSEngineK8sCluster_flattenMasterProfile(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenMasterProfile failed")
		}
	}()

	count := 1
	dnsNamePrefix := "testPrefix"
	vmSize := "Standard_D2_v2"
	fqdn := "abcdefg"
	profile := testExpandMasterProfile(count, dnsNamePrefix, vmSize, fqdn, 0)

	masterProfile, err := flattenMasterProfile(profile, "southcentralus")
	if err != nil {
		t.Fatalf("flattenServicePrincipal failed: %v", err)
	}

	if len(masterProfile) != 1 {
		t.Fatalf("flattenMasterProfile failed: did not find one master profile")
	}
	masterPf := masterProfile[0].(map[string]interface{})
	val, ok := masterPf["count"]
	assert.True(t, ok, "flattenMasterProfile failed: Master count does not exist")
	assert.Equal(t, val, int(count))
	if val, ok := masterPf["os_disk_size"]; ok {
		t.Fatalf("OS disk size should not be set but value is %d", val.(int))
	}
}

func TestACSEngineK8sCluster_flattenMasterProfileWithOSDiskSize(t *testing.T) {
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
	profile := testExpandMasterProfile(count, dnsNamePrefix, vmSize, fqdn, osDiskSize)

	masterProfile, err := flattenMasterProfile(profile, "southcentralus")
	if err != nil {
		t.Fatalf("flattenServicePrincipal failed: %v", err)
	}

	if len(masterProfile) != 1 {
		t.Fatalf("flattenMasterProfile failed: did not find one master profile")
	}
	masterPf := masterProfile[0].(map[string]interface{})
	val, ok := masterPf["count"]
	assert.True(t, ok, "flattenMasterProfile failed: Master count does not exist")
	assert.Equal(t, val, int(count))
	val, ok = masterPf["os_disk_size"]
	assert.True(t, ok, "OS disk size should was not set correctly")
	assert.Equal(t, val.(int), osDiskSize)
}

func TestACSEngineK8sCluster_flattenUnsetMasterProfile(t *testing.T) {
	profile := api.MasterProfile{}
	_, err := flattenMasterProfile(profile, "")

	if err == nil {
		t.Fatalf("flattenMasterProfile should have failed with unset values")
	}
}

func TestACSEngineK8sCluster_flattenAgentPoolProfiles(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenAgentPoolProfiles failed")
		}
	}()

	name := "agentpool1"
	count := 1
	vmSize := "Standard_D2_v2"
	osDiskSize := 200

	profile1 := testExpandAgentPoolProfile(name, count, vmSize, 0, false)

	name = "agentpool2"
	profile2 := testExpandAgentPoolProfile(name, count, vmSize, osDiskSize, false)

	profiles := []*api.AgentPoolProfile{profile1, profile2}
	agentPoolProfiles, err := flattenAgentPoolProfiles(profiles)
	if err != nil {
		t.Fatalf("flattenAgentPoolProfiles failed: %v", err)
	}

	if len(agentPoolProfiles) < 1 {
		t.Fatalf("flattenAgentPoolProfile failed: did not find any agent pool profiles")
	}
	agentPf0 := agentPoolProfiles[0].(map[string]interface{})
	val, ok := agentPf0["count"]
	assert.True(t, ok, "agent pool count does not exist")
	assert.Equal(t, val.(int), count)
	if val, ok := agentPf0["os_disk_size"]; ok {
		t.Fatalf("agent pool OS disk size should not be set, but is %d", val.(int))
	}
	agentPf1 := agentPoolProfiles[1].(map[string]interface{})
	val, ok = agentPf1["name"]
	assert.True(t, ok, "flattenAgentPoolProfile failed: agent pool count does not exist")
	assert.Equal(t, val.(string), name)
	val, ok = agentPf1["os_disk_size"]
	assert.True(t, ok, "agent pool os disk size is not set when it should be")
	assert.Equal(t, val.(int), osDiskSize)
}

func TestACSEngineK8sCluster_flattenAgentPoolProfilesWithOSType(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenAgentPoolProfiles failed")
		}
	}()

	name := "agentpool1"
	count := 1
	vmSize := "Standard_D2_v2"

	profile1 := testExpandAgentPoolProfile(name, count, vmSize, 0, false)

	name = "agentpool2"
	profile2 := testExpandAgentPoolProfile(name, count, vmSize, 0, true)

	profiles := []*api.AgentPoolProfile{profile1, profile2}
	agentPoolProfiles, err := flattenAgentPoolProfiles(profiles)
	if err != nil {
		t.Fatalf("flattenAgentPoolProfiles failed: %v", err)
	}

	if len(agentPoolProfiles) < 1 {
		t.Fatalf("flattenAgentPoolProfile failed: did not find any agent pool profiles")
	}
	agentPf0 := agentPoolProfiles[0].(map[string]interface{})
	val, ok := agentPf0["count"]
	assert.True(t, ok, "agent pool count does not exist")
	assert.Equal(t, val.(int), count)
	if val, ok := agentPf0["os_type"]; ok {
		t.Fatalf("agent pool OS type should not be set, but is %d", val.(int))
	}
	agentPf1 := agentPoolProfiles[1].(map[string]interface{})
	val, ok = agentPf1["name"]
	assert.True(t, ok, "flattenAgentPoolProfile failed: agent pool count does not exist")
	assert.Equal(t, val.(string), name)
	val, ok = agentPf1["os_type"]
	assert.True(t, ok, "'os_type' does not exist")
	assert.Equal(t, val.(string), "Windows")
}

func TestACSEngineK8sCluster_flattenUnsetAgentPoolProfiles(t *testing.T) {
	profile := &api.AgentPoolProfile{}
	profiles := []*api.AgentPoolProfile{profile}
	_, err := flattenAgentPoolProfiles(profiles)

	if err == nil {
		t.Fatalf("flattenAgentPoolProfiles should have failed with unset values")
	}
}

func TestACSEngineK8sCluster_expandLinuxProfile(t *testing.T) {
	r := resourceArmACSEngineKubernetesCluster()
	d := r.TestResourceData()

	adminUsername := "azureuser"
	linuxProfiles := testFlattenLinuxProfile(adminUsername)
	d.Set("linux_profile", &linuxProfiles)

	linuxProfile, err := expandLinuxProfile(d)
	if err != nil {
		t.Fatalf("expand linux profile failed: %v", err)
	}

	assert.Equal(t, linuxProfile.AdminUsername, "azureuser")
}

func TestACSEngineK8sCluster_expandWindowsProfile(t *testing.T) {
	r := resourceArmACSEngineKubernetesCluster()
	d := r.TestResourceData()

	adminUsername := "azureuser"
	adminPassword := "password"
	windowsProfiles := testFlattenWindowsProfile(adminUsername, adminPassword)
	d.Set("windows_profile", &windowsProfiles)

	windowsProfile, err := expandWindowsProfile(d)
	if err != nil {
		t.Fatalf("expand Windows profile failed: %v", err)
	}

	assert.Equal(t, windowsProfile.AdminUsername, adminUsername)
	assert.Equal(t, windowsProfile.AdminPassword, adminPassword)
}

func TestACSEngineK8sCluster_expandServicePrincipal(t *testing.T) {
	r := resourceArmACSEngineKubernetesCluster()
	d := r.TestResourceData()

	clientID := testClientID()
	servicePrincipals := testFlattenServicePrincipal()
	d.Set("service_principal", servicePrincipals)

	servicePrincipal, err := expandServicePrincipal(d)
	if err != nil {
		t.Fatalf("expand service principal failed: %v", err)
	}

	assert.Equal(t, servicePrincipal.ClientID, clientID)
}

func TestACSEngineK8sCluster_expandMasterProfile(t *testing.T) {
	r := resourceArmACSEngineKubernetesCluster()
	d := r.TestResourceData()

	dnsPrefix := "masterDNSPrefix"
	vmSize := "Standard_D2_v2"
	masterProfiles := testFlattenMasterProfile(1, dnsPrefix, vmSize)
	d.Set("master_profile", &masterProfiles)

	masterProfile, err := expandMasterProfile(d)
	if err != nil {
		t.Fatalf("expand master profile failed: %v", err)
	}

	assert.Equal(t, masterProfile.DNSPrefix, dnsPrefix)
	assert.Equal(t, masterProfile.VMSize, vmSize)
}

func TestACSEngineK8sCluster_expandAgentPoolProfiles(t *testing.T) {
	r := resourceArmACSEngineKubernetesCluster()
	d := r.TestResourceData()

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
	assert.Equal(t, profiles[1].Count, agentPool2Count)
	assert.Equal(t, profiles[1].OSDiskSizeGB, agentPool2osDiskSize)
	if profiles[1].OSType != api.Windows {
		t.Fatalf("The first agent pool profile has OS type %s when it should be %s", profiles[0].OSType, api.Windows)
	}
}

func testFlattenLinuxProfile(adminUsername string) []interface{} {
	sshKeys := []interface{}{}
	keys := map[string]interface{}{
		"key_data": testSSHPublicKey(),
	}
	sshKeys = append(sshKeys, keys)
	values := map[string]interface{}{
		"admin_username": adminUsername,
		"ssh":            sshKeys,
	}
	linuxProfiles := []interface{}{}
	linuxProfiles = append(linuxProfiles, values)

	return linuxProfiles
}

func testFlattenWindowsProfile(adminUsername string, adminPassword string) []interface{} {
	values := map[string]interface{}{
		"admin_username": adminUsername,
		"admin_password": adminPassword,
	}
	windowsProfiles := []interface{}{}
	windowsProfiles = append(windowsProfiles, values)

	return windowsProfiles
}

func testFlattenServicePrincipal() []interface{} {
	servicePrincipals := []interface{}{}

	spValues := map[string]interface{}{
		"client_id":     testClientID(),
		"client_secret": testClientSecret(),
	}

	servicePrincipals = append(servicePrincipals, spValues)

	return servicePrincipals
}

func testFlattenMasterProfile(count int, dnsNamePrefix string, vmSize string) []interface{} {
	masterProfiles := []interface{}{}

	masterProfile := make(map[string]interface{}, 5)

	masterProfile["count"] = count
	masterProfile["dns_name_prefix"] = dnsNamePrefix
	masterProfile["vm_size"] = vmSize
	masterProfile["fqdn"] = "f/q/d/n"

	masterProfiles = append(masterProfiles, masterProfile)

	return masterProfiles
}

func testFlattenAgentPoolProfiles(name string, count int, vmSize string, osDiskSizeGB int, windows bool) map[string]interface{} {
	agentPoolValues := map[string]interface{}{
		"name":    name,
		"count":   count,
		"vm_size": vmSize,
	}
	if osDiskSizeGB != 0 {
		agentPoolValues["os_disk_size"] = osDiskSizeGB
	}
	if windows {
		agentPoolValues["os_type"] = string(api.Windows)
	} else {
		agentPoolValues["os_type"] = string(api.Linux)
	}

	return agentPoolValues
}

func testExpandLinuxProfile(adminUsername string, keyData string) api.LinuxProfile {
	sshPublicKeys := []api.PublicKey{
		{KeyData: keyData},
	}
	profile := api.LinuxProfile{
		AdminUsername: adminUsername,
		SSH: struct {
			PublicKeys []api.PublicKey `json:"publicKeys"`
		}{
			PublicKeys: sshPublicKeys,
		},
	}

	return profile
}

func testExpandWindowsProfile(adminUsername string, adminPassword string) api.WindowsProfile {
	profile := api.WindowsProfile{
		AdminUsername: adminUsername,
		AdminPassword: adminPassword,
	}

	return profile
}

func testExpandServicePrincipal(clientID string, clientSecret string) api.ServicePrincipalProfile {
	profile := api.ServicePrincipalProfile{
		ClientID: clientID,
		Secret:   clientSecret,
	}

	return profile
}

func testExpandMasterProfile(count int, dnsPrefix string, vmSize string, fqdn string, osDiskSize int) api.MasterProfile {
	profile := api.MasterProfile{
		Count:     count,
		DNSPrefix: dnsPrefix,
		VMSize:    vmSize,
		FQDN:      fqdn,
	}

	if osDiskSize != 0 {
		profile.OSDiskSizeGB = osDiskSize
	}

	return profile
}

func testExpandAgentPoolProfile(name string, count int, vmSize string, osDiskSizeGB int, isWindows bool) *api.AgentPoolProfile {
	profile := &api.AgentPoolProfile{
		Name:   name,
		Count:  count,
		VMSize: vmSize,
	}

	if osDiskSizeGB > 0 {
		profile.OSDiskSizeGB = osDiskSizeGB
	}

	if isWindows {
		profile.OSType = api.Windows
	}

	return profile
}

func testExpandCertificateProfile() api.CertificateProfile {
	certificateProfile := api.CertificateProfile{
		CaCertificate:         "apple",
		CaPrivateKey:          "banana",
		APIServerCertificate:  "blueberry",
		APIServerPrivateKey:   "grape",
		ClientCertificate:     "blackberry",
		ClientPrivateKey:      "pomegranate",
		EtcdClientCertificate: "strawberry",
		EtcdClientPrivateKey:  "plum",
		EtcdPeerCertificates:  []string{"peach"},
		EtcdPeerPrivateKeys:   []string{"pear"},
	}
	return certificateProfile
}

func testCertificateProfile() *api.CertificateProfile {
	profile := &api.CertificateProfile{}

	return profile
}

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

	assert.Equal(t, apimodel.Name, name, "cluster name '%s' not found", name)
	assert.Equal(t, apimodel.Location, location, "cluster location '%s' not found", location)
}

func TestACSEngineCluster_setProfiles(t *testing.T) {
	dnsPrefix := "lessCreativeMasterDNSPrefix"
	d := mockClusterResourceData("name1", "westus", "testrg", "creativeMasterDNSPrefix")
	cluster := mockContainerService("name2", "southcentralus", dnsPrefix)

	if err := setProfiles(d, cluster); err != nil {
		t.Fatalf("setProfiles failed: %+v", err)
	}
	v, ok := d.GetOk("master_profile.0.dns_name_prefix")
	assert.True(t, ok, "failed to get 'master_profile.0.dns_name_prefix'")
	assert.Equal(t, v.(string), dnsPrefix, "'master_profile.0.dns_name_prefix' is not set correctly")
}

// These need to test linux profile...
func TestACSEngineCluster_setResourceProfiles(t *testing.T) {
	dnsPrefix := "lessCreativeMasterDNSPrefix"
	d := mockClusterResourceData("name1", "westus", "testrg", "creativeMasterDNSPrefix")
	cluster := mockContainerService("name2", "southcentralus", dnsPrefix)

	if err := setResourceProfiles(d, cluster); err != nil {
		t.Fatalf("setProfiles failed: %+v", err)
	}
	v, ok := d.GetOk("master_profile.0.dns_name_prefix")
	assert.True(t, ok, "failed to get 'master_profile.0.dns_name_prefix'")
	assert.Equal(t, v.(string), dnsPrefix, "'master_profile.0.dns_name_prefix' is not set correctly")
}

func TestACSEngineCluster_setDataSourceProfiles(t *testing.T) {
	dnsPrefix := "lessCreativeMasterDNSPrefix"
	d := mockClusterResourceData("name1", "westus", "testrg", "creativeMasterDNSPrefix")
	cluster := mockContainerService("name2", "southcentralus", dnsPrefix)

	if err := setDataSourceProfiles(d, cluster); err != nil {
		t.Fatalf("setProfiles failed: %+v", err)
	}
	v, ok := d.GetOk("master_profile.0.dns_name_prefix")
	assert.True(t, ok, "failed to get 'master_profile.0.dns_name_prefix'")
	assert.Equal(t, v.(string), dnsPrefix, "'master_profile.0.dns_name_prefix' is not set correctly")
}
