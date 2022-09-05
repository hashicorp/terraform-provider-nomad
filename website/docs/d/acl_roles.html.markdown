---
layout: "nomad"
page_title: "Nomad: nomad_acl_roles"
sidebar_current: "docs-nomad-datasource-acl_roles"
description: |-
Retrieve a list of ACL Roles.
---

# nomad_acl_roles

Retrieve a list of ACL Roles.

## Example Usage

```hcl
data "nomad_acl_roles" "example" {
  prefix = "a242"
}
```

## Argument Reference

The following arguments are supported:

* `prefix`: `(string)` An optional string to filter ACL Roles based on ID 
  prefix. If not provided, all policies are returned.

## Attribute Reference

The following attributes are exported:

* `roles`: `list of maps` a list of ACL Roles.
    * `id` `(string)` - The ACL Role unique identifier.
    * `name` `(string)` - Unique name of the ACL role.
    * `description` `(string)` - The description of the ACL Role.
    * `policies` `(set)` - The policies applied to the role.
