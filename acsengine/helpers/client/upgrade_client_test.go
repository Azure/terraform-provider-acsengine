package client

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/stretchr/testify/assert"
)

func TestACSEngineK8sCluster_setUpgradeClient(t *testing.T) {
	resourceGroup := "clusterResourceGroup"
	masterDNSPrefix := "masterDNSPrefix"
	cluster := utils.MockContainerService("clusterName", "southcentralus", masterDNSPrefix)
	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Resources/deployments/%s", os.Getenv("ARM_SUBSCRIPTION_ID"), resourceGroup, "clusterName")
	upgradeVersion := "1.9.8"

	uc := NewUpgradeClient()
	err := uc.SetUpgradeClient(cluster, id, upgradeVersion)
	if err != nil {
		t.Fatalf("setUpgradeClient failed: %+v", err)
	}

	assert.Equal(t, uc.ResourceGroupName, resourceGroup, "Resource group is not named correctly")
	assert.Equal(t, uc.UpgradeVersion, upgradeVersion, "Desired agent count is not named correctly")
	assert.Equal(t, uc.AuthArgs.SubscriptionID.String(), os.Getenv("ARM_SUBSCRIPTION_ID"), "Subscription ID is not set correctly")
}

func TestUpgradeValidate(t *testing.T) {
	cases := []struct {
		Client      UpgradeClient
		ExpectError bool
	}{
		{
			Client:      UpgradeClient{},
			ExpectError: true,
		},
		{
			Client: UpgradeClient{
				ACSEngineClient: ACSEngineClient{
					ResourceGroupName:   "rg",
					Location:            "westus",
					DeploymentDirectory: "directory",
				},
			},
			ExpectError: true,
		},
		{
			Client: UpgradeClient{
				ACSEngineClient: ACSEngineClient{
					ResourceGroupName:   "rg",
					Location:            "westus",
					DeploymentDirectory: "directory",
				},
				UpgradeVersion: "1.8.13",
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
