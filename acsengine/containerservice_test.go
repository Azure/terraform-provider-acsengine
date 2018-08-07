package acsengine

import (
	"testing"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/test"
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

	if len(linuxProfile) != 1 {
		t.Fatalf("flattenLinuxProfile failed: did not find one linux profile")
	}
	linuxPf := linuxProfile[0].(map[string]interface{})
	val, ok := linuxPf["admin_username"]
	if !ok {
		t.Fatalf("flattenLinuxProfile failed: Master count does not exist")
	}
	test.Equals(t, val, adminUsername)
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
	if !ok {
		t.Fatalf("flattenServicePrincipal failed: Master count does not exist")
	}
	test.Equals(t, val, clientID)
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
		t.Fatalf("flattenServicePrincipal failed: %v", err)
	}

	if len(servicePrincipal) != 1 {
		t.Fatalf("flattenServicePrincipal failed: did not find one master profile")
	}
	spPf := servicePrincipal[0].(map[string]interface{})
	val, ok := spPf["client_id"]
	if !ok {
		t.Fatalf("flattenServicePrincipal failed: Master count does not exist")
	}
	test.Equals(t, val, clientID)
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
	profile := testExpandMasterProfile(count, dnsNamePrefix, vmSize, fqdn)

	masterProfile, err := flattenMasterProfile(profile, "southcentralus")
	if err != nil {
		t.Fatalf("flattenServicePrincipal failed: %v", err)
	}

	if len(masterProfile) != 1 {
		t.Fatalf("flattenMasterProfile failed: did not find one master profile")
	}
	masterPf := masterProfile[0].(map[string]interface{})
	val, ok := masterPf["count"]
	if !ok {
		t.Fatalf("flattenMasterProfile failed: Master count does not exist")
	}
	test.Equals(t, val, int(count))
	if val, ok := masterPf["os_disk_size"]; ok {
		t.Fatalf("OS disk size should not be set but value is %d", val.(int))
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

	profile1 := testExpandAgentPoolProfile(name, count, vmSize, 0)

	name = "agentpool2"
	profile2 := testExpandAgentPoolProfile(name, count, vmSize, osDiskSize)

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
	if !ok {
		t.Fatalf("agent pool count does not exist")
	}
	test.Equals(t, val.(int), count)
	if val, ok := agentPf0["os_disk_size"]; ok {
		t.Fatalf("agent pool OS disk size should not be set, but is %d", val.(int))
	}
	agentPf1 := agentPoolProfiles[1].(map[string]interface{})
	val, ok = agentPf1["name"]
	if !ok {
		t.Fatalf("flattenAgentPoolProfile failed: agent pool count does not exist")
	}
	test.Equals(t, val.(string), name)
	val, ok = agentPf1["os_disk_size"]
	if !ok {
		t.Fatalf("agent pool os disk size is not set when it should be")
	}
	test.Equals(t, val.(int), osDiskSize)
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

	test.Equals(t, linuxProfile.AdminUsername, "azureuser")
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

	test.Equals(t, servicePrincipal.ClientID, clientID)
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

	test.Equals(t, masterProfile.DNSPrefix, dnsPrefix)
	test.Equals(t, masterProfile.VMSize, vmSize)
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

	test.Equals(t, len(profiles), 2)
	test.Equals(t, profiles[0].Name, agentPool1Name)
	test.Equals(t, profiles[0].Count, agentPool1Count)
	test.Equals(t, profiles[0].OSDiskSizeGB, 0)
	if profiles[0].OSType != api.Linux {
		t.Fatalf("The first agent pool profile has OS type %s when it should be %s", profiles[0].OSType, api.Linux)
	}
	test.Equals(t, profiles[1].Count, agentPool2Count)
	test.Equals(t, profiles[1].OSDiskSizeGB, agentPool2osDiskSize)
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

func testExpandServicePrincipal(clientID string, clientSecret string) api.ServicePrincipalProfile {
	profile := api.ServicePrincipalProfile{
		ClientID: clientID,
		Secret:   clientSecret,
	}

	return profile
}

func testExpandMasterProfile(count int, dnsPrefix string, vmSize string, fqdn string) api.MasterProfile {
	profile := api.MasterProfile{
		Count:     count,
		DNSPrefix: dnsPrefix,
		VMSize:    vmSize,
		FQDN:      fqdn,
	}

	return profile
}

func testExpandAgentPoolProfile(name string, count int, vmSize string, osDiskSizeGB int) *api.AgentPoolProfile {
	profile := &api.AgentPoolProfile{
		Name:   name,
		Count:  count,
		VMSize: vmSize,
	}

	if osDiskSizeGB > 0 {
		profile.OSDiskSizeGB = osDiskSizeGB
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

	if apimodel.Name != name {
		t.Fatalf("cluster name '%s' not found", name)
	}
	if apimodel.Location != location {
		t.Fatalf("cluster location '%s' not found", location)
	}
}

func TestACSEngineCluster_setProfiles(t *testing.T) {
	dnsPrefix := "lessCreativeMasterDNSPrefix"
	d := mockClusterResourceData("name1", "westus", "testrg", "creativeMasterDNSPrefix")
	cluster := mockContainerService("name2", "southcentralus", dnsPrefix)

	if err := setProfiles(d, cluster); err != nil {
		t.Fatalf("setProfiles failed: %+v", err)
	}
	v, ok := d.GetOk("master_profile.0.dns_name_prefix")
	if !ok {
		t.Fatalf("failed to get 'master_profile.0.dns_name_prefix'")
	}
	if v.(string) != dnsPrefix {
		t.Fatalf("'master_profile.0.dns_name_prefix' is not set correctly - actual: '%s', expected: '%s'", v.(string), dnsPrefix)
	}
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
	if !ok {
		t.Fatalf("failed to get 'master_profile.0.dns_name_prefix'")
	}
	if v.(string) != dnsPrefix {
		t.Fatalf("'master_profile.0.dns_name_prefix' is not set correctly - actual: '%s', expected: '%s'", v.(string), dnsPrefix)
	}
}

func TestACSEngineCluster_setDataSourceProfiles(t *testing.T) {
	dnsPrefix := "lessCreativeMasterDNSPrefix"
	d := mockClusterResourceData("name1", "westus", "testrg", "creativeMasterDNSPrefix")
	cluster := mockContainerService("name2", "southcentralus", dnsPrefix)

	if err := setDataSourceProfiles(d, cluster); err != nil {
		t.Fatalf("setProfiles failed: %+v", err)
	}
	v, ok := d.GetOk("master_profile.0.dns_name_prefix")
	if !ok {
		t.Fatalf("failed to get 'master_profile.0.dns_name_prefix'")
	}
	if v.(string) != dnsPrefix {
		t.Fatalf("'master_profile.0.dns_name_prefix' is not set correctly - actual: '%s', expected: '%s'", v.(string), dnsPrefix)
	}
}
