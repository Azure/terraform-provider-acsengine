resource "azurerm_resource_group" "testkvrg" {
  name     = "tfacsenginerg"
  location = "southcentralus"
}

resource "azurerm_key_vault" "testkv" {
  name = "tfacsenginekv"
  location = "${azurerm_resource_group.testkvrg.location}"
  resource_group_name = "${azurerm_resource_group.testkvrg.name}"
  tenant_id = "${var.tenant_id}"

  sku {
    name = "standard"
  }

  access_policy {
      tenant_id = "${var.tenant_id}"
      object_id = "${var.object_id}"

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
  value = "${var.client_secret}"
  vault_uri = "${azurerm_key_vault.testkv.vault_uri}"
}
