---
layout: "nomad"
page_title: "Nomad: nomad_acl_auth_method"
sidebar_current: "docs-nomad-resource-acl-auth-method"
description: |-
Manages an ACL Auth Method in Nomad.
---

# nomad_acl_auth_method

Manages an ACL Auth Method in Nomad.

## Example Usage

Creating an ALC Auth Method:

```hcl
resource "nomad_acl_auth_method" "my_nomad_acl_auth_method" {
  name              = "my-nomad-acl-auth-method"
  type              = "OIDC"
  token_locality    = "global"
  max_token_ttl     = "10m0s"
  token_name_format = "$${auth_method_type}-$${value.user}"
  default           = true

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
```

## Argument Reference

The following arguments are supported:

- `name` `(string: <required>)` - The identifier of the ACL Auth Method.

- `type` `(string: <required>)` - ACL Auth Method SSO workflow type. Valid values,
  are `OIDC` and `JWT`.

- `token_locality` `(string: <required>)` - Defines whether the ACL Auth Method
  creates a local or global token when performing SSO login. This field must be
  set to either `local` or `global`.

- `max_token_ttl` `(string: <required>)` - Defines the maximum life of a token
  created by this method and is specified as a time duration such as "15h".

- `token_name_format` `(string: "${auth_method_type}-${auth_method_name}")` -
  Defines the token name format for the generated tokens This can be lightly
  templated using HIL '${foo}' syntax.

- `default` `(bool: false)` - Defines whether this ACL Auth Method is to be set
  as default.

