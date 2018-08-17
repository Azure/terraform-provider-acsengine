package operations

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/acs-engine/pkg/api"
)

func TestValidateAuthArgs(t *testing.T) {
	cases := []struct {
		RawSubscriptionID string
		RawClientID       string
		ClientSecret      string
		AuthMethod        string
		ExpectError       bool
	}{
		{
			RawSubscriptionID: "",
			RawClientID:       "",
			ExpectError:       true,
		},
		{
			RawSubscriptionID: "12345678-9000-1000-1100-120000000000",
			RawClientID:       "12345678-9000-1000-1100-120000000000",
			AuthMethod:        "",
			ExpectError:       false,
		},
		{
			RawSubscriptionID: "12345678-9000-1000-1100-120000000000",
			RawClientID:       "12345678-9000-1000-1100-120000000000",
			AuthMethod:        "client_secret",
			ExpectError:       true,
		},
		{
			RawSubscriptionID: "12345678-9000-1000-1100-120000000000",
			AuthMethod:        "client_secret",
			RawClientID:       "12345678-9000-1000-1100-120000000000",
			ClientSecret:      "12345678-9000-1000-1100-120000000000",
			ExpectError:       false,
		},
	}

	for _, tc := range cases {
		id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Resources/deployments/%s", os.Getenv("ARM_SUBSCRIPTION_ID"), "rg", "clusterName")
		cluster := &api.ContainerService{
			Properties: &api.Properties{
				ServicePrincipalProfile: &api.ServicePrincipalProfile{
					ClientID: tc.RawClientID,
					Secret:   tc.ClientSecret,
				},
			},
		}
		auth := NewAuthArgs(tc.ClientSecret)
		auth.AddAuthArgs(cluster, id)
		auth.RawSubscriptionID = tc.RawSubscriptionID
		auth.AuthMethod = tc.AuthMethod
		err := auth.ValidateAuthArgs()
		if err == nil && tc.ExpectError {
			t.Fatalf("expected error")
		}
		if err != nil && !tc.ExpectError {
			t.Fatalf("error: %+v", err)
		}
	}
}
