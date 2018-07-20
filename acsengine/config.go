package acsengine

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	keyVault "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-09-01/locks"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	mainStorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/authentication"
	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/hashicorp/terraform/terraform"
)

// ArmClient contains the handles to all the specific Azure Resource Manager
// resource classes' respective clients.
type ArmClient struct {
	clientId                 string
	tenantId                 string
	subscriptionId           string
	usingServicePrincipal    bool
	environment              azure.Environment
	skipProviderRegistration bool

	StopContext context.Context

	// Container Management
	containerServicesClient  containerservice.ContainerServicesClient
	kubernetesClustersClient containerservice.ManagedClustersClient

	// KeyVault
	keyVaultClient           keyvault.VaultsClient
	keyVaultManagementClient keyVault.BaseClient

	// Resources
	managementLocksClient locks.ManagementLocksClient
	deploymentsClient     resources.DeploymentsClient
	providersClient       resources.ProvidersClient
	resourcesClient       resources.Client
	resourceGroupsClient  resources.GroupsClient
	subscriptionsClient   subscriptions.Client

	// Storage
	storageServiceClient storage.AccountsClient
}

func (c *ArmClient) configureClient(client *autorest.Client, auth autorest.Authorizer) {
	setUserAgent(client)
	client.Authorizer = auth
	client.Sender = autorest.CreateSender(withRequestLogging())
	client.SkipResourceProviderRegistration = c.skipProviderRegistration
	client.PollingDuration = 60 * time.Minute
}

func withRequestLogging() autorest.SendDecorator {
	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
			// dump request to wire format
			if dump, err := httputil.DumpRequestOut(r, true); err == nil {
				log.Printf("[DEBUG] AzureRM Request: \n%s\n", dump)
			} else {
				// fallback to basic message
				log.Printf("[DEBUG] AzureRM Request: %s to %s\n", r.Method, r.URL)
			}

			resp, err := s.Do(r)
			if resp != nil {
				// dump response to wire format
				if dump, err := httputil.DumpResponse(resp, true); err == nil {
					log.Printf("[DEBUG] AzureRM Response for %s: \n%s\n", r.URL, dump)
				} else {
					// fallback to basic message
					log.Printf("[DEBUG] AzureRM Response: %s for %s\n", resp.Status, r.URL)
				}
			} else {
				log.Printf("[DEBUG] Request to %s completed with no response", r.URL)
			}
			return resp, err
		})
	}
}

func setUserAgent(client *autorest.Client) {
	tfVersion := fmt.Sprintf("HashiCorp-Terraform-v%s", terraform.VersionString())

	// if the user agent already has a value append the Terraform user agent string
	if curUserAgent := client.UserAgent; curUserAgent != "" {
		client.UserAgent = fmt.Sprintf("%s;%s", curUserAgent, tfVersion)
	} else {
		client.UserAgent = tfVersion
	}

	// append the CloudShell version to the user agent if it exists
	if azureAgent := os.Getenv("AZURE_HTTP_USER_AGENT"); azureAgent != "" {
		client.UserAgent = fmt.Sprintf("%s;%s", client.UserAgent, azureAgent)
	}
}

func getAuthorizationToken(c *authentication.Config, oauthConfig *adal.OAuthConfig, endpoint string) (*autorest.BearerAuthorizer, error) {
	useServicePrincipal := c.ClientSecret != ""

	if useServicePrincipal {
		spt, err := adal.NewServicePrincipalToken(*oauthConfig, c.ClientID, c.ClientSecret, endpoint)
		if err != nil {
			return nil, err
		}

		auth := autorest.NewBearerAuthorizer(spt)
		return auth, nil
	}

	if c.UseMsi {
		spt, err := adal.NewServicePrincipalTokenFromMSI(c.MsiEndpoint, endpoint)
		if err != nil {
			return nil, err
		}
		auth := autorest.NewBearerAuthorizer(spt)
		return auth, nil
	}

	if c.IsCloudShell {
		// load the refreshed tokens from the Azure CLI
		err := c.LoadTokensFromAzureCLI()
		if err != nil {
			return nil, fmt.Errorf("Error loading the refreshed CloudShell tokens: %+v", err)
		}
	}

	spt, err := adal.NewServicePrincipalTokenFromManualToken(*oauthConfig, c.ClientID, endpoint, *c.AccessToken)
	if err != nil {
		return nil, err
	}

	auth := autorest.NewBearerAuthorizer(spt)
	return auth, nil
}

