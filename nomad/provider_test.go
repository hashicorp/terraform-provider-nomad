// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// How to run the acceptance tests for this provider:
//
// - Obtain an official Nomad release from https://nomadproject.io
//   and extract the "nomad" binary
//
// - Run the following to start the Nomad agent in development mode:
//       nomad agent -dev -acl-enabled
//
// - Bootstrap the cluster and get a management token:
//       nomad acl bootstrap
//
// - Set NOMAD_TOKEN to the Secret ID returned from the bootstrap
//
// - Run the Terraform acceptance tests as usual:
//       make testacc TEST=./builtin/providers/nomad
//
// The tests expect to be run in a fresh, empty Nomad server.

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = Provider()
}

var testProvider *schema.Provider
var testProviders map[string]*schema.Provider
var testProviderFactories map[string]func() (*schema.Provider, error)

func init() {
	testProvider = Provider()
	testProviders = map[string]*schema.Provider{
		"nomad": testProvider,
	}
	testProviderFactories = map[string]func() (*schema.Provider, error){
		"nomad": func() (*schema.Provider, error) {
			return testProvider, nil
		},
	}
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv("NOMAD_ADDR"); v == "" {
		os.Setenv("NOMAD_ADDR", "http://127.0.0.1:4646")
	}

	err := testProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
	if err != nil {
		t.Fatal(err)
	}
}

func testEntFeatures(t *testing.T, requiredFeatures ...string) {
	t.Helper()
	testCheckEnt(t)
	client := testProvider.Meta().(ProviderConfig).client
	resp, _, err := client.Operator().LicenseGet(nil)
	if err != nil {
		t.Fatal(err)
	}
	licensedFeatures := map[string]bool{}
	if resp != nil && resp.License != nil {
		for _, f := range resp.License.Features {
			licensedFeatures[f] = true
		}
	}
	for _, f := range requiredFeatures {
		if !licensedFeatures[f] {
			t.Skip(fmt.Sprintf("license doesn't include required feature %q", f))
		}
	}
}

func testCheckEnt(t *testing.T) {
	t.Helper()
	v := testGetVersion(t)
	if v.Metadata() != "ent" {
		t.Skipf("node version %q is not an enterprise build", v.String())
	}
}

func testGetVersion(t *testing.T) *version.Version {
	t.Helper()
	client := testProvider.Meta().(ProviderConfig).client
	if nodes, _, err := client.Nodes().List(nil); err == nil && len(nodes) > 0 {
		if version, err := version.NewVersion(nodes[0].Version); err != nil {
			t.Skip("could not parse node version: ", err)
		} else {
			version = version.Core()
			return version
		}
	} else {
		t.Skip("error listing nodes: ", err)
	}
	return nil
}

func testCheckMinVersion(t *testing.T, min string) {
	t.Helper()
	minVersion, err := version.NewVersion(min)
	if err != nil {
		t.Skipf("failed to check min version: %s", err)
		return
	}

	v := testGetVersion(t).Core()
	if !v.GreaterThanOrEqual(minVersion) {
		t.Skipf("node version %q is older than minimum for test %q",
			v.String(), minVersion.String())
	}
}

func testCheckCSIPluginAvailable(t *testing.T, pluginID string) {
	t.Helper()
	client := testProvider.Meta().(ProviderConfig).client
	plugins, _, err := client.CSIPlugins().List(nil)
	if err != nil {
		t.Skipf("failed to detect CSI plugin %s: %v", pluginID, err)
	}
	if len(plugins) == 0 {
		t.Skipf("no CSI plugins available")
	}

	for _, plugin := range plugins {
		if plugin.ID == pluginID {
			if plugin.ControllersHealthy == 0 {
				t.Skipf("CSI plugin %s has 0 healthy controllers", pluginID)
			}
			if plugin.NodesHealthy == 0 {
				t.Skipf("CSI plugin %s has 0 healthy nodes", pluginID)
			}

			// Plugin found and healthy.
			return
		}
	}
}

