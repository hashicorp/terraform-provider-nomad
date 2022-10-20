package nomad

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestResourceACLToken_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceACLToken_initialConfig(),
				Check:  testResourceACLToken_initialCheck(),
			},
			{
				ResourceName:      "nomad_acl_token.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},

		CheckDestroy: testResourceACLTokenCheckDestroy,
	})
}

func TestResourceACLToken_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceACLToken_initialConfig(),
				Check:  testResourceACLToken_initialCheck(),
			},
		},

		CheckDestroy: testResourceACLTokenCheckDestroy,
	})
}

func TestResourceACLToken_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceACLToken_initialConfig(),
				Check:  testResourceACLToken_initialCheck(),
			},
			{
				Config: testResourceACLToken_updateConfig(),
				Check:  testResourceACLToken_updateCheck(),
			},
		},

		CheckDestroy: testResourceACLTokenCheckDestroy,
	})
}

func TestResourceACLToken_Expiration(t *testing.T) {

	config, testFn := testResourceACLTokenExpiration()

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.4.0-beta.1") },
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testFn,
			},
		},
		CheckDestroy: testResourceACLTokenCheckDestroy,
	})
}

func TestResourceACLToken_RoleLink(t *testing.T) {

	config, testFn := testResourceACLTokenRoleLink()

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.4.0-beta.1") },
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testFn,
			},
		},
		CheckDestroy: testResourceACLTokenCheckDestroy,
	})
}

func testResourceACLToken_initialConfig() string {
	return `
resource "nomad_acl_token" "test" {
  name = "Terraform Test Token"
  type = "client"
  policies = ["dev", "qa"]
  global = false
}
`
}

func testResourceACLToken_initialCheck() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const (
			name     = "Terraform Test Token"
			typ      = "client"
			policies = "2"
			global   = "false"
		)
		resourceState := s.Modules[0].Resources["nomad_acl_token.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}

		if len(instanceState.ID) < 1 {
			return fmt.Errorf("expected ID to be set, got %q", instanceState.ID)
		}

		if len(instanceState.Attributes["secret_id"]) < 1 {
			return fmt.Errorf("expected secret_id to be set, got %q", instanceState.Attributes["secret_id"])
		}

		if instanceState.Attributes["name"] != name {
			return fmt.Errorf("expected name to be %q, is %q in state", name, instanceState.Attributes["name"])
		}

		if instanceState.Attributes["type"] != typ {
			return fmt.Errorf("expected type to be %q, is %q in state", typ, instanceState.Attributes["type"])
		}

		if instanceState.Attributes["global"] != global {
			return fmt.Errorf("expected global to be %q, is %q in state", global, instanceState.Attributes["global"])
		}

		if instanceState.Attributes["create_time"] == "" {
			return fmt.Errorf("expected create_time to be set, got %q", instanceState.Attributes["create_time"])
		}
		// because policies is a set, it's a pain to try and check the values here
		if instanceState.Attributes["policies.#"] != policies {
			return fmt.Errorf("expected policies.# to be %q, is %q in state", policies, instanceState.Attributes["policies.#"])
		}

		client := testProvider.Meta().(ProviderConfig).client
		token, _, err := client.ACLTokens().Info(instanceState.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back token %q: %s", instanceState.ID, err)
		}

		if token.Name != name {
			return fmt.Errorf("expected name to be %q, is %q in API", name, token.Name)
		}
		if token.Type != typ {
			return fmt.Errorf("expected type to be %q, is %q in API", typ, token.Type)
		}
		if token.Global != false {
			return fmt.Errorf("expected global to be %v, is %v in API", false, token.Global)
		}
		if len(token.Policies) != 2 {
			return fmt.Errorf("expected %d policies, got %v from the API", 2, token.Policies)
		}

		return nil
	}
}

func testResourceACLTokenCheckDestroy(s *terraform.State) error {
	for _, s := range s.Modules[0].Resources {
		if s.Type != "nomad_acl_token" {
			continue
		}
		if s.Primary == nil {
			continue
		}
		client := testProvider.Meta().(ProviderConfig).client
		token, _, err := client.ACLTokens().Info(s.Primary.ID, nil)
		if err != nil && strings.Contains(err.Error(), "404") || token == nil {
			continue
		}
		return fmt.Errorf("Token %q has not been deleted.", token.AccessorID)
	}

	return nil
}

