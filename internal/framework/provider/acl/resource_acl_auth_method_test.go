// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package acl_test

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const testResourceName = "nomad_acl_auth_method.test"

func TestACLAuthMethod(t *testing.T) {
	oidcName := acctest.RandomWithPrefix("tf-nomad-test")
	jwtName := oidcName + "-jwt"
	initialUICallback := "http://localhost:4646/ui/settings/tokens"
	updatedUICallback := "https://10.10.10.10:4646/ui/settings/tokens"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testCheckMinVersion(t, "1.4.4-dev")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(t),
		CheckDestroy:             testACLAuthMethodDestroyedAll(t, oidcName, jwtName),
		Steps: []resource.TestStep{
			{
				Config: testACLAuthMethodConfigFullOIDC(oidcName, initialUICallback, false),
				Check: resource.ComposeTestCheckFunc(
					testACLAuthMethodFullOIDCAPICheck(t, oidcName, initialUICallback, "false"),
					testACLAuthMethodExists(t, jwtName),
				),
			},
			{
				Config: testACLAuthMethodConfigFullOIDC(oidcName, updatedUICallback, true),
				Check: resource.ComposeTestCheckFunc(
					testACLAuthMethodFullOIDCAPICheck(t, oidcName, updatedUICallback, "true"),
					testACLAuthMethodExists(t, jwtName),
				),
			},
		},
	})
}

func TestACLAuthMethod_writeOnly(t *testing.T) {
	methodName := fmt.Sprintf("tf-test-wo-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(t),
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		CheckDestroy: testACLAuthMethodDestroyed(t, methodName),
		Steps: []resource.TestStep{
			{
				Config: testACLAuthMethodConfigOIDCWriteOnly(methodName, "initialsecret", "http://localhost:4649/oidc/callback"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testResourceName,
						tfjsonpath.New("config").AtMapKey("oidc_client_secret_wo_version"),
						knownvalue.Int64Exact(1),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testACLAuthMethodExists(t, methodName),
					resource.TestCheckNoResourceAttr(testResourceName, "config.0.oidc_client_secret_wo"),
				),
			},
			{
				Config: testACLAuthMethodConfigOIDCWriteOnly(methodName, "updatedsecret", "http://localhost:4649/oidc/callback"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testResourceName,
						tfjsonpath.New("config").AtMapKey("oidc_client_secret_wo_version"),
						knownvalue.Int64Exact(2),
					),
				},
			},
			{
				Config: testACLAuthMethodConfigOIDCWriteOnly(methodName, "updatedsecret", "http://new-callback.example.com/oidc/callback"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testResourceName,
						tfjsonpath.New("config").AtMapKey("oidc_client_secret_wo_version"),
						knownvalue.Int64Exact(2),
					),
				},
			},
		},
	})
}

