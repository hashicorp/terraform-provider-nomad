package nomad

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestDataSourceACLRoles(t *testing.T) {
	resourceName := "data.nomad_acl_roles.test"

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testDataSourceACLRolesConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "prefix"),
					resource.TestCheckResourceAttr(resourceName, "acl_roles.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "acl_roles.0.id"),
					resource.TestCheckResourceAttr(resourceName, "acl_roles.0.name", "acctest-acl-role"),
					resource.TestCheckResourceAttr(resourceName, "acl_roles.0.description", "A Terraform acctest ACL Role"),
					resource.TestCheckResourceAttr(resourceName, "acl_roles.0.policies.#", "1"),
				),
			},
		},
	})
}

const testDataSourceACLRolesConfig = `
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

data "nomad_acl_roles" "test" {
  prefix = split("-", nomad_acl_role.test.id)[0]
}
`
