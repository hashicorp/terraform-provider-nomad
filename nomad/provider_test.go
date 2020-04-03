package nomad

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
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
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

var testProvider *schema.Provider
var testProviders map[string]terraform.ResourceProvider

func init() {
	testProvider = Provider().(*schema.Provider)
	testProviders = map[string]terraform.ResourceProvider{
		"nomad": testProvider,
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("NOMAD_ADDR"); v == "" {
		os.Setenv("NOMAD_ADDR", "http://127.0.0.1:4646")
	}

	err := testProvider.Configure(terraform.NewResourceConfigRaw(nil))
	if err != nil {
		t.Fatal(err)
	}
}

func testCheckEnterpriseModules(t *testing.T) {
	testCheckVersion(t, func(v version.Version) bool { return v.Metadata() == "modules" })
}

func testCheckEnt(t *testing.T) {
	testCheckVersion(t, func(v version.Version) bool { return v.Metadata() == "ent" })
}

func testCheckEnterprisePlatform(t *testing.T) {
	testCheckVersion(t, func(v version.Version) bool { return v.Metadata() == "modules" || v.Metadata() == "platform" })
}

func testCheckVersion(t *testing.T, versionCheck func(version.Version) bool) {
	client := testProvider.Meta().(ProviderConfig).client
	if nodes, _, err := client.Nodes().List(nil); err == nil && len(nodes) > 0 {
		if version, err := version.NewVersion(nodes[0].Version); err != nil {
			t.Skip("could not parse node version: ", err)
		} else {
			if !versionCheck(*version) {
				t.Skip(fmt.Sprintf("node version '%v' not appropriate for test", version.String()))
			}
		}
	} else {
		t.Skip("error listing nodes: ", err)
	}
}

func testCheckMinVersion(t *testing.T, v string) {
	minVersion, err := version.NewVersion(v)
	if err != nil {
		t.Skipf("failed to check min version: %s", err)
		return
	}

	testCheckVersion(t, func(v version.Version) bool { return v.Compare(minVersion) >= 0 })
}

func testCheckVaultEnabled(t *testing.T) {
	client := testProvider.Meta().(ProviderConfig).client
	vaultEnabled := false
	if nodes, _, err := client.Nodes().List(nil); err == nil && len(nodes) > 0 {
		if node, _, err := client.Nodes().Info(nodes[0].ID, nil); err == nil {
			if va := node.Attributes["vault.accessible"]; va == "true" {
				vaultEnabled = true
			}
		}
	}
	if !vaultEnabled {
		t.Skip("vault not detected as accessible")
	}
}