func TestACLAuthMethod_conflictingOIDCSecretAndWO(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-conflict")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(t),
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		Steps: []resource.TestStep{
			{
				Config:      testACLAuthMethodConfigOIDCConflictingSecrets(name),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

func TestACLAuthMethod_conflictingPemKeyAndWO(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-conflict")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(t),
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"tls": {VersionConstraint: ">= 4.0.0", Source: "hashicorp/tls"},
		},
		Steps: []resource.TestStep{
			{
				Config:      testACLAuthMethodConfigPrivateKeyConflictingWO(name),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

func TestACLAuthMethod_OIDCClientAssertion(t *testing.T) {
	methodName := acctest.RandomWithPrefix("tf-nomad-test")
	resourceName := "nomad_acl_auth_method.client_assertion_test"
	ca := func(suffix string) string {
		return "config.oidc_client_assertion." + suffix
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testCheckMinVersion(t, "1.10.0")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(t),
		ExternalProviders: map[string]resource.ExternalProvider{
			"tls":   {VersionConstraint: ">= 4.0.0", Source: "hashicorp/tls"},
			"local": {VersionConstraint: ">= 2.5.0", Source: "hashicorp/local"},
		},
		Steps: []resource.TestStep{
			{
				Config: clientAssertionResourcesHCL(methodName, clientAssertionNomad, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, ca("audience.0"), "http://discovery.url"),
					resource.TestCheckResourceAttr(resourceName, ca("key_algorithm"), "RS256"),
					resource.TestCheckResourceAttr(resourceName, ca("key_source"), "nomad"),
					resource.TestCheckResourceAttr(resourceName, ca("extra_headers.dome"), "noggin"),
				),
			},
			{
				Config: clientAssertionResourcesHCL(methodName, clientAssertionPrivateKey, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, ca("audience.0"), "some-other-audience"),
					resource.TestCheckResourceAttr(resourceName, ca("key_algorithm"), "RS512"),
					resource.TestCheckResourceAttr(resourceName, ca("key_source"), "private_key"),
					resource.TestMatchResourceAttr(resourceName, ca("private_key.pem_key"),
						regexp.MustCompile("RSA PRIVATE KEY-----")),
					resource.TestMatchResourceAttr(resourceName, ca("private_key.pem_cert"),
						regexp.MustCompile("CERTIFICATE-----")),
					resource.TestCheckResourceAttr(resourceName, ca("extra_headers.%"), "0"),
				),
			},
			{
				Config: clientAssertionResourcesHCL(methodName, clientAssertionPrivateKeyFile, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, ca("audience.0"), "http://discovery.url"),
					resource.TestCheckResourceAttr(resourceName, ca("key_algorithm"), "RS256"),
					resource.TestCheckResourceAttr(resourceName, ca("key_source"), "private_key"),
					resource.TestMatchResourceAttr(resourceName, ca("private_key.pem_key_file"),
						regexp.MustCompile("/key.pem")),
					resource.TestMatchResourceAttr(resourceName, ca("private_key.pem_cert_file"),
						regexp.MustCompile("/cert.pem")),
				),
			},
			{
				Config: clientAssertionResourcesHCL(methodName, clientAssertionClientSecret, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, ca("audience.0"), "yet-another-aud"),
					resource.TestCheckResourceAttr(resourceName, ca("key_algorithm"), "HS256"),
					resource.TestCheckResourceAttr(resourceName, ca("key_source"), "client_secret"),
					resource.TestCheckResourceAttr(resourceName, "config.oidc_client_secret", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
					resource.TestCheckResourceAttr(resourceName, ca("private_key.%"), "0"),
				),
			},
		},
		CheckDestroy: testACLAuthMethodDestroyed(t, methodName),
	})
}

func testACLAuthMethodExists(t *testing.T, name string) resource.TestCheckFunc {
	t.Helper()
	return func(s *terraform.State) error {
		client := nomadClientFromMeta(t)
		am, _, err := client.ACLAuthMethods().Get(name, nil)
		if err != nil {
			return fmt.Errorf("error reading ACL auth method %q: %w", name, err)
		}
		if am == nil {
			return fmt.Errorf("ACL auth method %q not found", name)
		}
		return nil
	}
}

func testACLAuthMethodDestroyed(t *testing.T, name string) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		client := nomadClientFromMeta(t)
		am, _, err := client.ACLAuthMethods().Get(name, nil)
		if err != nil && strings.Contains(err.Error(), "404") {
			return nil
		}
		if err != nil {
			return fmt.Errorf("unexpected error checking ACL auth method %q: %w", name, err)
		}
		if am != nil {
			return fmt.Errorf("ACL auth method %q still exists", name)
		}
		return nil
	}
}

func testACLAuthMethodDestroyedAll(t *testing.T, names ...string) resource.TestCheckFunc {
	t.Helper()
	return func(s *terraform.State) error {
		for _, name := range names {
			if err := testACLAuthMethodDestroyed(t, name)(s); err != nil {
				return err
			}
		}
		return nil
	}
}

