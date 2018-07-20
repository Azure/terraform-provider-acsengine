package acsengine

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	"github.com/Azure/acs-engine/pkg/client"
	// "github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client"
	"github.com/Azure/acs-engine/pkg/i18n"
	"github.com/Azure/acs-engine/pkg/operations"
	"github.com/Azure/acs-engine/pkg/operations/kubernetesupgrade"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
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
							ValidateFunc: validation.IntBetween(1, 100),
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
		},
	}
}

const (
	acsEngineVersion = "0.19.1"
	apiVersion       = "vlabs"
)

/* CRUD operations for resource */

// how can I make sure that when VM is created, deleted, and created again the disk is reused
// I need to make sure this still works even when optional values are not given
func resourceAcsEngineK8sClusterCreate(d *schema.ResourceData, m interface{}) error {
	/* 1. Create resource group */
	err := createClusterResourceGroup(d, m)
	if err != nil {
		return err
	}

	/* 2. Create storage account */
	err = createClusterStorageAccount(d, m)
	if err != nil {
		return err
	}

	/* 3. Create storage container */
	err = createStorageContainer(d, m)
	if err != nil {
		return err
	}

	/* 4. Generate template w/ acs-engine */
	template, parameters, err := generateACSEngineTemplate(d, m, true)
	if err != nil {
		return err
	}

	/* 5. Deploy template using AzureRM */
	id, err := deployTemplate(d, m, template, parameters)
	if err != nil {
		return err
	}

	d.SetId(id)

	return resourceAcsEngineK8sClusterRead(d, m)
}

