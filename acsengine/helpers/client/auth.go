package client

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/armhelpers"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/satori/go.uuid"
)

const (
	emptyID = "00000000-0000-0000-0000-000000000000"
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

// AddAuthArgs initializes all string fields in an AuthArgs struct
func AddAuthArgs(a *AuthArgs) {
	a.RawAzureEnvironment = "AzurePublicCloud"
	a.RawSubscriptionID = ""
	a.AuthMethod = "device"
	a.RawClientID = ""
	a.CertificatePath = ""
	a.PrivateKeyPath = ""
	a.language = "en-us"
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

	if a.AuthMethod == "client_secret" {
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
	case "client_secret":
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
