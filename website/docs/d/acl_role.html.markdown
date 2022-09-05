---
layout: "nomad"
page_title: "Nomad: nomad_acl_role"
sidebar_current: "docs-nomad-datasource-acl-role"
description: |-
Get information on an ACL Role.
---

# nomad_acl_role

Get information on an ACL Role.

## Example Usage

```hcl
data "nomad_acl_role" "example" {
  id = "aa534e09-6a07-0a45-2295-a7f77063d429"
}
```

## Argument Reference

The following arguments are supported:

* `id`: `(string)` The unique identifier of the ACL Role.

## Attributes Reference

The following attributes are exported:

* `id` `(string)` - The ACL Role unique identifier.
* `name` `(string)` - Unique name of the ACL role.
* `description` `(string)` - The description of the ACL Role.
* `policies` `(set)` - The policies applied to the role.
