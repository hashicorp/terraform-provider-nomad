// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package mux

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"nomad": func() (tfprotov6.ProviderServer, error) {
		return MuxServer(context.Background())
	},
}

func TestMuxServer_MetadataAndSchema(t *testing.T) {
	t.Parallel()

	server, err := MuxServer(context.Background())
	if err != nil {
		t.Fatalf("expected mux server to initialize: %v", err)
	}

	metadata, err := server.GetMetadata(context.Background(), &tfprotov6.GetMetadataRequest{})
	if err != nil {
		t.Fatalf("expected metadata from mux server: %v", err)
	}

	if !containsDataSource(metadata.DataSources, "nomad_regions") {
		t.Fatalf("expected nomad_regions data source in mux metadata")
	}

	if !containsEphemeralResource(metadata.EphemeralResources, "nomad_node_intro_token") {
		t.Fatalf("expected nomad_node_intro_token ephemeral resource in mux metadata")
	}

	schema, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("expected provider schema from mux server: %v", err)
	}

	if schema.Provider == nil {
		t.Fatal("expected provider schema to be returned")
	}

	if _, ok := schema.DataSourceSchemas["nomad_regions"]; !ok {
		t.Fatal("expected nomad_regions schema from mux server")
	}

	if _, ok := schema.EphemeralResourceSchemas["nomad_node_intro_token"]; !ok {
		t.Fatal("expected nomad_node_intro_token schema from mux server")
	}
}

func TestAccMuxProvider_RegionsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			if os.Getenv("NOMAD_ADDR") == "" {
				os.Setenv("NOMAD_ADDR", "http://127.0.0.1:4646")
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "nomad" {}

data "nomad_regions" "test" {}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.nomad_regions.test", "regions.#"),
				),
			},
		},
	})
}

func containsDataSource(dataSources []tfprotov6.DataSourceMetadata, typeName string) bool {
	for _, dataSource := range dataSources {
		if dataSource.TypeName == typeName {
			return true
		}
	}
	return false
}

func containsEphemeralResource(ephemeralResources []tfprotov6.EphemeralResourceMetadata, typeName string) bool {
	for _, ephemeralResource := range ephemeralResources {
		if ephemeralResource.TypeName == typeName {
			return true
		}
	}
	return false
}
