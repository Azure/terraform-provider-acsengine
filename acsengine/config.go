package acsengine

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/mgmt/keyvault"
	vaultsvc "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/terraform-provider-acsengine/internal/authentication"
	"github.com/hashicorp/terraform/terraform"
)

// ArmClient contains the handles to all the specific Azure Resource Manager
// resource classes' respective clients.
type ArmClient struct {
	clientID                 string
	tenantID                 string
	subscriptionID           string
	usingServicePrincipal    bool
	environment              azure.Environment
	skipProviderRegistration bool

	StopContext context.Context

	deploymentsClient    resources.DeploymentsClient
	providersClient      resources.ProvidersClient
	resourceGroupsClient resources.GroupsClient

	keyVaultClient           keyvault.VaultsClient
	keyVaultManagementClient vaultsvc.BaseClient
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
			if dump, err := httputil.DumpRequestOut(r, true); err == nil {
				log.Printf("[DEBUG] AzureRM Request: \n%s\n", dump)
			} else {
				log.Printf("[DEBUG] AzureRM Request: %s to %s\n", r.Method, r.URL)
			}

			resp, err := s.Do(r)
			if resp != nil {
				var dump []byte
				if dump, err = httputil.DumpResponse(resp, true); err == nil {
					log.Printf("[DEBUG] AzureRM Response for %s: \n%s\n", r.URL, dump)
				} else {
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

func getArmClient(c *authentication.Config) (*ArmClient, error) {
	env, envErr := azureEnvironmentFromName(c.Environment)
	if envErr != nil {
		return nil, fmt.Errorf("did not detect cloud: %+v", envErr)
	}

	client := ArmClient{
		clientID:                 c.ClientID,
		tenantID:                 c.TenantID,
		subscriptionID:           c.SubscriptionID,
		environment:              env,
		usingServicePrincipal:    c.ClientSecret != "",
		skipProviderRegistration: c.SkipProviderRegistration,
	}

	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, c.TenantID)
	if err != nil {
		return nil, err
	}

	if oauthConfig == nil {
		return nil, fmt.Errorf("Unable to configure OAuthConfig for tenant %s", c.TenantID)
	}

	sender := autorest.CreateSender(withRequestLogging())

	endpoint := env.ResourceManagerEndpoint
	auth, err := getAuthorizationToken(c, oauthConfig, endpoint)
	if err != nil {
		return nil, err
	}

	keyVaultAuth := autorest.NewBearerAuthorizerCallback(sender, func(tenantID, resource string) (*autorest.BearerAuthorizer, error) {
		keyVaultSpt, err := getAuthorizationToken(c, oauthConfig, resource)
		if err != nil {
			return nil, err
		}
		return keyVaultSpt, nil
	})

	client.registerResourcesClients(endpoint, c.SubscriptionID, auth)
	client.registerKeyVaultClients(endpoint, c.SubscriptionID, auth, keyVaultAuth, sender)

	return &client, nil
}

func azureEnvironmentFromName(environment string) (azure.Environment, error) {
	// detect cloud from environment
	env, envErr := azure.EnvironmentFromName(environment)
	if envErr != nil {
		// try again with wrapped value to support readable values like german instead of AZUREGERMANCLOUD
		wrapped := fmt.Sprintf("AZURE%sCLOUD", environment)
		var innerErr error
		if env, innerErr = azure.EnvironmentFromName(wrapped); innerErr != nil {
			return azure.Environment{}, envErr
		}
	}
	return env, nil
}

func (c *ArmClient) registerResourcesClients(endpoint, subscriptionID string, auth autorest.Authorizer) {
	deploymentsClient := resources.NewDeploymentsClientWithBaseURI(endpoint, subscriptionID)
	c.configureClient(&deploymentsClient.Client, auth)
	c.deploymentsClient = deploymentsClient

	resourceGroupsClient := resources.NewGroupsClientWithBaseURI(endpoint, subscriptionID)
	c.configureClient(&resourceGroupsClient.Client, auth)
	c.resourceGroupsClient = resourceGroupsClient

	providersClient := resources.NewProvidersClientWithBaseURI(endpoint, subscriptionID)
	c.configureClient(&providersClient.Client, auth)
	c.providersClient = providersClient
}

func (c *ArmClient) registerKeyVaultClients(endpoint, subscriptionID string, auth autorest.Authorizer, keyVaultAuth autorest.Authorizer, sender autorest.Sender) {
	keyVaultClient := keyvault.NewVaultsClientWithBaseURI(endpoint, subscriptionID)
	setUserAgent(&keyVaultClient.Client)
	keyVaultClient.Authorizer = auth
	keyVaultClient.Sender = sender
	keyVaultClient.SkipResourceProviderRegistration = c.skipProviderRegistration
	c.keyVaultClient = keyVaultClient

	keyVaultManagementClient := vaultsvc.New()
	setUserAgent(&keyVaultManagementClient.Client)
	keyVaultManagementClient.Authorizer = keyVaultAuth
	keyVaultManagementClient.Sender = sender
	keyVaultManagementClient.SkipResourceProviderRegistration = c.skipProviderRegistration
	c.keyVaultManagementClient = keyVaultManagementClient
}
