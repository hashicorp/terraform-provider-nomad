// Copyright IBM Corp. 2017, 2026

package acl_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-nomad/internal/framework/provider/testutil"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

func TestAccEphemeralACLToken_basic(t *testing.T) {
	accessorID, tokenName := createTestACLToken(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(),
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccEphemeralACLTokenConfig(accessorID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("type"),
						knownvalue.StringExact("client"),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("name"),
						knownvalue.StringExact(tokenName),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("accessor_id"),
						knownvalue.StringExact(accessorID),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("secret_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("policies"),
						knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact("dev")}),
					),
				},
			},
		},
	})
}

func testAccEphemeralACLTokenConfig(accessorID string) string {
	return fmt.Sprintf(`
provider "nomad" {}

ephemeral "nomad_acl_token" "test" {
  accessor_id = %q
}

provider "echo" {
  data = ephemeral.nomad_acl_token.test
}

resource "echo" "test" {}
`, accessorID)
}

func createTestACLToken(t *testing.T) (string, string) {
	t.Helper()

	providerData := testutil.SDKV2ProviderMeta(t)()
	providerConfig, ok := providerData.(nomad.ProviderConfig)
	if !ok {
		t.Fatalf("expected nomad.ProviderConfig, got %T", providerData)
	}

	tokenName := fmt.Sprintf("acctest-ephemeral-token-%d", time.Now().UnixNano())
	ttl, err := time.ParseDuration("5m")
	if err != nil {
		t.Fatalf("failed to parse ttl: %v", err)
	}

	token, _, err := providerConfig.Client().ACLTokens().Create(&api.ACLToken{
		Name:          tokenName,
		Type:          "client",
		Policies:      []string{"dev"},
		ExpirationTTL: ttl,
	}, nil)
	if err != nil {
		t.Fatalf("failed to create test ACL token: %v", err)
	}

	t.Cleanup(func() {
		if _, err := providerConfig.Client().ACLTokens().Delete(token.AccessorID, nil); err != nil {
			t.Logf("failed to delete test ACL token %q: %v", token.AccessorID, err)
		}
	})

	return token.AccessorID, tokenName
}
