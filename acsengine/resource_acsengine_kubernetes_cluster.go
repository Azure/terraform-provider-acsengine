package acsengine

// TO DO
// - fix updateTags
// - add tests that check if cluster is running on nodes
// - use a CI tool in GitHub
// - Write documentation
// - get data source working (read from api model in resource state somehow)
// - OS type
// - make code more unit test-able and write more unit tests (plus clean up ones I have to use mock objects more?)
// - Important: fix dependency problems and use dep when acs-engine has been updated
// - do I need more translations?

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/acs-engine/pkg/acsengine" // make sure I'm using a recent release of acs-engine
	"github.com/Azure/acs-engine/pkg/acsengine/transform"
	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/api/common"
	acseutils "github.com/Azure/acs-engine/pkg/armhelpers/utils"
	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/acs-engine/pkg/operations"
	"github.com/Azure/acs-engine/pkg/operations/kubernetesupgrade"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client" // this is what I want to work
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/kubernetes"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/response"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"     // update version
	"github.com/hashicorp/terraform/helper/validation" // update version
)

func resourceArmAcsEngineKubernetesCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAcsEngineK8sClusterCreate,
		Read:   resourceAcsEngineK8sClusterRead,
		Delete: resourceAcsEngineK8sClusterDelete,
		Update: resourceAcsEngineK8sClusterUpdate,
		// Is importing possible when state is just stored in the state file?
		// Can I define my own function that will set things correctly?
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			// from resource_group_name.go: string, required, force new, and string validation
			"resource_group": resourceGroupNameSchema(),

			"kubernetes_version": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      common.GetDefaultKubernetesVersion(), // default is 1.8.13
				ValidateFunc: validateKubernetesVersion,
			},

			// from location.go: required, force new, and converted to lowercase w/ no spaces
			"location": locationSchema(),

			"linux_profile": {
				Type:     schema.TypeList,
				Required: true, // what about 'generate-ssh-keys' option?
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admin_username": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ssh": {
							Type:     schema.TypeSet,
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
							// looks like I accidentally deleted the hash function, do I care?
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
							ValidateFunc: validateMasterProfileCount, // checks if 1, 3, or 5
						},
						"dns_name_prefix": {
							Type:     schema.TypeString,
							Required: true, // force new?
							ForceNew: true,
						},
						"fqdn": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"vm_size": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          "Standard_DS1_v2",          // used by aks cli as default, I haven't looked into it
							ForceNew:         true,                       // really?
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc, // found in provider.go
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
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc, // found in provider.go
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
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc, // I think this is from provider.go
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

			// from tags.go: map, optional, computed, validated to make sure not too many, too long
			"tags": tagsSchema(),

			"api_model": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

const (
	acsEngineVersion = "0.19.1" // is this completely separate from the package that calls this?
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
	id, err := parseAzureResourceID(d.Id()) // from resourceid.go
	if err != nil {
		d.SetId("")
		return err
	}
	resourceGroup := id.ResourceGroup

	err = d.Set("resource_group", resourceGroup)
	if err != nil {
		return fmt.Errorf("Error setting `resource_group`: %+v", err)
	}

	// load from apimodel or configuration? apimodel is a better depiction of state of cluster
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

	// set linux profile, service principal, master profile, and agent pool profiles
	if err = setProfiles(d, cluster); err != nil {
		return err
	}

	// sets tags
	if err = setTags(d, cluster); err != nil {
		return err
	}

	// set kube config fields
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
	// if the deployment exists, that says something, but how do I check more for kubernetes cluster?
	_, err := parseAzureResourceID(d.Id()) // from resourceid.go
	if err != nil {
		d.SetId("")
		return err
	}

	d.Partial(true)

	// UPGRADE
	if d.HasChange("kubernetes_version") {
		// validate to make sure it's valid and > current version
		old, new := d.GetChange("kubernetes_version")
		if err = validateKubernetesVersionUpgrade(new.(string), old.(string)); err != nil {
			return fmt.Errorf("Error upgrading Kubernetes version: %+v", err)
		}
		if err = upgradeCluster(d, m, new.(string)); err != nil {
			return fmt.Errorf("Error upgrading Kubernetes version: %+v", err)
		}

		d.SetPartial("kubernetes_version")
	}

	// SCALE
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

	// I should make this come first so the tags can be updated at the same time
	// as another re-deployment (if one happens)
	if d.HasChange("tags") {
		// do I need to pass in "new" tags from d.GetChange? I'm pretty sure I don't
		if err = updateTags(d, m); err != nil {
			return fmt.Errorf("Error updating tags: %+v", err)
		}

		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAcsEngineK8sClusterRead(d, m)
}

/* HELPER FUNCTIONS */

/* 'Create' Helper Functions */

// Generates apimodel.json and other templates, saves these files along with certificates
func generateACSEngineTemplate(d *schema.ResourceData, write bool) (template string, parameters string, err error) {
	// create container service struct
	cluster, err := initializeContainerService(d)
	if err != nil {
		return "", "", err
	}

	// initialize values
	locale, err := i18n.LoadTranslations()
	if err != nil {
		return "", "", fmt.Errorf("error loading translation files: %+v", err)
	}
	ctx := acsengine.Context{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}

	// generate template
	templateGenerator, err := acsengine.InitializeTemplateGenerator(ctx, false)
	if err != nil {
		return "", "", fmt.Errorf("failed to initialize template generator: %+v", err)
	}
	template, parameters, certsGenerated, err := templateGenerator.GenerateTemplate(cluster, acsengine.DefaultGeneratorCode, false, false, acsEngineVersion)
	if err != nil {
		return "", "", fmt.Errorf("error generating template: %+v", err)
	}

	// format templates
	template, err = transform.PrettyPrintArmTemplate(template)
	if err != nil {
		return "", "", fmt.Errorf("error pretty printing template: %+v", err)
	}
	parameters, err = transform.BuildAzureParametersFile(parameters)
	if err != nil {
		return "", "", fmt.Errorf("error pretty printing template parameters: %+v", err)
	}

	// save templates
	if write { // this should be default but allow for more testing
		deploymentDirectory := path.Join("_output", cluster.Properties.MasterProfile.DNSPrefix)
		if err = writeTemplatesAndCerts(d, cluster, template, parameters, deploymentDirectory, certsGenerated); err != nil {
			return "", "", fmt.Errorf("error writing templates and certificates: %+v", err)
		}
	}

	return template, parameters, nil
}

// Initializes the acs-engine container service struct using Terraform input
// if update, this could set ca certificate and key
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

	var tags map[string]interface{}
	if v, ok = d.GetOk("tags"); ok {
		tags = v.(map[string]interface{})
	} else {
		tags = map[string]interface{}{}
	}

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

// Loads container service from apimodel JSON
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

// Deploys the templates generated by ACS Engine for creating a cluster
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

	azureDeployTemplate, err := expandTemplateBody(template)
	if err != nil {
		return "", fmt.Errorf("error expanding template body: %+v", err)
	}
	azureDeployParameters, err := expandParametersBody(parameters)
	if err != nil {
		return "", fmt.Errorf("error expanding parameters body: %+v", err)
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

// sets linux profile, service principal, master profile, and agent pool profiles
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

func setTags(d *schema.ResourceData, cluster *api.ContainerService) error {
	tags, err := flattenTags(cluster.Tags)
	if err != nil {
		return fmt.Errorf("Error flattening `tags`: %+v", err)
	}
	if err := d.Set("tags", tags); err != nil {
		return fmt.Errorf("Error setting 'tags': %+v", err)
	}

	return nil
}

// set `kube_config` and `kube_config_raw`
func setKubeConfig(d *schema.ResourceData, cluster *api.ContainerService) error {
	kubeConfigFile, err := getKubeConfig(cluster)
	if err != nil {
		return fmt.Errorf("Error getting kube config: %+v", err)
	}
	kubeConfigRaw, kubeConfig, err := flattenKubeConfig(kubeConfigFile)
	if err != nil {
		return fmt.Errorf("Error flattening kube config: %+v", err)
	}
	if err = d.Set("kube_config_raw", kubeConfigRaw); err != nil {
		return fmt.Errorf("Error setting `kube_config_raw`: %+v", err)
	}
	if err = d.Set("kube_config", kubeConfig); err != nil {
		return fmt.Errorf("Error setting `kube_config`: %+v", err)
	}

	return nil
}

/* 'Update' Helper Functions */

// Creates ScaleClient, loads ACS Engine templates, finds relevant node VM info, calls appropriate function for scaling up or down
func scaleCluster(d *schema.ResourceData, m interface{}, agentIndex int, agentCount int) error {
	// initialize scale client based on most recent values
	sc, err := initializeScaleClient(d, m, agentIndex, agentCount)
	if err != nil {
		return fmt.Errorf("failed to initialize scale client: %+v", err)
	}

	// find all VMs in agent pool
	var currentNodeCount, highestUsedIndex, windowsIndex int
	var indexToVM map[int]string
	if sc.AgentPool.IsAvailabilitySets() {
		if highestUsedIndex, currentNodeCount, windowsIndex, indexToVM, err = scaleVMAS(&sc, d); err != nil {
			return fmt.Errorf("failed to scale availability set: %+v", err)
		}

		if currentNodeCount == sc.DesiredAgentCount {
			log.Printf("Cluster is currently at the desired agent count")
			return nil
		}

		if currentNodeCount > sc.DesiredAgentCount {
			return scaleDownCluster(&sc, currentNodeCount, indexToVM, d)
		}
	} else {
		if highestUsedIndex, currentNodeCount, windowsIndex, err = scaleVMSS(&sc); err != nil {
			return fmt.Errorf("failed to scale scale set: %+v", err)
		}
	}

	return scaleUpCluster(&sc, highestUsedIndex, currentNodeCount, windowsIndex, d)
}

// Creates and initializes most fields in client.ScaleClient and returns it
func initializeScaleClient(d *schema.ResourceData, m interface{}, agentIndex int, agentCount int) (client.ScaleClient, error) {
	sc := client.ScaleClient{}
	var err error
	if v, ok := d.GetOk("resource_group"); ok {
		sc.ResourceGroupName = v.(string)
	}
	if v, ok := d.GetOk("master_profile.0.dns_name_prefix"); ok {
		sc.DeploymentDirectory = path.Join("_output", v.(string))
	}
	sc.DesiredAgentCount = agentCount
	if v, ok := d.GetOk("location"); ok {
		sc.Location = azureRMNormalizeLocation(v.(string))
	}
	if v, ok := d.GetOk("master_profile.0.fqdn"); ok {
		sc.MasterFQDN = v.(string)
	}
	sc.AgentPoolIndex = agentIndex
	if v, ok := d.GetOk("agent_pool_profiles." + strconv.Itoa(agentIndex) + ".name"); ok {
		sc.AgentPoolToScale = v.(string)
	} else {
		return sc, fmt.Errorf("agent pool profile name not found")
	}
	if err := sc.Validate(); err != nil {
		return sc, fmt.Errorf("error validating scale client: %+v", err)
	}

	if err = addScaleAuthArgs(d, &sc); err != nil {
		return sc, fmt.Errorf("failed to add auth args: %+v", err)
	}

	if sc.Locale, err = i18n.LoadTranslations(); err != nil {
		return sc, fmt.Errorf("error loading translation files: %+v", err)
	}
	apiloader := &api.Apiloader{
		Translator: &i18n.Translator{
			Locale: sc.Locale,
		},
	}
	if m != nil { // for testing purposes
		sc.K8sCluster, err = loadContainerServiceFromApimodel(d, true, true)
		if err != nil {
			return sc, fmt.Errorf("error parsing the api model: %+v", err)
		}
	} else {
		sc.APIModelPath = path.Join(sc.DeploymentDirectory, "apimodel.json")
		if _, err = os.Stat(sc.APIModelPath); os.IsNotExist(err) {
			return sc, fmt.Errorf("specified api model does not exist (%s)", sc.APIModelPath)
		}
		sc.K8sCluster, _, err = apiloader.LoadContainerServiceFromFile(sc.APIModelPath, true, true, nil)
		if err != nil {
			return sc, fmt.Errorf("error parsing the api model: %+v", err)
		}

	}
	if sc.K8sCluster.Location != sc.Location {
		return sc, fmt.Errorf("location does not match api model location") // this should probably never happen?
	}
	sc.AgentPool = sc.K8sCluster.Properties.AgentPoolProfiles[sc.AgentPoolIndex]

	sc.NameSuffix = acsengine.GenerateClusterID(sc.K8sCluster.Properties)

	return sc, nil
}

func addScaleAuthArgs(d *schema.ResourceData, sc *client.ScaleClient) error {
	client.AddAuthArgs(&sc.AuthArgs)
	id, err := parseAzureResourceID(d.Id()) // from resourceid.go
	if err != nil {
		return fmt.Errorf("error parsing resource ID: %+v", err)
	}
	sc.AuthArgs.RawSubscriptionID = id.SubscriptionID
	sc.AuthArgs.AuthMethod = "client_secret"
	if v, ok := d.GetOk("service_principal.0.client_id"); ok {
		sc.AuthArgs.RawClientID = v.(string)
	}
	if v, ok := d.GetOk("service_principal.0.client_secret"); ok {
		sc.AuthArgs.ClientSecret = v.(string)
	}
	if err = sc.AuthArgs.ValidateAuthArgs(); err != nil {
		return fmt.Errorf("error validating auth args: %+v", err)
	}

	if sc.Client, err = sc.AuthArgs.GetClient(); err != nil {
		return fmt.Errorf("failed to get client: %+v", err)
	}
	if _, err = sc.Client.EnsureResourceGroup(context.Background(), sc.ResourceGroupName, sc.Location, nil); err != nil {
		return fmt.Errorf("failed to get client: %+v", err)
	}

	return nil
}

// scale VM availability sets
func scaleVMAS(sc *client.ScaleClient, d *schema.ResourceData) (int, int, int, map[int]string, error) {
	var currentNodeCount, highestUsedIndex, vmNum int
	windowsIndex := -1
	highestUsedIndex = 0
	indexes := make([]int, 0)
	indexToVM := make(map[int]string)
	ctx := context.Background()
	vms, err := sc.Client.ListVirtualMachines(ctx, sc.ResourceGroupName)
	if err != nil {
		return highestUsedIndex, currentNodeCount, windowsIndex, indexToVM, fmt.Errorf("failed to get vms in the resource group. Error: %s", err.Error())
	} else if len(vms.Values()) < 1 {
		return highestUsedIndex, currentNodeCount, windowsIndex, indexToVM, fmt.Errorf("The provided resource group does not contain any vms")
	}
	index := 0
	for _, vm := range vms.Values() {
		vmTags := vm.Tags
		poolName := *vmTags["poolName"]
		nameSuf := *vmTags["resourceNameSuffix"]

		if err != nil || !strings.EqualFold(poolName, sc.AgentPoolToScale) || !strings.Contains(sc.NameSuffix, nameSuf) {
			continue
		}

		osPublisher := vm.StorageProfile.ImageReference.Publisher
		if osPublisher != nil && strings.EqualFold(*osPublisher, "MicrosoftWindowsServer") {
			_, _, windowsIndex, vmNum, err = acseutils.WindowsVMNameParts(*vm.Name)
		} else {
			_, _, vmNum, err = acseutils.K8sLinuxVMNameParts(*vm.Name) // this needs to be tested
		}
		if err != nil {
			return highestUsedIndex, currentNodeCount, windowsIndex, indexToVM, fmt.Errorf("error getting VM parts: %+v", err)
		}
		if vmNum > highestUsedIndex {
			highestUsedIndex = vmNum
		}

		indexToVM[index] = *vm.Name
		indexes = append(indexes, index)
		index++
	}
	sortedIndexes := sort.IntSlice(indexes)
	sortedIndexes.Sort()
	indexes = []int(sortedIndexes)
	currentNodeCount = len(indexes)

	return highestUsedIndex, currentNodeCount, windowsIndex, indexToVM, nil
}

// scale VM scale sets
func scaleVMSS(sc *client.ScaleClient) (int, int, int, error) {
	var currentNodeCount, highestUsedIndex int
	windowsIndex := -1
	highestUsedIndex = 0
	ctx := context.Background()
	vmssList, err := sc.Client.ListVirtualMachineScaleSets(ctx, sc.ResourceGroupName)
	if err != nil {
		return highestUsedIndex, currentNodeCount, windowsIndex, fmt.Errorf("failed to get vmss list in the resource group: %+v", err)
	}
	for _, vmss := range vmssList.Values() {
		vmTags := vmss.Tags
		poolName := *vmTags["poolName"]
		nameSuffix := *vmTags["resourceNameSuffix"]

		if err != nil || !strings.EqualFold(poolName, sc.AgentPoolToScale) || !strings.Contains(sc.NameSuffix, nameSuffix) {
			continue
		}

		osPublisher := *vmss.VirtualMachineProfile.StorageProfile.ImageReference.Publisher
		if strings.EqualFold(osPublisher, "MicrosoftWindowsServer") {
			_, _, windowsIndex, err = acseutils.WindowsVMSSNameParts(*vmss.Name)
			// log error here?
		}

		currentNodeCount = int(*vmss.Sku.Capacity)
		highestUsedIndex = 0
	}

	return highestUsedIndex, currentNodeCount, windowsIndex, nil
}

// Scales down a cluster by draining and deleting the nodes given as input
func scaleDownCluster(sc *client.ScaleClient, currentNodeCount int, indexToVM map[int]string, d *schema.ResourceData) error {
	if sc.MasterFQDN == "" {
		return fmt.Errorf("Master FQDN is required to scale down a Kubernetes cluster's agent pool")
	}

	vmsToDelete := make([]string, 0)
	for i := currentNodeCount - 1; i >= sc.DesiredAgentCount; i-- {
		vmsToDelete = append(vmsToDelete, indexToVM[i])
	}

	kubeconfig, err := acsengine.GenerateKubeConfig(sc.K8sCluster.Properties, sc.Location)
	if err != nil {
		return fmt.Errorf("failed to generate kube config: %+v", err)
	}
	if err = sc.DrainNodes(kubeconfig, vmsToDelete); err != nil {
		return fmt.Errorf("Got error while draining the nodes to be deleted: %+v", err)
	}

	errList := operations.ScaleDownVMs(
		sc.Client,
		sc.Logger,
		sc.AuthArgs.SubscriptionID.String(),
		sc.ResourceGroupName,
		vmsToDelete...)
	if errList != nil {
		errorMessage := ""
		for element := errList.Front(); element != nil; element = element.Next() {
			vmError, ok := element.Value.(*operations.VMScalingErrorDetails)
			if ok {
				error := fmt.Sprintf("Node '%s' failed to delete with error: '%s'", vmError.Name, vmError.Error.Error())
				errorMessage = errorMessage + error
			}
		}
		return fmt.Errorf(errorMessage)
	}

	return saveScaledApimodel(sc, d)
}

// Scales up clusters by creating new nodes within an agent pool
func scaleUpCluster(sc *client.ScaleClient, highestUsedIndex int, currentNodeCount int, windowsIndex int, d *schema.ResourceData) error {
	ctx := acsengine.Context{
		Translator: &i18n.Translator{
			Locale: sc.Locale,
		},
	}
	templateGenerator, err := acsengine.InitializeTemplateGenerator(ctx, false) // original uses classic mode
	if err != nil {
		return fmt.Errorf("failed to initialize template generator: %+v", err)
	}

	sc.K8sCluster.Properties.AgentPoolProfiles = []*api.AgentPoolProfile{sc.AgentPool} // how does this work when there's multiple agent pools?

	template, parameters, _, err := templateGenerator.GenerateTemplate(sc.K8sCluster, acsengine.DefaultGeneratorCode, false, true, acsEngineVersion)
	if err != nil {
		return fmt.Errorf("error generating template: %+v", err)
	}

	// format templates
	template, err = transform.PrettyPrintArmTemplate(template)
	if err != nil {
		return fmt.Errorf("error pretty printing template: %+v", err)
	}
	// don't format parameters! It messes things up

	// convert JSON to maps
	templateJSON, err := expandTemplateBody(template)
	if err != nil {
		return fmt.Errorf("error unmarshaling template: %+v", err)
	}
	parametersJSON, err := expandParametersBody(parameters)
	if err != nil {
		return fmt.Errorf("error unmarshaling template: %+v", err)
	}

	transformer := transform.Transformer{Translator: ctx.Translator}
	countForTemplate := sc.DesiredAgentCount
	if highestUsedIndex != 0 { // if not scale set
		countForTemplate += highestUsedIndex + 1 - currentNodeCount
	}
	addValue(parametersJSON, sc.AgentPoolToScale+"Count", countForTemplate)

	if windowsIndex != -1 {
		templateJSON["variables"].(map[string]interface{})[sc.AgentPool.Name+"Index"] = windowsIndex
	}

	if err = transformer.NormalizeForK8sVMASScalingUp(sc.Logger, templateJSON); err != nil {
		return fmt.Errorf("error transforming the template for scaling template: %+v", err)
	}
	if sc.AgentPool.IsAvailabilitySets() {
		addValue(parametersJSON, fmt.Sprintf("%sOffset", sc.AgentPoolToScale), highestUsedIndex+1)
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	deploymentSuffix := random.Int31()

	_, err = sc.Client.DeployTemplate(
		context.Background(),
		sc.ResourceGroupName,
		fmt.Sprintf("%s-%d", sc.ResourceGroupName, deploymentSuffix),
		templateJSON,
		parametersJSON)
	if err != nil {
		return fmt.Errorf("error deploying scaled template: %+v", err)
	}

	return saveScaledApimodel(sc, d)
}

func saveScaledApimodel(sc *client.ScaleClient, d *schema.ResourceData) error {
	var err error
	sc.K8sCluster, err = loadContainerServiceFromApimodel(d, false, true)
	if err != nil {
		return fmt.Errorf("failed to load container service from apimodel: %+v", err)
	}
	sc.K8sCluster.Properties.AgentPoolProfiles[sc.AgentPoolIndex].Count = sc.DesiredAgentCount

	return saveTemplates(sc.K8sCluster, sc.DeploymentDirectory, d)
}

func addValue(params map[string]interface{}, k string, v interface{}) {
	params[k] = map[string]interface{}{
		"value": v,
	}
}

// Upgrades a cluster to a higher Kubernetes version
func upgradeCluster(d *schema.ResourceData, m interface{}, upgradeVersion string) error {
	uc, err := initializeUpgradeClient(d, m, upgradeVersion)
	if err != nil {
		return fmt.Errorf("error initializing upgrade client: %+v", err)
	}

	// I already validated elsewhere, consider deleting
	kubernetesInfo, err := api.GetOrchestratorVersionProfile(uc.K8sCluster.Properties.OrchestratorProfile)
	if err != nil {
		return fmt.Errorf("error getting a list of the available upgrades: %+v", err)
	}
	found := false
	for _, up := range kubernetesInfo.Upgrades { // checking that version I want is within the allowed versions
		if up.OrchestratorVersion == uc.UpgradeVersion {
			uc.K8sCluster.Properties.OrchestratorProfile.OrchestratorVersion = uc.UpgradeVersion
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("version %s is not supported", uc.UpgradeVersion)
	}

	uc.AgentPoolsToUpgrade = []string{}
	for _, agentPool := range uc.K8sCluster.Properties.AgentPoolProfiles {
		uc.AgentPoolsToUpgrade = append(uc.AgentPoolsToUpgrade, agentPool.Name)
	}

	upgradeCluster := kubernetesupgrade.UpgradeCluster{
		Translator: &i18n.Translator{
			Locale: uc.Locale,
		},
		Logger:      uc.Logger,
		Client:      uc.Client,
		StepTimeout: uc.Timeout,
	}

	kubeconfig, err := acsengine.GenerateKubeConfig(uc.K8sCluster.Properties, uc.Location)
	if err != nil {
		return fmt.Errorf("failed to generate kube config: %+v", err)
	}

	err = upgradeCluster.UpgradeCluster(
		uc.AuthArgs.SubscriptionID,
		kubeconfig,
		uc.ResourceGroupName,
		uc.K8sCluster,
		uc.NameSuffix,
		uc.AgentPoolsToUpgrade,
		acsEngineVersion)
	if err != nil {
		return fmt.Errorf("failed to deploy upgraded cluster: %+v", err)
	}

	return saveUpgradedApimodel(&uc, d)
}

func initializeUpgradeClient(d *schema.ResourceData, m interface{}, upgradeVersion string) (client.UpgradeClient, error) {
	uc := client.UpgradeClient{}
	if v, ok := d.GetOk("resource_group"); ok {
		uc.ResourceGroupName = v.(string)
	}
	if v, ok := d.GetOk("master_profile.0.dns_name_prefix"); ok {
		uc.DeploymentDirectory = path.Join("_output", v.(string))
	}
	uc.UpgradeVersion = upgradeVersion
	if v, ok := d.GetOk("location"); ok {
		uc.Location = v.(string)
	}
	uc.TimeoutInMinutes = -1
	err := uc.Validate()
	if err != nil {
		return uc, fmt.Errorf(": %+v", err)
	}

	if err = addUpgradeAuthArgs(d, &uc); err != nil {
		return uc, fmt.Errorf("failure to add auth args: %+v", err)
	}

	apiloader := &api.Apiloader{
		Translator: &i18n.Translator{
			Locale: uc.Locale,
		},
	}
	if m != nil { // for testing purposes
		if uc.K8sCluster, err = loadContainerServiceFromApimodel(d, true, true); err != nil {
			return uc, fmt.Errorf("error parsing the api model: %+v", err)
		}
	} else {
		uc.APIModelPath = path.Join(uc.DeploymentDirectory, "apimodel.json")
		if _, err = os.Stat(uc.APIModelPath); os.IsNotExist(err) {
			return uc, fmt.Errorf("specified api model does not exist (%s)", uc.APIModelPath)
		}
		uc.K8sCluster, uc.APIVersion, err = apiloader.LoadContainerServiceFromFile(uc.APIModelPath, true, true, nil) // look into these parameters
		if err != nil {
			return uc, fmt.Errorf("error parsing the api model: %+v", err)
		}
	}
	if uc.K8sCluster.Location == "" { // not sure if this block is necessary, might only matter if people are messing w/ the apimodel
		uc.K8sCluster.Location = uc.Location
	} else if uc.K8sCluster.Location != uc.Location {
		return uc, fmt.Errorf("location does not match api model location") // this should probably never happen?
	}

	uc.NameSuffix = acsengine.GenerateClusterID(uc.K8sCluster.Properties)

	return uc, nil
}

func addUpgradeAuthArgs(d *schema.ResourceData, uc *client.UpgradeClient) error {
	client.AddAuthArgs(&uc.AuthArgs)
	id, err := parseAzureResourceID(d.Id()) // from resourceid.go
	if err != nil {
		return fmt.Errorf("error paring resource ID: %+v", err)
	}
	uc.AuthArgs.RawSubscriptionID = id.SubscriptionID
	uc.AuthArgs.AuthMethod = "client_secret"
	if v, ok := d.GetOk("service_principal.0.client_id"); ok {
		uc.AuthArgs.RawClientID = v.(string)
	}
	if v, ok := d.GetOk("service_principal.0.client_secret"); ok {
		uc.AuthArgs.ClientSecret = v.(string)
	}
	if err = uc.AuthArgs.ValidateAuthArgs(); err != nil {
		return fmt.Errorf("error validating auth args: %+v", err)
	}

	if uc.Client, err = uc.AuthArgs.GetClient(); err != nil {
		return fmt.Errorf("failed to get client: %+v", err)
	}
	if _, err = uc.Client.EnsureResourceGroup(context.Background(), uc.ResourceGroupName, uc.Location, nil); err != nil {
		return fmt.Errorf("error ensuring resource group: %+v", err)
	}

	return nil
}

func saveUpgradedApimodel(uc *client.UpgradeClient, d *schema.ResourceData) error {
	return saveTemplates(uc.K8sCluster, uc.DeploymentDirectory, d)
}

// not working yet
func updateTags(d *schema.ResourceData, m interface{}) error {
	// get new tags
	var tags map[string]interface{}
	if v, ok := d.GetOk("tags"); ok {
		tags = v.(map[string]interface{})
		fmt.Println(tags)
	} else {
		tags = map[string]interface{}{}
	}

	// get cluster apimodel
	locale, err := i18n.LoadTranslations()
	if err != nil {
		return fmt.Errorf("error loading translations: %+v", err)
	}
	cluster, err := loadContainerServiceFromApimodel(d, true, false)
	if err != nil {
		return fmt.Errorf("error parsing API model: %+v", err)
	}

	// set tags
	cluster.Tags = expandClusterTags(tags)

	// get templates
	ctx := acsengine.Context{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}
	deploymentDirectory := path.Join("_output", cluster.Properties.MasterProfile.DNSPrefix)
	templateGenerator, err := acsengine.InitializeTemplateGenerator(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to initialize template generator: %+v", err)
	}
	template, parameters, _, err := templateGenerator.GenerateTemplate(cluster, acsengine.DefaultGeneratorCode, false, false, acsEngineVersion)
	if err != nil {
		return fmt.Errorf("error generating templates: %+v", err)
	}

	template, err = transform.PrettyPrintArmTemplate(template)
	if err != nil {
		return fmt.Errorf("error pretty printing template: %+v", err)
	}
	parameters, err = transform.BuildAzureParametersFile(parameters)
	if err != nil {
		return fmt.Errorf("error building azure parameters file: %+v", err)
	}

	// deploy templates
	if _, err = deployTemplate(d, m, template, parameters); err != nil {
		return fmt.Errorf("failed to deploy updated tags template: %+v", err)
	}

	// do I really want to generate these templates all over again?
	return saveTemplates(cluster, deploymentDirectory, d)
}

// Save templates and certificates based on cluster struct
func saveTemplates(cluster *api.ContainerService, deploymentDirectory string, d *schema.ResourceData) error {
	locale, err := i18n.LoadTranslations()
	if err != nil {
		return fmt.Errorf("error loading translations: %+v", err)
	}

	ctx := acsengine.Context{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}

	// generate template
	templateGenerator, err := acsengine.InitializeTemplateGenerator(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to initialize template generator: %+v", err)
	}
	template, parameters, certsGenerated, err := templateGenerator.GenerateTemplate(cluster, acsengine.DefaultGeneratorCode, false, false, acsEngineVersion)
	if err != nil {
		return fmt.Errorf("error generating templates: %+v", err)
	}

	// format templates
	template, err = transform.PrettyPrintArmTemplate(template)
	if err != nil {
		return fmt.Errorf("error pretty printing template: %+v", err)
	}
	parameters, err = transform.BuildAzureParametersFile(parameters)
	if err != nil {
		return fmt.Errorf("error pretty printing template parameters: %+v", err)
	}

	// save templates and certificates
	return writeTemplatesAndCerts(d, cluster, template, parameters, deploymentDirectory, certsGenerated)
}

/* Misc. Helper Functions */

func writeTemplatesAndCerts(d *schema.ResourceData, cluster *api.ContainerService, template string, parameters string, deploymentDirectory string, certsGenerated bool) error {
	locale, err := i18n.LoadTranslations()
	if err != nil {
		return fmt.Errorf("error loading translations: %+v", err)
	}

	// save templates and certificates
	writer := &acsengine.ArtifactWriter{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}
	if err = writer.WriteTLSArtifacts(cluster, apiVersion, template, parameters, deploymentDirectory, certsGenerated, false); err != nil {
		return fmt.Errorf("error writing artifacts: %+v", err)
	}

	// set "api_model" to string of file
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

	sshKeys := &schema.Set{
		F: resourceLinuxProfilesSSHKeysHash,
	}

	keys := map[string]interface{}{}
	keys["key_data"] = keyData
	sshKeys.Add(keys)

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

func flattenTags(tags map[string]string) (map[string]interface{}, error) {
	output := make(map[string]interface{}, len(tags))

	for tag, val := range tags {
		output[tag] = val
	}

	return output, nil
}

func getKubeConfig(cluster *api.ContainerService) (string, error) {
	// maybe check that this is the same function being used when generating all of the templates
	// convert to base64?
	kubeConfig, err := acsengine.GenerateKubeConfig(cluster.Properties, cluster.Location)
	if err != nil {
		return "", fmt.Errorf("failed to generate kube config: %+v", err)
	}
	return kubeConfig, nil
}

func flattenKubeConfig(kubeConfigFile string) (string, []interface{}, error) {
	// Do I actually want all of this to be base64 encoded? I'm confused
	rawKubeConfig := base64Encode(kubeConfigFile)

	config, err := kubernetes.ParseKubeConfig(kubeConfigFile)
	if err != nil {
		return "", nil, fmt.Errorf("error parsing kube config: %+v", err)
	}

	kubeConfig := []interface{}{}
	cluster := config.Clusters[0].Cluster
	user := config.Users[0].User
	name := config.Users[0].Name

	values := map[string]interface{}{}
	values["host"] = cluster.Server
	values["username"] = name
	values["password"] = user.Token
	values["client_certificate"] = base64Encode(user.ClientCertificteData)
	values["client_key"] = base64Encode(user.ClientKeyData)
	values["cluster_ca_certificate"] = base64Encode(cluster.ClusterAuthorityData)

	kubeConfig = append(kubeConfig, values)

	return rawKubeConfig, kubeConfig, nil
}

func createWindowsProfile() (api.WindowsProfile, error) {
	// not implemented yet
	return api.WindowsProfile{}, nil
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
	linuxKeys := config["ssh"].(*schema.Set).List()

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

func expandTemplateBody(template string) (map[string]interface{}, error) {
	templateBody, err := expandBody(template)
	if err != nil {
		return nil, fmt.Errorf("error expanding the template_body for Azure RM Template Deployment: %+v", err)
	}
	return templateBody, nil
}

func expandParametersBody(parameters string) (map[string]interface{}, error) {
	parametersBody, err := expandBody(parameters)
	if err != nil {
		return nil, fmt.Errorf("error expanding the parameters_body for Azure RM Template Deployment: %+v", err)
	}
	return parametersBody, nil
}

func expandBody(body string) (map[string]interface{}, error) {
	var bodyMap map[string]interface{}
	if err := json.Unmarshal([]byte(body), &bodyMap); err != nil {
		return nil, err
	}
	return bodyMap, nil
}

// from resource_arm_container_service.go
func resourceLinuxProfilesSSHKeysHash(v interface{}) int {
	var buf bytes.Buffer

	if m, ok := v.(map[string]interface{}); ok {
		buf.WriteString(fmt.Sprintf("%s-", m["key_data"].(string)))
	}

	return hashcode.String(buf.String())
}
