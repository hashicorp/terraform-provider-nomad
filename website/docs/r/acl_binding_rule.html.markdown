---
layout: "nomad"
page_title: "Nomad: nomad_acl_binding_rule"
sidebar_current: "docs-nomad-resource-acl-binding-rule"
description: |-
Manages an ACL Binding Rule in Nomad.
---

# nomad_acl_binding_rule

Manages an ACL Binding Rule in Nomad.

~> **Warning:** this resource will store the sensitive value placed in
  `config.oidc_client_secret` in the Terraform's state file. Take care to
  [protect your state file](/docs/state/sensitive-data.html).

## Example Usage

Creating an ALC Binding Rule associated to an ACL Auth Method also created and
managed by Terraform:

```hcl
resource "nomad_acl_auth_method" "my_nomad_acl_auth_method" {
  name           = "my-nomad-acl-auth-method"
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

resource "nomad_acl_binding_rule" "my_nomad_acl_binding_rule" {
  description = "engineering rule"
  auth_method = nomad_acl_auth_method.my_nomad_acl_auth_method.name
  selector    = "engineering in list.roles"
  bind_type   = "role"
  bind_name   = "engineering-read-only"
}
```

## Argument Reference

The following arguments are supported:

- `description` `(string: "")` - Description for this ACL binding rule.

- `auth_method` `(string: <required>)` - Name of the auth method for which this
  rule applies to.

- `selector` `(string: "")` - A boolean expression that matches against verified
  identity attributes returned from the auth method during login.

- `bind_type` `(string: <required>)` - Adjusts how this binding rule is applied
  at login time. Valid values are `role`, `policy`, and `management`.

- `bind_name` `(string: "")` - Target of the binding.

- `config`: `(block: <required>)` - Configuration specific to the auth method
  provider.

  - `oidc_discovery_url`: `(string: <required>)` - The OIDC Discovery URL,
    without any .well-known component (base path).

  - `oidc_client_id`: `(string: <required>)` - The OAuth Client ID configured
    with the OIDC provider.

  - `oidc_client_secret`: `(string: <required>)` - The OAuth Client Secret
    configured with the OIDC provider.

  - `oidc_scopes`: `([]string: <optional>)` - List of OIDC scopes.

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
