package acsengine

// TO DO
// - move as much of upgrade and scale into the client package I made and make them struct methods
// - change import (and maybe other things?) to depend on deployment template which I should be able to get from DeploymentExtended in azure-sdk-for-go
// - Keep improving documentation
// - add code coverage
// - make code more unit test-able and write more unit tests (plus clean up ones I have to use mock objects more?)
// - do I need more translations?
// - refactor: better organization of functions, get rid of code duplication, inheritance where it makes sense, better function/variable naming
// - ask about additions to acs-engine: doesn't seem to allow tagging deployment, weird index problem
// - create a new struct for api.ContainerService so I can write methods for it?

import (
	"fmt"
	"strconv"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/kubernetes"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/response"
	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceArmACSEngineKubernetesCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceACSEngineK8sClusterCreate,
		Read:   resourceACSEngineK8sClusterRead,
		Delete: resourceACSEngineK8sClusterDelete,
		Update: resourceACSEngineK8sClusterUpdate,
		Importer: &schema.ResourceImporter{
			State: resourceACSEngineK8sClusterImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group": resourceGroupNameSchema(),

			"kubernetes_version": kubernetesVersionSchema(),

			"location": locationSchema(),

			"linux_profile": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admin_username": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ssh": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key_data": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},

			"windows_profile": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admin_username": {
							Type:     schema.TypeString,
							Required: true,
						},
						"admin_password": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},
					},
				},
			},

			"service_principal": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"client_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"client_secret": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},
					},
				},
			},

			"master_profile": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"count": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ForceNew:     true,
							ValidateFunc: validateMasterProfileCount,
						},
						"dns_name_prefix": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"fqdn": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"vm_size": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          "Standard_DS1_v2",
							ForceNew:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
						},
						"os_disk_size": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"agent_pool_profiles": {
				Type:     schema.TypeList, // may need to keep list sorted
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"count": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ValidateFunc: validateAgentPoolProfileCount,
						},
						"vm_size": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          "Standard_DS1_v2",
							ForceNew:         true,
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
						},
						"os_disk_size": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"os_type": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  api.Linux,
							ValidateFunc: validation.StringInSlice([]string{
								string(api.Linux),
								string(api.Windows),
							}, true),
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
						},
					},
				},
			},

			"kube_config": {
				Type:     schema.TypeList,
				Computed: true,
				MaxItems: 1,
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
				Computed:  true,
				Sensitive: true,
			},

			"tags": tagsSchema(),
		},
	}
}

const (
	acsEngineVersion = "0.20.4"
	apiVersion       = "vlabs"
)

func resourceACSEngineK8sClusterCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*ArmClient)

	err := createClusterResourceGroup(d, client)
	if err != nil {
		return fmt.Errorf("failed to create resource group: %+v", err)
	}

	template, parameters, err := generateACSEngineTemplate(d, true)
	if err != nil {
		return fmt.Errorf("failed to generate ACS Engine template: %+v", err)
	}

	id, err := deployTemplate(d, client, template, parameters)
	if err != nil {
		return fmt.Errorf("failed to deploy template: %+v", err)
	}

	d.SetId(id)

	return resourceACSEngineK8sClusterRead(d, m)
}

func resourceACSEngineK8sClusterRead(d *schema.ResourceData, m interface{}) error {
	id, err := utils.ParseAzureResourceID(d.Id())
	if err != nil {
		d.SetId("")
		return err
	}

	if err = d.Set("resource_group", id.ResourceGroup); err != nil {
		return fmt.Errorf("error setting `resource_group`: %+v", err)
	}

	cluster, err := loadContainerServiceFromApimodel(d, true, false)
	if err != nil {
		return fmt.Errorf("error parsing API model: %+v", err)
	}

	if err = d.Set("name", cluster.Name); err != nil {
		return fmt.Errorf("error setting `name`: %+v", err)
	}
	if err = d.Set("location", azureRMNormalizeLocation(cluster.Location)); err != nil {
		return fmt.Errorf("error setting `location`: %+v", err)
	}
	if err = d.Set("kubernetes_version", cluster.Properties.OrchestratorProfile.OrchestratorVersion); err != nil {
		return fmt.Errorf("error setting `kubernetes_version`: %+v", err)
	}

	if err = setResourceProfiles(d, cluster); err != nil {
		return err
	}

	if err = setTags(d, cluster.Tags); err != nil {
		return err
	}

	if err = setKubeConfig(d, cluster); err != nil {
		return err
	}

	fmt.Println("finished reading")

	return nil
}

func resourceACSEngineK8sClusterDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*ArmClient)
	rgClient := client.resourceGroupsClient
	ctx := client.StopContext

	id, err := utils.ParseAzureResourceID(d.Id())
	if err != nil {
		return fmt.Errorf("error parsing Azure Resource ID %q: %+v", d.Id(), err)
	}

	deleteFuture, err := rgClient.Delete(ctx, id.ResourceGroup)
	if err != nil {
		if response.WasNotFound(deleteFuture.Response()) {
			return nil
		}

		return fmt.Errorf("error deleting Resource Group %q: %+v", id.ResourceGroup, err)
	}

	err = deleteFuture.WaitForCompletion(ctx, rgClient.Client)
	if err != nil {
		if response.WasNotFound(deleteFuture.Response()) {
			return nil
		}

		return fmt.Errorf("error deleting Resource Group %q: %+v", id.ResourceGroup, err)
	}

	return nil
}

func resourceACSEngineK8sClusterUpdate(d *schema.ResourceData, m interface{}) error {
	_, err := utils.ParseAzureResourceID(d.Id())
	if err != nil {
		d.SetId("")
		return err
	}

	c := m.(*ArmClient)

	d.Partial(true)

	if d.HasChange("kubernetes_version") {
		old, new := d.GetChange("kubernetes_version")
		if err = kubernetes.ValidateKubernetesVersionUpgrade(new.(string), old.(string)); err != nil {
			return fmt.Errorf("error upgrading Kubernetes version: %+v", err)
		}
		if err = upgradeCluster(d, new.(string)); err != nil {
			return fmt.Errorf("error upgrading Kubernetes version: %+v", err)
		}

		d.SetPartial("kubernetes_version")
	}

	agentPoolProfiles := d.Get("agent_pool_profiles").([]interface{})
	for i := 0; i < len(agentPoolProfiles); i++ {
		profileCount := "agent_pool_profiles." + strconv.Itoa(i) + ".count"
		if d.HasChange(profileCount) {
			v, ok := d.GetOk(profileCount)
			if !ok {
				return fmt.Errorf("")
			}
			count := v.(int)
			if err = scaleCluster(d, i, count); err != nil {
				return fmt.Errorf("error scaling agent pool: %+v", err)
			}
		}

		d.SetPartial(profileCount)
	}

	if d.HasChange("tags") {
		if err = updateResourceGroupTags(d, c); err != nil {
			return fmt.Errorf("error updating tags: %+v", err)
		}

		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceACSEngineK8sClusterRead(d, m)
}
