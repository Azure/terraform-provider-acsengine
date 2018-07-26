package client

import (
	"fmt"

	"github.com/Azure/acs-engine/pkg/armhelpers"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/satori/go.uuid"
)

// AuthArgs includes various authentication arguments...
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
func AddAuthArgs(authArgs *AuthArgs) {
	authArgs.RawAzureEnvironment = "AzurePublicCloud"
	authArgs.RawSubscriptionID = ""
	authArgs.AuthMethod = "device"
	authArgs.RawClientID = ""
	authArgs.CertificatePath = ""
	authArgs.PrivateKeyPath = ""
	authArgs.language = "en-us"
}

// ValidateAuthArgs ...
func (authArgs *AuthArgs) ValidateAuthArgs() error {
	authArgs.ClientID, _ = uuid.FromString(authArgs.RawClientID)
	authArgs.SubscriptionID, _ = uuid.FromString(authArgs.RawSubscriptionID)

	if authArgs.AuthMethod == "client_secret" {
		if authArgs.ClientID.String() == "00000000-0000-0000-0000-000000000000" || authArgs.ClientSecret == "" {
			return fmt.Errorf("Client ID and client secret must be specified")
		}
	} else if authArgs.AuthMethod == "client_certificate" {
		if authArgs.ClientID.String() == "00000000-0000-0000-0000-000000000000" || authArgs.CertificatePath == "" || authArgs.PrivateKeyPath == "" {
			return fmt.Errorf("Client ID, certificate path, and private key path must be specified")
		}
	}

	if authArgs.SubscriptionID.String() == "00000000-0000-0000-0000-000000000000" {
		return fmt.Errorf("subscription ID is required (and must be valid UUID)")
	}

	_, err := azure.EnvironmentFromName(authArgs.RawAzureEnvironment)
	if err != nil {
		return fmt.Errorf("failed to parse a valid Azure cloud environment")
	}
	return nil
}

// GetClient ...
func (authArgs *AuthArgs) GetClient() (*armhelpers.AzureClient, error) {
	var client *armhelpers.AzureClient
	env, err := azure.EnvironmentFromName(authArgs.RawAzureEnvironment)
	if err != nil {
		return nil, err
	}
	switch authArgs.AuthMethod {
	case "device":
		client, err = armhelpers.NewAzureClientWithDeviceAuth(env, authArgs.SubscriptionID.String())
	case "client_secret":
		client, err = armhelpers.NewAzureClientWithClientSecret(env, authArgs.SubscriptionID.String(), authArgs.ClientID.String(), authArgs.ClientSecret)
	case "client_certificate":
		client, err = armhelpers.NewAzureClientWithClientCertificateFile(env, authArgs.SubscriptionID.String(), authArgs.ClientID.String(), authArgs.CertificatePath, authArgs.PrivateKeyPath)
	default:
		return nil, fmt.Errorf("ERROR: auth method unsupported. method=%q", authArgs.AuthMethod)
	}
	if err != nil {
		return nil, err
	}
	err = client.EnsureProvidersRegistered(authArgs.SubscriptionID.String())
	if err != nil {
		return nil, err
	}
	client.AddAcceptLanguages([]string{authArgs.language})
	return client, nil
}