func testResourceACLToken_updateConfig() string {
	return `
resource "nomad_acl_token" "test" {
  name = "Updated Terraform Test Token"
  type = "client"
  policies = ["dev", "qa", "prod"]
  global = false
}
`
}

func testResourceACLToken_updateCheck() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const (
			name     = "Updated Terraform Test Token"
			typ      = "client"
			policies = "3"
			global   = "false"
		)
		resourceState := s.Modules[0].Resources["nomad_acl_token.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}

		if len(instanceState.ID) < 1 {
			return fmt.Errorf("expected ID to be set, got %q", instanceState.ID)
		}

		if len(instanceState.Attributes["secret_id"]) < 1 {
			return fmt.Errorf("expected secret_id to be set, got %q", instanceState.Attributes["secret_id"])
		}

		if instanceState.Attributes["name"] != name {
			return fmt.Errorf("expected name to be %q, is %q in state", name, instanceState.Attributes["name"])
		}

		if instanceState.Attributes["type"] != typ {
			return fmt.Errorf("expected type to be %q, is %q in state", typ, instanceState.Attributes["type"])
		}

		if instanceState.Attributes["global"] != global {
			return fmt.Errorf("expected global to be %q, is %q in state", global, instanceState.Attributes["global"])
		}
		// because policies is a set, it's a pain to try and check the values here
		if instanceState.Attributes["policies.#"] != policies {
			return fmt.Errorf("expected policies.# to be %q, is %q in state", policies, instanceState.Attributes["policies.#"])
		}

		client := testProvider.Meta().(ProviderConfig).client
		token, _, err := client.ACLTokens().Info(instanceState.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back token %q: %s", instanceState.ID, err)
		}

		if token.Name != name {
			return fmt.Errorf("expected name to be %q, is %q in API", name, token.Name)
		}
		if token.Type != typ {
			return fmt.Errorf("expected type to be %q, is %q in API", typ, token.Type)
		}
		if token.Global != false {
			return fmt.Errorf("expected global to be %v, is %v in API", false, token.Global)
		}
		if len(token.Policies) != 3 {
			return fmt.Errorf("expected %d policies, got %v from the API", 3, token.Policies)
		}

		return nil
	}
}

func testResourceACLTokenExpiration() (string, resource.TestCheckFunc) {

	const (
		name          = "terraform-token-test"
		typ           = "client"
		global        = "false"
		numPolicies   = "3"
		expirationTTL = "10m0s"
	)

	config := fmt.Sprintf(`
resource "nomad_acl_token" "test" {
  name           = %q
  type           = %q
  policies       = ["dev", "qa", "prod"]
  global         = %s
  expiration_ttl = %q
}
`, name, typ, global, expirationTTL)

	checkFn := func(s *terraform.State) error {
		resourceState := s.Modules[0].Resources["nomad_acl_token.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}
		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}
		if len(instanceState.ID) < 1 {
			return fmt.Errorf("expected ID to be set, got %q", instanceState.ID)
		}
		if len(instanceState.Attributes["secret_id"]) < 1 {
			return fmt.Errorf("expected secret_id to be set, got %q", instanceState.Attributes["secret_id"])
		}
		if instanceState.Attributes["name"] != name {
			return fmt.Errorf("expected name to be %q, is %q in state",
				name, instanceState.Attributes["name"])
		}
		if instanceState.Attributes["type"] != typ {
			return fmt.Errorf("expected type to be %q, is %q in state",
				typ, instanceState.Attributes["type"])
		}
		if instanceState.Attributes["global"] != global {
			return fmt.Errorf("expected global to be %q, is %q in state",
				global, instanceState.Attributes["global"])
		}
		if instanceState.Attributes["policies.#"] != numPolicies {
			return fmt.Errorf("expected policies.# to be %q, is %q in state",
				numPolicies, instanceState.Attributes["policies.#"])
		}
		if instanceState.Attributes["expiration_ttl"] != expirationTTL {
			return fmt.Errorf("expected expiration_ttl to be %q, is %q in state",
				expirationTTL, instanceState.Attributes["expiration_ttl"])
		}
		if instanceState.Attributes["create_time"] == "" {
			return fmt.Errorf("expected create_time to be set, got %q",
				instanceState.Attributes["create_time"])
		}

		client := testProvider.Meta().(ProviderConfig).client
		token, _, err := client.ACLTokens().Info(instanceState.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back token %q: %s", instanceState.ID, err)
		}

		if token.Name != name {
			return fmt.Errorf("expected name to be %q, is %q in API", name, token.Name)
		}
		if token.Type != typ {
			return fmt.Errorf("expected type to be %q, is %q in API", typ, token.Type)
		}
		if token.Global != false {
			return fmt.Errorf("expected global to be %v, is %v in API", false, token.Global)
		}
		if len(token.Policies) != 3 {
			return fmt.Errorf("expected %d policies, got %v from the API", 3, token.Policies)
		}
		if token.ExpirationTTL.String() != expirationTTL {
			return fmt.Errorf("expected expiration ttl to be %q, is %q in API",
				expirationTTL, token.ExpirationTTL)
		}
		if token.ExpirationTime == nil || token.ExpirationTime.IsZero() {
			return fmt.Errorf("expected expiration time to be set, is %v in API",
				token.ExpirationTime)
		}
		return nil
	}

	return config, checkFn
}

