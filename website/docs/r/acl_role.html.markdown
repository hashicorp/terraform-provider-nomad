---
layout: "nomad"
page_title: "Nomad: nomad_acl_role"
sidebar_current: "docs-nomad-resource-acl-role"
description: |-
Manages an ACL Role in Nomad.
---

# nomad_acl_role

Manages an ACL Role in Nomad.

## Example Usage

Creating an ALC Role linked to an ACL policy also created by Terraform:

```hcl
resource "nomad_acl_policy" "my_nomad_acl_policy" {
  name        = "my-nomad-acl-policy"
  rules_hcl   = <<EOT
namespace "default" {
  policy       = "read"
  capabilities = ["submit-job"]
}
EOT
}

resource "nomad_acl_role" "my_nomad_acl_role" {
  name        = "my-nomad-acl-role"
  description = "An ACL Role for cluster developers"
  
  policies {
    name = nomad_acl_policy.my_nomad_acl_policy.name
  }
}
```

## Argument Reference

The following arguments are supported:

- `name` `(string: <required>)` - A human-friendly name for this ACL Role.

- `description` `(string: "")` - A description of the ACL Role.

- `policies` `(set: <required>)` - A set of policy names to associate with this
  ACL Role.
