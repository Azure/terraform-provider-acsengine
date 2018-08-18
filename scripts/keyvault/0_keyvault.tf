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
      object_id = "${var.sp_object_id}"

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

  access_policy {
      tenant_id = "${var.tenant_id}"
      object_id = "${var.user_object_id}"

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