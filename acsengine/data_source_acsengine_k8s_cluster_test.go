package acsengine

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceACSEngineK8sCluster_basic(t *testing.T) {
	ri := acctest.RandInt()
	clientID := os.Getenv("ARM_CLIENT_ID")
	clientSecret := os.Getenv("ARM_CLIENT_SECRET")
	location := os.Getenv("ARM_TEST_LOCATION")
	keyData := os.Getenv("SSH_KEY_PUB")
	config := testAccDataSourceACSEngineK8sClusterBasic(ri, clientID, clientSecret, location, keyData)
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

func testAccDataSourceACSEngineK8sClusterBasic(rInt int, clientID string, clientSecret string, location string, keyData string) string {
	resource := testAccACSEngineK8sClusterBasic(rInt, clientID, clientSecret, location, keyData)
	resourceName := resourceName(rInt)
	return fmt.Sprintf(`%s
	
	data "acsengine_kubernetes_cluster" "test%d" {
		name = "${%s.name}"
		resource_group  = "${%s.resource_group}"
		api_model = "${%s.api_model}"
	}`, resource, rInt, resourceName, resourceName, resourceName)
}
