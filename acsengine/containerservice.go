package acsengine

import (
	"encoding/base64"
	"fmt"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/api/common"
	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/hashicorp/terraform/helper/schema"
)

func flattenLinuxProfile(profile api.LinuxProfile) ([]interface{}, error) {
	adminUsername := profile.AdminUsername
	ssh := profile.SSH
	publicKey := ssh.PublicKeys[0]
	keyData := publicKey.KeyData
	if profile.AdminUsername == "" || keyData == "" {
		return nil, fmt.Errorf("Linux profile not set correctly")
	}

	profiles := []interface{}{}

	values := map[string]interface{}{}

	sshKeys := []interface{}{}

	keys := map[string]interface{}{}
	keys["key_data"] = keyData
	sshKeys = append(sshKeys, keys)

	values["admin_username"] = adminUsername
	values["ssh"] = sshKeys
	profiles = append(profiles, values)

	return profiles, nil
}

func flattenServicePrincipal(profile api.ServicePrincipalProfile) ([]interface{}, error) {
	clientID := profile.ClientID
	clientSecret := profile.Secret
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("Service principal not set correctly")
	}

	profiles := []interface{}{}

	values := map[string]interface{}{}
	values["client_id"] = clientID
	values["client_secret"] = clientSecret

	profiles = append(profiles, values)

	return profiles, nil
}

func flattenMasterProfile(profile api.MasterProfile, location string) ([]interface{}, error) {
	count := profile.Count
	dnsPrefix := profile.DNSPrefix
	vmSize := profile.VMSize
	// format is masterEndpointDNSNamePrefix.location.fqdnEndpointSuffix
	endpointSuffix := "cloudapp.azure.com"
	fqdn := dnsPrefix + "." + location + "." + endpointSuffix
	if count < 1 || dnsPrefix == "" || vmSize == "" {
		return nil, fmt.Errorf("Master profile not set correctly")
	}

	profiles := []interface{}{}

	values := map[string]interface{}{}
	values["count"] = count
	values["dns_name_prefix"] = dnsPrefix
	values["vm_size"] = vmSize
	values["fqdn"] = fqdn
	if profile.OSDiskSizeGB != 0 {
		values["os_disk_size"] = profile.OSDiskSizeGB
	}

	profiles = append(profiles, values)

	return profiles, nil
}

func flattenAgentPoolProfiles(profiles []*api.AgentPoolProfile) ([]interface{}, error) {
	agentPoolProfiles := []interface{}{}

	for _, pf := range profiles {
		profile := *pf
		if profile.Name == "" || profile.Count < 1 || profile.VMSize == "" { // debugging
			return nil, fmt.Errorf("Agent pool profiles not set correctly")
		}
		values := map[string]interface{}{}
		values["name"] = profile.Name
		values["count"] = profile.Count
		values["vm_size"] = profile.VMSize
		if profile.OSDiskSizeGB != 0 {
			values["os_disk_size"] = profile.OSDiskSizeGB
		}
		if profile.OSType != "" {
			values["os_type"] = profile.OSType
		}

		agentPoolProfiles = append(agentPoolProfiles, values)
	}

	return agentPoolProfiles, nil
}

func expandLinuxProfile(d *schema.ResourceData) (api.LinuxProfile, error) {
	var profiles []interface{}
	if v, ok := d.GetOk("linux_profile"); ok {
		profiles = v.([]interface{})
	} else {
		return api.LinuxProfile{}, fmt.Errorf("cluster 'linux_profile' not found")
	}
	config := profiles[0].(map[string]interface{})

	adminUsername := config["admin_username"].(string)
	linuxKeys := config["ssh"].([]interface{})

	sshPublicKeys := []api.PublicKey{}

	key := linuxKeys[0].(map[string]interface{})
	keyData := key["key_data"].(string)

	sshPublicKey := api.PublicKey{
		KeyData: keyData,
	}

	sshPublicKeys = append(sshPublicKeys, sshPublicKey)

	profile := api.LinuxProfile{
		AdminUsername: adminUsername,
		SSH: struct {
			PublicKeys []api.PublicKey `json:"publicKeys"`
		}{
			PublicKeys: sshPublicKeys,
		},
	}

	return profile, nil
}

func expandServicePrincipal(d *schema.ResourceData) (api.ServicePrincipalProfile, error) {
	var configs []interface{}
	if v, ok := d.GetOk("service_principal"); ok {
		configs = v.([]interface{})
	} else {
		return api.ServicePrincipalProfile{}, fmt.Errorf("cluster 'service_principal' not found")
	}
	config := configs[0].(map[string]interface{})

	clientID := config["client_id"].(string)
	clientSecret := config["client_secret"].(string)

	principal := api.ServicePrincipalProfile{
		ClientID: clientID,
		Secret:   clientSecret,
	}

	return principal, nil
}

func expandMasterProfile(d *schema.ResourceData) (api.MasterProfile, error) {
	var configs []interface{}
	if v, ok := d.GetOk("master_profile"); ok {
		configs = v.([]interface{})
	} else {
		return api.MasterProfile{}, fmt.Errorf("cluster 'master_profile' not found")
	}
	config := configs[0].(map[string]interface{})

	count := config["count"].(int)
	dnsPrefix := config["dns_name_prefix"].(string)
	vmSize := config["vm_size"].(string)

	profile := api.MasterProfile{
		Count:     count,
		DNSPrefix: dnsPrefix,
		VMSize:    vmSize,
	}

	if config["os_disk_size"] != nil {
		osDiskSizeGB := config["os_disk_size"].(int)
		profile.OSDiskSizeGB = osDiskSizeGB
	}

	return profile, nil
}

