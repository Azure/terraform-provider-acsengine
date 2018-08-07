package acsengine

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/test"
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

	if sc.ResourceGroupName != resourceGroup {
		t.Fatalf("Resource group is not named correctly")
	}
	if sc.DesiredAgentCount != desiredAgentCount {
		t.Fatalf("Desired agent count is not set correctly")
	}
	if sc.AuthArgs.SubscriptionID.String() != os.Getenv("ARM_SUBSCRIPTION_ID") {
		t.Fatalf("Subscription ID is not set correctly")
	}
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
		test.Equals(t, countForTemplate, tc.Expected)
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

		test.Equals(t, templateJSON["variables"].(map[string]interface{})[sc.AgentPool.Name+"Index"], tc.WindowsIndex)
	}
}
