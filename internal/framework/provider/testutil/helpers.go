// Copyright IBM Corp. 2017, 2026

package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	frameworkprovider "github.com/hashicorp/terraform-provider-nomad/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

func SDKV2ProviderMeta(t *testing.T) func() any {
	t.Helper()

	p := nomad.Provider()
	if err := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil)); err != nil {
		t.Fatalf("failed to configure sdkv2 provider: %v", err)
	}

	return p.Meta
}

func TestAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"nomad": func() (tfprotov6.ProviderServer, error) {
			return providerserver.NewProtocol6WithError(frameworkprovider.New(sdkv2ProviderMetaForFactory()))()
		},
		"echo": echoprovider.NewProviderServer(),
	}
}

func TestAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("NOMAD_ADDR") == "" {
		os.Setenv("NOMAD_ADDR", "http://127.0.0.1:4646")
	}

	_ = SDKV2ProviderMeta(t)
}

func sdkv2ProviderMetaForFactory() func() any {
	p := nomad.Provider()
	if err := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil)); err != nil {
		panic(fmt.Sprintf("failed to configure sdkv2 provider: %v", err))
	}

	return p.Meta
}