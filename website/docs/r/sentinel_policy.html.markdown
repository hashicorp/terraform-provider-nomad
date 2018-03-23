---
layout: "nomad"
page_title: "Nomad: nomad_sentinel_policy"
sidebar_current: "docs-nomad-resource-sentinel-policy"
description: |-
  Manages a Sentinel policy registered on the Nomad server.
---

# nomad_sentinel_policy

Manages a Sentinel policy registered in Nomad.

~> **Enterprise Only!** This API endpoint and functionality only exists in
   Nomad Enterprise. This is not present in the open source version of Nomad.

## Example Usage

```hcl
resource "nomad_sentinel_policy" "exec-only" {
  name = "exec-only"
  description = "Only allow jobs that are based on an exec driver."
  policy = <<EOT
main = rule { all_drivers_exec }

# all_drivers_exec checks that all the drivers in use are exec
all_drivers_exec = rule {
    all job.task_groups as tg {
        all tg.tasks as task {
            task.driver is "exec"
        }
    }
}
EOT
  scope = "submit-job"
  # allow administrators to override
  enforcement_level = "soft-mandatory"
}
```

## Argument Reference

The following arguments are supported:

- `name` `(string: <required>)` - A unique name for the policy.
- `policy` `(string: <required>)` - The contents of the policy to register.
- `enforcement_level` `(strings: <required>)` - The [enforcement level][enforcement-level]
  for this policy.
- `scope` `(strings: <required>)` - The [scope][scope] for this policy.
- `description` `(string: "")` - A description of the policy.

[scope]: https://www.nomadproject.io/guides/sentinel-policy.html#policy-scope
[enforcement-level]: https://www.nomadproject.io/guides/sentinel-policy.html#enforcement-level
