package acsengine

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccACSEngineK8sCluster_createBasic(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	config := testAccACSEngineK8sClusterBasic(ri, clientID, clientSecret, location, keyData)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "resource_group", "acctestRG-"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "location", location),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_id", clientID),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_secret", clientSecret),
					resource.TestCheckResourceAttr(tfResourceName, "linux_profile.0.admin_username", "acctestuser"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.dns_name_prefix", "acctestmaster"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.fqdn", "acctestmaster"+strconv.Itoa(ri)+"."+location+".cloudapp.azure.com"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.name", "agentpool1"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_createMultipleAgentPools(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	config := testAccACSEngineK8sClusterMultipleAgentPools(ri, clientID, clientSecret, location, keyData)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "resource_group", "acctestRG-"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "location", location),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_id", clientID),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_secret", clientSecret),
					resource.TestCheckResourceAttr(tfResourceName, "linux_profile.0.admin_username", "acctestuser"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.dns_name_prefix", "acctestmaster"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.fqdn", "acctestmaster"+strconv.Itoa(ri)+"."+location+".cloudapp.azure.com"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.name", "agentpool1"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.1.name", "agentpool2"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_createVersion10AndAbove(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	version := "1.10.0"
	vmSize := "Standard_D2_v2"
	agentCount := 1
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, version, agentCount, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "resource_group", "acctestRG-"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "location", location),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_id", clientID),
					resource.TestCheckResourceAttr(tfResourceName, "service_principal.0.client_secret", clientSecret),
					resource.TestCheckResourceAttr(tfResourceName, "linux_profile.0.admin_username", "acctestuser"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.dns_name_prefix", "acctestmaster"+strconv.Itoa(ri)),
					resource.TestCheckResourceAttr(tfResourceName, "master_profile.0.vm_size", vmSize),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", version),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.name", "agentpool1"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.vm_size", vmSize),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_scaleUp(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, 40)
	updatedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 2, vmSize, 40)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
		},
	})

}

func TestAccACSEngineK8sCluster_scaleDown(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 2, vmSize, 40)
	updatedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, 40)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_scaleUpDown(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	scaledUpConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 2, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: scaledUpConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_scaleDownUp(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 2, vmSize, osDiskSizeGB)
	scaledDownConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: scaledDownConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
		},
	})
}

// how can I test that cluster wasn't recreated instead of updated?
func TestAccACSEngineK8sCluster_upgradeOnce(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	upgradedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 1, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
				),
			},
			{
				Config: upgradedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_upgradeMultiple(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	upgradedConfig1 := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 1, vmSize, osDiskSizeGB)
	upgradedConfig2 := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.10.0", 1, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
				),
			},
			{
				Config: upgradedConfig1,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
				),
			},
			{
				Config: upgradedConfig2,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.10.0"),
				),
			},
		},
	})
}

// I need to look into what the expected behavior is, and if this is always a scale sets above a certain version
// also test below certain version upgraded to above, followed by scaling
func TestAccACSEngineK8sCluster_upgradeVersion10AndAbove(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.10.0", 1, vmSize, osDiskSizeGB)
	upgradedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.10.1", 1, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.10.0"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: upgradedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.10.1"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_updateUpgradeScaleUp(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	upgradedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 1, vmSize, osDiskSizeGB)
	scaledConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 2, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: upgradedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: scaledConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_updateScaleUpUpgrade(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	scaledConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 2, vmSize, osDiskSizeGB)
	upgradedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 2, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: scaledConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: upgradedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_updateUpgradeScaleDown(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 2, vmSize, osDiskSizeGB)
	upgradedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 2, vmSize, osDiskSizeGB)
	scaledConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 1, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: upgradedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: scaledConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_updateScaleDownUpgrade(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 2, vmSize, osDiskSizeGB)
	scaledConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	upgradedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 1, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
			{
				Config: scaledConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: upgradedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_updateScaleUpgradeInOne(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	vmSize := "Standard_D2_v2"
	osDiskSizeGB := 30
	config := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.8.13", 1, vmSize, osDiskSizeGB)
	updatedConfig := testAccACSEngineK8sClusterCustomized(ri, clientID, clientSecret, location, keyData, "1.9.8", 2, vmSize, osDiskSizeGB)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.8.13"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "1"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "kubernetes_version", "1.9.8"),
					resource.TestCheckResourceAttr(tfResourceName, "agent_pool_profiles.0.count", "2"),
				),
			},
		},
	})

}

// can I somehow check that az group show -g *rg* --query tags actually works
func TestAccACSEngineK8sCluster_updateTags(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	config := testAccACSEngineK8sClusterBasic(ri, clientID, clientSecret, location, keyData)
	newTagsConfig := testAccACSEngineK8sClusterTags(ri, clientID, clientSecret, location, keyData, "Prod", "IT")
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "tags.Environment", "Production"),
				),
			},
			{
				Config: newTagsConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
					resource.TestCheckResourceAttr(tfResourceName, "tags.Environment", "Prod"),
					resource.TestCheckResourceAttr(tfResourceName, "tags.Department", "IT"),
					testCheckACSEngineClusterTagsExists(tfResourceName),
				),
			},
		},
	})
}

