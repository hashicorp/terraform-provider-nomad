// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"

	"github.com/hashicorp/terraform-provider-nomad/internal/mux"
)

const (
	providerName = "registry.terraform.io/hashicorp/nomad"
)

func main() {
	ctx := context.Background()
	muxServer, err := mux.MuxServer(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if err := tf6server.Serve(providerName, func() tfprotov6.ProviderServer { return muxServer }); err != nil {
		log.Fatal(err)
	}
}
