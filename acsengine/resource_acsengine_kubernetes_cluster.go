package acsengine

// TO DO
// - add tests that check if cluster is running on nodes (I can basically only check if cluster API is there...)
// - use a CI tool in GitHub (seems to be mostly working, now I just need a successful build with acceptance tests)
// - Write documentation
// - add code coverage
// - make code more unit test-able and write more unit tests (plus clean up ones I have to use mock objects more?)
// - Important: fix dependency problems and use dep when acs-engine has been updated - DONE but update when acs-engine version has my change
// - do I need more translations?
// - get data source working (read from api model in resource state somehow)
// - OS type
// - refactor: better organization of functions, get rid of code duplication, inheritance where it makes sense, better function/variable naming
// - ask about additions to acs-engine: doesn't seem to allow tagging deployment, weird index problem

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/response"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceArmAcsEngineKubernetesCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAcsEngineK8sClusterCreate,
		Read:   resourceAcsEngineK8sClusterRead,
		Delete: resourceAcsEngineK8sClusterDelete,
		Update: resourceAcsEngineK8sClusterUpdate,
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
							ForceNew:         true, // really?
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

			"kube_config_raw": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},

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
	acsEngineVersion = "0.20.2" // is this completely separate from the package that calls this?
	apiVersion       = "vlabs"
)

/* CRUD operations for resource */

func resourceAcsEngineK8sClusterCreate(d *schema.ResourceData, m interface{}) error {
	/* 1. Create resource group */
	err := createClusterResourceGroup(d, m)
	if err != nil {
		return fmt.Errorf("Failed to create resource group: %+v", err)
	}

	/* 2. Generate template w/ acs-engine */
	template, parameters, err := generateACSEngineTemplate(d, true)
	if err != nil {
		return fmt.Errorf("Failed to generate ACS Engine template: %+v", err)
	}

	/* 3. Deploy template using AzureRM */
	id, err := deployTemplate(d, m, template, parameters)
	if err != nil {
		return fmt.Errorf("Failed to deploy template: %+v", err)
	}

	d.SetId(id)

	return resourceAcsEngineK8sClusterRead(d, m)
}

func resourceAcsEngineK8sClusterRead(d *schema.ResourceData, m interface{}) error {
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		d.SetId("")
		return err
	}
	resourceGroup := id.ResourceGroup

	err = d.Set("resource_group", resourceGroup)
	if err != nil {
		return fmt.Errorf("Error setting `resource_group`: %+v", err)
	}

	cluster, err := loadContainerServiceFromApimodel(d, true, false)
	if err != nil {
		return fmt.Errorf("error parsing API model: %+v", err)
	}

	err = d.Set("name", cluster.Name)
	if err != nil {
		return fmt.Errorf("error setting `name`: %+v", err)
	}
	err = d.Set("location", azureRMNormalizeLocation(cluster.Location))
	if err != nil {
		return fmt.Errorf("error setting `location`: %+v", err)
	}
	err = d.Set("kubernetes_version", cluster.Properties.OrchestratorProfile.OrchestratorVersion)
	if err != nil {
		return fmt.Errorf("error setting `kubernetes_version`: %+v", err)
	}

	if err = setProfiles(d, cluster); err != nil {
		return err
	}

	if err = setTags(d, cluster.Tags); err != nil {
		return err
	}

	if err = setKubeConfig(d, cluster); err != nil {
		return err
	}

	// set apimodel here? doesn't really make sense if I'm using that to set everything

	fmt.Println("finished reading")

	return nil
}

func resourceAcsEngineK8sClusterDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*ArmClient)
	rgClient := client.resourceGroupsClient
	ctx := client.StopContext

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return fmt.Errorf("Error parsing Azure Resource ID %q: %+v", d.Id(), err)
	}

	resourceGroupName := id.ResourceGroup

	deleteFuture, err := rgClient.Delete(ctx, resourceGroupName)
	if err != nil {
		if response.WasNotFound(deleteFuture.Response()) {
			return nil
		}

		return fmt.Errorf("Error deleting Resource Group %q: %+v", resourceGroupName, err)
	}

	err = deleteFuture.WaitForCompletion(ctx, rgClient.Client)
	if err != nil {
		if response.WasNotFound(deleteFuture.Response()) {
			return nil
		}

		return fmt.Errorf("Error deleting Resource Group %q: %+v", resourceGroupName, err)
	}

	return nil
}

