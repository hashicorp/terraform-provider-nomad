package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/terraform-providers/terraform-provider-nomad/nomad"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: nomad.Provider})
}
