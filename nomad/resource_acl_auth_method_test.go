// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"golang.org/x/exp/slices"
)

func TestResourceACLAuthMethod(t *testing.T) {

	testResourceName := acctest.RandomWithPrefix("tf-nomad-test")
	initialUICallback := "http://localhost:4646/ui/settings/tokens"
	updatedUICallback := "https://10.10.10.10:4646/ui/settings/tokens"

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.4.4-dev") },
		Steps: []resource.TestStep{
			{
				Config: testResourceACLAuthMethodConfig(testResourceName, initialUICallback, false),
				Check:  testResourceACLAuthMethodCheck(testResourceName, initialUICallback, "false"),
			},
			{
				Config: testResourceACLAuthMethodConfig(testResourceName, updatedUICallback, true),
				Check:  testResourceACLAuthMethodCheck(testResourceName, updatedUICallback, "true"),
			},
		},
		CheckDestroy: testResourceACLAuthMethodCheckDestroy(testResourceName),
	})
}

func testResourceACLAuthMethodConfig(name, uiCallback string, defaultVal bool) string {
	return fmt.Sprintf(`
resource "nomad_acl_auth_method" "test" {
  name           	= "%s"
  type           	= "OIDC"
  token_locality 	= "global"
  token_name_format	= "$${auth_method_type}-$${auth_method_name}-$${value.user}"
  max_token_ttl  	= "10m0s"
  default        	= %v

  config {
    oidc_discovery_url    = "https://uk.auth0.com/"
    oidc_client_id        = "someclientid"
    oidc_client_secret    = "someclientsecret-t"
    oidc_scopes           = ["email"]
    bound_audiences       = ["someclientid"]
    discovery_ca_pem      = ["secretpemcert"]
    signing_algs          = ["rsa256", "hs256"]
    oidc_disable_userinfo = true
    allowed_redirect_uris = [
      "http://localhost:4649/oidc/callback",
      %q,
    ]
    claim_mappings = {
      "http://nomad.internal/name": "name"
    }
    list_claim_mappings = {
      "http://nomad.internal/roles": "roles"
    }
  }
}
`, name, defaultVal, uiCallback)
}