func TestAccNomadProvider_namespace(t *testing.T) {
	defer func() {
		os.Unsetenv("NOMAD_NAMESPACE")
		os.Unsetenv("TFC_RUN_ID")
	}()
	var provider *schema.Provider

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactoryInternal(&provider),
		Steps: []resource.TestStep{
			// Provider reads NOMAD_NAMESPACE by default.
			{
				Config: `
provider "nomad" {}

// necessary to initialize the provider
data "nomad_namespaces" "test" {}
				`,
				PreConfig: func() {
					os.Setenv("NOMAD_NAMESPACE", "default")
				},
				Check: func(_ *terraform.State) error {
					providerConfig := provider.Meta().(ProviderConfig)
					config := providerConfig.config

					expect := os.Getenv("NOMAD_NAMESPACE")
					got := config.Namespace
					if got != expect {
						return fmt.Errorf("expected namespace to be %q, got %q", expect, got)
					}
					return nil
				},
			},
			// Provider ignores NOMAD_NAMESPACE if requested.
			{
				Config: `
provider "nomad" {
  ignore_env_vars = {
    NOMAD_NAMESPACE: true,
  }
}

// necessary to initialize the provider
data "nomad_namespaces" "test" {}
				`,
				PreConfig: func() {
					os.Setenv("NOMAD_NAMESPACE", "dev")
				},
				Check: func(_ *terraform.State) error {
					providerConfig := provider.Meta().(ProviderConfig)
					config := providerConfig.config

					expect := ""
					got := config.Namespace
					if got != expect {
						return fmt.Errorf("expected namespace to be %q, got %q", expect, got)
					}
					return nil
				},
			},
			// Provider ignores NOMAD_NAMESPACE if running in TFC.
			{
				Config: `
provider "nomad" {}

// necessary to initialize the provider
data "nomad_namespaces" "test" {}
				`,
				PreConfig: func() {
					os.Setenv("NOMAD_NAMESPACE", "tfc")
					os.Setenv("TFC_RUN_ID", "1")
				},
				Check: func(_ *terraform.State) error {
					providerConfig := provider.Meta().(ProviderConfig)
					config := providerConfig.config

					expect := ""
					got := config.Namespace
					if got != expect {
						return fmt.Errorf("expected namespace to be %q, got %q", expect, got)
					}
					return nil
				},
			},
			// Provider loads NOMAD_NAMESPACE even when running in TFC if user requests.
			{
				Config: `
provider "nomad" {
  ignore_env_vars = {
    NOMAD_NAMESPACE: false,
  }
}

// necessary to initialize the provider
data "nomad_namespaces" "test" {}
				`,
				PreConfig: func() {
					os.Setenv("NOMAD_NAMESPACE", "default")
					os.Setenv("TFC_RUN_ID", "1")
				},
				Check: func(_ *terraform.State) error {
					providerConfig := provider.Meta().(ProviderConfig)
					config := providerConfig.config

					expect := os.Getenv("NOMAD_NAMESPACE")
					got := config.Namespace
					if got != expect {
						return fmt.Errorf("expected namespace to be %q, got %q", expect, got)
					}
					return nil
				},
			},
		},
	})

}

func TestAccNomadProvider_region(t *testing.T) {
	defer func() {
		os.Unsetenv("NOMAD_REGION")
		os.Unsetenv("TFC_RUN_ID")
	}()
	var provider *schema.Provider

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactoryInternal(&provider),
		Steps: []resource.TestStep{
			// Provider reads NOMAD_REGION by default.
			{
				Config: `
provider "nomad" {}

// necessary to initialize the provider
data "nomad_namespaces" "test" {}
				`,
				PreConfig: func() {
					os.Setenv("NOMAD_REGION", "global")
				},
				Check: func(_ *terraform.State) error {
					providerConfig := provider.Meta().(ProviderConfig)
					config := providerConfig.config

					expect := os.Getenv("NOMAD_REGION")
					got := config.Region
					if got != expect {
						return fmt.Errorf("expected region to be %q, got %q", expect, got)
					}
					return nil
				},
			},
			// Provider overrides NOMAD_REGION if specified in the config.
			{
				Config: `
provider "nomad" {
  region = "global"
}

// necessary to initialize the provider
data "nomad_namespaces" "test" {}
				`,
				PreConfig: func() {
					os.Setenv("NOMAD_REGION", "us")
				},
				Check: func(_ *terraform.State) error {
					providerConfig := provider.Meta().(ProviderConfig)
					config := providerConfig.config

					expect := "global"
					got := config.Region
					if got != expect {
						return fmt.Errorf("expected region to be %q, got %q", expect, got)
					}
					return nil
				},
			},
			// Provider ignores NOMAD_REGION if requested.
			{
				Config: `
provider "nomad" {
  ignore_env_vars = {
    NOMAD_REGION: true,
  }
}

// necessary to initialize the provider
data "nomad_namespaces" "test" {}
				`,
				PreConfig: func() {
					os.Setenv("NOMAD_REGION", "us")
				},
				Check: func(_ *terraform.State) error {
					providerConfig := provider.Meta().(ProviderConfig)
					config := providerConfig.config

					expect := ""
					got := config.Region
					if got != expect {
						return fmt.Errorf("expected region to be %q, got %q", expect, got)
					}
					return nil
				},
			},
			// Provider ignores NOMAD_REGION if running in TFC.
			{
				Config: `
provider "nomad" {}

// necessary to initialize the provider
data "nomad_namespaces" "test" {}
				`,
				PreConfig: func() {
					os.Setenv("NOMAD_REGION", "tfc")
					os.Setenv("TFC_RUN_ID", "1")
				},
				Check: func(_ *terraform.State) error {
					providerConfig := provider.Meta().(ProviderConfig)
					config := providerConfig.config

					expect := ""
					got := config.Region
					if got != expect {
						return fmt.Errorf("expected region to be %q, got %q", expect, got)
					}
					return nil
				},
			},
			// Provider loads NOMAD_REGION even when running in TFC if user requests.
			{
				Config: `
provider "nomad" {
  ignore_env_vars = {
    NOMAD_REGION: false,
  }
}

// necessary to initialize the provider
data "nomad_namespaces" "test" {}
				`,
				PreConfig: func() {
					os.Setenv("NOMAD_REGION", "global")
					os.Setenv("TFC_RUN_ID", "1")
				},
				Check: func(_ *terraform.State) error {
					providerConfig := provider.Meta().(ProviderConfig)
					config := providerConfig.config

					expect := os.Getenv("NOMAD_REGION")
					got := config.Region
					if got != expect {
						return fmt.Errorf("expected region to be %q, got %q", expect, got)
					}
					return nil
				},
			},
		},
	})

}

