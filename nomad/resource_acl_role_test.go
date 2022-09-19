package nomad

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestResourceACLRole(t *testing.T) {

	testResourceName := acctest.RandomWithPrefix("tf-nomad-test")
	testACLRoleNameUpdated := testResourceName + "-updated"

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.4.0-beta.1") },
		Steps: []resource.TestStep{
			{
				Config: testResourceACLRoleConfig(testResourceName, testResourceName),
				Check:  testResourceACLRoleCheck(testResourceName),
			},
			{
				Config: testResourceACLRoleConfig(testResourceName, testACLRoleNameUpdated),
				Check:  testResourceACLRoleCheck(testACLRoleNameUpdated),
			},
		},
		CheckDestroy: resourceACLRoleCheckDestroy,
	})
}

func testResourceACLRoleConfig(policyName, roleName string) string {
	return fmt.Sprintf(`
resource "nomad_acl_policy" "test" {
  name        = %q
  description = "A Terraform acctest ACL policy"
  rules_hcl   = <<EOT
namespace "default" {
  policy       = "read"
  capabilities = ["submit-job"]
}
EOT
}

resource "nomad_acl_role" "test" {
  name        = %q
  description = "A Terraform acctest ACL role"
  depends_on  = [nomad_acl_policy.test]

  policies {
    name = nomad_acl_policy.test.name
  }
} `, policyName, roleName)
}

func testResourceACLRoleCheck(roleName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		resourceState := s.Modules[0].Resources["nomad_acl_role.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}

		if instanceState.ID == "" {
			return fmt.Errorf("expected ID to be set, got %q", instanceState.ID)
		}

		if len(instanceState.Attributes["id"]) < 1 {
			return fmt.Errorf("expected id to be set, got %q", instanceState.Attributes["id"])
		}

		if instanceState.Attributes["name"] != roleName {
			return fmt.Errorf("expected name to be %q, is %q in state", roleName, instanceState.Attributes["name"])
		}

		// because policies is a set, it's a pain to try and check the values here
		if instanceState.Attributes["policies.#"] != "1" {
			return fmt.Errorf(`expected policies.# to be "1", is %q in state`,
				instanceState.Attributes["policies.#"])
		}

		client := testProvider.Meta().(ProviderConfig).client
		role, _, err := client.ACLRoles().Get(instanceState.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back ACL role %q: %s", instanceState.ID, err)
		}

		if role.Name != roleName {
			return fmt.Errorf("expected name to be %q, is %q in API", roleName, role.Name)
		}
		if len(role.Policies) != 1 {
			return fmt.Errorf("expected %d policies, got %v from the API", 1, role.Policies)
		}

		return nil
	}
}

func resourceACLRoleCheckDestroy(s *terraform.State) error {

	client := testProvider.Meta().(ProviderConfig).client

	for _, s := range s.Modules[0].Resources {

		if s.Primary == nil {
			continue
		}

		switch s.Type {
		case "nomad_acl_role":
			role, _, err := client.ACLRoles().Get(s.Primary.ID, nil)
			if err != nil && strings.Contains(err.Error(), "404") || role == nil {
				continue
			}
			return fmt.Errorf("ACL Role %q has not been deleted.", role.ID)
		case "nomad_acl_policy":
			policy, _, err := client.ACLPolicies().Info(s.Primary.ID, nil)
			if err != nil && strings.Contains(err.Error(), "404") || policy == nil {
				continue
			}
			return fmt.Errorf("ACL Policy %q has not been deleted.", policy.Name)
		default:
			continue
		}
	}
	return nil
}
