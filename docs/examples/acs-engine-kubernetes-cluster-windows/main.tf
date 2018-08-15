provider "acsengine" {}

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
    count           = 1
    dns_name_prefix = "clustermaster"
    vm_size         = "Standard_D2_v2"
  }

  agent_pool_profiles {
    name         = "agentpool1"
    count        = 1
    vm_size      = "Standard_D2_v2"
    os_disk_size = 40
    os_type      = "Windows"
  }

  linux_profile {
    admin_username = "azureuser"

    ssh {
      key_data = ""
    }
  }

  windows_profile {
      admin_username = "azureuser"
      admin_password = ""
  }

  service_principal {
    client_id     = ""
    vault_id      = "${data.azurerm_key_vault.kv.id}"
    secret_name   = "${data.azurerm_key_vault_secret.spsecret.name}"
  }

  tags {
    Department = "IT"
  }
}