// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestDataSourceNamespace(t *testing.T) {
	resourceName := "data.nomad_namespace.test"
	name := acctest.RandomWithPrefix("tf-nomad-test")

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckEnt(t) },
		Steps: []resource.TestStep{
			{
				Config: testDataSourceDefaultNamespaceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "default"),
					resource.TestCheckResourceAttr(resourceName, "description", "Default shared namespace"),
					resource.TestCheckResourceAttr(resourceName, "quota", ""),
				),
			},
			{
				Config:      testDataSourceNamespaceConfig_doesNotExists,
				ExpectError: regexp.MustCompile("Namespace not found"),
			},
			{
				Config: testDataSourceNamespace_basicConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "A Terraform acctest namespace"),
					resource.TestCheckResourceAttr(resourceName, "quota", ""),
					resource.TestCheckResourceAttr(resourceName, "meta.key", "value"),
					resource.TestCheckResourceAttr(resourceName, "capabilities.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "capabilities.0.disabled_task_drivers.0", "raw_exec"),
					resource.TestCheckResourceAttr(resourceName, "capabilities.0.enabled_task_drivers.0", "docker"),
					resource.TestCheckResourceAttr(resourceName, "capabilities.0.enabled_task_drivers.1", "exec"),
				),
			},
		},
	})
}

const testDataSourceDefaultNamespaceConfig = `
data "nomad_namespace" "test" {
	name = "default"
}
`

const testDataSourceNamespaceConfig_doesNotExists = `
data "nomad_namespace" "test" {
	name = "does-not-exists"
}
`

func testDataSourceNamespace_basicConfig(name string) string {
	return fmt.Sprintf(`
resource "nomad_namespace" "test" {
  name = "%s"
  description = "A Terraform acctest namespace"

  meta = {
    key = "value",
  }

  capabilities {
    enabled_task_drivers  = ["docker", "exec"]
    disabled_task_drivers = ["raw_exec"]
  }
}

data "nomad_namespace" "test" {
  name = nomad_namespace.test.name
}
`, name)
}
