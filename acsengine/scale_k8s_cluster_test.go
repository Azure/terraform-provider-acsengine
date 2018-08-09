package acsengine

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client"
	"github.com/stretchr/testify/assert"
)

func TestACSEngineK8sCluster_initializeScaleClient(t *testing.T) {
	resourceGroup := "clusterResourceGroup"
	masterDNSPrefix := "masterDNSPrefix"
	d := mockClusterResourceData("clusterName", "southcentralus", resourceGroup, masterDNSPrefix)
	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Resources/deployments/%s", os.Getenv("ARM_SUBSCRIPTION_ID"), resourceGroup, masterDNSPrefix)
	d.SetId(id)

	agentIndex := 0
	desiredAgentCount := 2
	sc, err := initializeScaleClient(d, agentIndex, desiredAgentCount)
	if err != nil {
		t.Fatalf("initializeScaleClient failed: %+v", err)
	}

	assert.Equal(t, sc.ResourceGroupName, resourceGroup, "Resource group is not named correctly")
	assert.Equal(t, sc.DesiredAgentCount, desiredAgentCount, "Desired agent count is not set correctly")
	assert.Equal(t, sc.AuthArgs.SubscriptionID.String(), os.Getenv("ARM_SUBSCRIPTION_ID"), "Subscription ID is not set correctly")
}

func TestACSEngineK8sCluster_setCountForTemplate(t *testing.T) {
	cases := []struct {
		DesiredAgentCount int
		HighestUsedIndex  int
		CurrentNodeCount  int
		Expected          int
	}{
		{
			DesiredAgentCount: 2,
			HighestUsedIndex:  0,
			CurrentNodeCount:  1,
			Expected:          2,
		},
		{
			DesiredAgentCount: 2,
			HighestUsedIndex:  1,
			CurrentNodeCount:  1,
			Expected:          3,
		},
	}

	for _, tc := range cases {
		sc := client.ScaleClient{
			DesiredAgentCount: tc.DesiredAgentCount,
		}
		countForTemplate := setCountForTemplate(&sc, tc.HighestUsedIndex, tc.CurrentNodeCount)
		assert.Equal(t, countForTemplate, tc.Expected, "count for template should be the same")
	}
}

func TestACSEngineK8sCluster_setWindowsIndex(t *testing.T) {
	cases := []struct {
		WindowsIndex  int
		AgentPoolName string
	}{
		{
			WindowsIndex:  1,
			AgentPoolName: "agentpool1",
		},
		{
			WindowsIndex:  2,
			AgentPoolName: "agentpool2",
		},
	}

	templateJSON := map[string]interface{}{
		"variables": map[string]interface{}{},
	}

	for _, tc := range cases {
		sc := client.ScaleClient{
			AgentPool: &api.AgentPoolProfile{
				Name: tc.AgentPoolName,
			},
		}
		setWindowsIndex(&sc, tc.WindowsIndex, templateJSON)

		assert.Equal(t, templateJSON["variables"].(map[string]interface{})[sc.AgentPool.Name+"Index"], tc.WindowsIndex, "Windows index should be the same")
	}
}