func testResourceACLTokenRoleLink() (string, resource.TestCheckFunc) {

	const (
		name   = "terraform-token-test"
		typ    = "client"
		global = "false"
	)

	config := fmt.Sprintf(`
resource "nomad_acl_policy" "test" {
  name        = "terraform-token-test"
  description = "A Terraform acctest ACL policy"
  rules_hcl   = <<EOT
namespace "default" {
  policy       = "read"
  capabilities = ["submit-job"]
}
EOT
}

resource "nomad_acl_role" "test" {
  name        = "terraform-token-test"
  description = "A Terraform acctest ACL role"

  policy {
    name = nomad_acl_policy.test.name
  }
}

resource "nomad_acl_token" "test" {
  name   = %q
  type   = %q
  global = %s

  role {
    id = nomad_acl_role.test.id
  }
}
`, name, typ, global)

	checkFn := func(s *terraform.State) error {
		resourceState := s.Modules[0].Resources["nomad_acl_token.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}
		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}
		if len(instanceState.ID) < 1 {
			return fmt.Errorf("expected ID to be set, got %q", instanceState.ID)
		}
		if len(instanceState.Attributes["secret_id"]) < 1 {
			return fmt.Errorf("expected secret_id to be set, got %q", instanceState.Attributes["secret_id"])
		}
		if instanceState.Attributes["name"] != name {
			return fmt.Errorf("expected name to be %q, is %q in state",
				name, instanceState.Attributes["name"])
		}
		if instanceState.Attributes["type"] != typ {
			return fmt.Errorf("expected type to be %q, is %q in state",
				typ, instanceState.Attributes["type"])
		}
		if instanceState.Attributes["global"] != global {
			return fmt.Errorf("expected global to be %q, is %q in state",
				global, instanceState.Attributes["global"])
		}
		if instanceState.Attributes["create_time"] == "" {
			return fmt.Errorf("expected create_time to be set, got %q",
				instanceState.Attributes["create_time"])
		}
		if instanceState.Attributes["role.#"] != "1" {
			return fmt.Errorf("expected roles.# to be %q, is %q in state",
				"1", instanceState.Attributes["role.#"])
		}

		client := testProvider.Meta().(ProviderConfig).client
		token, _, err := client.ACLTokens().Info(instanceState.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back token %q: %s", instanceState.ID, err)
		}

		if token.Name != name {
			return fmt.Errorf("expected name to be %q, is %q in API", name, token.Name)
		}
		if token.Type != typ {
			return fmt.Errorf("expected type to be %q, is %q in API", typ, token.Type)
		}
		if token.Global != false {
			return fmt.Errorf("expected global to be %v, is %v in API", false, token.Global)
		}
		if len(token.Policies) != 0 {
			return fmt.Errorf("expected %d policies, got %v from the API", 0, token.Policies)
		}
		if len(token.Roles) != 1 {
			return fmt.Errorf("expected %d roles, got %v from the API", 1, token.Roles)
		}
		return nil
	}

	return config, checkFn
}
