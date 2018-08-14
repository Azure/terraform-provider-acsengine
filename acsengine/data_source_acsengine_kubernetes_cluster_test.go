package acsengine

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceACSEngineK8sCluster_basic(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	location := testLocation()
	keyData := testSSHPublicKey()
	vaultID := testKeyVaultID()
	config := testAccDataSourceACSEngineK8sClusterBasic(ri, clientID, location, keyData, vaultID)
	dataSourceName := fmt.Sprintf("data.acsengine_kubernetes_cluster.test%d", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(dataSourceName),
					resource.TestCheckResourceAttrSet(dataSourceName, "kube_config.0.client_key"),
					resource.TestCheckResourceAttrSet(dataSourceName, "kube_config.0.client_certificate"),
					resource.TestCheckResourceAttrSet(dataSourceName, "kube_config.0.cluster_ca_certificate"),
					resource.TestCheckResourceAttrSet(dataSourceName, "kube_config.0.username"),
					resource.TestCheckResourceAttr(dataSourceName, "kube_config.0.host", fmt.Sprintf("https://acctestmaster%s.%s.cloudapp.azure.com", strconv.Itoa(ri), location)),
				),
			},
		},
	})
}

func testAccDataSourceACSEngineK8sClusterBasic(rInt int, clientID, location, keyData, vaultID string) string {
	resource := testAccACSEngineK8sClusterBasic(rInt, clientID, location, keyData, vaultID)
	resourceName := resourceName(rInt)
	return fmt.Sprintf(`%s
	
	data "acsengine_kubernetes_cluster" "test%d" {
		name = "${%s.name}"
		resource_group  = "${%s.resource_group}"
		api_model = "${%s.api_model}"
	}`, resource, rInt, resourceName, resourceName, resourceName)
}
