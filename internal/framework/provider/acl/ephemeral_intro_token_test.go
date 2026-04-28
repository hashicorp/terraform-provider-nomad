// Copyright IBM Corp. 2017, 2026

package acl_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-nomad/internal/framework/provider/testutil"
)

func TestAccEphemeralIntroToken_basic(t *testing.T) {
	const nodeName = "acctest-node-intro"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
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
