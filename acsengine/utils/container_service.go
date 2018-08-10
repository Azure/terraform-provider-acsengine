package utils

import (
	"os"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/api/common"
)

// FlattenLinuxProfile creates test linux profile interface
func FlattenLinuxProfile(adminUsername string) []interface{} {
	sshKeys := []interface{}{}
	keys := map[string]interface{}{
		// "key_data": testSSHPublicKey(),
		"key_data": os.Getenv("SSH_KEY_PUB"), // for now
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

// FlattenWindowsProfile creates test windows profile interface
func FlattenWindowsProfile(adminUsername string, adminPassword string) []interface{} {
	values := map[string]interface{}{
		"admin_username": adminUsername,
		"admin_password": adminPassword,
	}
	windowsProfiles := []interface{}{}
	windowsProfiles = append(windowsProfiles, values)

	return windowsProfiles
}

// FlattenServicePrincipal creates test service principal interface
func FlattenServicePrincipal() []interface{} {
	servicePrincipals := []interface{}{}

	spValues := map[string]interface{}{
		// "client_id":     testClientID(),
		"client_id": os.Getenv("ARM_CLIENT_ID"), // for now
		// "client_secret": testClientSecret(),
		"client_secret": os.Getenv("ARM_CLIENT_SECRET"), // for now
	}

	servicePrincipals = append(servicePrincipals, spValues)

	return servicePrincipals
}

// FlattenMasterProfile creates test master profile interface
func FlattenMasterProfile(count int, dnsNamePrefix string, vmSize string) []interface{} {
	masterProfiles := []interface{}{}

	masterProfile := make(map[string]interface{}, 5)

	masterProfile["count"] = count
	masterProfile["dns_name_prefix"] = dnsNamePrefix
	masterProfile["vm_size"] = vmSize
	masterProfile["fqdn"] = "f/q/d/n"

	masterProfiles = append(masterProfiles, masterProfile)

	return masterProfiles
}

// FlattenAgentPoolProfiles creates test agent pool profiles interface
func FlattenAgentPoolProfiles(name string, count int, vmSize string, osDiskSizeGB int, windows bool) map[string]interface{} {
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

// ExpandLinuxProfile creates test api.LinuxProfile
func ExpandLinuxProfile(adminUsername string, keyData string) api.LinuxProfile {
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

// ExpandWindowsProfile creates test api.WindowsProfile
func ExpandWindowsProfile(adminUsername string, adminPassword string) api.WindowsProfile {
	profile := api.WindowsProfile{
		AdminUsername: adminUsername,
		AdminPassword: adminPassword,
	}

	return profile
}

// ExpandServicePrincipal creates test api.ServicePrincipalProfile
func ExpandServicePrincipal(clientID string, clientSecret string) api.ServicePrincipalProfile {
	profile := api.ServicePrincipalProfile{
		ClientID: clientID,
		Secret:   clientSecret,
	}

	return profile
}

// ExpandMasterProfile creates test api.MasterProfile
func ExpandMasterProfile(count int, dnsPrefix string, vmSize string, fqdn string, osDiskSize int) api.MasterProfile {
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

// ExpandAgentPoolProfile creates test api.AgentPoolProfile pointer
func ExpandAgentPoolProfile(name string, count int, vmSize string, osDiskSizeGB int, isWindows bool) *api.AgentPoolProfile {
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

// ExpandCertificateProfile creates test api.CertificateProfile
func ExpandCertificateProfile() api.CertificateProfile {
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

// MockContainerService returns a container service with mostly pre-initialized values
func MockContainerService(name string, location string, dnsPrefix string) *api.ContainerService {
	linuxProfile := ExpandLinuxProfile("azureuser", "public key")
	servicePrincipal := ExpandServicePrincipal(os.Getenv("ARM_CLIENT_ID"), os.Getenv("ARM_CLIENT_SECRET"))
	masterProfile := ExpandMasterProfile(1, dnsPrefix, "Standard_D2_v2", "fqdn", 0)

	agentPoolProfile1 := ExpandAgentPoolProfile("agentpool1", 1, "Standard_D2_v2", 0, false)
	agentPoolProfile2 := ExpandAgentPoolProfile("agentpool2", 2, "Standard_D2_v2", 30, false)
	agentPoolProfiles := []*api.AgentPoolProfile{agentPoolProfile1, agentPoolProfile2}

	orchestratorProfile := api.OrchestratorProfile{
		OrchestratorType:    "Kubernetes",
		OrchestratorVersion: common.GetDefaultKubernetesVersion(),
	}

	certificateProfile := ExpandCertificateProfile()

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