func TestAccNomadProvider_Headers(t *testing.T) {
	var provider *schema.Provider

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactoryInternal(&provider),
		CheckDestroy:      testAccCheckNomadProviderConfigWithHeadersCrashCheckDestroy(provider),
		Steps: []resource.TestStep{
			{
				Config: testAccNomadProviderConfigWithHeaders,
				Check:  testAccCheckNomadProviderConfigWithHeaders(provider),
			},
			{
				// Test concurrent access to Nomad API headers.
				// On fail the provider would panic, so there's no check necessary.
				// https://github.com/hashicorp/terraform-provider-nomad/issues/215
				Config: testAccNomadProviderConfigWithHeadersCrash,
			},
		},
	})
}

func testAccCheckNomadProviderConfigWithHeaders(provider *schema.Provider) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if provider == nil || provider.Meta() == nil {
			return fmt.Errorf("provider was not initialized")
		}
		nomadClientConfig := provider.Meta().(ProviderConfig).config
		expectedHeaders := http.Header(map[string][]string{
			"Test-Header-1": {"a", "b"},
			"Test-Header-2": {"c"},
		})
		if !reflect.DeepEqual(nomadClientConfig.Headers, expectedHeaders) {
			return fmt.Errorf("expected headers %v, got: %v", expectedHeaders, nomadClientConfig.Headers)
		}

		return nil
	}
}

func testAccCheckNomadProviderConfigWithHeadersCrashCheckDestroy(provider *schema.Provider) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		providerConfig := provider.Meta().(ProviderConfig)
		client := providerConfig.client
		namespaces, _, err := client.Namespaces().List(nil)
		if err != nil {
			return err
		}

		count := 0
		namespacePrefix := "headers-crash-test"
		for _, ns := range namespaces {
			if strings.HasPrefix(ns.Name, namespacePrefix) {
				count++
			}
		}

		if count != 0 {
			return fmt.Errorf("%d namespaces still registered", count)
		}
		return nil
	}
}

var testAccNomadProviderConfigWithHeaders = `
provider "nomad" {
  headers {
    name = "Test-Header-1"
	value = "a"
  }
  headers {
    name = "Test-header-1"
	value = "b"
  }
  headers {
    name = "test-header-2"
	value = "c"
  }
}

// necessary to initialize the provider
data "nomad_namespaces" "test" {}
`

var testAccNomadProviderConfigWithHeadersCrash = `
provider "nomad" {
  headers {
    name = "Test-Header-1"
	value = "a"
  }
  headers {
    name = "Test-header-1"
	value = "b"
  }
  headers {
    name = "test-header-2"
	value = "c"
  }
}

// necessary to initialize the provider
data "nomad_namespaces" "test" {}

resource "nomad_namespace" "test" {
  count = 100
  name  = "headers-crash-test-${count.index}"
}
`

// testAccProviderFactoryInternal creates ProviderFactories for provider configuration testing
//
// This should only be used for TestAccNomadProvider_ tests which need to
// reference the provider instance itself. Other testing should use
// testAccProviderFactories or other related functions.
func testAccProviderFactoryInternal(provider **schema.Provider) map[string]func() (*schema.Provider, error) {
	p := Provider()
	factories := map[string]func() (*schema.Provider, error){
		"nomad": func() (*schema.Provider, error) {
			return p, nil
		},
	}
	if provider != nil {
		*provider = p
	}
	return factories
}
