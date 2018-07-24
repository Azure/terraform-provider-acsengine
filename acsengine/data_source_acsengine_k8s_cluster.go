package acsengine

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAcsEngineKubernetesCluster() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceACSEngineK8sClusterRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"resource_group": resourceGroupNameForDataSourceSchema(), // type string, required

			"location": locationForDataSourceSchema(),

			"kubernetes_version": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"linux_profile": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admin_username": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ssh": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key_data": { // public SSH key
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},

			"service_principal": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"client_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"master_profile": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"count": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"dns_name_prefix": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"fqdn": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"vm_size": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"os_disk_size": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},

			"agent_pool_profiles": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"count": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"os_disk_size": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"vm_size": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"os_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"tags": tagsForDataSourceSchema(), // probably from tags.go

			"kube_config": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": { // I think this is what was meant by 'host'
							Type:     schema.TypeString,
							Computed: true,
						},
						"username": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"password": {
							Type:      schema.TypeString,
							Computed:  true,
							Sensitive: true,
						},
						"client_certificate": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"client_key": {
							Type:      schema.TypeString,
							Computed:  true,
							Sensitive: true,
						},
						"cluster_ca_certificate": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"kube_config_raw": { // do I need this?
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func dataSourceACSEngineK8sClusterRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*ArmClient)
	deployClient := client.deploymentsClient
	ctx := client.StopContext

	var name, resourceGroup string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	}
	if v, ok := d.GetOk("resource_group"); ok {
		resourceGroup = v.(string)
	}

	// this could be a problem because the deployment name changes
	resp, err := deployClient.Get(ctx, resourceGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			return fmt.Errorf("Error: cluster %s (with resource group %s) was not found", name, resourceGroup)
		}
	}
	d.SetId(*resp.ID)

	if err = d.Set("name", name); err != nil {
		return err
	}
	if err = d.Set("resource_group", resourceGroup); err != nil {
		return err
	}

	apimodel, err := getBlob(d, m, "apimodel.json")
	if err != nil {
		return err
	}

	locale, err := i18n.LoadTranslations()
	if err != nil {
		return err
	}
	apiloader := &api.Apiloader{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}

	cluster, err := apiloader.LoadContainerService([]byte(apimodel), apiVersion, true, false, nil)
	if err != nil {
		return fmt.Errorf("error parsing API model")
	}

	if err = d.Set("location", azureRMNormalizeLocation(cluster.Location)); err != nil {
		return err
	}
	if err = d.Set("kubernetes_version", cluster.Properties.OrchestratorProfile.OrchestratorVersion); err != nil {
		return err
	}

	linuxProfile, err := flattenLinuxProfile(*cluster.Properties.LinuxProfile)
	if err != nil {
		return err
	}
	if err = d.Set("linux_profile", linuxProfile); err != nil {
		return fmt.Errorf("Error setting 'linux_profile': %+v", err)
	}

	servicePrincipal, err := flattenDataSourceServicePrincipal(*cluster.Properties.ServicePrincipalProfile)
	if err != nil {
		return err
	}
	if err = d.Set("service_principal", servicePrincipal); err != nil {
		return fmt.Errorf("Error setting 'service_principal': %+v", err)
	}

	masterProfile, err := flattenMasterProfile(*cluster.Properties.MasterProfile, cluster.Location)
	if err != nil {
		return err
	}
	if err = d.Set("master_profile", masterProfile); err != nil {
		return fmt.Errorf("Error setting 'master_profile': %+v", err)
	}

	agentPoolProfiles, err := flattenAgentPoolProfiles(cluster.Properties.AgentPoolProfiles)
	if err != nil {
		return err
	}
	if err = d.Set("agent_pool_profiles", agentPoolProfiles); err != nil {
		return fmt.Errorf("Error setting 'agent_pool_profiles': %+v", err)
	}

	tags, err := flattenTags(cluster.Tags)
	if err != nil {
		return err
	}
	if err = d.Set("tags", tags); err != nil {
		return fmt.Errorf("Error setting `tags`: %+v", err)
	}

	kubeConfigFile, err := getKubeConfig(cluster)
	if err != nil {
		return err
	}
	kubeConfigRaw, kubeConfig, err := flattenKubeConfig(kubeConfigFile)
	if err != nil {
		return err
	}
	if err = d.Set("kube_config_raw", kubeConfigRaw); err != nil {
		return fmt.Errorf("Error setting `kube_config_raw`: %+v", err)
	}
	if err := d.Set("kube_config", kubeConfig); err != nil {
		return fmt.Errorf("Error setting `kube_config`: %+v", err)
	}

	fmt.Println("finished reading")

	return nil
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
