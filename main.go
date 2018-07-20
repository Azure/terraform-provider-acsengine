package main

import (
	"github.com/Azure/terraform-provider-acsengine/acsengine"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: acsengine.Provider})
}