// failing because I haven't implemented yet
func TestAccACSEngineK8sCluster_windowsCreateWindowsAgentCluster(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	kubernetesVersion := "1.9.0"
	count := 1
	config := testAccACSEngineK8sClusterOSType(ri, clientID, clientSecret, location, keyData, kubernetesVersion, count)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
				),
			},
		},
	})
}

// failing because I haven't implemented yet
func TestAccACSEngineK8sCluster_windowsScaleUpDownWindowsAgentCluster(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	kubernetesVersion := "1.9.0"
	config := testAccACSEngineK8sClusterOSType(ri, clientID, clientSecret, location, keyData, kubernetesVersion, 1)
	scaledUpConfig := testAccACSEngineK8sClusterOSType(ri, clientID, clientSecret, location, keyData, kubernetesVersion, 2)
	scaledDownConfig := testAccACSEngineK8sClusterOSType(ri, clientID, clientSecret, location, keyData, kubernetesVersion, 1)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
				),
			},
			{
				Config: scaledUpConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
				),
			},
			{
				Config: scaledDownConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
				),
			},
		},
	})
}

func TestAccACSEngineK8sCluster_windowsUpgradeScaleUpWindowsAgentCluster(t *testing.T) {
	ri := acctest.RandInt()
	clientID := testClientID()
	clientSecret := testClientSecret()
	location := testLocation()
	keyData := testSSHPublicKey()
	config := testAccACSEngineK8sClusterOSType(ri, clientID, clientSecret, location, keyData, "1.9.0", 1)
	upgradeConfig := testAccACSEngineK8sClusterOSType(ri, clientID, clientSecret, location, keyData, "1.10.0", 1)
	scaledConfig := testAccACSEngineK8sClusterOSType(ri, clientID, clientSecret, location, keyData, "1.10.0", 2)
	tfResourceName := resourceName(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckACSEngineClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
				),
			},
			{
				Config: upgradeConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
				),
			},
			{
				Config: scaledConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckACSEngineClusterExists(tfResourceName),
				),
			},
		},
	})
}

// createHybridAgentCluster

// test validation (incorrect commands should not let you do 'apply')

/* HELPER FUNCTIONS */

func testAccACSEngineK8sClusterBasic(rInt int, clientID string, clientSecret string, location string, keyData string) string {
	return fmt.Sprintf(`resource "acsengine_kubernetes_cluster" "test%d" {
		name               = "acctest"
		resource_group     = "acctestRG-%d"
		location           = "%s"

		master_profile {
			count           = 1
			dns_name_prefix = "acctestmaster%d"
			vm_size         = "Standard_D2_v2"
		}
	
		agent_pool_profiles {
			name    = "agentpool1"
			count   = 1
			vm_size = "Standard_D2_v2"
		}
	
		linux_profile {
			admin_username = "acctestuser%d"
			ssh {
				key_data = "%s"
			}
		}

		service_principal {
			client_id     = "%s"
			client_secret = "%s"
		}

		tags {
			Environment = "Production"
		}
	}`, rInt, rInt, location, rInt, rInt, keyData, clientID, clientSecret)
}

func testAccACSEngineK8sClusterMultipleAgentPools(rInt int, clientID string, clientSecret string, location string, keyData string) string {
	return fmt.Sprintf(`resource "acsengine_kubernetes_cluster" "test%d" {
		name               = "acctest"
		resource_group     = "acctestRG-%d"
		location           = "%s"
	
		master_profile {
			count           = 1
			dns_name_prefix = "acctestmaster%d"
			vm_size         = "Standard_D2_v2"
		}
	
		agent_pool_profiles {
			name    = "agentpool1"
			count   = 2
			vm_size = "Standard_D2_v2"
		}
	
		agent_pool_profiles {
			name    = "agentpool2"
			count   = 1
			vm_size = "Standard_D2_v2"
		}
	
		linux_profile {
			admin_username = "acctestuser%d"
			ssh {
				key_data = "%s"
			}
		}

		service_principal {
			client_id     = "%s"
			client_secret = "%s"
		}
	}`, rInt, rInt, location, rInt, rInt, keyData, clientID, clientSecret)
}

