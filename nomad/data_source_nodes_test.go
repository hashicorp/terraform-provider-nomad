// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestDataSourceNodes_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testDataSourceNodes_config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.nomad_nodes.all", "nodes.#"),
				),
			},
		},
	})
}

func TestDataSourceNodes_filter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testDataSourceNodes_filter,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.nomad_nodes.ready", "nodes.#"),
				),
			},
		},
	})
}

const testDataSourceNodes_config = `
data "nomad_nodes" "all" {}
`

const testDataSourceNodes_filter = `
data "nomad_nodes" "ready" {
  filter = "Status == \"ready\""
}
`
