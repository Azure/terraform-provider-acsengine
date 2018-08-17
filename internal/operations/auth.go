package operations

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/armhelpers"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/terraform-provider-acsengine/internal/resource"
	"github.com/satori/go.uuid"
)

const (
	defaultAuthMethod = "client_secret"
	emptyID           = "00000000-0000-0000-0000-000000000000"
)

// AuthArgs includes various authentication arguments for Azure client
type AuthArgs struct {
	RawAzureEnvironment string
	RawSubscriptionID   string
	SubscriptionID      uuid.UUID
	AuthMethod          string
	RawClientID         string

	ClientID        uuid.UUID
	ClientSecret    string
	CertificatePath string
	PrivateKeyPath  string
	language        string
}

// NewAuthArgs returns a new authorization arguments struct
func NewAuthArgs(secret string) *AuthArgs {
	return &AuthArgs{
		ClientSecret: secret,
	}
}

// AddAuthArgs initializes auth args (which can be changed)
func (a *AuthArgs) AddAuthArgs(cluster *api.ContainerService, azureID string) error {
	a.RawAzureEnvironment = "AzurePublicCloud"
	a.language = "en-us"

	id, err := resource.ParseAzureResourceID(azureID)
	if err != nil {
		return fmt.Errorf("error parsing resource ID: %+v", err)
	}
	a.RawSubscriptionID = id.SubscriptionID
	a.AuthMethod = defaultAuthMethod
	a.RawClientID = cluster.Properties.ServicePrincipalProfile.ClientID
	if err = a.ValidateAuthArgs(); err != nil {
		return fmt.Errorf("error validating auth args: %+v", err)
	}
	return nil
}

// GetClient returns an Azure client using the auth args and auth method provided
func (a *AuthArgs) GetClient() (*armhelpers.AzureClient, error) {
	var c *armhelpers.AzureClient
	env, err := azure.EnvironmentFromName(a.RawAzureEnvironment)
	if err != nil {
		return nil, err
	}
	switch a.AuthMethod {
	case "device":
		c, err = armhelpers.NewAzureClientWithDeviceAuth(env, a.SubscriptionID.String())
	case defaultAuthMethod:
		c, err = armhelpers.NewAzureClientWithClientSecret(env, a.SubscriptionID.String(), a.ClientID.String(), a.ClientSecret)
	case "client_certificate":
		c, err = armhelpers.NewAzureClientWithClientCertificateFile(env, a.SubscriptionID.String(), a.ClientID.String(), a.CertificatePath, a.PrivateKeyPath)
	default:
		return nil, fmt.Errorf("ERROR: auth method unsupported. method=%q", a.AuthMethod)
	}
	if err != nil {
		return nil, err
	}
	err = c.EnsureProvidersRegistered(a.SubscriptionID.String())
	if err != nil {
		return nil, err
	}
	c.AddAcceptLanguages([]string{a.language})
	return c, nil
}

// ValidateAuthArgs handles error checking for auth args
func (a *AuthArgs) ValidateAuthArgs() error {
	var err error
	a.ClientID, err = uuid.FromString(a.RawClientID)
	if err != nil {
		return err
	}
	a.SubscriptionID, err = uuid.FromString(a.RawSubscriptionID)
	if err != nil {
		return err
	}

	if a.AuthMethod == defaultAuthMethod {
		if a.ClientID.String() == emptyID || a.ClientSecret == "" {
			return fmt.Errorf("Client ID and client secret must be specified")
		}
	} else if a.AuthMethod == "client_certificate" {
		if a.ClientID.String() == emptyID || a.CertificatePath == "" || a.PrivateKeyPath == "" {
			return fmt.Errorf("Client ID, certificate path, and private key path must be specified")
		}
	}

	if a.SubscriptionID.String() == emptyID {
		return fmt.Errorf("subscription ID is required (and must be valid UUID)")
	}

	_, err = azure.EnvironmentFromName(a.RawAzureEnvironment)
	if err != nil {
		return fmt.Errorf("failed to parse a valid Azure cloud environment")
	}
	return nil
}
