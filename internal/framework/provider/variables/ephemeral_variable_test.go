// Copyright IBM Corp. 2017, 2026

package variables_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/nomad/api"
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

func TestAccEphemeralVariable_basic(t *testing.T) {
	path := fmt.Sprintf("acctest-variable-%d", time.Now().UnixNano())
	createTestVariable(t, path, api.DefaultNamespace)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccEphemeralVariableConfig(path, api.DefaultNamespace),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("path"),
						knownvalue.StringExact(path),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("namespace"),
						knownvalue.StringExact(api.DefaultNamespace),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("items").AtMapKey("test_key"),
						knownvalue.StringExact("test_value"),
					),
				},
			},
		},
	})
}

func testAccEphemeralVariableConfig(path, namespace string) string {
	return fmt.Sprintf(`
provider "nomad" {}

ephemeral "nomad_variable" "test" {
  path      = %q
  namespace = %q
}

provider "echo" {
  data = ephemeral.nomad_variable.test
}

resource "echo" "test" {}
`, path, namespace)
}

func createTestVariable(t *testing.T, path, namespace string) {
	t.Helper()

	providerData := sdkv2providerMeta(t)()
	providerConfig, ok := providerData.(nomad.ProviderConfig)
	if !ok {
		t.Fatalf("expected nomad.ProviderConfig, got %T", providerData)
	}

	variable := &api.Variable{
		Namespace: namespace,
		Path:      path,
		Items: map[string]string{
			"test_key": "test_value",
		},
	}

	if _, _, err := providerConfig.Client().Variables().Create(variable, nil); err != nil {
		t.Fatalf("failed to create test variable: %v", err)
	}

	t.Cleanup(func() {
		if _, err := providerConfig.Client().Variables().Delete(path, &api.WriteOptions{Namespace: namespace}); err != nil {
			t.Logf("failed to delete test variable %q@%q: %v", path, namespace, err)
		}
	})
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
