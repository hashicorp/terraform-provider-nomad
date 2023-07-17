// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestDataSourceNodePools_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.6.0-beta.1") },
		Steps: []resource.TestStep{
			{
				Config: testDataSourceNodePools_basic(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nomad_node_pools.all_pools", "node_pools.#", "5"),
					resource.TestCheckResourceAttr("data.nomad_node_pools.prefix", "node_pools.#", "2"),
					resource.TestCheckResourceAttr("data.nomad_node_pools.filter", "node_pools.#", "2"),
					resource.TestCheckResourceAttr("data.nomad_node_pools.filter_with_prefix", "node_pools.#", "1"),
				),
			},
		},
		CheckDestroy: testResourceNodePool_checkDestroy(name),
	})
}

func testDataSourceNodePools_basic(prefix string) string {
	return fmt.Sprintf(`
resource "nomad_node_pool" "basic" {
  name        = "%[1]s-basic"
  description = "Terraform test node pool"

  meta = {
    test = "%[1]s"
  }
}

resource "nomad_node_pool" "simple" {
  name = "%[1]s-simple"
}

resource "nomad_node_pool" "different_prefix" {
  name = "other-%[1]s"

  meta = {
    test = "%[1]s"
  }
}

data "nomad_node_pools" "all_pools" {
  depends_on = [
    nomad_node_pool.basic,
    nomad_node_pool.simple,
    nomad_node_pool.different_prefix,
  ]
}

data "nomad_node_pools" "prefix" {
  depends_on = [
    nomad_node_pool.basic,
    nomad_node_pool.simple,
    nomad_node_pool.different_prefix,
  ]

  prefix = "%[1]s"
}

data "nomad_node_pools" "filter" {
  depends_on = [
    nomad_node_pool.basic,
    nomad_node_pool.simple,
    nomad_node_pool.different_prefix,
  ]

  filter = "Meta.test == \"%[1]s\""
}

data "nomad_node_pools" "filter_with_prefix" {
  depends_on = [
    nomad_node_pool.basic,
    nomad_node_pool.simple,
    nomad_node_pool.different_prefix,
  ]

  prefix = "%[1]s"
  filter = "Meta.test == \"%[1]s\""
}
`, prefix)
}
