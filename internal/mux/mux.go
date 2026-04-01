// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package mux

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	framework "github.com/hashicorp/terraform-provider-nomad/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

func MuxServer(ctx context.Context) (tfprotov6.ProviderServer, error) {
	sdkProvider := nomad.Provider()

	upgradedSDKProvider, err := tf5to6server.UpgradeServer(ctx, sdkProvider.GRPCProvider)
	if err != nil {
		return nil, err
	}

	providers := []func() tfprotov6.ProviderServer{
		func() tfprotov6.ProviderServer { return upgradedSDKProvider },
		providerserver.NewProtocol6(framework.New(sdkProvider.Meta)),
	}

	return tf6muxserver.NewMuxServer(ctx, providers...)
}
