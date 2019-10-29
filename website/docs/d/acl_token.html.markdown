---
layout: "nomad"
page_title: "Nomad: nomad_acl_token"
sidebar_current: "docs-nomad-datasource-acl-token"
description: |-
  Get information on an ACL token.
---

# nomad_acl_token

Get information on an ACL token.

~> **Warning:** this data source will store tokens in the Terraform state. Take care to
  [protect your state file](/docs/state/sensitive-data.html).

## Example Usage

```hcl
data "nomad_acl_token" "my_token" {
  accessor_id = "aa534e09-6a07-0a45-2295-a7f77063d429"
}
```

## Argument Reference

The following arguments are supported:

* `accessor_id`: `(string)` Non-sensitive identifier for this token.

## Attributes Reference

The following attributes are exported:

* `name`: `(string)` Non-sensitive identifier for this token.
* `accessor_id`: `(string)` Non-sensitive identifier for this token.
* `secret_id`: `(string)` The token value itself.
* `type`: `(string)` The type of the token.
* `policies`: `(list of strings)` List of policy names associated with this token.
* `global`: `(bool)` Whether the token is replicated to all regions, or if it will only be used in the region it was created.
* `create_time`: `(string)` Date and time the token was created.
