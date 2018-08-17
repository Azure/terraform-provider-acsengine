package acsengine

import (
	"fmt"
	"path"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccImportACSEngineK8sCluster_importBasic(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	location := testLocation()
	keyData := testSSHPublicKey()
	vaultID := testKeyVaultID()
	config := testAccACSEngineK8sClusterBasic(ri, clientID, location, keyData, vaultID)
	resourceName := fmt.Sprintf("acsengine_kubernetes_cluster.test%d", ri)

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
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return importStateID(s, ri)
				},
			},
		},
	})
}

func importStateID(s *terraform.State, ri int) (string, error) {
	name := fmt.Sprintf("acsengine_kubernetes_cluster.test%d", ri)
	is, err := primaryInstanceState(s, name)
	if err != nil {
		return "", err
	}

	azureID := is.ID

	dnsPrefix, hasDNSPrefix := is.Attributes["master_profile.0.dns_name_prefix"]
	if !hasDNSPrefix {
		return "", fmt.Errorf("%s: Attribute 'master_profile.0.dns_name_prefix' not found", name)
	}
	deploymentDirectory := path.Join("_output", dnsPrefix)

	id := fmt.Sprintf("%s %s", azureID, deploymentDirectory)

	return id, nil
}
