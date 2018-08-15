package acsengine

import (
	"encoding/base64"
	"fmt"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/api/common"
	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/hashicorp/terraform/helper/schema"
)

// ResourceData ...
type ResourceData struct {
	*schema.ResourceData
}

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

func flattenWindowsProfile(profile *api.WindowsProfile) ([]interface{}, error) {
	if profile == nil {
		return []interface{}{}, nil
	}
	adminUsername := profile.AdminUsername
	adminPassword := profile.AdminPassword
	if profile.AdminUsername == "" || adminPassword == "" {
		return nil, fmt.Errorf("Windows profile not set correctly")
	}

	profiles := []interface{}{}

	values := map[string]interface{}{}
	values["admin_username"] = adminUsername
	values["admin_password"] = adminPassword
	profiles = append(profiles, values)

	return profiles, nil
}

func flattenServicePrincipal(profile api.ServicePrincipalProfile) ([]interface{}, error) {
	clientID := profile.ClientID
	keyVaultSecretRef := profile.KeyvaultSecretRef
	if keyVaultSecretRef == nil {
		return nil, fmt.Errorf("Key vault secret ref should be set")
	}
	vaultID := keyVaultSecretRef.VaultID
	secretName := keyVaultSecretRef.SecretName
	if clientID == "" || vaultID == "" || secretName == "" {
		return nil, fmt.Errorf("Service principal not set correctly")
	}

	profiles := []interface{}{}

	values := map[string]interface{}{}
	values["client_id"] = clientID
	values["vault_id"] = vaultID
	values["secret_name"] = secretName

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
			values["os_type"] = string(profile.OSType)
		}

		agentPoolProfiles = append(agentPoolProfiles, values)
	}

	return agentPoolProfiles, nil
}

func flattenDataSourceServicePrincipal(profile api.ServicePrincipalProfile) ([]interface{}, error) {
	clientID := profile.ClientID
	if clientID == "" {
		return nil, fmt.Errorf("Service principal not set correctly")
	}

	profiles := []interface{}{}

	values := map[string]interface{}{}
	values["client_id"] = clientID

	profiles = append(profiles, values)

	return profiles, nil
}

