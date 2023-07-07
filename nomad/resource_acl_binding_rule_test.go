// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceACLBindingRule(t *testing.T) {

	initialDescription := ""
	updatedDescription := "updated description"

	initialRoleName := "engineering-read-only"
	updatedRoleName := "engineering-ro"

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.4.4-dev") },
		Steps: []resource.TestStep{
			{
				Config: testResourceACLBindingRuleConfig(initialDescription, initialRoleName),
				Check:  testResourceACLBindingRuleCheck(initialDescription, initialRoleName),
			},
			{
				Config: testResourceACLBindingRuleConfig(updatedDescription, updatedRoleName),
				Check:  testResourceACLBindingRuleCheck(updatedDescription, updatedRoleName),
			},
		},

		CheckDestroy: testResourceACLBindingRuleCheckDestroy,
	})
}

func TestResourceACLManagementBindingRule(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.4.4-dev") },
		Steps: []resource.TestStep{
			{
				Config: testResourceACLBindingManagementRuleConfig(),
				Check:  testResourceACLBindingManagementRuleCheck(),
			},
		},

		CheckDestroy: testResourceACLBindingRuleCheckDestroy,
	})
}

func testResourceACLBindingRuleConfig(description, bindingName string) string {
	return fmt.Sprintf(`
resource "nomad_acl_auth_method" "test" {
	name           = "tf-provider-acl-binding-rule-test-auth-method"
	type           = "OIDC"
	token_locality = "global"
	max_token_ttl  = "10m0s"
	default        = true

	config {
		oidc_discovery_url    = "https://uk.auth0.com/"
		oidc_client_id        = "someclientid"
		oidc_client_secret    = "someclientsecret-t"
		bound_audiences       = ["someclientid"]
		allowed_redirect_uris = [
			"http://localhost:4649/oidc/callback",
			"http://localhost:4646/ui/settings/tokens",
		]
		list_claim_mappings = {
			"http://nomad.internal/roles" : "roles"
		}
	}
}

resource "nomad_acl_binding_rule" "test" {
	description = %q
	auth_method = nomad_acl_auth_method.test.name
	selector    = "engineering in list.roles"
	bind_type   = "role"
	bind_name   = %q

	depends_on = [nomad_acl_auth_method.test]
}

`, description, bindingName)
}

func testResourceACLBindingManagementRuleConfig() string {
	return `
resource "nomad_acl_auth_method" "test" {
	name           = "tf-provider-acl-binding-rule-test-auth-method"
	type           = "OIDC"
	token_locality = "global"
	max_token_ttl  = "10m0s"
	default        = true

	config {
		oidc_discovery_url    = "https://uk.auth0.com/"
		oidc_client_id        = "someclientid"
		oidc_client_secret    = "someclientsecret-t"
		bound_audiences       = ["someclientid"]
		allowed_redirect_uris = [
			"http://localhost:4649/oidc/callback",
			"http://localhost:4646/ui/settings/tokens",
		]
		list_claim_mappings = {
			"http://nomad.internal/roles" : "roles"
		}
	}
}

resource "nomad_acl_binding_rule" "test" {
	description = "management token test"
	auth_method = "tf-provider-acl-binding-rule-test-auth-method"
	selector    = "engineering in list.roles"
	bind_type   = "management"

	depends_on = [nomad_acl_auth_method.test]
}
`
}

func testResourceACLBindingRuleCheck(description, bindName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const (
			expectedAuthMethod = "tf-provider-acl-binding-rule-test-auth-method"
			expectedBindType   = "role"
			expectedSelector   = "engineering in list.roles"
		)

		resourceState := s.Modules[0].Resources["nomad_acl_binding_rule.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}
		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}

		if instanceState.Attributes["description"] != description {
			return fmt.Errorf("expected description to be %q, is %q in state", description,
				instanceState.Attributes["description"])
		}
		if instanceState.Attributes["auth_method"] != expectedAuthMethod {
			return fmt.Errorf("expected auth method to be %q, is %q in state", expectedAuthMethod,
				instanceState.Attributes["auth_method"])
		}
		if instanceState.Attributes["selector"] != expectedSelector {
			return fmt.Errorf("expected selector to be %q, is %q in state", expectedSelector,
				instanceState.Attributes["selector"])
		}
		if instanceState.Attributes["bind_type"] != expectedBindType {
			return fmt.Errorf("expected bind type to be %q, is %q in state", expectedBindType,
				instanceState.Attributes["bind_type"])
		}
		if instanceState.Attributes["bind_name"] != bindName {
			return fmt.Errorf("expected bind name to be %q, is %q in state", bindName,
				instanceState.Attributes["bind_name"])
		}

		bindingRule, _, err := testProvider.Meta().(ProviderConfig).client.ACLBindingRules().Get(instanceState.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back binding rule %q: %s", instanceState.ID, err)
		}
		if bindingRule.ID != instanceState.ID {
			return fmt.Errorf("expected ID to be %q, is %q in API", instanceState.ID, bindingRule.ID)
		}
		if bindingRule.Description != description {
			return fmt.Errorf("expected description to be %q, is %q in API",
				description, bindingRule.Description)
		}
		if bindingRule.AuthMethod != expectedAuthMethod {
			return fmt.Errorf("expected auth method to be %q, is %q in API",
				expectedAuthMethod, bindingRule.AuthMethod)
		}
		if bindingRule.Selector != expectedSelector {
			return fmt.Errorf("expected selector to be %q, is %q in API",
				expectedSelector, bindingRule.Selector)
		}
		if bindingRule.BindType != expectedBindType {
			return fmt.Errorf("expected bind type to be %q, is %q in API",
				expectedBindType, bindingRule.BindType)
		}
		if bindingRule.BindName != bindName {
			return fmt.Errorf("expected bind name to be %q, is %q in API",
				bindName, bindingRule.BindName)
		}

		return nil
	}
}

func testResourceACLBindingManagementRuleCheck() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState := s.Modules[0].Resources["nomad_acl_binding_rule.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}

		return nil
	}
}

func testResourceACLBindingRuleCheckDestroy(s *terraform.State) error {
	for _, s := range s.Modules[0].Resources {
		if s.Type != "nomad_acl_binding_rule" {
			continue
		}
		if s.Primary == nil {
			continue
		}
		client := testProvider.Meta().(ProviderConfig).client
		bindingRule, _, err := client.ACLBindingRules().Get(s.Primary.ID, nil)
		if err != nil && strings.Contains(err.Error(), "404") || bindingRule == nil {
			continue
		}
		return fmt.Errorf("Binding Rule %q has not been deleted.", bindingRule.ID)
	}
	return nil
}
