package acsengine

import (
	"testing"

	"github.com/Azure/acs-engine/pkg/api"
)

func TestACSEngineK8sCluster_flattenLinuxProfile(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flattenLinuxProfile failed")
		}
	}()

	adminUsername := "adminUser"
	keyData := "public key data"
	profile := fakeExpandLinuxProfile(adminUsername, keyData)

	linuxProfile, err := flattenLinuxProfile(profile)
	if err != nil {
		t.Fatalf("flattenLinuxProfile failed: %v", err)
	}

	if len(linuxProfile) != 1 {
		t.Fatalf("flattenLinuxProfile failed: did not find one linux profile")
	}
	linuxPf := linuxProfile[0].(map[string]interface{})
	if val, ok := linuxPf["admin_username"]; ok {
		if val != adminUsername {
			t.Fatalf("flattenLinuxProfile failed: Master count is innaccurate")
		}
	} else {
		t.Fatalf("flattenLinuxProfile failed: Master count does not exist")
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
	profile := fakeExpandServicePrincipal(clientID, clientSecret)

	servicePrincipal, err := flattenServicePrincipal(profile)
	if err != nil {
		t.Fatalf("flattenServicePrincipal failed: %v", err)
	}

	if len(servicePrincipal) != 1 {
		t.Fatalf("flattenServicePrincipal failed: did not find one master profile")
	}
	spPf := servicePrincipal[0].(map[string]interface{})
	if val, ok := spPf["client_id"]; ok {
		if val != clientID {
			t.Fatalf("flattenServicePrincipal failed: Master count is innaccurate")
		}
	} else {
		t.Fatalf("flattenServicePrincipal failed: Master count does not exist")
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
	profile := fakeExpandMasterProfile(count, dnsNamePrefix, vmSize, fqdn)

	masterProfile, err := flattenMasterProfile(profile, "southcentralus")
	if err != nil {
		t.Fatalf("flattenServicePrincipal failed: %v", err)
	}

	if len(masterProfile) != 1 {
		t.Fatalf("flattenMasterProfile failed: did not find one master profile")
	}
	masterPf := masterProfile[0].(map[string]interface{})
	if val, ok := masterPf["count"]; ok {
		if val != int(count) {
			t.Fatalf("flattenMasterProfile failed: Master count is innaccurate")
		}
	} else {
		t.Fatalf("flattenMasterProfile failed: Master count does not exist")
	}
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
	fqdn := "abcdefg"
	osDiskSize := 200

	profile1 := fakeExpandAgentPoolProfile(name, count, vmSize, fqdn, 0)

	name = "agentpool2"
	profile2 := fakeExpandAgentPoolProfile(name, count, vmSize, fqdn, osDiskSize)

	profiles := []*api.AgentPoolProfile{profile1, profile2}
	agentPoolProfiles, err := flattenAgentPoolProfiles(profiles)
	if err != nil {
		t.Fatalf("flattenAgentPoolProfiles failed: %v", err)
	}

	if len(agentPoolProfiles) < 1 {
		t.Fatalf("flattenAgentPoolProfile failed: did not find any agent pool profiles")
	}
	agentPf0 := agentPoolProfiles[0].(map[string]interface{})
	if val, ok := agentPf0["count"]; ok {
		if val.(int) != count {
			t.Fatalf("agent pool count is inaccurate. %d != %d", val.(int), count)
		}
	} else {
		t.Fatalf("agent pool count does not exist")
	}
	if val, ok := agentPf0["os_disk_size"]; ok {
		t.Fatalf("agent pool OS disk size should not be set, but is %d", val.(int))
	}
	agentPf1 := agentPoolProfiles[1].(map[string]interface{})
	if val, ok := agentPf1["name"]; ok {
		if val.(string) != name {
			t.Fatalf("flattenAgentPoolProfile failed: agent pool name is innaccurate. %s != %s.", val, name)
		}
	} else {
		t.Fatalf("flattenAgentPoolProfile failed: agent pool count does not exist")
	}
	if val, ok := agentPf1["os_disk_size"]; ok {
		if val.(int) != osDiskSize {
			t.Fatalf("agent pool os disk size is %d when it should be %d", val.(int), osDiskSize)
		}
	} else {
		t.Fatalf("agent pool os disk size is not set when it should be")
	}
}

func TestACSEngineK8sCluster_expandLinuxProfile(t *testing.T) {
	r := resourceArmAcsEngineKubernetesCluster()
	d := r.TestResourceData()

	adminUsername := "azureuser"
	linuxProfiles := fakeFlattenLinuxProfile(adminUsername)
	d.Set("linux_profile", &linuxProfiles)

	linuxProfile, err := expandLinuxProfile(d)
	if err != nil {
		t.Fatalf("expand linux profile failed: %v", err)
	}

	if linuxProfile.AdminUsername != "azureuser" {
		t.Fatalf("linux profile admin username is not '%s' as expected", adminUsername)
	}
}

func TestACSEngineK8sCluster_expandServicePrincipal(t *testing.T) {
	r := resourceArmAcsEngineKubernetesCluster()
	d := r.TestResourceData()

	clientID := testClientID()
	servicePrincipals := fakeFlattenServicePrincipal()
	d.Set("service_principal", servicePrincipals)

	servicePrincipal, err := expandServicePrincipal(d)
	if err != nil {
		t.Fatalf("expand service principal failed: %v", err)
	}

	if servicePrincipal.ClientID != clientID {
		t.Fatalf("service principal client ID is not '%s' as expected", clientID)
	}
}

func TestACSEngineK8sCluster_expandMasterProfile(t *testing.T) {
	r := resourceArmAcsEngineKubernetesCluster()
	d := r.TestResourceData()

	dnsPrefix := "masterDNSPrefix"
	vmSize := "Standard_D2_v2"
	masterProfiles := fakeFlattenMasterProfile(1, dnsPrefix, vmSize)
	d.Set("master_profile", &masterProfiles)

	masterProfile, err := expandMasterProfile(d)
	if err != nil {
		t.Fatalf("expand master profile failed: %v", err)
	}

	if masterProfile.DNSPrefix != dnsPrefix {
		t.Fatalf("master profile dns prefix is not '%s' as expected", dnsPrefix)
	}
	if masterProfile.VMSize != vmSize {
		t.Fatalf("master profile VM size is not '%s' as expected", vmSize)
	}
}

func TestACSEngineK8sCluster_expandAgentPoolProfiles(t *testing.T) {
	r := resourceArmAcsEngineKubernetesCluster()
	d := r.TestResourceData()

	agentPool1Name := "agentpool1"
	agentPool1Count := 1
	agentPool2Name := "agentpool2"
	agentPool2Count := 2
	agentPool2osDiskSize := 30

	agentPoolProfiles := []interface{}{}
	agentPoolProfile0 := fakeFlattenAgentPoolProfiles(agentPool1Name, agentPool1Count, "Standard_D2_v2", 0, false)
	agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile0)
	agentPoolProfile1 := fakeFlattenAgentPoolProfiles(agentPool2Name, agentPool2Count, "Standard_D2_v2", agentPool2osDiskSize, true)
	agentPoolProfiles = append(agentPoolProfiles, agentPoolProfile1)
	d.Set("agent_pool_profiles", &agentPoolProfiles)

	profiles, err := expandAgentPoolProfiles(d)
	if err != nil {
		t.Fatalf("expand agent pool profiles failed: %v", err)
	}

	if len(profiles) != 2 {
		t.Fatalf("Length of agent pool profiles array is not %d as expected", 2)
	}
	if profiles[0].Name != agentPool1Name {
		t.Fatalf("The first agent pool profile is not named '%s' as expected", agentPool1Name)
	}
	if profiles[0].Count != agentPool1Count {
		t.Fatalf("%s does not have count = %d as expected", agentPool1Name, agentPool1Count)
	}
	if profiles[0].OSDiskSizeGB != 0 {
		t.Fatalf("The first agent pool profile has OSDiskSizeGB = %d when it should not be set", profiles[1].OSDiskSizeGB)
	}
	if profiles[0].OSType != api.Linux {
		t.Fatalf("The first agent pool profile has OS type %s when it should be %s", profiles[0].OSType, api.Linux)
	}
	if profiles[1].Count != agentPool2Count {
		t.Fatalf("%s does not have count = %d as expected", agentPool2Name, agentPool2Count)
	}
	if profiles[1].OSDiskSizeGB != agentPool2osDiskSize {
		t.Fatalf("The second agent pool profile has OSDiskSizeGB = %d when it should not be %d", profiles[1].OSDiskSizeGB, agentPool2osDiskSize)
	}
	if profiles[1].OSType != api.Windows {
		t.Fatalf("The first agent pool profile has OS type %s when it should be %s", profiles[0].OSType, api.Windows)
	}
}

