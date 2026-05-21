// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDataSourceVariable_basic(t *testing.T) {
	path := acctest.RandomWithPrefix("tf-nomad-test")

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.4.0") },
		Steps: []resource.TestStep{
			{
				Config: testDataSourceVariable_config(path),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nomad_variable.test", "path", path),
					resource.TestCheckResourceAttr("data.nomad_variable.test", "items.test_key", "test_value"),
				),
			},
		},
	})
}

func testDataSourceVariable_config(path string) string {
	return fmt.Sprintf(`
resource "nomad_variable" "test" {
  path = "%s"

  items = {
    test_key = "test_value"
  }
}

data "nomad_variable" "test" {
  path       = "%s"
  depends_on = [nomad_variable.test]
}
`, path, path)
}