func testACLAuthMethodFullOIDCAPICheck(t *testing.T, name, uiCallback, defaultVal string) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
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
		expectedBoundAudiences := []string{"someclientid"}
		expectedAllowedRedirectURIs := []string{"http://localhost:4649/oidc/callback", uiCallback}
		expectedClaimMappings := map[string]string{"http://nomad.internal/name": "name"}
		expectedListClaimMappings := map[string]string{"http://nomad.internal/roles": "roles"}

		authMethod, _, err := nomadClientFromMeta(t).ACLAuthMethods().Get(name, nil)
		if err != nil {
			return fmt.Errorf("error reading back auth method %q: %w", name, err)
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
			return fmt.Errorf(`expected OIDC client ID to be %q, is %q in API`,
				expectedOIDCClientID, authMethod.Config.OIDCClientID)
		}
		if authMethod.Config.OIDCEnablePKCE {
			return fmt.Errorf("expected PKCE default to be false, is %v in API", authMethod.Config.OIDCEnablePKCE)
		}
		if authMethod.Config.OIDCDisableUserInfo != expectedOIDCDisableUserInfo {
			return fmt.Errorf(`expected OIDC disable userinfo to be %t, is %t in API`,
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
				expectedAllowedRedirectURIs, authMethod.Config.AllowedRedirectURIs)
		}
		if !reflect.DeepEqual(authMethod.Config.ClaimMappings, expectedClaimMappings) {
			return fmt.Errorf(`expected claim mappings to be %v, is %v in API`,
				expectedClaimMappings, authMethod.Config.ClaimMappings)
		}
		if !reflect.DeepEqual(authMethod.Config.ListClaimMappings, expectedListClaimMappings) {
			return fmt.Errorf(`expected list claim mappings to be %v, is %v in API`,
				expectedListClaimMappings, authMethod.Config.ListClaimMappings)
		}
		if !authMethod.Config.VerboseLogging {
			return fmt.Errorf("expected VerboseLogging to be true, is %v in API", authMethod.Config.VerboseLogging)
		}

		return nil
	}
}

func TestACLAuthMethod_upgradeToFramework(t *testing.T) {
	methodName := fmt.Sprintf("tf-test-jwt-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },

		CheckDestroy: testACLAuthMethodDestroyed(t, methodName),
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"nomad": {
						Source:            "hashicorp/nomad",
						VersionConstraint: "2.5.2",
					},
				},
				Config: testACLAuthMethodConfigOIDCWithMaxTTL(methodName, "someclientsecret", "http://localhost:4649/oidc/callback", false, "10m0s"),
				Check: resource.ComposeTestCheckFunc(
					testACLAuthMethodExists(t, methodName),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(t),
				Config:                   testACLAuthMethodConfigOIDCWithMaxTTL(methodName, "someclientsecret", "http://localhost:4649/oidc/callback", false, "10m0s"),
			},
		},
	})
}

func TestACLAuthMethod_writeOnlyManualVersion(t *testing.T) {
	methodName := fmt.Sprintf("tf-test-wov-%d", time.Now().UnixNano())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(t),
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		CheckDestroy: testACLAuthMethodDestroyed(t, methodName),
		Steps: []resource.TestStep{
			{
				Config: testACLAuthMethodConfigOIDCWriteOnlyManualVersion(methodName, "secret", 1),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testResourceName,
						tfjsonpath.New("config").AtMapKey("oidc_client_secret_wo_version"),
						knownvalue.Int64Exact(1),
					),
				},
			},
			{
				Config: testACLAuthMethodConfigOIDCWriteOnlyManualVersion(methodName, "newsecret", 2),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(testResourceName,
						tfjsonpath.New("config").AtMapKey("oidc_client_secret_wo_version"),
						knownvalue.Int64Exact(2),
					),
				},
			},
		},
	})
}

