package acsengine

// This file may end up being deleted

import (
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

// Why is this failing??
func TestAccImportACSEngineK8sCluster_importBasic(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	config := testAccACSEngineK8sClusterBasic(ri, clientID, clientSecret, location, keyData)
	resourceName := "acsengine_kubernetes_cluster.test" + strconv.Itoa(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
