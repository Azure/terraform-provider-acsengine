package acsengine

import (
	"fmt"
	"os"
	"testing"
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
