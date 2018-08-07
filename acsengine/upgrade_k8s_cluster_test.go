package acsengine

import (
	"fmt"
	"os"
	"testing"
)

// very similar to initializeScaleClient test, get rid of duplicate code (with mock ResourceData function?)
func TestACSEngineK8sCluster_initializeUpgradeClient(t *testing.T) {
	resourceGroup := "clusterResourceGroup"
	masterDNSPrefix := "masterDNSPrefix"
	d := mockClusterResourceData("clusterName", "southcentralus", resourceGroup, masterDNSPrefix)
	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Resources/deployments/%s", os.Getenv("ARM_SUBSCRIPTION_ID"), resourceGroup, masterDNSPrefix)
	d.SetId(id)

	upgradeVersion := "1.9.8"
	uc, err := initializeUpgradeClient(d, upgradeVersion)
	if err != nil {
		t.Fatalf("initializeScaleClient failed: %+v", err)
	}

	if uc.ResourceGroupName != resourceGroup {
		t.Fatalf("Resource group is not named correctly")
	}
	if uc.UpgradeVersion != upgradeVersion {
		t.Fatalf("Desired agent count is not set correctly")
	}
	if uc.AuthArgs.SubscriptionID.String() != os.Getenv("ARM_SUBSCRIPTION_ID") {
		t.Fatalf("Subscription ID is not set correctly")
	}
}
