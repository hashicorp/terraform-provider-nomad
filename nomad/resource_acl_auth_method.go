// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceACLAuthMethod() *schema.Resource {
	return &schema.Resource{
		Create: resourceACLAuthMethodCreate,
		Update: resourceACLAuthMethodUpdate,
		Delete: resourceACLAuthMethodDelete,
		Read:   resourceACLAuthMethodRead,
		Exists: resourceACLAuthMethodExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The identifier of the ACL Auth Method.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"type": {
				Description: `ACL Auth Method SSO workflow type. Currently, the only supported type is "OIDC."`,
				Required:    true,
				Type:        schema.TypeString,
				ValidateFunc: validation.StringInSlice([]string{
					api.ACLAuthMethodTypeOIDC,
					api.ACLAuthMethodTypeJWT,
				}, false),
			},
			"token_locality": {
				Description: `Defines whether the ACL Auth Method creates a local or global token when performing SSO login. This field must be set to either "local" or "global".`,
				Required:    true,
				Type:        schema.TypeString,
				ValidateFunc: validation.StringInSlice([]string{
					api.ACLAuthMethodTokenLocalityGlobal,
					api.ACLAuthMethodTokenLocalityLocal,
				}, false),
			},
			"max_token_ttl": {
				Description: "Defines the maximum life of a token created by this method.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"token_name_format": {
				Description: "Defines the token format for the authenticated users. This can be lightly templated using HIL '${foo}' syntax.",
				Optional:    true,
				Type:        schema.TypeString,
				Default:     "${auth_method_type}-${auth_method_name}",
			},
			"default": {
				Description: "Defines whether this ACL Auth Method is to be set as default.",
				Optional:    true,
				Type:        schema.TypeBool,
			},
			"config": {
				Description: "Configuration specific to the auth method provider.",
				Required:    true,
				Type:        schema.TypeList,
				MaxItems:    1,
				MinItems:    1,
				Elem:        resourceACLAuthMethodConfig(),
			},
		},
	}
}