func fakeFlattenLinuxProfile(adminUsername string) []interface{} {
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

func fakeFlattenServicePrincipal() []interface{} {
	servicePrincipals := []interface{}{}

	spValues := map[string]interface{}{
		"client_id":     testClientID(),
		"client_secret": testClientSecret(),
	}

	servicePrincipals = append(servicePrincipals, spValues)

	return servicePrincipals
}

func fakeFlattenMasterProfile(count int, dnsNamePrefix string, vmSize string) []interface{} {
	masterProfiles := []interface{}{}

	masterProfile := make(map[string]interface{}, 5)

	masterProfile["count"] = count
	masterProfile["dns_name_prefix"] = dnsNamePrefix
	masterProfile["vm_size"] = vmSize
	masterProfile["fqdn"] = "f/q/d/n"

	masterProfiles = append(masterProfiles, masterProfile)

	return masterProfiles
}

func fakeFlattenAgentPoolProfiles(name string, count int, vmSize string, osDiskSizeGB int, windows bool) map[string]interface{} {
	agentPoolValues := map[string]interface{}{
		"name":    name,
		"count":   count,
		"vm_size": vmSize,
	}
	if osDiskSizeGB != 0 {
		agentPoolValues["os_disk_size"] = osDiskSizeGB
	}
	if windows {
		agentPoolValues["os_type"] = api.Windows
	} else {
		agentPoolValues["os_type"] = api.Linux
	}

	return agentPoolValues
}

func fakeExpandLinuxProfile(adminUsername string, keyData string) api.LinuxProfile {
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

func fakeExpandServicePrincipal(clientID string, clientSecret string) api.ServicePrincipalProfile {
	profile := api.ServicePrincipalProfile{
		ClientID: clientID,
		Secret:   clientSecret,
	}

	return profile
}

func fakeExpandMasterProfile(count int, dnsPrefix string, vmSize string, fqdn string) api.MasterProfile {
	profile := api.MasterProfile{
		Count:     count,
		DNSPrefix: dnsPrefix,
		VMSize:    vmSize,
		FQDN:      fqdn,
	}

	return profile
}

func fakeExpandAgentPoolProfile(name string, count int, vmSize string, fqdn string, osDiskSizeGB int) *api.AgentPoolProfile {
	profile := &api.AgentPoolProfile{
		Name:   name,
		Count:  count,
		VMSize: vmSize,
		FQDN:   fqdn,
	}

	if osDiskSizeGB > 0 {
		profile.OSDiskSizeGB = osDiskSizeGB
	}

	return profile
}
