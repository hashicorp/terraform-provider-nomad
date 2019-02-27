package nomad

import (
	"fmt"
	"github.com/hashicorp/go-version"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
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

	err := testProvider.Configure(terraform.NewResourceConfig(nil))
	if err != nil {
		t.Fatal(err)
	}
}

func testCheckEnt(t *testing.T) {
	testCheckVersion(t, func(v version.Version) bool { return v.Metadata() == "ent" })
}

func testCheckPro(t *testing.T) {
	testCheckVersion(t, func(v version.Version) bool { return v.Metadata() == "pro" || v.Metadata() == "ent" })
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
