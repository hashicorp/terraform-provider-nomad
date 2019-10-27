---
layout: "nomad"
page_title: "Nomad: nomad_acl_token"
sidebar_current: "docs-nomad-resource-acl-token"
description: |-
  Manages an ACL token in Nomad.
---

# nomad_acl_token

Manages an ACL token in Nomad.

~> **Warning:** this resource will store any tokens it creates in
  Terraform's state file. Take care to
  [protect your state file](/docs/state/sensitive-data.html).

## Example Usage

Creating a token with limited policies:

```hcl
resource "nomad_acl_token" "ron" {
  name     = "Ron Weasley"
  type     = "client"
  policies = ["dev", "qa"]
}
```

Creating a global token that will be replicated to all regions:

```hcl
resource "nomad_acl_token" "hermione" {
  name     = "Hermione Granger"
  type     = "client"
  policies = ["dev", "qa"]
  global   = true
}
```

Creating a token with full access to the cluster:

```hcl
resource "nomad_acl_token" "hagrid" {
  name = "Rubeus Hagrid"

  # Hagrid is the keeper of the keys
  type = "management"
}
```

Accessing the token:

```hcl
resource "nomad_acl_token" "token" {
  type     = "client"
  policies = ["dev"]
}

output "nomad_token" {
  value = "${nomad_acl_token.token.secret_id}"
}
```

## Argument Reference

The following arguments are supported:

- `type` `(string: <required>)` - The type of token this is. Use `client`
  for tokens that will have policies associated with them. Use `management`
  for tokens that can perform any action.

- `name` `(string: "")` - A human-friendly name for this token.

- `policies` `(set: [])` - A set of policy names to associate with this
  token. Must be set on `client`-type tokens, must not be set on
  `management`-type tokens. Policies do not need to exist before being
  used here.

- `global` `(bool: false)` - Whether the token should be replicated to all
  regions, or if it will only be used in the region it was created in.

In addition to the above arguments, the following attributes are exported and
can be referenced:

- `accessor_id` `(string)` - A non-sensitive identifier for this token that
  can be logged and shared safely without granting any access to the cluster.

- `secret_id` `(string)` - The token value itself, which is presented for
  access to the cluster.

- `create_time` `(string)` - The timestamp the token was created.
