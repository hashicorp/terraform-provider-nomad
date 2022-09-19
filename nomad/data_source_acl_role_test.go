package nomad

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestDataSourceACLRole(t *testing.T) {
	resourceName := "data.nomad_acl_role.test"

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.4.0-beta.1") },
		Steps: []resource.TestStep{
			{
				Config: testDataSourceACLRoleConfig,
				Check: resource.ComposeTestCheckFunc(
					testDataSourceACLRoleExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "acctest-acl-role"),
					resource.TestCheckResourceAttr(resourceName, "description", "A Terraform acctest ACL Role"),
					resource.TestCheckResourceAttr(resourceName, "policies.#", "1"),
				),
			},
		},
	})
}

func testDataSourceACLRoleExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}
		return nil
	}
}

const testDataSourceACLRoleConfig = `
resource "nomad_acl_policy" "test" {
  name        = "acctest-acl-policy"
  description = "A Terraform acctest ACL Policy"
  rules_hcl   = <<EOT
namespace "default" {
  policy       = "read"
  capabilities = ["submit-job"]
}
EOT
}

resource "nomad_acl_role" "test" {
  name        = "acctest-acl-role"
  description = "A Terraform acctest ACL Role"
  depends_on  = [nomad_acl_policy.test]

  policies {
    name = nomad_acl_policy.test.name
  }
}

data "nomad_acl_role" "test" {
  id = nomad_acl_role.test.id
}
`
