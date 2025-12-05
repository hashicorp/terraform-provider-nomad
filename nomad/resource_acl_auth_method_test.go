// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
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
  max_token_ttl  	= "10m"
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
    verbose_logging = true
  }
}

resource "nomad_acl_auth_method" "test-jwt" {
  name = "%s-jwt"
  type = "JWT"
  config {
    jwks_url = "https://somewhere/.well-known/jwks.json"
  }
  token_locality = "global"
  max_token_ttl  = "10m"
}
`, name, defaultVal, uiCallback, name)
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
			"config.0.oidc_enable_pkce":                                "false",
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
			"config.0.verbose_logging":                                 "true",
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
		if authMethod.Config.OIDCEnablePKCE {
			return fmt.Errorf("expected PKCE default to be false, is %v in API", authMethod.Config.OIDCEnablePKCE)
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
		if !authMethod.Config.VerboseLogging {
			return fmt.Errorf("expected VerboseLogging to be true, is %v in API", authMethod.Config.VerboseLogging)
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

// client assertion permutations are complex, so test them separately
func TestResourceACLAuthMethod_OIDCClientAssertion(t *testing.T) {
	methodName := acctest.RandomWithPrefix("tf-nomad-test")
	resourceName := "nomad_acl_auth_method.client_assertion_test"
	attrPrefix := "config.0.oidc_client_assertion.0."

	resource.Test(t, resource.TestCase{
		ProviderFactories: testProviderFactories,
		PreCheck:          func() { testAccPreCheck(t); testCheckMinVersion(t, "1.10.0") },
		ExternalProviders: map[string]resource.ExternalProvider{
			"tls":   {VersionConstraint: ">= 4.0.0", Source: "hashicorp/tls"},
			"local": {VersionConstraint: ">= 2.5.0", Source: "hashicorp/local"},
		},
		Steps: []resource.TestStep{
			{
				Config: clientAssertionResourcesHCL(methodName, clientAssertionNomad, false),
				Check: resource.ComposeTestCheckFunc(
					// audience and algorithm are optional and computed
					// nomad server defaults audience = [oidc_discovery_url]
					// and alg = RS256 for the nomad key_source
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"audience.0", "http://discovery.url"),
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"key_algorithm", "RS256"),
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"key_source", "nomad"),
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"extra_headers.dome", "noggin"),
				),
			},
			{
				Config: clientAssertionResourcesHCL(methodName, clientAssertionPrivateKey, true),
				Check: resource.ComposeTestCheckFunc(
					// aud and algo set explicitly
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"audience.0", "some-other-audience"),
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"key_algorithm", "RS512"),
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"key_source", "private_key"),
					resource.TestMatchResourceAttr(resourceName, attrPrefix+"private_key.0.pem_key",
						regexp.MustCompile("RSA PRIVATE KEY-----")),
					resource.TestMatchResourceAttr(resourceName, attrPrefix+"private_key.0.pem_cert",
						regexp.MustCompile("CERTIFICATE-----")),
					// headers should be removed
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"extra_headers.%", "0"),
				),
			},
			{
				Config: clientAssertionResourcesHCL(methodName, clientAssertionPrivateKeyFile, true),
				Check: resource.ComposeTestCheckFunc(
					// aud and algo implicitly remain, due to being optional and computed
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"audience.0", "some-other-audience"),
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"key_algorithm", "RS512"),
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"key_source", "private_key"),
					resource.TestMatchResourceAttr(resourceName, attrPrefix+"private_key.0.pem_key_file",
						regexp.MustCompile("/key.pem")),
					resource.TestMatchResourceAttr(resourceName, attrPrefix+"private_key.0.pem_cert_file",
						regexp.MustCompile("/cert.pem")),
				),
			},
			{
				Config: clientAssertionResourcesHCL(methodName, clientAssertionClientSecret, false),
				Check: resource.ComposeTestCheckFunc(
					// aud and algo can still be changed explicitly
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"audience.0", "yet-another-aud"),
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"key_algorithm", "HS256"),
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"key_source", "client_secret"),
					resource.TestCheckResourceAttr(resourceName, "config.0.oidc_client_secret", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
					resource.TestCheckResourceAttr(resourceName, attrPrefix+"private_key.%", "0"),
				),
			},
		},
		CheckDestroy: testResourceACLAuthMethodCheckDestroy(methodName),
	})
}

func clientAssertionResourcesHCL(authMethodName string, block clientAssertionBlock, withTLS bool) string {
	conf := fmt.Sprintf(clientAssertionHCLFormat, authMethodName, block)
	if withTLS {
		conf += tlsResourcesHCL
	}
	return conf
}

type clientAssertionBlock string

const (
	clientAssertionHCLFormat = `
resource "nomad_acl_auth_method" "client_assertion_test" {
  name           = "%s"
  type           = "OIDC"
  token_locality = "global"
  max_token_ttl  = "10m0s"
  default        = true

  config {
    # required
    oidc_discovery_url = "http://discovery.url"
    oidc_client_id     = "someclientid"

    # really ought to be required
    allowed_redirect_uris = [
      "http://localhost:4649/oidc/callback",
    ]

# CLIENT ASSERTION BLOCK GOES HERE
%s

  }
}
`
	clientAssertionClientSecret clientAssertionBlock = `
    oidc_client_secret = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" # 32 bytes
    oidc_client_assertion {
      audience      = ["yet-another-aud"]
      key_algorithm = "HS256"
      key_source    = "client_secret"
    }
`
	clientAssertionNomad clientAssertionBlock = `
    oidc_client_assertion {
      key_source = "nomad"
      extra_headers = {
        dome = "noggin"
      }
    }
`
	clientAssertionPrivateKey clientAssertionBlock = `
    oidc_client_assertion {
      audience      = ["some-other-audience"]
      key_source    = "private_key"
      key_algorithm = "RS512"
      private_key {
        pem_key  = tls_private_key.test.private_key_pem
        pem_cert = tls_self_signed_cert.test.cert_pem
      }
    }
`
	clientAssertionPrivateKeyFile clientAssertionBlock = `
    oidc_client_assertion {
      key_source = "private_key"
      private_key {
        pem_key_file  = abspath(local_sensitive_file.key.filename)
        pem_cert_file = abspath(local_file.cert.filename)
      }
    }
`
	tlsResourcesHCL = `
resource "tls_private_key" "test" {
  algorithm = "RSA"
  rsa_bits  = 4096
}
resource "tls_self_signed_cert" "test" {
  private_key_pem = tls_private_key.test.private_key_pem
  subject {
    common_name  = "nomadproject.io"
    organization = "HashiCorp"
  }
  validity_period_hours = 1
  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "server_auth", # TODO: ?
  ]
}
resource "local_sensitive_file" "key" {
  content  = tls_private_key.test.private_key_pem
  filename = "${path.module}/key.pem"
}
resource "local_file" "cert" {
  content  = tls_self_signed_cert.test.cert_pem
  filename = "${path.module}/cert.pem"
}
`
)
