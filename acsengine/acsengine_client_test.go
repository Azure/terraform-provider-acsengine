package acsengine

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client"
)

func TestInitializeACSEngineClient(t *testing.T) {
	resourceGroup := "clusterResourceGroup"
	masterDNSPrefix := "masterDNSPrefix"
	d := mockClusterResourceData("clusterName", "southcentralus", resourceGroup, masterDNSPrefix)
	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Resources/deployments/%s", os.Getenv("ARM_SUBSCRIPTION_ID"), resourceGroup, masterDNSPrefix)
	d.SetId(id)

	c := client.ACSEngineClient{}

	if err := initializeACSEngineClient(d, &c); err != nil {
		t.Fatalf("initializeScaleClient failed: %+v", err)
	}

	if c.ResourceGroupName != resourceGroup {
		t.Fatalf("Resource group is not named correctly")
	}
	if c.AuthArgs.SubscriptionID.String() != os.Getenv("ARM_SUBSCRIPTION_ID") {
		t.Fatalf("Subscription ID is not set correctly")
	}
}

func TestInitializeACSEngineClientBadID(t *testing.T) {
	resourceGroup := "clusterResourceGroup"
	masterDNSPrefix := "masterDNSPrefix"
	d := mockClusterResourceData("clusterName", "southcentralus", resourceGroup, masterDNSPrefix)
	d.SetId("")

	c := client.ACSEngineClient{}

	if err := initializeACSEngineClient(d, &c); err == nil {
		t.Fatalf("initializeScaleClient should have failed")
	}
}

// func TestInitializeACSEngineClientInvalidAuthArgs(t *testing.T) {
// 	resourceGroup := "clusterResourceGroup"
// 	masterDNSPrefix := "masterDNSPrefix"
// 	d := mockClusterResourceData("clusterName", "southcentralus", resourceGroup, masterDNSPrefix)
// 	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Resources/deployments/%s", os.Getenv("ARM_SUBSCRIPTION_ID"), resourceGroup, masterDNSPrefix)
// 	d.SetId(id)
// 	if err := d.Set("service_principal.0.client_secret", ""); err != nil {
// 		t.Fatalf("setting service principal failed")
// 	}

// 	c := client.ACSEngineClient{}

// 	err := initializeACSEngineClient(d, &c)
// 	if err == nil {
// 		t.Fatalf("initializeScaleClient should have failed")
// 	}
// }