func resourceAcsEngineK8sClusterRead(d *schema.ResourceData, m interface{}) error {
	// make sure cluster exists first?
	id, err := parseAzureResourceID(d.Id()) // from resourceid.go
	if err != nil {
		return err
	}
	resourceGroup := id.ResourceGroup

	d.Set("resource_group", resourceGroup)

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

	d.Set("name", cluster.Name)
	d.Set("location", cluster.Location)
	d.Set("kubernetes_version", cluster.Properties.OrchestratorProfile.OrchestratorVersion)

	linuxProfile, err := flattenLinuxProfile(*cluster.Properties.LinuxProfile)
	if err != nil {
		return err
	}
	if err := d.Set("linux_profile", linuxProfile); err != nil {
		return fmt.Errorf("Error setting 'linux_profile': %+v", err)
	}

	servicePrincipal, err := flattenServicePrincipal(*cluster.Properties.ServicePrincipalProfile)
	if err != nil {
		return err
	}
	if err := d.Set("service_principal", servicePrincipal); err != nil {
		return fmt.Errorf("Error setting 'service_principal': %+v", err)
	}

	masterProfile, err := flattenMasterProfile(*cluster.Properties.MasterProfile, cluster.Location)
	if err != nil {
		return err
	}
	if err := d.Set("master_profile", masterProfile); err != nil {
		return fmt.Errorf("Error setting 'master_profile': %+v", err)
	}

	agentPoolProfiles, err := flattenAgentPoolProfiles(cluster.Properties.AgentPoolProfiles)
	if err != nil {
		return err
	}
	if err := d.Set("agent_pool_profiles", agentPoolProfiles); err != nil {
		return fmt.Errorf("Error setting 'agent_pool_profiles': %+v", err)
	}

	tags, err := flattenTags(cluster.Tags)
	if err != nil {
		return err
	}
	d.Set("tags", tags)

	kubeConfigFile, err := getKubeConfig(cluster)
	if err != nil {
		return err
	}
	kubeConfigRaw, kubeConfig, err := flattenKubeConfig(kubeConfigFile)
	if err != nil {
		return err
	}
	d.Set("kube_config_raw", kubeConfigRaw)
	if err := d.Set("kube_config", kubeConfig); err != nil {
		return fmt.Errorf("Error setting `kube_config`: %+v", err)
	}

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
	// check that cluster exists? Not so sure I need this, if read is called before
	// if the deployment exists, that says something, but how do I check more for kubernetes cluster?
	// generate template again? do I want to make sure all templates are updated, not just apimodel.json?
	_, err := parseAzureResourceID(d.Id()) // from resourceid.go
	if err != nil {
		return err
	}

	d.Partial(true)

	// UPGRADE
	if d.HasChange("kubernetes_version") {
		// validate to make sure it's valid and > current version
		old, new := d.GetChange("kubernetes_version")
		if err = validateKubernetesVersionUpgrade(new.(string), old.(string)); err != nil {
			return fmt.Errorf("Error updating Kubernetes version: %+v", err)
		}
		err = upgradeCluster(d, m, new.(string))
		if err != nil {
			return err
		}

		d.SetPartial("kubernetes_version")
	}

	// SCALE
	agentPoolProfiles := d.Get("agent_pool_profiles").([]interface{})
	for i := 0; i < len(agentPoolProfiles); i++ {
		profileCount := "agent_pool_profiles." + strconv.Itoa(i) + ".count"
		if d.HasChange(profileCount) {
			count := d.Get(profileCount).(int)
			err = scaleCluster(d, m, i, count)
			if err != nil {
				return fmt.Errorf("Error scaling agent pool: %+v", err)
			}
		}

		d.SetPartial(profileCount)
	}

	if d.HasChange("tags") {
		err = updateTags(d, m)
		if err != nil {
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
func generateACSEngineTemplate(d *schema.ResourceData, m interface{}, write bool) (template string, parameters string, err error) {
	var name string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return "", "", fmt.Errorf("cluster 'name' not found")
	}
	var location string // from location.go
	if v, ok := d.GetOk("location"); ok {
		location = azureRMNormalizeLocation(v.(string))
	} else {
		return "", "", fmt.Errorf("cluster 'location' not found")
	}
	var kubernetesVersion string
	if v, ok := d.GetOk("kubernetes_version"); ok {
		kubernetesVersion = v.(string)
	} else {
		kubernetesVersion = common.GetDefaultKubernetesVersion() // will this case ever be needed?
	}

	linuxProfile, err := expandLinuxProfile(d)
	if err != nil {
		return "", "", err
	}
	servicePrincipal, err := expandServicePrincipal(d)
	if err != nil {
		return "", "", err
	}
	masterProfile, err := expandMasterProfile(d)
	if err != nil {
		return "", "", err
	}
	agentProfiles, err := expandAgentPoolProfiles(d)
	if err != nil {
		return "", "", err
	}

	// do I need to add a Windows profile is osType = Windows?
	// adminUser = masterProfile.adminUser
	// adminPassword = ?

	var tags map[string]interface{}
	if v, ok := d.GetOk("tags"); ok {
		tags = v.(map[string]interface{})
	} else {
		tags = map[string]interface{}{}
	}

	cluster := api.ContainerService{
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

	locale, err := i18n.LoadTranslations()
	if err != nil {
		return "", "", fmt.Errorf(fmt.Sprintf("error loading translation files: %s", err.Error()))
	}

	ctx := acsengine.Context{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}

	templateGenerator, err := acsengine.InitializeTemplateGenerator(ctx, false)
	if err != nil {
		return "", "", fmt.Errorf("failed to initialize template generator: %s", err.Error())
	}

	template, parameters, certsGenerated, err := templateGenerator.GenerateTemplate(&cluster, acsengine.DefaultGeneratorCode, false, acsEngineVersion)
	if err != nil {
		return "", "", fmt.Errorf("error generating template: %s", err.Error()) // is there a way I can say which terraform file change caused this?
	}

	if template, err = transform.PrettyPrintArmTemplate(template); err != nil {
		return "", "", fmt.Errorf("error pretty printing template: %s", err.Error())
	}
	if parameters, err = transform.BuildAzureParametersFile(parameters); err != nil {
		return "", "", fmt.Errorf("error pretty printing template parameters: %s", err.Error())
	}

	if write { // this should be default but allow for more testing
		writer := &acsengine.ArtifactWriter{
			Translator: &i18n.Translator{
				Locale: locale,
			},
		}
		deploymentDirectory := "_output/k8scluster"
		if err = writer.WriteTLSArtifacts(&cluster, apiVersion, template, parameters, deploymentDirectory, certsGenerated, false); err != nil {
			return "", "", fmt.Errorf("error writing artifacts: %s", err.Error())
		}
		if m != nil { // for testing purposes again
			err = createBlob(d, m, deploymentDirectory, "apimodel.json") // seeing if new implementation works
			if err != nil {
				return "", "", fmt.Errorf("Uploading 'apimodel.json' blob failed: %+v", err)
			}
			err = createBlob(d, m, deploymentDirectory, "azuredeploy.json")
			if err != nil {
				return "", "", fmt.Errorf("Uploading 'azuredeploy.json' blob failed: %+v", err)
			}
		}
	}

	return template, parameters, nil
}

// Deploys the templates generated by ACS Engine for creating a cluster
func deployTemplate(d *schema.ResourceData, m interface{}, template string, parameters string) (id string, err error) {
	client := m.(*ArmClient)
	deployClient := client.deploymentsClient
	ctx := client.StopContext

	var name string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return "", fmt.Errorf("cluster 'name' not found")
	}
	var resourceGroup string
	if v, ok := d.GetOk("resource_group"); ok {
		resourceGroup = v.(string)
	} else {
		return "", fmt.Errorf("cluster 'resource_group' not found")
	}

	azureDeployTemplate, err := expandTemplateBody(template)
	if err != nil {
		return "", err
	}
	azureDeployParameters, err := expandParametersBody(parameters)
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("Error creating deployment: %+v", err)
	}

	fmt.Println("Deployment created (1)")

	err = future.WaitForCompletion(ctx, deployClient.Client)
	if err != nil {
		return "", fmt.Errorf("Error creating deployment: %+v", err)
	}

	fmt.Println("Deployment created (2)")

	read, err := deployClient.Get(ctx, resourceGroup, name)
	if err != nil {
		return "", err
	}
	if read.ID == nil {
		return "", fmt.Errorf("Cannot read ACS Engine Kubernetes cluster deployment %s (resource group %s) ID", name, resourceGroup)
	}
	log.Printf("[INFO] cluster %q ID: %q", name, *read.ID)

	return *read.ID, nil
}

/* 'Update' Helper Functions */

// Creates ScaleClient, loads ACS Engine templates, finds relevant node VM info, calls appropriate function for scaling up or down
func scaleCluster(d *schema.ResourceData, m interface{}, agentIndex int, agentCount int) error {
	sc, err := initializeScaleClient(d, m, agentIndex, agentCount)
	if err != nil {
		return err
	}

	var currentNodeCount, highestUsedIndex, vmNum int
	windowsIndex := -1
	indexes := make([]int, 0)
	indexToVM := make(map[int]string)
	highestUsedIndex = 0
	if sc.AgentPool.IsAvailabilitySets() {
		vms, err := sc.Client.ListVirtualMachines(sc.ResourceGroupName)
		if err != nil {
			return fmt.Errorf("failed to get vms in the resource group. Error: %s", err.Error())
		} else if len(*vms.Value) < 1 {
			return fmt.Errorf("The provided resource group does not contain any vms")
		}
		index := 0
		for _, vm := range *vms.Value {
			vmTags := *vm.Tags
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
				return err
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

		if currentNodeCount == sc.DesiredAgentCount {
			log.Printf("Cluster is currently at the desired agent count")
			return nil
		}

		if currentNodeCount > sc.DesiredAgentCount {
			return scaleDownCluster(&sc, currentNodeCount, indexToVM, d, m)
		}
	} else {
		vmssList, err := sc.Client.ListVirtualMachineScaleSets(sc.ResourceGroupName)
		if err != nil {
			return fmt.Errorf("failed to get vmss list in the resource group: %v", err)
		}
		for _, vmss := range *vmssList.Value {
			vmTags := *vmss.Tags
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
	}

	return scaleUpCluster(&sc, highestUsedIndex, currentNodeCount, windowsIndex, d, m)
}

// Creates and initializes most fields in client.ScaleClient and returns it
func initializeScaleClient(d *schema.ResourceData, m interface{}, agentIndex int, agentCount int) (client.ScaleClient, error) {
	sc := client.ScaleClient{}
	if v, ok := d.GetOk("resource_group"); ok {
		sc.ResourceGroupName = v.(string)
	}
	sc.DeploymentDirectory = "_output/k8scluster"
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
	err := sc.Validate()
	if err != nil {
		return sc, err
	}

	client.AddAuthArgs(&sc.AuthArgs)
	id, err := parseAzureResourceID(d.Id()) // from resourceid.go
	if err != nil {
		return sc, err
	}
	sc.AuthArgs.RawSubscriptionID = id.SubscriptionID
	sc.AuthArgs.AuthMethod = "client_secret"
	if v, ok := d.GetOk("service_principal.0.client_id"); ok {
		sc.AuthArgs.RawClientID = v.(string)
	}
	if v, ok := d.GetOk("service_principal.0.client_secret"); ok {
		sc.AuthArgs.ClientSecret = v.(string)
	}
	err = sc.AuthArgs.ValidateAuthArgs()
	if err != nil {
		return sc, err
	}

	sc.Client, err = sc.AuthArgs.GetClient()
	if err != nil {
		return sc, fmt.Errorf("Failed to get client: %s", err.Error())
	}
	_, err = sc.Client.EnsureResourceGroup(sc.ResourceGroupName, sc.Location, nil)
	if err != nil {
		return sc, err
	}

	sc.Locale, err = i18n.LoadTranslations()
	if err != nil {
		return sc, fmt.Errorf(fmt.Sprintf("error loading translation files: %s", err.Error()))
	}
	apiloader := &api.Apiloader{
		Translator: &i18n.Translator{
			Locale: sc.Locale,
		},
	}
	if m != nil { // for testing purposes
		apimodel, err := getBlob(d, m, "apimodel.json")
		if err != nil {
			return sc, err
		}
		sc.K8sCluster, err = apiloader.LoadContainerService([]byte(apimodel), apiVersion, true, true, nil)
	} else {
		sc.APIModelPath = path.Join(sc.DeploymentDirectory, "apimodel.json")
		if _, err := os.Stat(sc.APIModelPath); os.IsNotExist(err) {
			return sc, fmt.Errorf("specified api model does not exist (%s)", sc.APIModelPath)
		}
		sc.K8sCluster, _, err = apiloader.LoadContainerServiceFromFile(sc.APIModelPath, true, true, nil)
	}
	if err != nil {
		return sc, fmt.Errorf("error parsing the api model: %s", err.Error())
	}
	if sc.K8sCluster.Location != sc.Location {
		return sc, fmt.Errorf("location does not match api model location") // this should probably never happen?
	}
	sc.AgentPool = sc.K8sCluster.Properties.AgentPoolProfiles[sc.AgentPoolIndex]

	templateParameters := generateParametersMap(sc.DeploymentDirectory)

	nameSuffixParam := templateParameters["nameSuffix"].(map[string]interface{}) // do I actually need this?
	sc.NameSuffix = nameSuffixParam["defaultValue"].(string)

	return sc, nil
}

// Scales down a cluster by draining and deleting the nodes given as input
func scaleDownCluster(sc *client.ScaleClient, currentNodeCount int, indexToVM map[int]string, d *schema.ResourceData, m interface{}) error {
	if sc.MasterFQDN == "" {
		return fmt.Errorf("Master FQDN is required to scale down a Kubernetes cluster's agent pool")
	}

	vmsToDelete := make([]string, 0)
	for i := currentNodeCount - 1; i >= sc.DesiredAgentCount; i-- {
		vmsToDelete = append(vmsToDelete, indexToVM[i])
	}

	kubeconfig, err := acsengine.GenerateKubeConfig(sc.K8sCluster.Properties, sc.Location)
	if err != nil {
		return fmt.Errorf("failed to generate kube config: %v", err)
	}
	err = sc.DrainNodes(kubeconfig, vmsToDelete)
	if err != nil {
		return fmt.Errorf("Got error %+v, while draining the nodes to be deleted", err)
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

	return saveScaledApimodel(sc, d, m)
}

// Scales up clusters by creating new nodes within an agent pool
func scaleUpCluster(sc *client.ScaleClient, highestUsedIndex int, currentNodeCount int, windowsIndex int, d *schema.ResourceData, m interface{}) error {
	ctx := acsengine.Context{
		Translator: &i18n.Translator{
			Locale: sc.Locale,
		},
	}
	templateGenerator, err := acsengine.InitializeTemplateGenerator(ctx, false) // original uses classic mode
	if err != nil {
		return fmt.Errorf("failed to initialize template generator: %s", err.Error())
	}

	sc.K8sCluster.Properties.AgentPoolProfiles = []*api.AgentPoolProfile{sc.AgentPool} // how does this work when there's multiple agent pools?

	template, parameters, _, err := templateGenerator.GenerateTemplate(sc.K8sCluster, acsengine.DefaultGeneratorCode, false, acsEngineVersion)
	if err != nil {
		return fmt.Errorf("error generating template %s: %s", sc.APIModelPath, err.Error())
	}

	if template, err = transform.PrettyPrintArmTemplate(template); err != nil {
		return fmt.Errorf("error pretty printing template: %s", err.Error())
	}

	templateJSON := make(map[string]interface{})
	parametersJSON := make(map[string]interface{})

	err = json.Unmarshal([]byte(template), &templateJSON)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(parameters), &parametersJSON)
	if err != nil {
		return err
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

	err = transformer.NormalizeForK8sVMASScalingUp(sc.Logger, templateJSON)
	if err != nil {
		return fmt.Errorf("error transforming the template for scaling template %s: %s", sc.APIModelPath, err.Error())
	}
	if sc.AgentPool.IsAvailabilitySets() {
		addValue(parametersJSON, fmt.Sprintf("%sOffset", sc.AgentPoolToScale), highestUsedIndex+1)
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	deploymentSuffix := random.Int31()

	_, err = sc.Client.DeployTemplate(
		sc.ResourceGroupName,
		fmt.Sprintf("%s-%d", sc.ResourceGroupName, deploymentSuffix),
		templateJSON,
		parametersJSON,
		nil)
	if err != nil {
		return err
	}

	return saveScaledApimodel(sc, d, m)
}

// this needs to be a change in ACS Engine repo instead. Also I know this is ugly.
// Meant to get around error about master node data disk create option being changed
func removeDataDiskCreateOption(templateJSON map[string]interface{}) error {
	// ["resources"][some index]["properties"]["storageProfile"]["dataDisks"]
	found := false
	if v, ok := templateJSON["resources"]; ok {
		resources := v.([]interface{})
		for _, r := range resources {
			resource := r.(map[string]interface{})
			if apiVer, ok := resource["apiVersion"]; ok {
				if apiVer == "[variables('apiVersionStorageManagedDisks')]" {
					if p, ok := resource["properties"]; ok {
						properties := p.(map[string]interface{})
						if sp, ok := properties["storageProfile"]; ok {
							storageProfile := sp.(map[string]interface{})
							if _, ok := storageProfile["dataDisks"]; ok {
								delete(storageProfile, "dataDisks")
								found = true
							}
						}
					}
				}
			}
		}
	}
	if found == false {
		return fmt.Errorf("Removing data disk create option failed: dataDisk not found")
	}
	return nil
}

func saveScaledApimodel(sc *client.ScaleClient, d *schema.ResourceData, m interface{}) error {
	apimodel, err := getBlob(d, m, "apimodel.json")
	if err != nil {
		return err
	}
	apiloader := &api.Apiloader{
		Translator: &i18n.Translator{
			Locale: sc.Locale,
		},
	}
	k8sCluster, err := apiloader.LoadContainerService([]byte(apimodel), apiVersion, false, true, nil)
	if err != nil {
		return err
	}
	sc.K8sCluster = k8sCluster
	sc.K8sCluster.Properties.AgentPoolProfiles[sc.AgentPoolIndex].Count = sc.DesiredAgentCount

	return saveTemplates(sc.K8sCluster, sc.DeploymentDirectory, d, m)
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
		return err
	}

	// I'm not sure that I actually want this, since I'm already validating (but I guess this doesn't hurt?)
	kubernetesInfo, err := api.GetOrchestratorVersionProfile(uc.K8sCluster.Properties.OrchestratorProfile)
	if err != nil {
		return fmt.Errorf("error getting a list of the available upgrades: %s", err.Error())
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
		if agentPool.Name == "" {
			fmt.Println("uc.AgentPoolsToUpgrade has nameless node being appended from AgentPoolProfiles")
		}
		uc.AgentPoolsToUpgrade = append(uc.AgentPoolsToUpgrade, agentPool.Name)
	}

	upgradeCluster := kubernetesupgrade.UpgradeCluster{ // is it a problem that this assumes scale sets as opposed to availability sets?
		Translator: &i18n.Translator{
			Locale: uc.Locale,
		},
		Logger:      uc.Logger,
		Client:      uc.Client,
		StepTimeout: uc.Timeout,
	}

	kubeconfig, err := acsengine.GenerateKubeConfig(uc.K8sCluster.Properties, uc.Location)
	if err != nil {
		return fmt.Errorf("Failed to generate kube config: %v", err)
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
		return fmt.Errorf("Error upgrading cluster: %v", err)
	}

	return saveUpgradedApimodel(&uc, d, m)
}

func initializeUpgradeClient(d *schema.ResourceData, m interface{}, upgradeVersion string) (client.UpgradeClient, error) {
	uc := client.UpgradeClient{}
	if v, ok := d.GetOk("resource_group"); ok {
		uc.ResourceGroupName = v.(string)
	}
	uc.DeploymentDirectory = "_output/k8scluster"
	uc.UpgradeVersion = upgradeVersion
	if v, ok := d.GetOk("location"); ok {
		uc.Location = v.(string)
	}
	uc.TimeoutInMinutes = -1
	err := uc.Validate()
	if err != nil {
		return uc, err
	}

	client.AddAuthArgs(&uc.AuthArgs)
	id, err := parseAzureResourceID(d.Id()) // from resourceid.go
	if err != nil {
		return uc, err
	}
	uc.AuthArgs.RawSubscriptionID = id.SubscriptionID
	uc.AuthArgs.AuthMethod = "client_secret"
	if v, ok := d.GetOk("service_principal.0.client_id"); ok {
		uc.AuthArgs.RawClientID = v.(string)
	}
	if v, ok := d.GetOk("service_principal.0.client_secret"); ok {
		uc.AuthArgs.ClientSecret = v.(string)
	}
	err = uc.AuthArgs.ValidateAuthArgs()
	if err != nil {
		return uc, err
	}

	uc.Client, err = uc.AuthArgs.GetClient()
	if err != nil {
		return uc, fmt.Errorf("Failed to get client: %s", err.Error())
	}
	_, err = uc.Client.EnsureResourceGroup(uc.ResourceGroupName, uc.Location, nil)
	if err != nil {
		return uc, fmt.Errorf("Error ensuring resource group: %s", err.Error())
	}

	apiloader := &api.Apiloader{
		Translator: &i18n.Translator{
			Locale: uc.Locale,
		},
	}
	if m != nil { // for testing purposes
		apimodel, err := getBlob(d, m, "apimodel.json")
		if err != nil {
			return uc, err
		}
		uc.K8sCluster, err = apiloader.LoadContainerService([]byte(apimodel), apiVersion, true, true, nil)
	} else {
		uc.APIModelPath = path.Join(uc.DeploymentDirectory, "apimodel.json")
		if _, err := os.Stat(uc.APIModelPath); os.IsNotExist(err) {
			return uc, fmt.Errorf("specified api model does not exist (%s)", uc.APIModelPath)
		}
		uc.K8sCluster, uc.APIVersion, err = apiloader.LoadContainerServiceFromFile(uc.APIModelPath, true, true, nil) // look into these parameters
	}
	if err != nil {
		return uc, fmt.Errorf("error parsing the api model: %s", err.Error())
	}
	if uc.K8sCluster.Location == "" { // not sure if this block is necessary, might only matter if people are messing w/ the apimodel
		uc.K8sCluster.Location = uc.Location
	} else if uc.K8sCluster.Location != uc.Location {
		return uc, fmt.Errorf("location does not match api model location") // this should probably never happen?
	}

	templateParameters := generateParametersMap(uc.DeploymentDirectory)

	nameSuffixParam := templateParameters["nameSuffix"].(map[string]interface{})
	uc.NameSuffix = nameSuffixParam["defaultValue"].(string)

	return uc, nil
}

func saveUpgradedApimodel(uc *client.UpgradeClient, d *schema.ResourceData, m interface{}) error {
	return saveTemplates(uc.K8sCluster, uc.DeploymentDirectory, d, m)
}

// not working yet
func updateTags(d *schema.ResourceData, m interface{}) error {
	// am I somehow deleting other tags, including acsengineVersion?
	var tags map[string]interface{}
	if v, ok := d.GetOk("tags"); ok {
		tags = v.(map[string]interface{})
	} else {
		tags = map[string]interface{}{}
	}

	// get cluster apimodel
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

	// set tags
	cluster.Tags = expandClusterTags(tags)

	// get templates
	ctx := acsengine.Context{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}
	deploymentDirectory := "_output/k8scluster"
	templateGenerator, err := acsengine.InitializeTemplateGenerator(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to initialize template generator: %s", err.Error())
	}
	template, parameters, _, err := templateGenerator.GenerateTemplate(cluster, acsengine.DefaultGeneratorCode, false, acsEngineVersion)
	if err != nil {
		return fmt.Errorf("error generating templates: %s", err.Error())
	}

	if template, err = transform.PrettyPrintArmTemplate(template); err != nil {
		return fmt.Errorf("error pretty printing template: %s", err.Error())
	}
	if parameters, err = transform.BuildAzureParametersFile(parameters); err != nil {
		return fmt.Errorf("error pretty printing template parameters: %s", err.Error())
	}

	// deploy templates
	// does this know it's updating? I'm not sure
	_, err = deployTemplate(d, m, template, parameters)
	if err != nil {
		return err
	}

	// do I really want to generate these templates all over again?
	return saveTemplates(cluster, deploymentDirectory, d, m)
}

func saveTemplates(cluster *api.ContainerService, deploymentDirectory string, d *schema.ResourceData, m interface{}) error {
	locale, err := i18n.LoadTranslations()
	if err != nil {
		return err
	}

	ctx := acsengine.Context{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}

	templateGenerator, err := acsengine.InitializeTemplateGenerator(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to initialize template generator: %s", err.Error())
	}

	template, parameters, certsGenerated, err := templateGenerator.GenerateTemplate(cluster, acsengine.DefaultGeneratorCode, false, acsEngineVersion)
	if err != nil {
		return fmt.Errorf("error generating templates at %s: %s", deploymentDirectory, err.Error())
	}

	if template, err = transform.PrettyPrintArmTemplate(template); err != nil {
		return fmt.Errorf("error pretty printing template: %s", err.Error())
	}
	if parameters, err = transform.BuildAzureParametersFile(parameters); err != nil {
		log.Fatalf("error pretty printing template parameters: %s \n", err.Error())
	}
	writer := &acsengine.ArtifactWriter{
		Translator: &i18n.Translator{
			Locale: locale,
		},
	}
	if err = writer.WriteTLSArtifacts(cluster, apiVersion, template, parameters, deploymentDirectory, certsGenerated, false); err != nil {
		return fmt.Errorf("error writing artifacts: %s", err.Error())
	}
	err = createBlob(d, m, deploymentDirectory, "apimodel.json")
	if err != nil {
		return fmt.Errorf("Uploading 'apimodel.json' blob failed: %+v", err)
	}
	err = createBlob(d, m, deploymentDirectory, "azuredeploy.json")
	if err != nil {
		return fmt.Errorf("Uploading 'azuredeploy.json' blob failed: %+v", err)
	}

	return nil
}

// if I can get rid of this then I only need to store apimodel.json
func generateParametersMap(deploymentDirectory string) map[string]interface{} {
	templatePath := path.Join(deploymentDirectory, "azuredeploy.json")
	contents, _ := ioutil.ReadFile(templatePath)

	var templateInter interface{}
	json.Unmarshal(contents, &templateInter)

	templateMap := templateInter.(map[string]interface{})
	templateParameters := templateMap["parameters"].(map[string]interface{})

	return templateParameters
}

/* Misc. Helper Functions */

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
		return "", fmt.Errorf("failed to generate kube config: %v", err)
	}
	return kubeConfig, nil
}

func flattenKubeConfig(kubeConfigFile string) (string, []interface{}, error) {
	// Do I actually want all of this to be base64 encoded? I'm confused
	rawKubeConfig := base64.StdEncoding.EncodeToString([]byte(kubeConfigFile))

	config, err := kubernetes.ParseKubeConfig(kubeConfigFile)
	if err != nil {
		return "", nil, err
	}

	kubeConfig := []interface{}{}
	cluster2 := config.Clusters[0].Cluster
	user := config.Users[0].User
	name := config.Users[0].Name

	values := map[string]interface{}{}
	values["host"] = cluster2.Server
	values["username"] = name
	values["password"] = user.Token
	values["client_certificate"] = base64.StdEncoding.EncodeToString([]byte(user.ClientCertificteData))
	values["client_key"] = base64.StdEncoding.EncodeToString([]byte(user.ClientKeyData))
	values["cluster_ca_certificate"] = base64.StdEncoding.EncodeToString([]byte(cluster2.ClusterAuthorityData))

	kubeConfig = append(kubeConfig, values)

	return rawKubeConfig, kubeConfig, nil
}

func createWindowsProfile() (api.WindowsProfile, error) {
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

// based on expandTags in tags.go
func expandClusterTags(tagsMap map[string]interface{}) map[string]string {
	output := make(map[string]string, len(tagsMap))

	for i, v := range tagsMap {
		// Validate should have ignored this error already
		value, _ := tagValueToString(v)
		output[i] = value
	}

	return output
}

func expandTemplateBody(template string) (map[string]interface{}, error) {
	var templateBody map[string]interface{}
	err := json.Unmarshal([]byte(template), &templateBody)
	if err != nil {
		return nil, fmt.Errorf("Error Expanding the template_body for Azure RM Template Deployment")
	}
	return templateBody, nil
}

// from resource_arm_template_deployment.go
func expandParametersBody(body string) (map[string]interface{}, error) {
	var parametersBody map[string]interface{}
	err := json.Unmarshal([]byte(body), &parametersBody)
	if err != nil {
		return nil, fmt.Errorf("Error Expanding the parameters_body for Azure RM Template Deployment")
	}
	return parametersBody, nil
}

// from resource_arm_template_deployment.go
func validateKubernetesVersion(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	capacities := common.AllKubernetesSupportedVersions

	if !capacities[value] {
		errors = append(errors, fmt.Errorf("ACS Engine Kubernetes Cluster: Kubernetes version %s is invalid or not supported", value))
	}
	return
}

func validateKubernetesVersionUpgrade(newVersion string, currentVersion string) error {
	kubernetesProfile := api.OrchestratorProfile{
		OrchestratorType:    "Kubernetes",
		OrchestratorVersion: currentVersion,
	}
	kubernetesInfo, err := api.GetOrchestratorVersionProfile(&kubernetesProfile)
	if err != nil {
		return fmt.Errorf("error getting a list of the available upgrades: %s", err.Error())
	}
	found := false
	for _, up := range kubernetesInfo.Upgrades { // checking that version I want is within the allowed versions
		if up.OrchestratorVersion == newVersion {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("version %s is not supported (either doesn't exist, is a downgrade or same version, or is an upgrade by more than 1 minor version)", newVersion)
	}

	return nil
}

// from resource_arm_container_service.go
func validateMasterProfileCount(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	capacities := map[int]bool{
		1: true,
		3: true,
		5: true,
	}

	if !capacities[value] {
		errors = append(errors, fmt.Errorf("the number of master nodes must be 1, 3 or 5"))
	}
	return
}

// I'm not actually using this... using validation.IntBetween(1, 100)
func validateAgentPoolProfileCount(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value > 100 || value <= 0 {
		errors = append(errors, fmt.Errorf("the count for an agent pool profile can only be between 1 and 100"))
	}
	return
}

// from resource_arm_container_service.go
func resourceLinuxProfilesSSHKeysHash(v interface{}) int {
	var buf bytes.Buffer

	if m, ok := v.(map[string]interface{}); ok {
		buf.WriteString(fmt.Sprintf("%s-", m["key_data"].(string)))
	}

	return hashcode.String(buf.String())
}
