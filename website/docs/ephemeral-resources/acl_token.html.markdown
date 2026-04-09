---
layout: "nomad"
page_title: "Nomad: nomad_acl_token"
sidebar_current: "docs-nomad-ephemeral-acl-token"
description: |-
  Reads an existing Nomad ACL token for token usage during Terraform runs without storing the token secret in state.
---

# nomad_acl_token

Reads an existing Nomad ACL token for token usage during Terraform runs
without storing the token secret in state.

Use the `nomad_acl_token` resource to create and manage ACL token metadata.
For token usage during Terraform runs, use this ephemeral resource so the
token secret is not persisted in Terraform state.

## Example Usage

```hcl
resource "nomad_acl_token" "build" {
  type     = "client"
  policies = ["dev"]
}

ephemeral "nomad_acl_token" "build" {
  accessor_id = nomad_acl_token.build.accessor_id
}
```

## Argument Reference

The following arguments are supported:

- `accessor_id` `(string: <required>)` - Non-sensitive identifier for this token.

## Attributes Reference

The following attributes are exported:

- `accessor_id` `(string)` - A non-sensitive identifier for this token that
  can be logged and shared safely without granting any access to the cluster.

- `secret_id` `(string)` - The token value itself, which is presented for
  access to the cluster.

- `type` `(string)` - The type of the token.

- `name` `(string)` - Human-friendly name of the ACL token.

- `policies` `(list of strings)` - List of policy names associated with this token.

- `roles` `(set: [])` - The roles that are applied to the token. Each entry has
  `name` and `id` attributes.

- `global` `(bool)` - Whether the token is replicated to all regions, or if it
  will only be used in the region it was created.

- `create_time` `(string)` - The timestamp the token was created.

- `expiration_ttl` `(string)` - The expiration TTL for the token.

- `expiration_time` `(string)` - The timestamp after which the token is
  considered expired and eligible for destruction.