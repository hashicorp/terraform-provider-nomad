// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package services_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-provider-nomad/internal/framework/provider/testutil"
	"github.com/shoenig/test/must"
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
					resource.TestCheckResourceAttrSet("data.nomad_service.test", "registrations.#"),
					testCheckServiceRegistrations(t, "webapp"),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.nomad_service.test",
						tfjsonpath.New("service_name"),
						knownvalue.StringExact("webapp"),
					),
					statecheck.ExpectKnownValue(
						"data.nomad_service.test",
						tfjsonpath.New("namespace"),
						knownvalue.StringExact("default"),
					),
					statecheck.ExpectKnownValue(
						"data.nomad_service.test",
						tfjsonpath.New("registrations"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"data.nomad_service.test",
						tfjsonpath.New("registrations").AtSliceIndex(0).AtMapKey("namespace"),
						knownvalue.StringExact("default"),
					),
					statecheck.ExpectKnownValue(
						"data.nomad_service.test",
						tfjsonpath.New("registrations").AtSliceIndex(0).AtMapKey("tags"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("http"),
							knownvalue.StringExact("test"),
						}),
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
					resource.TestCheckResourceAttr("data.nomad_service.test", "registrations.#", "1"),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.nomad_service.test",
						tfjsonpath.New("registrations"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"data.nomad_service.test",
						tfjsonpath.New("registrations").AtSliceIndex(0).AtMapKey("address"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.nomad_service.test",
						tfjsonpath.New("registrations").AtSliceIndex(0).AtMapKey("node_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.nomad_service.test",
						tfjsonpath.New("registrations").AtSliceIndex(0).AtMapKey("alloc_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.nomad_service.test",
						tfjsonpath.New("registrations").AtSliceIndex(0).AtMapKey("job_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.nomad_service.test",
						tfjsonpath.New("registrations").AtSliceIndex(0).AtMapKey("datacenter"),
						knownvalue.StringExact("dc1"),
					),
				},
			},
		},
	})
}

// testCheckServiceRegistrations verifies that registrations exist and have
// expected fields populated.
func testCheckServiceRegistrations(t *testing.T, serviceName string) resource.TestCheckFunc {
	t.Helper()
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["data.nomad_service.test"]
		must.True(t, ok, must.Sprintf("data.nomad_service.test not found in state"))

		attrs := rs.Primary.Attributes
		must.Eq(t, serviceName, attrs["service_name"])
		must.NotEq(t, "0", attrs["registrations.#"],
			must.Sprintf("expected at least one registration"))

		// Verify the first registration has required fields set.
		must.NotEq(t, "", attrs["registrations.0.id"],
			must.Sprintf("registration id should not be empty"))
		must.NotEq(t, "", attrs["registrations.0.node_id"],
			must.Sprintf("registration node_id should not be empty"))
		must.NotEq(t, "", attrs["registrations.0.alloc_id"],
			must.Sprintf("registration alloc_id should not be empty"))
		must.NotEq(t, "", attrs["registrations.0.job_id"],
			must.Sprintf("registration job_id should not be empty"))
		must.NotEq(t, "", attrs[fmt.Sprintf("registrations.0.port")],
			must.Sprintf("registration port should not be empty"))

		return nil
	}
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
