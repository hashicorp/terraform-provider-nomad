// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package acl_test

import (
	"os"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-nomad/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("NOMAD_ADDR") == "" {
		os.Setenv("NOMAD_ADDR", "http://127.0.0.1:4646")
	}
	_ = sdkv2providerMeta(t)
}

func testAccProtoV6ProviderFactories(t *testing.T) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"nomad": func() (tfprotov6.ProviderServer, error) {
			return providerserver.NewProtocol6WithError(provider.New(sdkv2providerMeta(t)))()
		},
	}
}

// sdkv2providerMeta configures the SDKv2 provider and returns a func() any.
func sdkv2providerMeta(t *testing.T) func() any {
	t.Helper()
	p := nomad.Provider()
	if err := p.Configure(t.Context(), sdkv2.NewResourceConfigRaw(nil)); err != nil {
		t.Fatalf("failed to configure sdkv2 provider: %v", err)
	}
	return p.Meta
}

// nomadClientFromMeta returns a configured Nomad API client for use in test check functions.
func nomadClientFromMeta(t *testing.T) *api.Client {
	t.Helper()
	metaFunc := sdkv2providerMeta(t)
	providerConfig, ok := metaFunc().(nomad.ProviderConfig)
	if !ok {
		t.Fatalf("expected nomad.ProviderConfig, got %T", metaFunc())
	}
	return providerConfig.Client()
}

// testGetVersion returns the Nomad agent version from the first node in the cluster.
func testGetVersion(t *testing.T) *version.Version {
	t.Helper()
	client := nomadClientFromMeta(t)
	nodes, _, err := client.Nodes().List(nil)
	if err != nil || len(nodes) == 0 {
		t.Skip("error listing nodes: ", err)
		return nil
	}
	v, err := version.NewVersion(nodes[0].Version)
	if err != nil {
		t.Skip("could not parse node version: ", err)
		return nil
	}
	return v
}

// testCheckMinVersion skips the test when the Nomad cluster is older than min.
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