func expandLinuxProfile(d *ResourceData) (api.LinuxProfile, error) {
	var profiles []interface{}
	v, ok := d.GetOk("linux_profile")
	if !ok {
		return api.LinuxProfile{}, fmt.Errorf("cluster 'linux_profile' not found")
	}
	profiles = v.([]interface{})
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

func expandWindowsProfile(d *ResourceData) (*api.WindowsProfile, error) {
	var profiles []interface{}
	v, ok := d.GetOk("windows_profile")
	if !ok { // don't return error because this shows there's no Windows profile
		return nil, nil
	}
	profiles = v.([]interface{})
	config := profiles[0].(map[string]interface{})

	adminUsername := config["admin_username"].(string)
	adminPassword := config["admin_password"].(string)

	profile := &api.WindowsProfile{
		AdminUsername: adminUsername,
		AdminPassword: adminPassword,
	}

	return profile, nil
}

func expandServicePrincipal(d *ResourceData) (api.ServicePrincipalProfile, error) {
	var configs []interface{}
	v, ok := d.GetOk("service_principal")
	if !ok {
		return api.ServicePrincipalProfile{}, fmt.Errorf("cluster 'service_principal' not found")
	}
	configs = v.([]interface{})
	config := configs[0].(map[string]interface{})

	clientID := config["client_id"].(string)
	vaultID := config["vault_id"].(string)
	secretName := config["secret_name"].(string)
	// secretVersion := config["version"].(string)

	principal := api.ServicePrincipalProfile{
		ClientID: clientID,
		KeyvaultSecretRef: &api.KeyvaultSecretRef{
			VaultID:    vaultID,
			SecretName: secretName,
			// SecretVersion: version,
		},
	}

	return principal, nil
}

func expandMasterProfile(d *ResourceData) (api.MasterProfile, error) {
	var configs []interface{}
	v, ok := d.GetOk("master_profile")
	if !ok {
		return api.MasterProfile{}, fmt.Errorf("cluster 'master_profile' not found")
	}
	configs = v.([]interface{})
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

func expandAgentPoolProfiles(d *ResourceData) ([]*api.AgentPoolProfile, error) {
	var configs []interface{}
	v, ok := d.GetOk("agent_pool_profiles")
	if !ok {
		return []*api.AgentPoolProfile{}, fmt.Errorf("cluster 'agent_pool_profiles' not found")
	}
	configs = v.([]interface{})
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

func newResourceData(data *schema.ResourceData) *ResourceData {
	return &ResourceData{
		ResourceData: data,
	}
}

func (d *ResourceData) setContainerService() (Cluster, error) {
	var name, location, resourceGroup, kubernetesVersion string
	var v interface{}
	var ok bool

	if v, ok = d.GetOk("name"); !ok {
		return Cluster{}, fmt.Errorf("cluster 'name' not found")
	}
	name = v.(string)

	if v, ok = d.GetOk("location"); !ok {
		return Cluster{}, fmt.Errorf("cluster 'location' not found")
	}
	location = azureRMNormalizeLocation(v.(string)) // from location.go

	if v, ok = d.GetOk("resource_group"); !ok {
		return Cluster{}, fmt.Errorf("cluster 'resource_group' not found")
	}
	resourceGroup = v.(string)

	if v, ok = d.GetOk("kubernetes_version"); ok {
		kubernetesVersion = v.(string)
	} else {
		kubernetesVersion = common.GetDefaultKubernetesVersion() // will this case ever be needed?
	}

	linuxProfile, err := expandLinuxProfile(d)
	if err != nil {
		return Cluster{}, fmt.Errorf("error expanding `linux_profile: %+v`", err)
	}
	windowsProfile, err := expandWindowsProfile(d)
	if err != nil {
		return Cluster{}, fmt.Errorf("error expanding `windows_profile: %+v`", err)
	}
	servicePrincipal, err := expandServicePrincipal(d)
	if err != nil {
		return Cluster{}, fmt.Errorf("error expanding `service_principal: %+v`", err)
	}
	masterProfile, err := expandMasterProfile(d)
	if err != nil {
		return Cluster{}, fmt.Errorf("error expanding `master_profile: %+v`", err)
	}
	agentProfiles, err := expandAgentPoolProfiles(d)
	if err != nil {
		return Cluster{}, fmt.Errorf("error expanding `agent_pool_profiles: %+v`", err)
	}

	tags := d.getTags()

	cluster := Cluster{
		ContainerService: &api.ContainerService{
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
		},
		ResourceGroup: resourceGroup,
	}

	if windowsProfile != nil {
		cluster.Properties.WindowsProfile = windowsProfile
	}

	return cluster, nil
}

func (d *ResourceData) loadContainerServiceFromApimodel(validate, isUpdate bool) (Cluster, error) {
	locale, err := i18n.LoadTranslations()
	if err != nil {
		return Cluster{}, fmt.Errorf("error loading translations: %+v", err)
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
			return Cluster{}, fmt.Errorf("error decoding `api_model`: %+v", err)
		}
	}

	cluster := Cluster{}
	cluster.ContainerService, err = apiloader.LoadContainerService(apimodel, apiVersion, validate, isUpdate, nil)
	if err != nil {
		return Cluster{}, fmt.Errorf("error loading container service from apimodel bytes: %+v", err)
	}

	// make sure the location is normalized

	return cluster, nil
}

func (d *ResourceData) setStateAPIModel(cluster *Cluster) error {
	locale, err := i18n.LoadTranslations()
	if err != nil {
		return fmt.Errorf("error loading translations: %+v", err)
	}

	apiloader := &api.Apiloader{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}
	apimodel, err := apiloader.SerializeContainerService(cluster.ContainerService, apiVersion)
	if err != nil {
		return fmt.Errorf("error serializing API model: %+v", err)
	}
	if err = d.Set("api_model", base64.StdEncoding.EncodeToString(apimodel)); err != nil {
		return fmt.Errorf("error setting API model: %+v", err)
	}

	return nil
}

func (d *ResourceData) setStateProfiles(cluster *Cluster) error {
	linuxProfile, err := flattenLinuxProfile(*cluster.Properties.LinuxProfile)
	if err != nil {
		return fmt.Errorf("Error flattening `linux_profile`: %+v", err)
	}
	if err = d.Set("linux_profile", linuxProfile); err != nil {
		return fmt.Errorf("Error setting 'linux_profile': %+v", err)
	}

	windowsProfile, err := flattenWindowsProfile(cluster.Properties.WindowsProfile)
	if err != nil {
		return fmt.Errorf("Error flattening `windows_profile`: %+v", err)
	}
	if len(windowsProfile) > 0 {
		if err = d.Set("windows_profile", windowsProfile); err != nil {
			return fmt.Errorf("Error setting 'windows_profile': %+v", err)
		}
	}

	masterProfile, err := flattenMasterProfile(*cluster.Properties.MasterProfile, cluster.Location)
	if err != nil {
		return fmt.Errorf("Error flattening `master_profile`: %+v", err)
	}
	if err = d.Set("master_profile", masterProfile); err != nil {
		return fmt.Errorf("Error setting 'master_profile': %+v", err)
	}

	agentPoolProfiles, err := flattenAgentPoolProfiles(cluster.Properties.AgentPoolProfiles)
	if err != nil {
		return fmt.Errorf("Error flattening `agent_pool_profiles`: %+v", err)
	}
	if err = d.Set("agent_pool_profiles", agentPoolProfiles); err != nil {
		return fmt.Errorf("Error setting 'agent_pool_profiles': %+v", err)
	}

	return nil
}

func (d *ResourceData) setResourceStateProfiles(cluster *Cluster) error {
	if err := d.setStateProfiles(cluster); err != nil {
		return err
	}

	servicePrincipal, err := flattenServicePrincipal(*cluster.Properties.ServicePrincipalProfile)
	if err != nil {
		return fmt.Errorf("Error flattening `service_principal`: %+v", err)
	}
	if err = d.Set("service_principal", servicePrincipal); err != nil {
		return fmt.Errorf("Error setting 'service_principal': %+v", err)
	}

	return nil
}

func (d *ResourceData) setDataSourceStateProfiles(cluster *Cluster) error {
	if err := d.setStateProfiles(cluster); err != nil {
		return err
	}

	servicePrincipal, err := flattenDataSourceServicePrincipal(*cluster.Properties.ServicePrincipalProfile)
	if err != nil {
		return fmt.Errorf("Error flattening `service_principal`: %+v", err)
	}
	if err = d.Set("service_principal", servicePrincipal); err != nil {
		return fmt.Errorf("Error setting 'service_principal': %+v", err)
	}

	return nil
}
