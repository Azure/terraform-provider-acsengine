package client

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/stretchr/testify/assert"
)

func TestSetACSEngineClient(t *testing.T) {
	resourceGroup := "clusterResourceGroup"
	masterDNSPrefix := "masterDNSPrefix"
	cluster := utils.MockContainerService("clusterName", "southcentralus", masterDNSPrefix)
	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Resources/deployments/%s", os.Getenv("ARM_SUBSCRIPTION_ID"), resourceGroup, "clusterName")

	c := NewACSEngineClient()

	if err := c.SetACSEngineClient(cluster, id); err != nil {
		t.Fatalf("initializeScaleClient failed: %+v", err)
	}

	assert.Equal(t, c.ResourceGroupName, resourceGroup, "Resource group is not named correctly")
	assert.Equal(t, c.SubscriptionID.String(), os.Getenv("ARM_SUBSCRIPTION_ID"), "Subscription ID is not set correctly")
}

func TestSetACSEngineClientBadID(t *testing.T) {
	masterDNSPrefix := "masterDNSPrefix"
	cluster := utils.MockContainerService("clusterName", "southcentralus", masterDNSPrefix)

	c := NewACSEngineClient()

	if err := c.SetACSEngineClient(cluster, ""); err == nil {
		t.Fatalf("initializeScaleClient should have failed")
	}
}

// func TestsetACSEngineClientInvalidAuthArgs(t *testing.T) {
// 	resourceGroup := "clusterResourceGroup"
// 	masterDNSPrefix := "masterDNSPrefix"
// 	d := mockClusterResourceData("clusterName", "southcentralus", resourceGroup, masterDNSPrefix)
// 	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Resources/deployments/%s", os.Getenv("ARM_SUBSCRIPTION_ID"), resourceGroup, masterDNSPrefix)
// 	d.SetId(id)
// 	if err := d.Set("service_principal.0.client_secret", ""); err != nil {
// 		t.Fatalf("setting service principal failed")
// 	}

// 	c := client.ACSEngineClient{}

// 	err := setACSEngineClient(d, &c)
// 	if err == nil {
// 		t.Fatalf("initializeScaleClient should have failed")
// 	}
// }
func TestValidate(t *testing.T) {
	cases := []struct {
		Client      ACSEngineClient
		ExpectError bool
	}{
		{
			Client:      ACSEngineClient{},
			ExpectError: true,
		},
		{
			Client: ACSEngineClient{
				ResourceGroupName: "rg",
			},
			ExpectError: true,
		},
		{
			Client: ACSEngineClient{
				ResourceGroupName: "rg",
				Location:          "westus",
			},
			ExpectError: true,
		},
		{
			Client: ACSEngineClient{
				ResourceGroupName:   "rg",
				Location:            "westus",
				DeploymentDirectory: "directory",
			},
			ExpectError: false,
		},
	}
	for _, tc := range cases {
		err := tc.Client.Validate()
		if err == nil && tc.ExpectError {
			t.Fatalf("expected error")
		}
		if err != nil && !tc.ExpectError {
			t.Fatalf("error: %+v", err)
		}
	}
}
