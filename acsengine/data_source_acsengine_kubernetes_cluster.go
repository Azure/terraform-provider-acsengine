package acsengine

import (
	"fmt"

	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceACSEngineKubernetesCluster() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceACSEngineK8sClusterRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"resource_group": resourceGroupNameForDataSourceSchema(),

			"location": locationForDataSourceSchema(),

			"kubernetes_version": kubernetesVersionForDataSourceSchema(),

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
							Type:     schema.TypeList,
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

			// this will usually not be set, how do I deal with that?
			"windows_profile": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admin_username": {
							Type:     schema.TypeString,
							Computed: true,
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

			"kube_config": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": {
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

			"kube_config_raw": kubeConfigRawSchema(),

			"api_model": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},

			"tags": tagsForDataSourceSchema(),
		},
	}
}

func dataSourceACSEngineK8sClusterRead(data *schema.ResourceData, m interface{}) error {
	d := newResourceData(data)
	client := m.(*ArmClient)
	deployClient := client.deploymentsClient

	var name, resourceGroup, apimodel string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	}
	if v, ok := d.GetOk("resource_group"); ok {
		resourceGroup = v.(string)
	}

	resp, err := deployClient.Get(client.StopContext, resourceGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			return fmt.Errorf("Error: cluster %s (with resource group %s) was not found", name, resourceGroup)
		}
	}
	d.SetId(*resp.ID)

	if err = d.Set("name", name); err != nil {
		return fmt.Errorf("Error setting name: %+v", err)
	}

	if err = d.Set("resource_group", resourceGroup); err != nil {
		return fmt.Errorf("Error setting resource group: %+v", err)
	}

	cluster, err := d.loadContainerServiceFromApimodel(true, false)
	if err != nil {
		return fmt.Errorf("Error parsing API model: %+v", err)
	}

	if err = d.Set("location", azureRMNormalizeLocation(cluster.Location)); err != nil {
		return fmt.Errorf("Error setting location: %+v", err)
	}

	if err = d.Set("kubernetes_version", cluster.Properties.OrchestratorProfile.OrchestratorVersion); err != nil {
		return fmt.Errorf("Error setting kubernetes_version: %+v", err)
	}

	if err := d.setDataSourceStateProfiles(&cluster); err != nil {
		return err
	}

	if err := d.setTags(cluster.Tags); err != nil {
		return err
	}

	if err := d.setKubeConfig(client, &cluster, true); err != nil {
		return err
	}

	// I don't need to do this but it's nice that base64Encode checks if it's encoded yet
	apimodelBase64 := base64Encode(apimodel)
	if err := d.Set("api_model", apimodelBase64); err != nil {
		return fmt.Errorf("Error setting `api_model`: %+v", err)
	}

	fmt.Println("finished reading")

	return nil
}
