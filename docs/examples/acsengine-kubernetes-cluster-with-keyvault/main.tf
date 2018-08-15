provider "acsengine" {}

provider "azurerm" {}

resource "azurerm_resource_group" "testkvrg" {
  name     = "testkv"
  location = "southcentralus"
}

resource "azurerm_key_vault" "testkv" {
  name = "testkvsm"
  location = "${azurerm_resource_group.testkvrg.location}"
  resource_group_name = "${azurerm_resource_group.testkvrg.name}"
  tenant_id = ""

  sku {
    name = "standard"
  }

  access_policy {
      tenant_id = ""
      object_id = ""

      key_permissions = [
          "get",
          "create",
      ]

      secret_permissions = [
          "get",
          "delete",
          "set",
      ]
  }

  enabled_for_template_deployment = true
}

resource "azurerm_key_vault_secret" "spsecret" {
  name = "spsecret"
  value = ""
  vault_uri = "${azurerm_key_vault.testkv.vault_uri}"
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
  }

  linux_profile {
    admin_username = "azureuser"

    ssh {
      key_data = ""
    }
  }

  service_principal {
    client_id     = ""
    vault_id      = "${azurerm_key_vault.kv.id}"
    secret_name   = "${azurerm_key_vault_secret.spsecret.name}"
  }

  tags {
    Environment = "Production"
  }
}
