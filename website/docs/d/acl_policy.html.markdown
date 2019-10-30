---
layout: "nomad"
page_title: "Nomad: nomad_acl_policy"
sidebar_current: "docs-nomad-datasource-acl_policy"
description: |-
  Retrieve information on an ACL Policy.
---

# nomad_acl_policy

Retrieve information on an ACL Policy.

## Example Usage

```hcl
data "nomad_acl_policy" "my_policy" {
  name = "my-policy"
}
```

## Attribute Reference

The following attributes are exported:

- `name` `(string)` - the name of the ACL Policy.
- `description` `(string)` - the description of the ACL Policy.
- `rules` `(string)` - the ACL Policy rules in HCL format.