func expandAgentPoolProfiles(d *schema.ResourceData) ([]*api.AgentPoolProfile, error) {
	var configs []interface{}
	if v, ok := d.GetOk("agent_pool_profiles"); ok {
		configs = v.([]interface{})
	} else {
		return []*api.AgentPoolProfile{}, fmt.Errorf("cluster 'agent_pool_profiles' not found")
	}
	profiles := make([]*api.AgentPoolProfile, 0, len(configs))

	for _, c := range configs {
		config := c.(map[string]interface{})
		name := config["name"].(string)
		count := config["count"].(int)
		vmSize := config["vm_size"].(string)
		osType := config["os_type"].(string)

		profile := &api.AgentPoolProfile{
			Name:   name,
			Count:  count,
			VMSize: vmSize,
			OSType: api.OSType(osType),
		}

		if config["os_disk_size"] != nil {
			osDiskSizeGB := config["os_disk_size"].(int)
			profile.OSDiskSizeGB = osDiskSizeGB
		}

		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// I feel kind of funny about having these functions here

func initializeContainerService(d *schema.ResourceData) (*api.ContainerService, error) {
	var name, location, kubernetesVersion string
	var v interface{}
	var ok bool

	if v, ok = d.GetOk("name"); !ok {
		return &api.ContainerService{}, fmt.Errorf("cluster 'name' not found")
	}
	name = v.(string)

	if v, ok = d.GetOk("location"); !ok {
		return &api.ContainerService{}, fmt.Errorf("cluster 'location' not found")
	}
	location = azureRMNormalizeLocation(v.(string)) // from location.go

	if v, ok = d.GetOk("kubernetes_version"); ok {
		kubernetesVersion = v.(string)
	} else {
		kubernetesVersion = common.GetDefaultKubernetesVersion() // will this case ever be needed?
	}

	linuxProfile, err := expandLinuxProfile(d)
	if err != nil {
		return nil, fmt.Errorf("error expanding `linux_profile: %+v`", err)
	}
	servicePrincipal, err := expandServicePrincipal(d)
	if err != nil {
		return nil, fmt.Errorf("error expanding `service_principal: %+v`", err)
	}
	masterProfile, err := expandMasterProfile(d)
	if err != nil {
		return nil, fmt.Errorf("error expanding `master_profile: %+v`", err)
	}
	agentProfiles, err := expandAgentPoolProfiles(d)
	if err != nil {
		return nil, fmt.Errorf("error expanding `agent_pool_profiles: %+v`", err)
	}

	// do I need to add a Windows profile is osType = Windows?
	// adminUser = masterProfile.adminUser
	// adminPassword = ?
	if _, err := createWindowsProfile(); err != nil {
		return nil, err
	}

	tags := getTags(d)

	cluster := &api.ContainerService{
		Name:     name,
		Location: location,
		Properties: &api.Properties{
			LinuxProfile:            &linuxProfile,
			ServicePrincipalProfile: &servicePrincipal,
			MasterProfile:           &masterProfile,
			AgentPoolProfiles:       agentProfiles,
			OrchestratorProfile: &api.OrchestratorProfile{
				OrchestratorType:    "Kubernetes",
				OrchestratorVersion: kubernetesVersion,
			},
		},
		Tags: expandClusterTags(tags),
	}

	return cluster, nil
}

func loadContainerServiceFromApimodel(d *schema.ResourceData, validate bool, isUpdate bool) (*api.ContainerService, error) {
	locale, err := i18n.LoadTranslations()
	if err != nil {
		return &api.ContainerService{}, fmt.Errorf("error loading translations: %+v", err)
	}
	apiloader := &api.Apiloader{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}
	var apimodel []byte
	if v, ok := d.GetOk("api_model"); ok {
		apimodel, err = base64.StdEncoding.DecodeString(v.(string))
		if err != nil {
			return &api.ContainerService{}, fmt.Errorf("error decoding `api_model`: %+v", err)
		}
	}

	cluster, err := apiloader.LoadContainerService(apimodel, apiVersion, validate, isUpdate, nil)
	if err != nil {
		return &api.ContainerService{}, fmt.Errorf("error loading container service from apimodel bytes: %+v", err)
	}

	return cluster, nil
}

func setAPIModel(d *schema.ResourceData, cluster *api.ContainerService) error {
	locale, err := i18n.LoadTranslations()
	if err != nil {
		return fmt.Errorf("error loading translations: %+v", err)
	}

	apiloader := &api.Apiloader{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}
	apimodel, err := apiloader.SerializeContainerService(cluster, apiVersion)
	if err != nil {
		return fmt.Errorf("error serializing API model: %+v", err)
	}
	if err = d.Set("api_model", base64.StdEncoding.EncodeToString(apimodel)); err != nil {
		return fmt.Errorf("error setting API model: %+v", err)
	}

	return nil
}

func createWindowsProfile() (api.WindowsProfile, error) {
	// not implemented yet
	return api.WindowsProfile{}, nil
}
