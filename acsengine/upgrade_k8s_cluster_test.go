package acsengine

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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

	assert.Equal(t, uc.ResourceGroupName, resourceGroup, "Resource group is not named correctly")
	assert.Equal(t, uc.UpgradeVersion, upgradeVersion, "Desired agent count is not named correctly")
	assert.Equal(t, uc.AuthArgs.SubscriptionID.String(), os.Getenv("ARM_SUBSCRIPTION_ID"), "Subscription ID is not set correctly")
}
