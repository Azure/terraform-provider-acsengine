resource "azurerm_key_vault_secret" "spsecret" {
  name = "spsecret"
  value = "${var.client_secret}"
  vault_uri = "${azurerm_key_vault.testkv.vault_uri}"
}
