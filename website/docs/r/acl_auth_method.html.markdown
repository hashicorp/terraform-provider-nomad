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

- `default` `(bool: false)` - Defines whether this ACL Auth Method is to be set
  as default.