- `config`: `(block: <required>)` - Configuration specific to the auth method
  provider.

  - `jwt_validation_pub_keys`: `([]string: <optional>)` - List of PEM-encoded 
    public keys to use to authenticate signatures locally.

  - `jwks_url`: `(string: <optional>)` - JSON Web Key Sets url for authenticating
    signatures.
			
  - `jwks_ca_cert`: `(string: <optional>)` - PEM encoded CA cert for use by the 
    TLS client used to talk with the JWKS server.

  - `oidc_discovery_url`: `(string: <optional>)` - The OIDC Discovery URL,
    without any .well-known component (base path).

  - `oidc_client_id`: `(string: <optional>)` - The OAuth Client ID configured
    with the OIDC provider.

  - `oidc_client_secret`: `(string: <optional>)` - The OAuth Client Secret
    configured with the OIDC provider.

  - `oidc_client_assertion` `(OIDCClientAssertion: <optional>)` - Optionally
    send a signed JWT ("[private key jwt][]") as a client assertion to the OIDC
    provider. Browse to the [OIDC concepts][concepts-assertions] page to learn
    more.

    - `audience` `([]string: optional)` - Who processes the assertion.
      Defaults to the auth method's `oidc_discovery_url`.

    - `key_source` `(string: <required>)` - Specifies where to get the private
      key to sign the JWT.
      Available sources:
      - "nomad": Use current active key in Nomad's keyring
      - "private_key": Use key material in the `private_key` field
      - "client_secret": Use the `oidc_client_secret` as an HMAC key

    - `key_algorithm` `(string: <optional>)` is the key's algorithm.
      Its default values are based on the `key_source`:
      - "nomad": "RS256"; this is from Nomad's keyring and must not be changed
      - "private_key": "RS256"; must be RS256, RS384, or RS512
      - "client_secret": "HS256"; must be HS256, HS384, or HS512

    - `private_key` `(OIDCClientAssertionKey: <optional>)` - External key
      to sign the JWT. `key_source` must be "private_key" to enable this.

      - `pem_key` `(string: <optional>)` - An RSA private key, in pem format.
        It is used to sign the JWT. Mutually exclusive with `pem_key`.

      - `pem_key_file` `(string: optional)` - An absolute path to a private key
        on Nomad servers' disk, in pem format. It is used to sign the JWT.
        Mutually exclusive with `pem_key_file`.

      - `key_id_header` `(string: optional)` - Which header the provider uses
        to find the public key to verify the signed JWT.
        The default and allowed values depend on whether you set `key_id`,
        `pem_cert`, or `pem_cert_file`. You must set exactly one of those
        options, so refer to them for their requirements.

      - `key_id` `(string: optional)` - Becomes the JWT's "kid" header.
        Mutually exclusive with `pem_cert` and `pem_cert_file`.
        Allowed `key_id_header` values: "kid" (the default)

      - `pem_cert` `(string: optional)` - An x509 certificate, signed by the
        private key or a CA, in pem format. Nomad uses this certificate to
        derive an [x5t#S256][] (or [x5t][]) key_id.
        Mutually exclusive with `pem_cert_file` and `key_id`.
        Allowed `key_id_header` values: "x5t", "x5t#S256" (default "x5t#S256")

      - `pem_cert_file` `(string: optional)` - An absolute path to an x509
        certificate on Nomad servers' disk, signed by the private key or a CA,
        in pem format.
        Nomad uses this certificate to derive an [x5t#S256][] (or [x5t][])
        header. Mutually exclusive with `pem_cert` and key_id.
        Allowed `key_id_header` values: "x5t", "x5t#S256" (default "x5t#S256")

    - `extra_headers` `(map[string]string: optional)` - Add to the JWT headers,
      alongside "kid" and "type". Setting the "kid" header here is not allowed;
      use `private_key.key_id`.

  - `oidc_enable_pkce` `(bool: false)` - When set to `true`, Nomad will include
    [PKCE][] verification in the auth flow. Even with PKCE enabled in Nomad,
    you may still need to enable it in your OIDC provider.
  - `oidc_scopes`: `([]string: <optional>)` - List of OIDC scopes.

  - `oidc_disable_userinfo`: `(bool: false)` - When set to `true`, Nomad will
     not make a request to the identity provider to get OIDC `UserInfo`.
     You may wish to set this if your identity provider doesn't send any
     additional claims from the `UserInfo` endpoint.

  - `bound_audiences`: `([]string: <optional>)` - List of auth claims that are
    valid for login.

  - `bound_issuer`: `([]string: <optional>)` - The value against which to match
    the iss claim in a JWT.

  - `allowed_redirect_uris`: `([]string: <optional>)` - A list of allowed values
    that can be used for the redirect URI.

  - `discovery_ca_pem`: `([]string: <optional>)` - PEM encoded CA certs for use
    by the TLS client used to talk with the OIDC Discovery URL.

  - `signing_algs`: `([]string: <optional>)` - A list of supported signing
    algorithms.

  - `expiration_leeway`: `(string: <optional>)` - Duration of leeway when validating
    expiration of a JWT in the form of a time duration such as "5m" or "1h".

	- `not_before_leeway`: `(string: <optional>)` - Duration of leeway when validating
    not before values of a token in the form of a time duration such as "5m" or "1h".
    
	- `clock_skew_leeway`: `(string: <optional>)` - Duration of leeway when validating
    all claims in the form of a time duration such as "5m" or "1h".

  - `claim_mappings`: `(map[string]string: <optional>)` - Mappings of claims (key)
    that will be copied to a metadata field (value).

  - `list_claim_mappings`: `(map[string]string: <optional>)` - Mappings of list
    claims (key) that will be copied to a metadata field (value).

  - `verbose_logging` `(bool: false)` - When set to `true`, Nomad will log token
    claims, information related to binding-rule and role/policy evaluations,
    and client assertion JWTs, if applicable. Not recommended in production,
    since sensitive information may be present.

[private key jwt]: https://oauth.net/private-key-jwt/
[concepts-assertions]: /nomad/docs/concepts/acl/auth-methods/oidc#client-assertions
[x5t]: https://datatracker.ietf.org/doc/html/rfc7515#section-4.1.7
[x5t#S256]: https://datatracker.ietf.org/doc/html/rfc7515#section-4.1.8
[pkce]: https://oauth.net/2/pkce/
