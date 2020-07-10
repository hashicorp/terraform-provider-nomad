---
layout: "nomad"
page_title: "Nomad: nomad_acl_tokens"
sidebar_current: "docs-nomad-datasource-acl-token"
description: |-
  Get a list of ACL tokens.
---

# nomad_acl_tokens

Get a list of ACL tokens.

## Example Usage

```hcl
data "nomad_acl_tokens" "tokens" {
  prefix = "a242"
}
```

## Argument Reference

The following arguments are supported:

* `prefix`: `(string)` Optional prefix to filter the tokens.

## Attributes Reference

The following attributes are exported:

* `acl_tokens`: `(list of objects)` The list of tokens found in the given prefix.

The objects in the `acl_tokens` list have the following attributes:

* `accessor_id`: `(TypeString)` Non-sensitive identifier for the token.
* `name`: `(TypeString)` The name of the token.
* `type`: `(TypeString)` The type of the token.
* `policies`: `(list of strings)` The list of policies attached to the token.
* `global`: `(bool)` Whether the token is replicated to all regions.
* `create_time`: `(string)` Date and time the token was created at.
