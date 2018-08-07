package acsengine

import (
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/authentication"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"acsengine": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func TestDetermineAzureResourceProvidersToRegister(t *testing.T) {
	namespace := "Microsoft.Compute"
	namespace2 := "imaginary namespace"
	registrationState := "registered"
	provider := resources.Provider{
		Namespace:         &namespace,
		RegistrationState: &registrationState,
	}
	provider2 := resources.Provider{
		Namespace:         &namespace2,
		RegistrationState: &registrationState,
	}
	providerList := []resources.Provider{provider, provider2}

	providerMap := determineAzureResourceProvidersToRegister(providerList)
	if _, ok := providerMap[namespace]; ok {
		t.Fatalf("%s should have been deleted", namespace)
	}
	if _, ok := providerMap["Microsoft.Authorization"]; !ok {
		t.Fatalf("Microsoft.Authorization should be in provider map")
	}
}

func TestBase64Encode(t *testing.T) {
	cases := []struct {
		Input  string
		Output string
	}{
		{
			Input:  "hello",
			Output: base64.StdEncoding.EncodeToString([]byte("hello")),
		},
		{
			Input:  base64.StdEncoding.EncodeToString([]byte("hello")),
			Output: base64.StdEncoding.EncodeToString([]byte("hello")),
		},
	}

	for _, tc := range cases {
		output := base64Encode(tc.Input)

		if output != tc.Output {
			t.Fatalf("expected %s but got %s", output, tc.Output)
		}
	}
}

func TestIgnoreCaseDiffSuppressFunc(t *testing.T) {
	cases := []struct {
		New      string
		Old      string
		Expected bool
	}{
		{
			New:      "testRG",
			Old:      "testrg",
			Expected: true,
		},
		{
			New:      "testrg1",
			Old:      "testrg",
			Expected: false,
		},
	}

	for _, tc := range cases {
		diff := ignoreCaseDiffSuppressFunc("", tc.Old, tc.New, nil)

		if diff != tc.Expected {
			t.Fatalf("")
		}
	}

}

func TestIsBase64Encoded(t *testing.T) {
	cases := []struct {
		Input  string
		Output bool
	}{
		{
			Input:  "hello",
			Output: false,
		},
		{
			Input:  base64.StdEncoding.EncodeToString([]byte("hello")),
			Output: true,
		},
	}

	for _, tc := range cases {
		output := isBase64Encoded(tc.Input)

		if output != tc.Output {
			t.Fatalf("expected %t but got %t", output, tc.Output)
		}
	}
}

func testAccPreCheck(t *testing.T) {
	variables := []string{
		"ARM_CLIENT_ID",
		"ARM_CLIENT_SECRET",
		"ARM_SUBSCRIPTION_ID",
		"ARM_TENANT_ID",
		"ARM_TEST_LOCATION",
		"SSH_KEY_PUB",
	}

	for _, variable := range variables {
		value := os.Getenv(variable)
		if value == "" {
			t.Fatalf("`%s` must be set for acceptance tests!", variable)
		}
	}
}

func testClientID() string {
	return os.Getenv("ARM_CLIENT_ID")
}

func testClientSecret() string {
	return os.Getenv("ARM_CLIENT_SECRET")
}

func testLocation() string {
	return os.Getenv("ARM_TEST_LOCATION")
}

func testSSHPublicKey() string {
	return os.Getenv("SSH_KEY_PUB")
}

func testArmEnvironmentName() string {
	envName, exists := os.LookupEnv("ARM_ENVIRONMENT")
	if !exists {
		envName = "public"
	}

	return envName
}

func testGetAzureConfig(t *testing.T) *authentication.Config {
	if os.Getenv(resource.TestEnvVar) == "" {
		t.Skip(fmt.Sprintf("Integration test skipped unless env '%s' set", resource.TestEnvVar))
		return nil
	}

	environment := testArmEnvironmentName()

	// we deliberately don't use the main config - since we care about
	config := authentication.Config{
		SubscriptionID:           os.Getenv("ARM_SUBSCRIPTION_ID"),
		ClientID:                 os.Getenv("ARM_CLIENT_ID"),
		TenantID:                 os.Getenv("ARM_TENANT_ID"),
		ClientSecret:             os.Getenv("ARM_CLIENT_SECRET"),
		Environment:              environment,
		SkipProviderRegistration: false,
	}
	return &config
}

func TestAccAzureRMResourceProviderRegistration(t *testing.T) {
	config := testGetAzureConfig(t)
	if config == nil {
		return
	}

	armClient, err := getArmClient(config)
	if err != nil {
		t.Fatalf("Error building ARM Client: %+v", err)
	}

	client := armClient.providersClient
	ctx := testAccProvider.StopContext()
	providerList, err := client.List(ctx, nil, "")
	if err != nil {
		t.Fatalf("Unable to list provider registration status, it is possible that this is due to invalid "+
			"credentials or the service principal does not have permission to use the Resource Manager API, Azure "+
			"error: %s", err)
	}

	err = registerAzureResourceProvidersWithSubscription(ctx, providerList.Values(), client)
	if err != nil {
		t.Fatalf("Error registering Resource Providers: %+v", err)
	}

	needingRegistration := determineAzureResourceProvidersToRegister(providerList.Values())
	if len(needingRegistration) > 0 {
		t.Fatalf("'%d' Resource Providers are still Pending Registration: %v", len(needingRegistration), needingRegistration)
	}
}