// getArmClient is a helper method which returns a fully instantiated
// *ArmClient based on the Config's current settings.
func getArmClient(c *authentication.Config) (*ArmClient, error) {
	// detect cloud from environment
	env, envErr := azure.EnvironmentFromName(c.Environment)
	if envErr != nil {
		// try again with wrapped value to support readable values like german instead of AZUREGERMANCLOUD
		wrapped := fmt.Sprintf("AZURE%sCLOUD", c.Environment)
		var innerErr error
		if env, innerErr = azure.EnvironmentFromName(wrapped); innerErr != nil {
			return nil, envErr
		}
	}

	// client declarations:
	client := ArmClient{
		clientId:                 c.ClientID,
		tenantId:                 c.TenantID,
		subscriptionId:           c.SubscriptionID,
		environment:              env,
		usingServicePrincipal:    c.ClientSecret != "",
		skipProviderRegistration: c.SkipProviderRegistration,
	}

	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, c.TenantID)
	if err != nil {
		return nil, err
	}

	// OAuthConfigForTenant returns a pointer, which can be nil.
	if oauthConfig == nil {
		return nil, fmt.Errorf("Unable to configure OAuthConfig for tenant %s", c.TenantID)
	}

	sender := autorest.CreateSender(withRequestLogging())

	// Resource Manager endpoints
	endpoint := env.ResourceManagerEndpoint
	auth, err := getAuthorizationToken(c, oauthConfig, endpoint)
	if err != nil {
		return nil, err
	}

	// Key Vault Endpoints
	keyVaultAuth := autorest.NewBearerAuthorizerCallback(sender, func(tenantID, resource string) (*autorest.BearerAuthorizer, error) {
		keyVaultSpt, err := getAuthorizationToken(c, oauthConfig, resource)
		if err != nil {
			return nil, err
		}

		return keyVaultSpt, nil
	})

	client.registerContainerServicesClients(endpoint, c.SubscriptionID, auth)
	client.registerKeyVaultClients(endpoint, c.SubscriptionID, auth, keyVaultAuth, sender)
	client.registerResourcesClients(endpoint, c.SubscriptionID, auth)
	client.registerStorageServiceClients(endpoint, c.SubscriptionID, auth)

	return &client, nil
}

func (c *ArmClient) registerContainerServicesClients(endpoint, subscriptionId string, auth autorest.Authorizer) {
	// ACS
	containerServicesClient := containerservice.NewContainerServicesClientWithBaseURI(endpoint, subscriptionId)
	c.configureClient(&containerServicesClient.Client, auth)
	c.containerServicesClient = containerServicesClient

	// AKS
	kubernetesClustersClient := containerservice.NewManagedClustersClientWithBaseURI(endpoint, subscriptionId)
	c.configureClient(&kubernetesClustersClient.Client, auth)
	c.kubernetesClustersClient = kubernetesClustersClient
}

func (c *ArmClient) registerKeyVaultClients(endpoint, subscriptionId string, auth autorest.Authorizer, keyVaultAuth autorest.Authorizer, sender autorest.Sender) {
	keyVaultClient := keyvault.NewVaultsClientWithBaseURI(endpoint, subscriptionId)
	setUserAgent(&keyVaultClient.Client)
	keyVaultClient.Authorizer = auth
	keyVaultClient.Sender = sender
	keyVaultClient.SkipResourceProviderRegistration = c.skipProviderRegistration
	c.keyVaultClient = keyVaultClient

	keyVaultManagementClient := keyVault.New()
	setUserAgent(&keyVaultManagementClient.Client)
	keyVaultManagementClient.Authorizer = keyVaultAuth
	keyVaultManagementClient.Sender = sender
	keyVaultManagementClient.SkipResourceProviderRegistration = c.skipProviderRegistration
	c.keyVaultManagementClient = keyVaultManagementClient
}

func (c *ArmClient) registerResourcesClients(endpoint, subscriptionId string, auth autorest.Authorizer) {
	locksClient := locks.NewManagementLocksClientWithBaseURI(endpoint, subscriptionId)
	c.configureClient(&locksClient.Client, auth)
	c.managementLocksClient = locksClient

	deploymentsClient := resources.NewDeploymentsClientWithBaseURI(endpoint, subscriptionId)
	c.configureClient(&deploymentsClient.Client, auth)
	c.deploymentsClient = deploymentsClient

	resourcesClient := resources.NewClientWithBaseURI(endpoint, subscriptionId)
	c.configureClient(&resourcesClient.Client, auth)
	c.resourcesClient = resourcesClient

	resourceGroupsClient := resources.NewGroupsClientWithBaseURI(endpoint, subscriptionId)
	c.configureClient(&resourceGroupsClient.Client, auth)
	c.resourceGroupsClient = resourceGroupsClient

	providersClient := resources.NewProvidersClientWithBaseURI(endpoint, subscriptionId)
	c.configureClient(&providersClient.Client, auth)
	c.providersClient = providersClient

	subscriptionsClient := subscriptions.NewClientWithBaseURI(endpoint)
	c.configureClient(&subscriptionsClient.Client, auth)
	c.subscriptionsClient = subscriptionsClient
}

func (c *ArmClient) registerStorageServiceClients(endpoint, subscriptionId string, auth autorest.Authorizer) {
	accountsClient := storage.NewAccountsClientWithBaseURI(endpoint, subscriptionId)
	c.configureClient(&accountsClient.Client, auth)
	c.storageServiceClient = accountsClient
}

