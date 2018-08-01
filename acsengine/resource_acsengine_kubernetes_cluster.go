package acsengine

// TO DO
// - fix updateTags
// - add tests that check if cluster is running on nodes (I can basically only check if cluster API is there...)
// - use a CI tool in GitHub (seems to be mostly working, now I just need a successful build with acceptance tests)
// - Write documentation
// - add code coverage
// - make code more unit test-able and write more unit tests (plus clean up ones I have to use mock objects more?)
// - Important: fix dependency problems and use dep when acs-engine has been updated - DONE but update when acs-engine version has my change
// - do I need more translations?
// - get data source working (read from api model in resource state somehow)
// - OS type
// - make sure DataDisk.CreateOption problem is still sorted out
// - refactor: better organization of functions, get rid of code duplication, inheritance where it makes sense, better function/variable naming
// - ask about additions to acs-engine: doesn't seem to allow tagging deployment, weird index problem

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/acs-engine/pkg/acsengine"
	"github.com/Azure/acs-engine/pkg/acsengine/transform"
	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/api/common"
	acseutils "github.com/Azure/acs-engine/pkg/armhelpers/utils"
	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/acs-engine/pkg/operations"
	"github.com/Azure/acs-engine/pkg/operations/kubernetesupgrade"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/kubernetes"
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

	if err = setTags(d, cluster); err != nil {
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

	// UPGRADE
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

	// currently just adding to resource group, add to deployment or VMs as well?
	if d.HasChange("tags") {
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

func generateACSEngineTemplate(d *schema.ResourceData, write bool) (template string, parameters string, err error) {
	cluster, err := initializeContainerService(d)
	if err != nil {
		return "", "", err
	}

	locale, err := i18n.LoadTranslations()
	if err != nil {
		return "", "", fmt.Errorf("error loading translation files: %+v", err)
	}
	ctx := acsengine.Context{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}

	templateGenerator, err := acsengine.InitializeTemplateGenerator(ctx, false)
	if err != nil {
		return "", "", fmt.Errorf("failed to initialize template generator: %+v", err)
	}
	template, parameters, certsGenerated, err := templateGenerator.GenerateTemplate(cluster, acsengine.DefaultGeneratorCode, false, false, acsEngineVersion)
	if err != nil {
		return "", "", fmt.Errorf("error generating template: %+v", err)
	}

	template, err = transform.PrettyPrintArmTemplate(template)
	if err != nil {
		return "", "", fmt.Errorf("error pretty printing template: %+v", err)
	}
	parameters, err = transform.BuildAzureParametersFile(parameters)
	if err != nil {
		return "", "", fmt.Errorf("error pretty printing template parameters: %+v", err)
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
	sc, err := initializeScaleClient(d, m, agentIndex, agentCount)
	if err != nil {
		return fmt.Errorf("failed to initialize scale client: %+v", err)
	}

	var currentNodeCount, highestUsedIndex, windowsIndex int
	var indexToVM []string
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
		sc.Cluster, err = loadContainerServiceFromApimodel(d, true, true)
		if err != nil {
			return sc, fmt.Errorf("error parsing the api model: %+v", err)
		}
	} else {
		sc.APIModelPath = path.Join(sc.DeploymentDirectory, "apimodel.json")
		if _, err = os.Stat(sc.APIModelPath); os.IsNotExist(err) {
			return sc, fmt.Errorf("specified api model does not exist (%s)", sc.APIModelPath)
		}
		sc.Cluster, _, err = apiloader.LoadContainerServiceFromFile(sc.APIModelPath, true, true, nil)
		if err != nil {
			return sc, fmt.Errorf("error parsing the api model: %+v", err)
		}

	}
	if sc.Cluster.Location != sc.Location {
		return sc, fmt.Errorf("location does not match api model location") // this should probably never happen?
	}
	sc.AgentPool = sc.Cluster.Properties.AgentPoolProfiles[sc.AgentPoolIndex]

	sc.NameSuffix = acsengine.GenerateClusterID(sc.Cluster.Properties)

	return sc, nil
}

func addScaleAuthArgs(d *schema.ResourceData, sc *client.ScaleClient) error {
	client.AddAuthArgs(&sc.AuthArgs)
	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return fmt.Errorf("error parsing resource ID: %+v", err)
	}
	sc.RawSubscriptionID = id.SubscriptionID
	sc.AuthMethod = "client_secret"
	if v, ok := d.GetOk("service_principal.0.client_id"); ok {
		sc.RawClientID = v.(string)
	}
	if v, ok := d.GetOk("service_principal.0.client_secret"); ok {
		sc.ClientSecret = v.(string)
	}
	if err = sc.ValidateAuthArgs(); err != nil {
		return fmt.Errorf("error validating auth args: %+v", err)
	}

	if sc.Client, err = sc.GetClient(); err != nil {
		return fmt.Errorf("failed to get client: %+v", err)
	}
	if _, err = sc.Client.EnsureResourceGroup(context.Background(), sc.ResourceGroupName, sc.Location, nil); err != nil {
		return fmt.Errorf("failed to get client: %+v", err)
	}

	return nil
}

// scale VM availability sets
func scaleVMAS(sc *client.ScaleClient, d *schema.ResourceData) (int, int, int, []string, error) {
	var currentNodeCount, highestUsedIndex, vmNum int
	windowsIndex := -1
	highestUsedIndex = 0
	indexToVM := make([]string, 0)
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

		indexToVM = append(indexToVM, *vm.Name)
		index++
	}
	currentNodeCount = len(indexToVM)

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
func scaleDownCluster(sc *client.ScaleClient, currentNodeCount int, indexToVM []string, d *schema.ResourceData) error {
	if sc.MasterFQDN == "" {
		return fmt.Errorf("Master FQDN is required to scale down a Kubernetes cluster's agent pool")
	}

	vmsToDelete := make([]string, 0)
	for i := currentNodeCount - 1; i >= sc.DesiredAgentCount; i-- {
		vmsToDelete = append(vmsToDelete, indexToVM[i])
	}

	kubeconfig, err := acsengine.GenerateKubeConfig(sc.Cluster.Properties, sc.Location)
	if err != nil {
		return fmt.Errorf("failed to generate kube config: %+v", err)
	}
	if err = sc.DrainNodes(kubeconfig, vmsToDelete); err != nil {
		return fmt.Errorf("Got error while draining the nodes to be deleted: %+v", err)
	}

	errList := operations.ScaleDownVMs(
		sc.Client,
		sc.Logger,
		sc.SubscriptionID.String(),
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

	sc.Cluster.Properties.AgentPoolProfiles = []*api.AgentPoolProfile{sc.AgentPool} // how does this work when there's multiple agent pools?

	template, parameters, _, err := templateGenerator.GenerateTemplate(sc.Cluster, acsengine.DefaultGeneratorCode, false, true, acsEngineVersion)
	if err != nil {
		return fmt.Errorf("error generating template: %+v", err)
	}

	template, err = transform.PrettyPrintArmTemplate(template)
	if err != nil {
		return fmt.Errorf("error pretty printing template: %+v", err)
	}
	// don't format parameters! It messes things up

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
	sc.Cluster, err = loadContainerServiceFromApimodel(d, false, true)
	if err != nil {
		return fmt.Errorf("failed to load container service from apimodel: %+v", err)
	}
	sc.Cluster.Properties.AgentPoolProfiles[sc.AgentPoolIndex].Count = sc.DesiredAgentCount

	return saveTemplates(d, sc.Cluster, sc.DeploymentDirectory)
}

// Upgrades a cluster to a higher Kubernetes version
func upgradeCluster(d *schema.ResourceData, m interface{}, upgradeVersion string) error {
	uc, err := initializeUpgradeClient(d, m, upgradeVersion)
	if err != nil {
		return fmt.Errorf("error initializing upgrade client: %+v", err)
	}

	// I already validated elsewhere, consider deleting
	kubernetesInfo, err := api.GetOrchestratorVersionProfile(uc.Cluster.Properties.OrchestratorProfile)
	if err != nil {
		return fmt.Errorf("error getting a list of the available upgrades: %+v", err)
	}
	found := false
	for _, up := range kubernetesInfo.Upgrades { // checking that version I want is within the allowed versions
		if up.OrchestratorVersion == uc.UpgradeVersion {
			uc.Cluster.Properties.OrchestratorProfile.OrchestratorVersion = uc.UpgradeVersion
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("version %s is not supported", uc.UpgradeVersion)
	}

	uc.AgentPoolsToUpgrade = []string{}
	for _, agentPool := range uc.Cluster.Properties.AgentPoolProfiles {
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

	kubeconfig, err := acsengine.GenerateKubeConfig(uc.Cluster.Properties, uc.Location)
	if err != nil {
		return fmt.Errorf("failed to generate kube config: %+v", err)
	}

	err = upgradeCluster.UpgradeCluster(
		uc.SubscriptionID,
		kubeconfig,
		uc.ResourceGroupName,
		uc.Cluster,
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
		if uc.Cluster, err = loadContainerServiceFromApimodel(d, true, true); err != nil {
			return uc, fmt.Errorf("error parsing the api model: %+v", err)
		}
	} else {
		uc.APIModelPath = path.Join(uc.DeploymentDirectory, "apimodel.json")
		if _, err = os.Stat(uc.APIModelPath); os.IsNotExist(err) {
			return uc, fmt.Errorf("specified api model does not exist (%s)", uc.APIModelPath)
		}
		uc.Cluster, uc.APIVersion, err = apiloader.LoadContainerServiceFromFile(uc.APIModelPath, true, true, nil) // look into these parameters
		if err != nil {
			return uc, fmt.Errorf("error parsing the api model: %+v", err)
		}
	}
	if uc.Cluster.Location == "" { // not sure if this block is necessary, might only matter if people are messing w/ the apimodel
		uc.Cluster.Location = uc.Location
	} else if uc.Cluster.Location != uc.Location {
		return uc, fmt.Errorf("location does not match api model location") // this should probably never happen?
	}

	uc.NameSuffix = acsengine.GenerateClusterID(uc.Cluster.Properties)

	return uc, nil
}

func addUpgradeAuthArgs(d *schema.ResourceData, uc *client.UpgradeClient) error {
	client.AddAuthArgs(&uc.AuthArgs)
	id, err := parseAzureResourceID(d.Id()) // from resourceid.go
	if err != nil {
		return fmt.Errorf("error paring resource ID: %+v", err)
	}
	uc.RawSubscriptionID = id.SubscriptionID
	uc.AuthMethod = "client_secret"
	if v, ok := d.GetOk("service_principal.0.client_id"); ok {
		uc.RawClientID = v.(string)
	}
	if v, ok := d.GetOk("service_principal.0.client_secret"); ok {
		uc.ClientSecret = v.(string)
	}
	if err = uc.ValidateAuthArgs(); err != nil {
		return fmt.Errorf("error validating auth args: %+v", err)
	}

	if uc.Client, err = uc.GetClient(); err != nil {
		return fmt.Errorf("failed to get client: %+v", err)
	}
	if _, err = uc.Client.EnsureResourceGroup(context.Background(), uc.ResourceGroupName, uc.Location, nil); err != nil {
		return fmt.Errorf("error ensuring resource group: %+v", err)
	}

	return nil
}

func saveUpgradedApimodel(uc *client.UpgradeClient, d *schema.ResourceData) error {
	return saveTemplates(d, uc.Cluster, uc.DeploymentDirectory)
}

// only updates resource group tags
func updateTags(d *schema.ResourceData, m interface{}) error {
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

// Save templates and certificates based on cluster struct
func saveTemplates(d *schema.ResourceData, cluster *api.ContainerService, deploymentDirectory string) error {
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
	if err = writeTemplatesAndCerts(cluster, template, parameters, deploymentDirectory, certsGenerated); err != nil {
		return fmt.Errorf("error writing templates and certificates: %+v", err)
	}
	if err = setAPIModel(d, cluster); err != nil {
		return fmt.Errorf("error setting API model: %+v", err)
	}

	return nil
}

func setAPIModel(d *schema.ResourceData, cluster *api.ContainerService) error {
	locale, err := i18n.LoadTranslations()
	if err != nil {
		return fmt.Errorf("error loading translations: %+v", err)
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

func getKubeConfig(cluster *api.ContainerService) (string, error) {
	kubeConfig, err := acsengine.GenerateKubeConfig(cluster.Properties, cluster.Location)
	if err != nil {
		return "", fmt.Errorf("failed to generate kube config: %+v", err)
	}
	return kubeConfig, nil
}

func flattenKubeConfig(kubeConfigFile string) (string, []interface{}, error) {
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
