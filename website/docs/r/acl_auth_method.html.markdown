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

- `type` `(string: <required>)` - ACL Auth Method SSO workflow type. Currently,
  the only supported type is `OIDC`.

- `token_locality` `(string: <required>)` - Defines whether the ACL Auth Method
  creates a local or global token when performing SSO login. This field must be
  set to either `local` or `global`.

- `max_token_ttl` `(string: <required>)` - Defines the maximum life of a token
  created by this method and is specified as a time duration such as "15h".

- `token_name_format` `(string: <optional>)` - Defines the token name format for the
  generated tokens This can be lightly templated using HIL '${foo}' syntax.
  Defaults to `${auth_method_type}-${auth_method_name}`.

- `default` `(bool: false)` - Defines whether this ACL Auth Method is to be set
  as default.

- `config`: `(block: <required>)` - Configuration specific to the auth method
  provider.

  - `oidc_discovery_url`: `(string: <required>)` - The OIDC Discovery URL,
    without any .well-known component (base path).

  - `oidc_client_id`: `(string: <required>)` - The OAuth Client ID configured
    with the OIDC provider.

  - `oidc_client_secret`: `(string: <required>)` - The OAuth Client Secret
    configured with the OIDC provider.

  - `oidc_scopes`: `([]string: <optional>)` - List of OIDC scopes.

  - `oidc_disable_userinfo`: `(bool: false)` - When set to `true`, Nomad will
     not make a request to the identity provider to get OIDC `UserInfo`.
     You may wish to set this if your identity provider doesn't send any
     additional claims from the `UserInfo` endpoint.

  - `bound_audiences`: `([]string: <optional>)` - List of auth claims that are
    valid for login.

  - `allowed_redirect_uris`: `([]string: <optional>)` - A list of allowed values
    that can be used for the redirect URI.

  - `discovery_ca_pem`: `([]string: <optional>)` - PEM encoded CA certs for use
    by the TLS client used to talk with the OIDC Discovery URL.

  - `signing_algs`: `([]string: <optional>)` - A list of supported signing
    algorithms.

  - `claim_mappings`: `(map[string]string: <optional>)` - Mappings of claims (key)
    that will be copied to a metadata field (value).

  - `list_claim_mappings`: `(map[string]string: <optional>)` - Mappings of list
    claims (key) that will be copied to a metadata field (value).
