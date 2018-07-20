package acsengine

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceACSEngineK8sCluster_basic(t *testing.T) {
	dataSourceName := "data.azurerm_kubernetes_cluster.test"
	ri := acctest.RandInt()
	clientID := os.Getenv("ARM_CLIENT_ID")
	clientSecret := os.Getenv("ARM_CLIENT_SECRET")
	location := os.Getenv("ARM_TEST_LOCATION")
	keyData := os.Getenv("SSH_KEY_PUB")
	config := testAccDataSourceACSEngineK8sCluster_basic(ri, clientID, clientSecret, location, keyData)

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
					resource.TestCheckResourceAttrSet(dataSourceName, "kube_config.0.host"),
					resource.TestCheckResourceAttrSet(dataSourceName, "kube_config.0.username"),
					resource.TestCheckResourceAttrSet(dataSourceName, "kube_config.0.password"),
				),
			},
		},
	})
}

func testAccDataSourceACSEngineK8sCluster_basic(rInt int, clientID string, clientSecret string, location string, keyData string) string {
	resource := testAccACSEngineK8sClusterBasic(rInt, clientID, clientSecret, location, keyData)
	return fmt.Sprintf(`%s`, resource)
}
