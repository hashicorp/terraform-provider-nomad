// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package services_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-provider-nomad/internal/framework/provider/testutil"
)

func TestAccDataSourceNomadService_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() { registerTestService(t, "webapp", "default") },
				Config:    testAccDataSourceNomadServiceConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nomad_service.test", "service_name", "webapp"),
					resource.TestCheckResourceAttr("data.nomad_service.test", "namespace", "default"),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.nomad_service.test",
						tfjsonpath.New("service_name"),
						knownvalue.StringExact("webapp"),
					),
				},
			},
		},
	})
}

func TestAccDataSourceNomadService_choose(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() { registerTestService(t, "webapp-choose", "default") },
				Config:    testAccDataSourceNomadServiceChooseConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nomad_service.test", "service_name", "webapp-choose"),
				),
			},
		},
	})
}

func testAccDataSourceNomadServiceConfig() string {
	return `
data "nomad_service" "test" {
  service_name = "webapp"
}
`
}

func testAccDataSourceNomadServiceChooseConfig() string {
	return `
data "nomad_service" "test" {
  service_name = "webapp-choose"
  choose       = "1|mykey"
}
`
}
