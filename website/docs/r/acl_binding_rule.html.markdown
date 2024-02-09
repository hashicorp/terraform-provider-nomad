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

- `bind_name` `(string: <optional>)` - Target of the binding. If `bind_type` is
  `role` or `policy` then `bind_name` is required. If `bind_type` is
  `management` than `bind_name` must not be defined.
