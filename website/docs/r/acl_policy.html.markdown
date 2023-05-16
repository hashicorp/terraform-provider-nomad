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

Registering a policy from a HCL file:

```hcl
resource "nomad_acl_policy" "dev" {
  name        = "dev"
  description = "Submit jobs to the dev environment."
  rules_hcl   = file("${path.module}/dev.hcl")
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
- `job_acl`: `(`[`JobACL`](#jobacl-1)`: <optional>)` - Options for assigning the ACL rules to a job, group, or task.

### JobACL

The `job_acl` block is used to associate the ACL policy with a given namespace,
job, group, or task. Refer to [Workload Associated ACL Policies][nomad_docs_wi]
for more information. The following arguments are supported.

- `namespace` `(string: "default")` - The namespace to attach the policy.
  Required if `job_id` is set.
- `job_id` `(string: <optional>` - The job to attach the policy. Required if
  `group` is set.
- `group` `(string: <optional>` - The group to attach the policy. Required if
  `task` is set.
- `task` `(string: <optional>` - The task to attach the policy.

[nomad_docs_wi]: https://www.nomadproject.io/docs/concepts/workload-identity#workload-associated-acl-policies
