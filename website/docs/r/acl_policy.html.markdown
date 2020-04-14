---
layout: "nomad"
page_title: "Nomad: nomad_acl_policy"
sidebar_current: "docs-nomad-resource-acl-policy"
description: |-
  Manages an ACL policy registered on the Nomad server.
---

# nomad_acl_policy

Manages an ACL policy registered in Nomad.

## Example Usage

Registering a policy from a HCL file with Terraform 0.12 or greater:

```hcl
resource "nomad_acl_policy" "dev" {
  name        = "dev"
  description = "Submit jobs to the dev environment."
  rules_hcl   = file("${path.module}/dev.hcl"
}
```

Registering a policy from a HCL file with Terraform 0.11 or prior:

```hcl
resource "nomad_acl_policy" "dev" {
  name        = "dev"
  description = "Submit jobs to the dev environment."
  rules_hcl   = "${file("${path.module}/dev.hcl")}"
}
```

Registering a policy from inline HCL:

```hcl
resource "nomad_acl_policy" "dev" {
  name        = "dev"
  description = "Submit jobs to the dev environment."

  rules_hcl = <<EOT
namespace "dev" {
  policy = "write"
}
EOT
}
```

## Argument Reference

The following arguments are supported:

- `name` `(string: <required>)` - A unique name for the policy.
- `rules_hcl` `(string: <required>)` - The contents of the policy to register,
   as HCL or JSON.
- `description` `(string: "")` - A description of the policy.