func testACLAuthMethodConfigFullOIDC(name, uiCallback string, defaultVal bool) string {
	return fmt.Sprintf(`
resource "nomad_acl_auth_method" "test" {
  name            = "%s"
  type            = "OIDC"
  token_locality  = "global"
  token_name_format = "$${auth_method_type}-$${auth_method_name}-$${value.user}"
  max_token_ttl   = "10m"
  default         = %v

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

func testACLAuthMethodConfigOIDCWithMaxTTL(name, secret, redirectURI string, defaultVal bool, maxTokenTTL string) string {
	return fmt.Sprintf(`
resource "nomad_acl_auth_method" "test" {
  name           = %q
  type           = "OIDC"
  token_locality = "global"
  max_token_ttl  = %q
  default        = %v

  config {
    oidc_discovery_url    = "https://uk.auth0.com/"
    oidc_client_id        = "someclientid"
    oidc_client_secret    = %q
    oidc_scopes           = ["email"]
    bound_audiences       = ["someclientid"]
    allowed_redirect_uris = [%q]
    signing_algs          = ["RS256"]
  }
}
`, name, maxTokenTTL, defaultVal, secret, redirectURI)
}

func testACLAuthMethodConfigOIDCWriteOnly(name, secret, redirectURI string) string {
	return fmt.Sprintf(`
resource "nomad_acl_auth_method" "test" {
  name           = %q
  type           = "OIDC"
  token_locality = "global"
  max_token_ttl  = "10m"

  config {
    oidc_discovery_url    = "https://uk.auth0.com/"
    oidc_client_id        = "someclientid"
    oidc_client_secret_wo = %q
    oidc_scopes           = ["email"]
    bound_audiences       = ["someclientid"]
    allowed_redirect_uris = [%q]
    signing_algs          = ["RS256"]
  }
}
`, name, secret, redirectURI)
}

func testACLAuthMethodConfigOIDCConflictingSecrets(name string) string {
	return fmt.Sprintf(`
resource "nomad_acl_auth_method" "test" {
  name           = %q
  type           = "OIDC"
  token_locality = "global"
  max_token_ttl  = "10m"

  config {
    oidc_discovery_url    = "https://uk.auth0.com/"
    oidc_client_id        = "someclientid"
    oidc_client_secret    = "stored-secret-value"
    oidc_client_secret_wo = "write-only-secret-value"
    oidc_scopes           = ["email"]
    bound_audiences       = ["someclientid"]
    allowed_redirect_uris = ["http://localhost:4649/oidc/callback"]
    signing_algs          = ["RS256"]
  }
}
`, name)
}

func testACLAuthMethodConfigPrivateKeyConflictingWO(name string) string {
	return fmt.Sprintf(`
resource "tls_private_key" "conflict_test" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

resource "nomad_acl_auth_method" "test" {
  name           = %q
  type           = "OIDC"
  token_locality = "global"
  max_token_ttl  = "10m"

  config {
    oidc_discovery_url = "http://discovery.url"
    oidc_client_id     = "someclientid"
    allowed_redirect_uris = [
      "http://localhost:4649/oidc/callback",
    ]
    oidc_client_assertion {
      key_source = "private_key"
      private_key {
        pem_key    = tls_private_key.conflict_test.private_key_pem
        pem_key_wo = "write-only-pem-not-allowed-with-pem_key"
      }
    }
  }
}
`, name)
}

func testACLAuthMethodConfigOIDCWriteOnlyManualVersion(name, secret string, version int) string {
	return fmt.Sprintf(`
resource "nomad_acl_auth_method" "test" {
  name           = %q
  type           = "OIDC"
  token_locality = "global"
  max_token_ttl  = "10m"

  config {
    oidc_discovery_url           = "https://uk.auth0.com/"
    oidc_client_id               = "someclientid"
    oidc_client_secret_wo        = %q
    oidc_client_secret_wo_version = %d
    oidc_scopes                  = ["email"]
    bound_audiences              = ["someclientid"]
    allowed_redirect_uris        = ["http://localhost:4649/oidc/callback"]
    signing_algs                 = ["RS256"]
  }
}
`, name, secret, version)
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
    oidc_discovery_url = "http://discovery.url"
    oidc_client_id     = "someclientid"

    allowed_redirect_uris = [
      "http://localhost:4649/oidc/callback",
    ]

%s

  }
}
`
	clientAssertionClientSecret clientAssertionBlock = `
    oidc_client_secret = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
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
    "server_auth",
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
