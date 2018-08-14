package client

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/terraform-provider-acsengine/acsengine/utils"
	"github.com/stretchr/testify/assert"
)

func TestSetScaleClient(t *testing.T) {
	resourceGroup := "clusterResourceGroup"
	masterDNSPrefix := "masterDNSPrefix"
	cluster := utils.MockContainerService("clusterName", "southcentralus", masterDNSPrefix)
	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Resources/deployments/%s", os.Getenv("ARM_SUBSCRIPTION_ID"), resourceGroup, "clusterName")

	agentIndex := 0
	desiredAgentCount := 2
	sc := NewScaleClient(os.Getenv("ARM_CLIENT_SECRET"))
	if err := sc.SetScaleClient(cluster, id, agentIndex, desiredAgentCount); err != nil {
		t.Fatalf("setScaleClient failed: %+v", err)
	}

	assert.Equal(t, sc.ResourceGroupName, resourceGroup, "Resource group is not named correctly")
	assert.Equal(t, sc.DesiredAgentCount, desiredAgentCount, "Desired agent count is not set correctly")
	assert.Equal(t, sc.AuthArgs.SubscriptionID.String(), os.Getenv("ARM_SUBSCRIPTION_ID"), "Subscription ID is not set correctly")
}

func TestScaleValidate(t *testing.T) {
	cases := []struct {
		Client      ScaleClient
		ExpectError bool
	}{
		{
			Client:      ScaleClient{},
			ExpectError: true,
		},
		{
			Client: ScaleClient{
				ACSEngineClient: ACSEngineClient{
					ResourceGroupName:   "rg",
					Location:            "westus",
					DeploymentDirectory: "directory",
				},
			},
			ExpectError: true,
		},
		{
			Client: ScaleClient{
				ACSEngineClient: ACSEngineClient{
					ResourceGroupName:   "rg",
					Location:            "westus",
					DeploymentDirectory: "directory",
				},
				DesiredAgentCount: 1,
				DeploymentName:    "testdeploy",
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
