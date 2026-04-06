// Copyright IBM Corp. 2017, 2026

package acl_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-nomad/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

func TestAccEphemeralIntroToken_basic(t *testing.T) {
	const nodeName = "acctest-node-intro"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		Steps: []resource.TestStep{
			{
				// Ephemeral values must be consumed by another part of the graph rather
				// than asserted directly from state. The echo provider gives the test a
				// safe sink for the ephemeral object and reflects the consumed data back
				// into test-checkable state. See
				// https://developer.hashicorp.com/terraform/plugin/testing/acceptance-tests/ephemeral-resources#testing-ephemeral-data-with-echo-provider
				Config: testAccEphemeralIntroTokenConfig(nodeName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("node_name"),
						knownvalue.StringExact(nodeName),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("ttl"),
						knownvalue.StringExact("5m"),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("jwt"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccEphemeralIntroTokenConfig(nodeName string) string {
	return fmt.Sprintf(`
provider "nomad" {}

ephemeral "nomad_node_intro_token" "test" {
  node_name = %q
  ttl       = "5m"
}

provider "echo" {
  data = ephemeral.nomad_node_intro_token.test
}

resource "echo" "test" {}
`, nodeName)
}

func sdkv2providerMeta(t *testing.T) func() any {
	t.Helper()

	p := nomad.Provider()
	if err := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil)); err != nil {
		t.Fatalf("failed to configure sdkv2 provider: %v", err)
	}

	return p.Meta
}

func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"nomad": func() (tfprotov6.ProviderServer, error) {
			return providerserver.NewProtocol6WithError(provider.New(sdkv2providerMetaForFactory()))()
		},
		"echo": echoprovider.NewProviderServer(),
	}
}

func sdkv2providerMetaForFactory() func() any {
	p := nomad.Provider()
	if err := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil)); err != nil {
		panic(fmt.Sprintf("failed to configure sdkv2 provider: %v", err))
	}

	return p.Meta
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("NOMAD_ADDR") == "" {
		os.Setenv("NOMAD_ADDR", "http://127.0.0.1:4646")
	}

	_ = sdkv2providerMeta(t)
}