func resourceACLAuthMethodConfig() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"jwt_validation_pub_keys": {
				Description:  "List of PEM-encoded public keys to use to authenticate signatures locally.",
				Type:         schema.TypeList,
				Elem:         &schema.Schema{Type: schema.TypeString},
				Optional:     true,
				ExactlyOneOf: []string{"config.0.jwks_url", "config.0.oidc_discovery_url"},
			},
			"jwks_url": {
				Description: "JSON Web Key Sets url for authenticating signatures.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"jwks_ca_cert": {
				Description: "PEM encoded CA cert for use by the TLS client used to talk with the JWKS server.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"oidc_discovery_url": {
				Description:  "The OIDC Discovery URL, without any .well-known component (base path).",
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"config.0.oidc_client_id", "config.0.oidc_client_secret"},
			},
			"oidc_client_id": {
				Description: "The OAuth Client ID configured with the OIDC provider.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"oidc_client_secret": {
				Description: "The OAuth Client Secret configured with the OIDC provider.",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
			},
			"oidc_disable_userinfo": {
				Description: "Nomad will not make a request to the identity provider to get OIDC UserInfo.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"oidc_scopes": {
				Description: "List of OIDC scopes.",
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
			"bound_audiences": {
				Description: "List of auth claims that are valid for login.",
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
			"bound_issuer": {
				Description: "The value against which to match the iss claim in a JWT.",
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
			"allowed_redirect_uris": {
				Description: "A list of allowed values that can be used for the redirect URI.",
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
			"discovery_ca_pem": {
				Description: "PEM encoded CA certs for use by the TLS client used to talk with the OIDC Discovery URL.",
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
			"signing_algs": {
				Description: "A list of supported signing algorithms.",
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
			"expiration_leeway": {
				Description: `Duration of leeway when validating expiration of a JWT in the form of a time duration such as "5m" or "1h".`,
				Type:        schema.TypeString,
				Default:     "0s",
				Optional:    true,
			},
			"not_before_leeway": {
				Description: `Duration of leeway when validating not before values of a token in the form of a time duration such as "5m" or "1h".`,
				Type:        schema.TypeString,
				Default:     "0s",
				Optional:    true,
			},
			"clock_skew_leeway": {
				Description: `Duration of leeway when validating all claims in the form of a time duration such as "5m" or "1h".`,
				Type:        schema.TypeString,
				Default:     "0s",
				Optional:    true,
			},
			"claim_mappings": {
				Description: "Mappings of claims (key) that will be copied to a metadata field (value).",
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
			"list_claim_mappings": {
				Description: "Mappings of list claims (key) that will be copied to a metadata field (value).",
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
		},
	}
}

func resourceACLAuthMethodCreate(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	aclAuthMethod, err := generateNomadACLAuthMethod(d)
	if err != nil {
		return err
	}

	// Create our ACL auth method.
	log.Print("[DEBUG] Creating ACL Auth Method")
	aclBindingRuleCreateResp, _, err := client.ACLAuthMethods().Create(aclAuthMethod, nil)
	if err != nil {
		return fmt.Errorf("error creating ACL Auth Method: %s", err.Error())
	}
	log.Printf("[DEBUG] Created ACL Auth Method %q", aclBindingRuleCreateResp.Name)
	d.SetId(aclBindingRuleCreateResp.Name)

	return resourceACLAuthMethodRead(d, meta)
}

func resourceACLAuthMethodDelete(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	authMethodName := d.Id()

	// Delete the ACL auth method.
	log.Printf("[DEBUG] Deleting ACL Auth Method %q", authMethodName)
	_, err := client.ACLAuthMethods().Delete(authMethodName, nil)
	if err != nil {
		return fmt.Errorf("error deleting ACL Auth Method %q: %s", authMethodName, err.Error())
	}
	log.Printf("[DEBUG] Deleted ACL Auth Method %q", authMethodName)

	d.SetId("")

	return nil
}

func resourceACLAuthMethodUpdate(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	aclBindingRule, err := generateNomadACLAuthMethod(d)
	if err != nil {
		return err
	}

	// Perform the in-place update of the ACL auth method.
	log.Printf("[DEBUG] Updating ACL Auth Method %q", aclBindingRule.Name)
	_, _, err = client.ACLAuthMethods().Update(aclBindingRule, nil)
	if err != nil {
		return fmt.Errorf("error updating ACL Auth Method %q: %s", aclBindingRule.Name, err.Error())
	}
	log.Printf("[DEBUG] Updated ACL Auth Method %q", aclBindingRule.Name)

	return resourceACLAuthMethodRead(d, meta)
}

func resourceACLAuthMethodRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	authMethodName := d.Id()

	// If the auth method has not been created, the ID will be an empty string
	// which means we can skip attempting to perform the lookup.
	if authMethodName == "" {
		return nil
	}

	log.Printf("[DEBUG] Reading ACL Auth Method %q", authMethodName)
	authMethod, _, err := client.ACLAuthMethods().Get(authMethodName, nil)
	if err != nil {
		return fmt.Errorf("error reading ACL Auth Method %q: %s", authMethodName, err.Error())
	}
	log.Printf("[DEBUG] Read ACL Auth Method %q", authMethod.Name)

	_ = d.Set("name", authMethod.Name)
	_ = d.Set("type", authMethod.Type)
	_ = d.Set("token_locality", authMethod.TokenLocality)
	_ = d.Set("max_token_ttl", authMethod.MaxTokenTTL.String())
	_ = d.Set("token_name_format", authMethod.TokenNameFormat)
	_ = d.Set("default", authMethod.Default)
	_ = d.Set("config", flattenACLAuthMethodConfig(authMethod.Config))

	return nil
}

func resourceACLAuthMethodExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	authMethodName := d.Id()

	log.Printf("[DEBUG] Checking if ACL Auth Method %q exists", authMethodName)
	_, _, err := client.ACLAuthMethods().Get(authMethodName, nil)
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return true, fmt.Errorf("error checking for ACL Auth Method %q: %#v", authMethodName, err)
	}

	return true, nil
}

func generateNomadACLAuthMethod(d *schema.ResourceData) (*api.ACLAuthMethod, error) {

	aclAuthMethod := api.ACLAuthMethod{
		Name:            d.Get("name").(string),
		Type:            d.Get("type").(string),
		TokenLocality:   d.Get("token_locality").(string),
		TokenNameFormat: d.Get("token_name_format").(string),
		Default:         d.Get("default").(bool),
	}

	// Pull the string value of the token TTL and parse this as a time
	// duration.
	if ttlString := d.Get("max_token_ttl").(string); ttlString != "" {
		ttl, err := time.ParseDuration(ttlString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse max_token_ttl: %v", err)
		}
		aclAuthMethod.MaxTokenTTL = ttl
	}

	configList := d.Get("config").([]interface{})

	for _, cfg := range configList {
		authMethodConfig, err := generateNomadACLAuthMethodConfig(cfg)
		if err != nil {
			return nil, err
		}
		aclAuthMethod.Config = authMethodConfig
	}

	return &aclAuthMethod, nil
}

func generateNomadACLAuthMethodConfig(intf interface{}) (*api.ACLAuthMethodConfig, error) {

	configMap, ok := intf.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type %T for auth method config, expected map[string]interface{}", intf)
	}

	var authMethodConfig api.ACLAuthMethodConfig

	for k, v := range configMap {
		switch k {
		case "jwt_validation_pub_keys":
			unpacked, err := unpackStringArray(v, "jwt_validation_pub_keys")
			if err != nil {
				return nil, err
			}
			authMethodConfig.JWTValidationPubKeys = unpacked
		case "jwks_url":
			authMethodConfig.JWKSURL = v.(string)
		case "jwks_ca_cert":
			authMethodConfig.JWKSCACert = v.(string)
		case "oidc_discovery_url":
			authMethodConfig.OIDCDiscoveryURL = v.(string)
		case "oidc_client_id":
			authMethodConfig.OIDCClientID = v.(string)
		case "oidc_client_secret":
			authMethodConfig.OIDCClientSecret = v.(string)
		case "oidc_disable_userinfo":
			authMethodConfig.OIDCDisableUserInfo = v.(bool)
		case "oidc_scopes":
			unpacked, err := unpackStringArray(v, "oidc_scopes")
			if err != nil {
				return nil, err
			}
			authMethodConfig.OIDCScopes = unpacked
		case "bound_audiences":
			unpacked, err := unpackStringArray(v, "bound_audiences")
			if err != nil {
				return nil, err
			}
			authMethodConfig.BoundAudiences = unpacked
		case "bound_issuer":
			unpacked, err := unpackStringArray(v, "bound_issuer")
			if err != nil {
				return nil, err
			}
			authMethodConfig.BoundIssuer = unpacked
		case "allowed_redirect_uris":
			unpacked, err := unpackStringArray(v, "allowed_redirect_uris")
			if err != nil {
				return nil, err
			}
			authMethodConfig.AllowedRedirectURIs = unpacked
		case "discovery_ca_pem":
			unpacked, err := unpackStringArray(v, "discovery_ca_pem")
			if err != nil {
				return nil, err
			}
			authMethodConfig.DiscoveryCaPem = unpacked
		case "signing_algs":
			unpacked, err := unpackStringArray(v, "signing_algs")
			if err != nil {
				return nil, err
			}
			authMethodConfig.SigningAlgs = unpacked
		case "expiration_leeway":
			dur, err := parseDuration(v.(string), "expiration_leeway")
			if err != nil {
				return nil, err
			}
			authMethodConfig.ExpirationLeeway = dur
		case "not_before_leeway":
			dur, err := parseDuration(v.(string), "not_before_leeway")
			if err != nil {
				return nil, err
			}
			authMethodConfig.NotBeforeLeeway = dur
		case "clock_skew_leeway":
			dur, err := parseDuration(v.(string), "clock_skew_leeway")
			if err != nil {
				return nil, err
			}
			authMethodConfig.ClockSkewLeeway = dur
		case "claim_mappings":
			unpacked, err := unpackStringMap(v, "claim_mappings")
			if err != nil {
				return nil, err
			}
			authMethodConfig.ClaimMappings = unpacked
		case "list_claim_mappings":
			unpacked, err := unpackStringMap(v, "list_claim_mappings")
			if err != nil {
				return nil, err
			}
			authMethodConfig.ListClaimMappings = unpacked
		}
	}

	return &authMethodConfig, nil
}

func flattenACLAuthMethodConfig(cfg *api.ACLAuthMethodConfig) []any {
	if cfg == nil {
		return nil
	}
	result := map[string]interface{}{
		"jwt_validation_pub_keys": packStringArray(cfg.JWTValidationPubKeys),
		"jwks_url":                cfg.JWKSURL,
		"jwks_ca_cert":            cfg.JWKSCACert,
		"oidc_discovery_url":      cfg.OIDCDiscoveryURL,
		"oidc_client_id":          cfg.OIDCClientID,
		"oidc_client_secret":      cfg.OIDCClientSecret,
		"oidc_scopes":             packStringArray(cfg.OIDCScopes),
		"oidc_disable_userinfo":   cfg.OIDCDisableUserInfo,
		"bound_audiences":         packStringArray(cfg.BoundAudiences),
		"bound_issuer":            packStringArray(cfg.BoundIssuer),
		"allowed_redirect_uris":   packStringArray(cfg.AllowedRedirectURIs),
		"discovery_ca_pem":        packStringArray(cfg.DiscoveryCaPem),
		"signing_algs":            packStringArray(cfg.SigningAlgs),
		"expiration_leeway":       cfg.ExpirationLeeway.String(),
		"not_before_leeway":       cfg.NotBeforeLeeway.String(),
		"clock_skew_leeway":       cfg.ClockSkewLeeway.String(),
		"claim_mappings":          packStringMap(cfg.ClaimMappings),
		"list_claim_mappings":     packStringMap(cfg.ListClaimMappings),
	}
	return []any{result}
}

func unpackStringArray(v interface{}, name string) ([]string, error) {
	array, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to unpack %s configuration block", name)
	}

	var unpacked []string

	for _, entry := range array {
		unpacked = append(unpacked, entry.(string))
	}
	return unpacked, nil
}

func unpackStringMap(v interface{}, name string) (map[string]string, error) {
	existingMap, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to unpack %s configuration block", name)
	}

	unpacked := make(map[string]string)

	for k, v := range existingMap {
		unpacked[k] = v.(string)
	}
	return unpacked, nil
}

func packStringArray(array []string) []interface{} {
	var packed []interface{}
	for _, entry := range array {
		packed = append(packed, entry)
	}
	return packed
}

func packStringMap(stringMap map[string]string) map[string]interface{} {
	packed := make(map[string]interface{})
	for k, v := range stringMap {
		packed[k] = v
	}
	return packed
}

func parseDuration(durStr, name string) (time.Duration, error) {
	if durStr == "" {
		return 0, nil
	}
	dur, err := time.ParseDuration(durStr)
	if err != nil {
		return dur, fmt.Errorf("failed to parse %s duration: %v", name, err)
	}
	return dur, nil
}
