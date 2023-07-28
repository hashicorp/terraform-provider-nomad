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

func TestDataSourceNodePool(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.6.0") },
		Steps: []resource.TestStep{
			{
				Config: testDataSourceNodePoolConfig_builtIn,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nomad_node_pool.all", "name", "all"),
					resource.TestCheckResourceAttr("data.nomad_node_pool.default", "name", "default"),
				),
			},
			{
				Config:      testDataSourceNodePoolConfig_doesntExist,
				ExpectError: regexp.MustCompile("node pool not found"),
			},
			{
				Config: testDataSourceNodePoolConfig_basic(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nomad_node_pool.test", "name", name),
					resource.TestCheckResourceAttr("data.nomad_node_pool.test", "description", "Terraform test node pool"),
					resource.TestCheckResourceAttr("data.nomad_node_pool.test", "meta.%", "1"),
					resource.TestCheckResourceAttr("data.nomad_node_pool.test", "meta.test", "true"),
					resource.TestCheckResourceAttr("data.nomad_node_pool.test", "scheduler_config.#", "0"),
				),
			},
		},
		CheckDestroy: testResourceNodePool_checkDestroy(name),
	})
}

func TestDataSourceNodePool_schedConfig(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.6.0"); testCheckEnt(t) },
		Steps: []resource.TestStep{
			{
				Config: testDataSourceNodePoolConfig_schedConfig(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nomad_node_pool.no_mem_oversub", "name", fmt.Sprintf("%s-no-mem-oversub", name)),
					resource.TestCheckResourceAttr("data.nomad_node_pool.no_mem_oversub", "scheduler_config.0.scheduler_algorithm", "spread"),
					resource.TestCheckResourceAttr("data.nomad_node_pool.no_mem_oversub", "scheduler_config.0.memory_oversubscription", ""),

					resource.TestCheckResourceAttr("data.nomad_node_pool.mem_oversub_disabled", "name", fmt.Sprintf("%s-mem-oversub-disabled", name)),
					resource.TestCheckResourceAttr("data.nomad_node_pool.mem_oversub_disabled", "scheduler_config.0.scheduler_algorithm", "binpack"),
					resource.TestCheckResourceAttr("data.nomad_node_pool.mem_oversub_disabled", "scheduler_config.0.memory_oversubscription", "disabled"),

					resource.TestCheckResourceAttr("data.nomad_node_pool.mem_oversub_enabled", "name", fmt.Sprintf("%s-mem-oversub-enabled", name)),
					resource.TestCheckResourceAttr("data.nomad_node_pool.mem_oversub_enabled", "scheduler_config.0.scheduler_algorithm", "binpack"),
					resource.TestCheckResourceAttr("data.nomad_node_pool.mem_oversub_enabled", "scheduler_config.0.memory_oversubscription", "enabled"),
				),
			},
		},
		CheckDestroy: testResourceNodePool_checkDestroy(name),
	})
}

const testDataSourceNodePoolConfig_builtIn = `
data "nomad_node_pool" "all" {
  name = "all"
}

data "nomad_node_pool" "default" {
  name = "default"
}
`

const testDataSourceNodePoolConfig_doesntExist = `
data "nomad_node_pool" "doesnt_exist" {
  name = "doesnt-exist"
}
`

func testDataSourceNodePoolConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "nomad_node_pool" "test" {
  name        = "%s"
  description = "Terraform test node pool"

  meta = {
    test = "true"
  }
}

data "nomad_node_pool" "test" {
  name = nomad_node_pool.test.name
}
`, name)
}

func testDataSourceNodePoolConfig_schedConfig(prefix string) string {
	return fmt.Sprintf(`
resource "nomad_node_pool" "no_mem_oversub" {
  name = "%[1]s-no-mem-oversub"

  scheduler_config {
    scheduler_algorithm = "spread"
  }
}

data "nomad_node_pool" "no_mem_oversub" {
  name = nomad_node_pool.no_mem_oversub.name
}


resource "nomad_node_pool" "mem_oversub_disabled" {
  name = "%[1]s-mem-oversub-disabled"

  scheduler_config {
    scheduler_algorithm     = "binpack"
	memory_oversubscription = "disabled"
  }
}

data "nomad_node_pool" "mem_oversub_disabled" {
  name = nomad_node_pool.mem_oversub_disabled.name
}

resource "nomad_node_pool" "mem_oversub_enabled" {
  name = "%[1]s-mem-oversub-enabled"

  scheduler_config {
    scheduler_algorithm     = "binpack"
	memory_oversubscription = "enabled"
  }
}

data "nomad_node_pool" "mem_oversub_enabled" {
  name = nomad_node_pool.mem_oversub_enabled.name
}
`, prefix)
}
