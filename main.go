// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider { return nomad.Provider() },
	})
}
