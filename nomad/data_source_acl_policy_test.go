package nomad

import (
	"fmt"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"testing"
)

func TestAccDataSourceNomadAclPolicy_Basic(t *testing.T) {
	policyName := "testpolicy"
	resourceName := "data.nomad_acl_policy.test-policy"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testProviders,
		CheckDestroy: testResourceACLPolicy_checkDestroy(policyName),
		Steps: []resource.TestStep{
			{
				Config: testAccNomadAclPolicyConfig(policyName),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceNomadAclPolicyExists(resourceName),
					resource.TestCheckResourceAttr(
						resourceName, "name", policyName),
					resource.TestCheckResourceAttr(
						resourceName, "description", "Test ACL Policy"),
					resource.TestCheckResourceAttrSet(resourceName, "rules"),
				),
			},
		},
	})
}

func testAccNomadAclPolicyConfig(name string) string {
	return `
resource "nomad_acl_policy" "policy-instance" {
	name        = "` + name + `"
	description = "Test ACL Policy"
    rules_hcl   = <<EOT
namespace "default" {
  policy = "write"
}
EOT
}

data "nomad_acl_policy" "test-policy" {
	name = "${nomad_acl_policy.policy-instance.name}"
}
`
}

func testAccDataSourceNomadAclPolicyExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("ACL Policy not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ACL Policy Name is set")
		}

		providerConfig := testProvider.Meta().(ProviderConfig)
		client := providerConfig.client

		name := rs.Primary.ID

		// Try to get the Policy
		policy, _, err := client.ACLPolicies().Info(name, &api.QueryOptions{})
		if err != nil {
			return err
		}

		if policy.Name != name {
			return fmt.Errorf("ACL Policy not found")
		}

		return nil
	}
}