func resourceAcsEngineK8sClusterUpdate(d *schema.ResourceData, m interface{}) error {
	_, err := parseAzureResourceID(d.Id())
	if err != nil {
		d.SetId("")
		return err
	}

	d.Partial(true)

	/* UPGRADE */
	if d.HasChange("kubernetes_version") {
		old, new := d.GetChange("kubernetes_version")
		if err = validateKubernetesVersionUpgrade(new.(string), old.(string)); err != nil {
			return fmt.Errorf("Error upgrading Kubernetes version: %+v", err)
		}
		if err = upgradeCluster(d, m, new.(string)); err != nil {
			return fmt.Errorf("Error upgrading Kubernetes version: %+v", err)
		}

		d.SetPartial("kubernetes_version")
	}

	/* SCALE */
	agentPoolProfiles := d.Get("agent_pool_profiles").([]interface{})
	for i := 0; i < len(agentPoolProfiles); i++ {
		profileCount := "agent_pool_profiles." + strconv.Itoa(i) + ".count"
		if d.HasChange(profileCount) {
			count := d.Get(profileCount).(int)
			if err = scaleCluster(d, m, i, count); err != nil {
				return fmt.Errorf("Error scaling agent pool: %+v", err)
			}
		}

		d.SetPartial(profileCount)
	}

	if d.HasChange("tags") {
		if err = updateResourceGroupTags(d, m); err != nil {
			return fmt.Errorf("Error updating tags: %+v", err)
		}

		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAcsEngineK8sClusterRead(d, m)
}

/* HELPER FUNCTIONS */

/* 'Create' Helper Functions */

func generateACSEngineTemplate(d *schema.ResourceData, write bool) (template string, parameters string, err error) {
	cluster, err := initializeContainerService(d)
	if err != nil {
		return "", "", err
	}

	template, parameters, certsGenerated, err := formatTemplates(cluster)
	if err != nil {
		return "", "", fmt.Errorf("failed to format templates using cluster: %+v", err)
	}

	if write { // this should be default but allow for more testing
		deploymentDirectory := path.Join("_output", cluster.Properties.MasterProfile.DNSPrefix)
		if err = writeTemplatesAndCerts(cluster, template, parameters, deploymentDirectory, certsGenerated); err != nil {
			return "", "", fmt.Errorf("error writing templates and certificates: %+v", err)
		}
	}
	if err = setAPIModel(d, cluster); err != nil {
		return "", "", fmt.Errorf("error setting API model: %+v", err)
	}

	return template, parameters, nil
}

func deployTemplate(d *schema.ResourceData, m interface{}, template string, parameters string) (id string, err error) {
	client := m.(*ArmClient)
	deployClient := client.deploymentsClient
	ctx := client.StopContext

	var name, resourceGroup string
	var v interface{}
	var ok bool

	if v, ok = d.GetOk("name"); !ok {
		return "", fmt.Errorf("cluster 'name' not found")
	}
	name = v.(string)

	if v, ok = d.GetOk("resource_group"); !ok {
		return "", fmt.Errorf("cluster 'resource_group' not found")
	}
	resourceGroup = v.(string)

	azureDeployTemplate, azureDeployParameters, err := expandTemplates(template, parameters)
	if err != nil {
		return "", fmt.Errorf("failed to expand template and parameters: %+v", err)
	}

	properties := resources.DeploymentProperties{
		Mode:       resources.Incremental,
		Parameters: azureDeployParameters["parameters"],
		Template:   azureDeployTemplate,
	}

	deployment := resources.Deployment{
		Properties: &properties,
	}

	future, err := deployClient.CreateOrUpdate(ctx, resourceGroup, name, deployment)
	if err != nil {
		return "", fmt.Errorf("error creating deployment: %+v", err)
	}
	fmt.Println("Deployment created (1)")

	if err = future.WaitForCompletion(ctx, deployClient.Client); err != nil {
		return "", fmt.Errorf("error creating deployment: %+v", err)
	}
	fmt.Println("Deployment created (2)")

	read, err := deployClient.Get(ctx, resourceGroup, name)
	if err != nil {
		return "", fmt.Errorf("error getting deployment: %+v", err)
	}
	if read.ID == nil {
		return "", fmt.Errorf("Cannot read ACS Engine Kubernetes cluster deployment %s (resource group %s) ID", name, resourceGroup)
	}
	log.Printf("[INFO] cluster %q ID: %q", name, *read.ID)

	return *read.ID, nil
}

/* 'Read' Helper Functions */

func setProfiles(d *schema.ResourceData, cluster *api.ContainerService) error {
	linuxProfile, err := flattenLinuxProfile(*cluster.Properties.LinuxProfile)
	if err != nil {
		return fmt.Errorf("Error flattening `linux_profile`: %+v", err)
	}
	if err = d.Set("linux_profile", linuxProfile); err != nil {
		return fmt.Errorf("Error setting 'linux_profile': %+v", err)
	}

	servicePrincipal, err := flattenServicePrincipal(*cluster.Properties.ServicePrincipalProfile)
	if err != nil {
		return fmt.Errorf("Error flattening `service_principal`: %+v", err)
	}
	if err = d.Set("service_principal", servicePrincipal); err != nil {
		return fmt.Errorf("Error setting 'service_principal': %+v", err)
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

/* 'Update' Helper Functions */

// only updates resource group tags
func updateResourceGroupTags(d *schema.ResourceData, m interface{}) error {
	if err := createClusterResourceGroup(d, m); err != nil { // this should update... let's see if it works
		return fmt.Errorf("failed to update resource group: %+v", err)
	}

	// do I want to tag deployment as well?

	tags := getTags(d)

	cluster, err := loadContainerServiceFromApimodel(d, true, false)
	if err != nil {
		return fmt.Errorf("error parsing API model: %+v", err)
	}
	cluster.Tags = expandClusterTags(tags)

	deploymentDirectory := path.Join("_output", cluster.Properties.MasterProfile.DNSPrefix)

	return saveTemplates(d, cluster, deploymentDirectory)
}

/* Misc. Helper Functions */

func saveTemplates(d *schema.ResourceData, cluster *api.ContainerService, deploymentDirectory string) error {
	template, parameters, certsGenerated, err := formatTemplates(cluster)
	if err != nil {
		return fmt.Errorf("failed to format templates: %+v", err)
	}

	// save templates and certificates
	if err = writeTemplatesAndCerts(cluster, template, parameters, deploymentDirectory, certsGenerated); err != nil {
		return fmt.Errorf("error writing templates and certificates: %+v", err)
	}
	if err = setAPIModel(d, cluster); err != nil {
		return fmt.Errorf("error setting API model: %+v", err)
	}

	return nil
}

// the ID passed will be a string of format "AZURE_RESOURCE_ID*space*APIMODEL_DIRECTORY"
func resourceACSEngineK8sClusterImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	client := m.(*ArmClient)
	deployClient := client.deploymentsClient
	ctx := client.StopContext

	azureID, deploymentDirectory, err := parseImportID(d.Id())
	if err != nil {
		return nil, err
	}

	id, err := parseAzureResourceID(azureID)
	if err != nil {
		return nil, err
	}
	name := id.Path["deployments"]
	if name == "" {
		name = id.Path["Deployments"]
	}
	resourceGroup := id.ResourceGroup

	read, err := deployClient.Get(ctx, resourceGroup, name)
	if err != nil {
		return nil, fmt.Errorf("error getting deployment: %+v", err)
	}
	if read.ID == nil {
		return nil, fmt.Errorf("Cannot read ACS Engine Kubernetes cluster deployment %s (resource group %s) ID", name, resourceGroup)
	}
	log.Printf("[INFO] cluster %q ID: %q", name, *read.ID)

	d.SetId(*read.ID)

	apimodel, err := getAPIModelFromFile(deploymentDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to get apimodel.json: %+v", err)
	}
	d.Set("api_model", apimodel)

	return []*schema.ResourceData{d}, nil
}

func parseImportID(dID string) (string, string, error) {
	input := strings.Split(dID, " ")
	if len(input) != 2 {
		return "", "", fmt.Errorf("")
	}

	azureID := input[0]
	deploymentDirectory := input[1]

	return azureID, deploymentDirectory, nil
}

func getAPIModelFromFile(deploymentDirectory string) (string, error) {
	APIModelPath := path.Join(deploymentDirectory, "apimodel.json")
	if _, err := os.Stat(APIModelPath); os.IsNotExist(err) {
		return "", fmt.Errorf("specified api model does not exist (%s)", APIModelPath)
	}
	f, err := os.Open(APIModelPath)
	if err != nil {
		return "", fmt.Errorf("")
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %+v", err)
	}
	apimodel := base64Encode(string(b))

	return apimodel, nil
}
