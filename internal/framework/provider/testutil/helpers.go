// Copyright IBM Corp. 2017, 2026

package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	sdkv2helper "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	frameworkprovider "github.com/hashicorp/terraform-provider-nomad/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

func SDKV2ProviderMeta(t *testing.T) func() any {
	t.Helper()
	ensureNomadAddrEnv()

	p := nomad.Provider()
	if err := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil)); err != nil {
		t.Fatalf("failed to configure sdkv2 provider: %v", err)
	}

	return p.Meta
}

// TestAccProtoV6ProviderFactories returns a factory map that serves only the
// framework provider. Use this for tests that don't need SDKv2-only resources
// like nomad_namespace or nomad_sentinel_policy.
func TestAccProtoV6ProviderFactories(t *testing.T) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"nomad": func() (tfprotov6.ProviderServer, error) {
			return providerserver.NewProtocol6WithError(frameworkprovider.New(sdkv2ProviderMetaForFactory(t)))()
		},
		"echo": echoprovider.NewProviderServer(),
	}
}

// TestAccMuxedProviderFactories returns a factory map that muxes the SDKv2
// provider and the framework provider under the same "nomad" provider name.
// Use this for tests that reference SDKv2-only resource types such as
// nomad_namespace or nomad_sentinel_policy alongside nomad_job.
func TestAccMuxedProviderFactories(t *testing.T) map[string]func() (tfprotov6.ProviderServer, error) {
	metaFn := sdkv2ProviderMetaForFactory(t)
	return map[string]func() (tfprotov6.ProviderServer, error){
		"nomad": func() (tfprotov6.ProviderServer, error) {
			sdkv2Provider := nomad.Provider()
			sdkv2Server, err := tf5to6server.UpgradeServer(
				context.Background(),
				sdkv2Provider.GRPCProvider,
			)
			if err != nil {
				return nil, err
			}
			frameworkServer, err := providerserver.NewProtocol6WithError(
				frameworkprovider.New(metaFn),
			)()
			if err != nil {
				return nil, err
			}
			return tf6muxserver.NewMuxServer(context.Background(), func() tfprotov6.ProviderServer {
				return sdkv2Server
			}, func() tfprotov6.ProviderServer {
				return frameworkServer
			})
		},
		"echo": echoprovider.NewProviderServer(),
	}
}

// sdkv2SchemaProvider is used only to satisfy GRPCProvider interface lookup.
var _ = (*sdkv2helper.Provider)(nil)

func TestAccPreCheck(t *testing.T) {
	t.Helper()
	ensureNomadAddrEnv()

	_ = SDKV2ProviderMeta(t)
}

func sdkv2ProviderMetaForFactory(t *testing.T) func() any {
	ensureNomadAddrEnv()
	p := nomad.Provider()
	if err := p.Configure(t.Context(), sdkv2.NewResourceConfigRaw(nil)); err != nil {
		t.Fatalf("failed to configure sdkv2 provider: %v", err)
	}

	return p.Meta
}

func ensureNomadAddrEnv() {
	if os.Getenv("NOMAD_ADDR") == "" {
		os.Setenv("NOMAD_ADDR", "http://127.0.0.1:4646")
	}
}

// NomadClient returns a Nomad API client using the configured SDKv2 provider meta.
func NomadClient(t *testing.T) *api.Client {
	t.Helper()
	return SDKV2ProviderMeta(t)().(nomad.ProviderConfig).Client()
}

// CheckMinVersion skips the test if the first Nomad node's version is older than min.
func CheckMinVersion(t *testing.T, min string) {
	t.Helper()
	minVer, err := version.NewVersion(min)
	if err != nil {
		t.Skipf("failed to parse min version %q: %v", min, err)
	}
	client := NomadClient(t)
	nodes, _, err := client.Nodes().List(nil)
	if err != nil || len(nodes) == 0 {
		t.Skip("could not list nodes to check version")
	}
	nodeVer, err := version.NewVersion(nodes[0].Version)
	if err != nil {
		t.Skip("could not parse node version: ", err)
	}
	if !nodeVer.Core().GreaterThanOrEqual(minVer) {
		t.Skipf("node version %q is older than minimum %q", nodeVer, minVer)
	}
}

// CheckEnt skips the test if the Nomad build is not enterprise.
func CheckEnt(t *testing.T) {
	t.Helper()
	client := NomadClient(t)
	nodes, _, err := client.Nodes().List(nil)
	if err != nil || len(nodes) == 0 {
		t.Skip("could not list nodes to check enterprise")
	}
	nodeVer, err := version.NewVersion(nodes[0].Version)
	if err != nil {
		t.Skip("could not parse node version: ", err)
	}
	if nodeVer.Metadata() != "ent" {
		t.Skipf("node version %q is not an enterprise build", nodeVer)
	}
}

// CheckEntFeatures skips the test if the Nomad enterprise license does not include all required features.
func CheckEntFeatures(t *testing.T, requiredFeatures ...string) {
	t.Helper()
	CheckEnt(t)
	client := NomadClient(t)
	resp, _, err := client.Operator().LicenseGet(nil)
	if err != nil {
		t.Fatal(err)
	}
	licensed := map[string]bool{}
	if resp != nil && resp.License != nil {
		for _, f := range resp.License.Features {
			licensed[f] = true
		}
	}
	for _, f := range requiredFeatures {
		if !licensed[f] {
			t.Skipf("license doesn't include required feature %q", f)
		}
	}
}
