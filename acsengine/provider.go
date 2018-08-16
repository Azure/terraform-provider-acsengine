package acsengine

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/authentication"
	azschema "github.com/Azure/terraform-provider-acsengine/acsengine/helpers/schema"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_SUBSCRIPTION_ID", ""),
			},

			"client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_ID", ""),
			},

			"client_secret": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_SECRET", ""),
			},

			"tenant_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_TENANT_ID", ""),
			},

			"environment": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_ENVIRONMENT", "public"),
			},

			// CR: see if this is supported, if not remove
			"skip_credentials_validation": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_SKIP_CREDENTIALS_VALIDATION", false),
			},

			"skip_provider_registration": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_SKIP_PROVIDER_REGISTRATION", false),
			},
			"use_msi": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_USE_MSI", false),
			},
			"msi_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_MSI_ENDPOINT", ""),
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"acsengine_kubernetes_cluster": dataSourceACSEngineKubernetesCluster(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"acsengine_kubernetes_cluster": resourceArmACSEngineKubernetesCluster(),
		},
	}

	p.ConfigureFunc = providerConfigure(p)

	return p
}

func providerConfigure(p *schema.Provider) schema.ConfigureFunc {
	return func(d *schema.ResourceData) (interface{}, error) {
		var err error
		config := &authentication.Config{
			SubscriptionID:            d.Get("subscription_id").(string),
			ClientID:                  d.Get("client_id").(string),
			ClientSecret:              d.Get("client_secret").(string),
			TenantID:                  d.Get("tenant_id").(string),
			Environment:               d.Get("environment").(string),
			UseMsi:                    d.Get("use_msi").(bool),
			MsiEndpoint:               d.Get("msi_endpoint").(string),
			SkipCredentialsValidation: d.Get("skip_credentials_validation").(bool),
			SkipProviderRegistration:  d.Get("skip_provider_registration").(bool),
		}

		if config.UseMsi {
			log.Printf("[DEBUG] use_msi specified - using MSI Authentication")
			if config.MsiEndpoint == "" {
				msiEndpoint, err := adal.GetMSIVMEndpoint()
				if err != nil {
					return nil, fmt.Errorf("Could not retrieve MSI endpoint from VM settings."+
						"Ensure the VM has MSI enabled, or try setting msi_endpoint. Error: %s", err)
				}
				config.MsiEndpoint = msiEndpoint
			}
			log.Printf("[DEBUG] Using MSI endpoint %s", config.MsiEndpoint)
			if err = config.ValidateMsi(); err != nil {
				return nil, err
			}
		} else if config.ClientSecret != "" {
			log.Printf("[DEBUG] Client Secret specified - using Service Principal for Authentication")
			if err = config.ValidateServicePrincipal(); err != nil {
				return nil, err
			}
		} else {
			log.Printf("[DEBUG] No Client Secret specified - loading credentials from Azure CLI")
			if err = config.LoadTokensFromAzureCLI(); err != nil {
				return nil, err
			}

			if err = config.ValidateBearerAuth(); err != nil {
				return nil, fmt.Errorf("Please specify either a Service Principal, or log in with the Azure CLI (using `az login`): %+v", err)
			}
		}

		client, err := getArmClient(config)
		if err != nil {
			return nil, err
		}

		client.StopContext = p.StopContext()

		// replaces the context between tests
		p.MetaReset = func() error {
			client.StopContext = p.StopContext()
			return nil
		}

		if !config.SkipCredentialsValidation {
			// List all the available providers and their registration state to avoid unnecessary
			// requests. This also lets us check if the provider credentials are correct.
			providerList, err := client.providersClient.List(client.StopContext, nil, "")
			if err != nil {
				return nil, fmt.Errorf("Unable to list provider registration status, it is possible that this is due to invalid "+
					"credentials or the service principal does not have permission to use the Resource Manager API, Azure "+
					"error: %s", err)
			}

			if !config.SkipProviderRegistration {
				if err = registerAzureResourceProvidersWithSubscription(client.StopContext, providerList.Values(), client.providersClient); err != nil {
					return nil, err
				}
			}
		}

		return client, nil
	}
}

func registerProviderWithSubscription(ctx context.Context, providerName string, client resources.ProvidersClient) error {
	_, err := client.Register(ctx, providerName)
	if err != nil {
		return fmt.Errorf("cannot register provider %s with Azure Resource Manager: %s", providerName, err)
	}

	return nil
}

func determineAzureResourceProvidersToRegister(providerList []resources.Provider) map[string]struct{} {
	// only Compute and Network are used to make a cluster, do I still need KeyVault and Storage?
	providers := map[string]struct{}{
		"Microsoft.Compute":  {},
		"Microsoft.KeyVault": {},
		"Microsoft.Network":  {},
		"Microsoft.Storage":  {},
	}

	// filter out any providers already registered
	for _, p := range providerList {
		if _, ok := providers[*p.Namespace]; !ok {
			continue
		}

		if strings.ToLower(*p.RegistrationState) == "registered" {
			log.Printf("[DEBUG] Skipping provider registration for namespace %s\n", *p.Namespace)
			delete(providers, *p.Namespace)
		}
	}

	return providers
}

func registerAzureResourceProvidersWithSubscription(ctx context.Context, providerList []resources.Provider, client resources.ProvidersClient) error {
	providers := determineAzureResourceProvidersToRegister(providerList)

	var err error
	var wg sync.WaitGroup
	wg.Add(len(providers))

	for providerName := range providers {
		go func(p string) {
			defer wg.Done()
			log.Printf("[DEBUG] Registering provider with namespace %s\n", p)
			if innerErr := registerProviderWithSubscription(ctx, p, client); err != nil {
				err = innerErr
			}
		}(providerName)
	}

	wg.Wait()

	return err
}

func ignoreCaseDiffSuppressFunc(k, old, new string, d *schema.ResourceData) bool {
	return azschema.IgnoreCaseDiffSuppressFunc(k, old, new, d)
}

func base64Encode(data string) string {
	// Check whether the data is already Base64 encoded; don't double-encode
	if isBase64Encoded(data) {
		return data
	}
	// data has not been encoded encode and return
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func base64Decode(data string) string {
	if !isBase64Encoded(data) {
		return data
	}
	result, _ := base64.StdEncoding.DecodeString(data)
	return string(result)
}

func isBase64Encoded(data string) bool {
	_, err := base64.StdEncoding.DecodeString(data)
	return err == nil
}