var (
	storageKeyCacheMu sync.RWMutex
	storageKeyCache   = make(map[string]string)
)

func (armClient *ArmClient) getKeyForStorageAccount(ctx context.Context, resourceGroupName, storageAccountName string) (string, bool, error) {
	cacheIndex := resourceGroupName + "/" + storageAccountName
	storageKeyCacheMu.RLock()
	key, ok := storageKeyCache[cacheIndex]
	storageKeyCacheMu.RUnlock()

	if ok {
		return key, true, nil
	}

	storageKeyCacheMu.Lock()
	defer storageKeyCacheMu.Unlock()
	key, ok = storageKeyCache[cacheIndex]
	if !ok {
		accountKeys, err := armClient.storageServiceClient.ListKeys(ctx, resourceGroupName, storageAccountName)
		if utils.ResponseWasNotFound(accountKeys.Response) {
			return "", false, nil
		}
		if err != nil {
			// We assume this is a transient error rather than a 404 (which is caught above),  so assume the
			// account still exists.
			return "", true, fmt.Errorf("Error retrieving keys for storage account %q: %s", storageAccountName, err)
		}

		if accountKeys.Keys == nil {
			return "", false, fmt.Errorf("Nil key returned for storage account %q", storageAccountName)
		}

		keys := *accountKeys.Keys
		if len(keys) <= 0 {
			return "", false, fmt.Errorf("No keys returned for storage account %q", storageAccountName)
		}

		keyPtr := keys[0].Value
		if keyPtr == nil {
			return "", false, fmt.Errorf("The first key returned is nil for storage account %q", storageAccountName)
		}

		key = *keyPtr
		storageKeyCache[cacheIndex] = key
	}

	return key, true, nil
}

func (armClient *ArmClient) getBlobStorageClientForStorageAccount(ctx context.Context, resourceGroupName, storageAccountName string) (*mainStorage.BlobStorageClient, bool, error) {
	key, accountExists, err := armClient.getKeyForStorageAccount(ctx, resourceGroupName, storageAccountName)
	if err != nil {
		return nil, accountExists, err
	}
	if accountExists == false {
		return nil, false, nil
	}

	storageClient, err := mainStorage.NewClient(storageAccountName, key, armClient.environment.StorageEndpointSuffix,
		mainStorage.DefaultAPIVersion, true)
	if err != nil {
		return nil, true, fmt.Errorf("Error creating storage client for storage account %q: %s", storageAccountName, err)
	}

	blobClient := storageClient.GetBlobService()
	return &blobClient, true, nil
}

func (armClient *ArmClient) getFileServiceClientForStorageAccount(ctx context.Context, resourceGroupName, storageAccountName string) (*mainStorage.FileServiceClient, bool, error) {
	key, accountExists, err := armClient.getKeyForStorageAccount(ctx, resourceGroupName, storageAccountName)
	if err != nil {
		return nil, accountExists, err
	}
	if accountExists == false {
		return nil, false, nil
	}

	storageClient, err := mainStorage.NewClient(storageAccountName, key, armClient.environment.StorageEndpointSuffix,
		mainStorage.DefaultAPIVersion, true)
	if err != nil {
		return nil, true, fmt.Errorf("Error creating storage client for storage account %q: %s", storageAccountName, err)
	}

	fileClient := storageClient.GetFileService()
	return &fileClient, true, nil
}

func (armClient *ArmClient) getTableServiceClientForStorageAccount(ctx context.Context, resourceGroupName, storageAccountName string) (*mainStorage.TableServiceClient, bool, error) {
	key, accountExists, err := armClient.getKeyForStorageAccount(ctx, resourceGroupName, storageAccountName)
	if err != nil {
		return nil, accountExists, err
	}
	if accountExists == false {
		return nil, false, nil
	}

	storageClient, err := mainStorage.NewClient(storageAccountName, key, armClient.environment.StorageEndpointSuffix,
		mainStorage.DefaultAPIVersion, true)
	if err != nil {
		return nil, true, fmt.Errorf("Error creating storage client for storage account %q: %s", storageAccountName, err)
	}

	tableClient := storageClient.GetTableService()
	return &tableClient, true, nil
}

func (armClient *ArmClient) getQueueServiceClientForStorageAccount(ctx context.Context, resourceGroupName, storageAccountName string) (*mainStorage.QueueServiceClient, bool, error) {
	key, accountExists, err := armClient.getKeyForStorageAccount(ctx, resourceGroupName, storageAccountName)
	if err != nil {
		return nil, accountExists, err
	}
	if accountExists == false {
		return nil, false, nil
	}

	storageClient, err := mainStorage.NewClient(storageAccountName, key, armClient.environment.StorageEndpointSuffix,
		mainStorage.DefaultAPIVersion, true)
	if err != nil {
		return nil, true, fmt.Errorf("Error creating storage client for storage account %q: %s", storageAccountName, err)
	}

	queueClient := storageClient.GetQueueService()
	return &queueClient, true, nil
}
