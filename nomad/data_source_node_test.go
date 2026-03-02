// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestDataSourceNode_basic(t *testing.T) {
	testAccPreCheck(t) // required to configure provider to get node ID
	nodeID := testDataSourceNode_getNodeID(t)

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testDataSourceNode_config(nodeID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nomad_node.test", "node_id", nodeID),
					resource.TestCheckResourceAttrSet("data.nomad_node.test", "name"),
					resource.TestCheckResourceAttrSet("data.nomad_node.test", "datacenter"),
					resource.TestCheckResourceAttrSet("data.nomad_node.test", "status"),
					resource.TestCheckResourceAttrSet("data.nomad_node.test", "scheduling_eligibility"),
				),
			},
		},
	})
}

func testDataSourceNode_getNodeID(t *testing.T) string {
	client := testProvider.Meta().(ProviderConfig).client

	nodes, _, err := client.Nodes().List(nil)
	if err != nil {
		t.Fatalf("unexpected error when listing nodes: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatalf("no nodes available for testing")
	}
	return nodes[0].ID
}

func testDataSourceNode_config(nodeID string) string {
	return fmt.Sprintf(`
data "nomad_node" "test" {
  node_id = "%s"
}
`, nodeID)
}