func testResourceACLAuthMethodCheck(name, uiCallback, defaultVal string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const (
			expectedType                = "OIDC"
			expectedTokenLocality       = "global"
			expectedMaxTokenTTL         = "10m0s"
			expectedTokenNameFormat     = "${auth_method_type}-${auth_method_name}-${value.user}"
			expectedOIDCDiscoveryURL    = "https://uk.auth0.com/"
			expectedOIDCClientID        = "someclientid"
			expectedOIDCDisableUserInfo = true
			expectedOIDCClientSecret    = "redacted"
		)
		var (
			expectedBoundAudiences      = []string{"someclientid"}
			expectedAllowedRedirectURIs = []string{"http://localhost:4649/oidc/callback", uiCallback}
			expectedListClaimMappings   = map[string]string{"http://nomad.internal/roles": "roles"}
		)
		resourceState := s.Modules[0].Resources["nomad_acl_auth_method.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}
		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}
		if instanceState.ID != name {
			return fmt.Errorf("expected ID to be %q, got %q", name, instanceState.ID)
		}
		if instanceState.Attributes["name"] != name {
			return fmt.Errorf("expected name to be %q, is %q in state", name, instanceState.Attributes["name"])
		}
		if instanceState.Attributes["type"] != expectedType {
			return fmt.Errorf("expected type to be %q, is %q in state", name, instanceState.Attributes["type"])
		}
		if instanceState.Attributes["token_locality"] != expectedTokenLocality {
			return fmt.Errorf("expected token_locality to be %q, is %q in state", name, instanceState.Attributes["token_locality"])
		}
		if instanceState.Attributes["max_token_ttl"] != expectedMaxTokenTTL {
			return fmt.Errorf("expected max_token_ttl to be %q, is %q in state", name, instanceState.Attributes["max_token_ttl"])
		}
		if instanceState.Attributes["default"] != defaultVal {
			return fmt.Errorf("expected default to be %q, is %q in state", name, instanceState.Attributes["default"])
		}

		// Use a map to check the config list entries, so it's a little easier
		// to manage.
		configExpectedEntries := map[string]string{
			"config.#":                                                 "1",
			"config.0.oidc_discovery_url":                              "https://uk.auth0.com/",
			"config.0.oidc_client_id":                                  "someclientid",
			"config.0.oidc_client_secret":                              "someclientsecret-t",
			"config.0.bound_audiences.#":                               "1",
			"config.0.bound_audiences.0":                               "someclientid",
			"config.0.allowed_redirect_uris.#":                         "2",
			"config.0.allowed_redirect_uris.0":                         expectedAllowedRedirectURIs[0],
			"config.0.allowed_redirect_uris.1":                         expectedAllowedRedirectURIs[1],
			"config.0.oidc_scopes.#":                                   "1",
			"config.0.oidc_scopes.0":                                   "email",
			"config.0.discovery_ca_pem.#":                              "1",
			"config.0.discovery_ca_pem.0":                              "secretpemcert",
			"config.0.signing_algs.#":                                  "2",
			"config.0.signing_algs.0":                                  "rsa256",
			"config.0.signing_algs.1":                                  "hs256",
			"config.0.claim_mappings.%":                                "1",
			"config.0.claim_mappings.http://nomad.internal/name":       "name",
			"config.0.list_claim_mappings.%":                           "1",
			"config.0.list_claim_mappings.http://nomad.internal/roles": "roles",
		}

		for testKey, testValue := range configExpectedEntries {
			actualValue, ok := instanceState.Attributes[testKey]
			if !ok {
				return fmt.Errorf("expected key %q not found in attributes", testKey)
			}
			if actualValue != testValue {
				return fmt.Errorf("expected config key %s to be %q, is %q in state",
					testKey, testValue, actualValue)
			}
		}

		authMethod, _, err := testProvider.Meta().(ProviderConfig).client.ACLAuthMethods().Get(name, nil)
		if err != nil {
			return fmt.Errorf("error reading back auth method %q: %s", name, err)
		}
		if authMethod.Name != name {
			return fmt.Errorf("expected name to be %q, is %q in API", name, authMethod.Name)
		}
		if authMethod.Type != expectedType {
			return fmt.Errorf("expected type to be %q, is %q in API", expectedType, authMethod.Type)
		}
		if authMethod.TokenLocality != expectedTokenLocality {
			return fmt.Errorf("expected token locality to be %q, is %q in API", expectedTokenLocality, authMethod.TokenLocality)
		}
		if authMethod.MaxTokenTTL.String() != expectedMaxTokenTTL {
			return fmt.Errorf("expected max token TTL to be %q, is %q in API", expectedMaxTokenTTL, authMethod.MaxTokenTTL)
		}
		if authMethod.TokenNameFormat != expectedTokenNameFormat {
			return fmt.Errorf("expected token name format to be %q, is %q in API", expectedTokenNameFormat, authMethod.TokenNameFormat)
		}
		if strconv.FormatBool(authMethod.Default) != defaultVal {
			return fmt.Errorf(`expected default to be %q, is "%v" in API`, defaultVal, authMethod.Default)
		}
		if authMethod.Config.OIDCDiscoveryURL != expectedOIDCDiscoveryURL {
			return fmt.Errorf(`expected OIDC discovery URL to be %q, is %q in API`,
				expectedOIDCDiscoveryURL, authMethod.Config.OIDCDiscoveryURL)
		}
		if authMethod.Config.OIDCClientID != expectedOIDCClientID {
			return fmt.Errorf(`expected OIDC client ID to be %q, is %q" in API`,
				expectedOIDCClientID, authMethod.Config.OIDCClientID)
		}
		if authMethod.Config.OIDCDisableUserInfo != expectedOIDCDisableUserInfo {
			return fmt.Errorf(`expected OIDC disable userinfo to be %t, is %t" in API`,
				expectedOIDCDisableUserInfo, authMethod.Config.OIDCDisableUserInfo)
		}
		if authMethod.Config.OIDCClientSecret != expectedOIDCClientSecret {
			return fmt.Errorf(`expected OIDC client secret to be %q, is %q in API`,
				expectedOIDCClientSecret, authMethod.Config.OIDCClientSecret)
		}
		if !slices.Equal(authMethod.Config.BoundAudiences, expectedBoundAudiences) {
			return fmt.Errorf(`expected bound audiences to be %q, is %q in API`,
				expectedBoundAudiences, authMethod.Config.BoundAudiences)
		}
		if !slices.Equal(authMethod.Config.AllowedRedirectURIs, expectedAllowedRedirectURIs) {
			return fmt.Errorf(`expected allowed redirect URIs to be %q, is %q in API`,
				expectedBoundAudiences, authMethod.Config.BoundAudiences)
		}
		if !reflect.DeepEqual(authMethod.Config.ListClaimMappings, expectedListClaimMappings) {
			return fmt.Errorf(`expected list claim mappings to be %q, is %q in API`,
				expectedListClaimMappings, authMethod.Config.ListClaimMappings)
		}

		return nil
	}
}

func testResourceACLAuthMethodCheckDestroy(name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client
		authMethod, _, err := client.ACLAuthMethods().Get(name, nil)
		if err != nil && strings.Contains(err.Error(), "404") || authMethod == nil {
			return nil
		}
		return fmt.Errorf("Auth Method %q has not been deleted", name)
	}
}
