provider "acsengine" {}

provider "azurerm" {}

data "azurerm_resource_group" "testkvrg" {
  name = "testkv"
}

data "azurerm_key_vault" "testkv" {
  name = "testkvrg"
  resource_group_name = "${data.azurerm_resource_group.testkvrg.name}"
}

data "azurerm_key_vault_secret" "spsecret" {
  name = "spsecret"
  vault_uri = "${data.azurerm_key_vault.testkv.vault_uri}"
}

resource "acsengine_kubernetes_cluster" "cluster" {
  name               = "testcluster"
  resource_group     = "testRG"
  location           = "southcentralus"
  kubernetes_version = "1.9.0"

  master_profile {
    count           = 3
    dns_name_prefix = "clustermaster"
    vm_size         = "Standard_D2_v2"
  }

  agent_pool_profiles {
    name         = "agentpool1"
    count        = 2
    vm_size      = "Standard_D2_v2"
    os_disk_size = 40
  }

  agent_pool_profiles {
    name         = "agentpool2"
    count        = 3
    vm_size      = "Standard_D2_v2"
    os_disk_size = 40
  }

  linux_profile {
    admin_username = "azureuser"

    ssh {
      key_data = "${var.ssh_public_key}"
    }
  }

  service_principal {
    client_id     = "${var.sp_id}"
    vault_id      = "${data.azurerm_key_vault.testkv.id}"
    secret_name   = "${data.azurerm_key_vault_secret.spsecret.name}"
  }

  tags {
    Environment = "Production"
  }
}