func testAccACSEngineK8sClusterCustomized(rInt int, clientID string, clientSecret string, location string, keyData string, k8sVersion string, agentCount int, vmSize string, osDiskSize int) string {
	return fmt.Sprintf(`resource "acsengine_kubernetes_cluster" "test%d" {
		name               = "acctest"
		resource_group     = "acctestRG-%d"
		location           = "%s"
		kubernetes_version = "%s"
	
		master_profile {
			count           = 1
			dns_name_prefix = "acctestmaster%d"
			vm_size         = "%s"
			os_disk_size    = "%d"
		}
	
		agent_pool_profiles {
			name         = "agentpool1"
			count        = "%d"
			vm_size      = "%s"
			os_disk_size = "%d"
		}
	
		linux_profile {
			admin_username = "acctestuser%d"
			ssh {
				key_data = "%s"
			}
		}

		service_principal {
			client_id     = "%s"
			client_secret = "%s"
		}
	
		tags {
			Environment = "Production"
		}
	}`, rInt, rInt, location, k8sVersion, rInt, vmSize, osDiskSize, agentCount, vmSize, osDiskSize, rInt, keyData, clientID, clientSecret)
}

func testAccACSEngineK8sClusterTags(rInt int, clientID string, clientSecret string, location string, keyData string, tag1 string, tag2 string) string {
	return fmt.Sprintf(`resource "acsengine_kubernetes_cluster" "test%d" {
		name               = "acctest"
		resource_group     = "acctestRG-%d"
		location           = "%s"

		master_profile {
			count           = 1
			dns_name_prefix = "acctestmaster%d"
			vm_size         = "Standard_D2_v2"
		}
	
		agent_pool_profiles {
			name    = "agentpool1"
			count   = 1
			vm_size = "Standard_D2_v2"
		}
	
		linux_profile {
			admin_username = "acctestuser%d"
			ssh {
				key_data = "%s"
			}
		}

		service_principal {
			client_id     = "%s"
			client_secret = "%s"
		}

		tags {
			Environment = "%s"
			Department  = "%s"
		}
	}`, rInt, rInt, location, rInt, rInt, keyData, clientID, clientSecret, tag1, tag2)
}

func testAccACSEngineK8sClusterOSType(rInt int, clientID string, clientSecret string, location string, keyData string, kubernetesVersion string, agentCount int) string {
	rStr := fmt.Sprintf("%d", rInt)[0:10]
	return fmt.Sprintf(`resource "acsengine_kubernetes_cluster" "test%d" {
		name               = "acctest"
		resource_group     = "acctestRG-%d"
		location           = "%s"
		kubernetes_version = "%s"

		master_profile {
			count           = 1
			dns_name_prefix = "acctestmaster%d"
			vm_size         = "Standard_D2_v2"
		}
	
		agent_pool_profiles {
			name    = "windowspool1"
			count   = "%d"
			vm_size = "Standard_D2_v2"
			os_type = "Windows"
		}
	
		linux_profile {
			admin_username = "acctestuser%d"
			ssh {
				key_data = "%s"
			}
		}

		windows_profile {
			admin_username = "acctstusr%s"
			admin_password = "password%d!"
		}

		service_principal {
			client_id     = "%s"
			client_secret = "%s"
		}

		tags {
			Environment = "Production"
		}
	}`, rInt, rInt, location, kubernetesVersion, rInt, agentCount, rInt, keyData, rStr, rInt, clientID, clientSecret)
}

func testCheckACSEngineClusterExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		is, err := primaryInstanceState(s, name)
		if err != nil {
			return err
		}

		name := is.Attributes["name"]
		resourceGroup, hasResourceGroup := is.Attributes["resource_group"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for Kubernetes cluster: %s", name)
		}

		client := testAccProvider.Meta().(*ArmClient)
		deployClient := client.deploymentsClient

		resp, err := deployClient.Get(client.StopContext, resourceGroup, name) // is this the best way to test for cluster existence?
		if err != nil {
			return fmt.Errorf("Bad: Get on deploymentsClient: %+v", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Kubernetes cluster %q (resource group: %q) does not exist", name, resourceGroup)
		}

		// check if cluster is actually running (not just that Terraform resource exists and deployment exists)
		if err = clusterIsRunning(is, name); err != nil {
			return fmt.Errorf("Bad: cluster not found to be running: %+v", err)
		}

		return nil
	}
}

func testCheckACSEngineClusterDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ArmClient)
	deployClient := client.deploymentsClient

	for _, rs := range s.RootModule().Resources { // for each resource
		if rs.Type != "acsengine_kubernetes_cluster" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group"]

		resp, err := deployClient.Get(client.StopContext, resourceGroup, name)
		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Kubernetes cluster still exists:\n%#v", resp)
		}
	}

	return nil
}

func resourceName(rInt int) string {
	return fmt.Sprintf("acsengine_kubernetes_cluster.test%d", rInt)
}

func primaryInstanceState(s *terraform.State, name string) (*terraform.InstanceState, error) {
	ms := s.RootModule()
	rs, ok := ms.Resources[name]
	if !ok {
		return nil, fmt.Errorf("Not found: %s", name)
	}
	is := rs.Primary
	if is == nil {
		return nil, fmt.Errorf("Bad: could not get primary instance state: %s in %s", name, ms.Path)
	}
	return is, nil
}
