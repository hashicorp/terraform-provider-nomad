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
		PreCheck:  func() { testAccPreCheck(t) },
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

func TestDataSourceNamespace_nodePoolConfig(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.6.0"); testCheckEnt(t) },
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "nomad_namespace" "test" {
  name = "%s"

  node_pool_config {
    default = "dev"
    allowed = ["prod", "qa"]
  }
}

data "nomad_namespace" "test" {
  name = nomad_namespace.test.name
}
`, name),

				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "name", name),
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "node_pool_config.#", "1"),
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "node_pool_config.0.default", "dev"),
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "node_pool_config.0.allowed.#", "2"),
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "node_pool_config.0.allowed.0", "prod"),
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "node_pool_config.0.allowed.1", "qa"),
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "node_pool_config.0.denied.#", "0"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "nomad_namespace" "test" {
  name = "%s"

  node_pool_config {
    default = "dev"
    denied  = ["prod", "qa"]
  }
}

data "nomad_namespace" "test" {
  name = nomad_namespace.test.name
}
`, name),

				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "name", name),
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "node_pool_config.#", "1"),
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "node_pool_config.0.default", "dev"),
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "node_pool_config.0.denied.#", "2"),
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "node_pool_config.0.denied.0", "prod"),
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "node_pool_config.0.denied.1", "qa"),
					resource.TestCheckResourceAttr("data.nomad_namespace.test", "node_pool_config.0.allowed.#", "0"),
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
