---
layout: "nomad"
page_title: "Nomad: nomad_scheduler_config"
sidebar_current: "docs-nomad-scheduler-config"
description: |-
  Manages scheduler configuration on the Nomad server.
---

# nomad_scheduler_config

Manages scheduler configuration of the Nomad cluster.

~> **Warning:** destroying this resource will not have any effect in the
cluster configuration, since there's no clear definition of what a destroy
action should do. The cluster will be left as-is and only the state reference
will be removed.

## Example Usage

Set cluster scheduler configuration:

```hcl
resource "nomad_scheduler_config" "config" {
  scheduler_algorithm = "spread"
  preemption_config = {
    system_scheduler_enabled  = true
    batch_scheduler_enabled   = true
    service_scheduler_enabled = true
  }
}
```

## Argument Reference

The following arguments are supported:

- `algorithm` `(string: "binpack")` - Specifies whether scheduler binpacks or spreads allocations on available nodes. Possible values are `binpack` and `spread`.
- `preemption_config` `(map[string]bool)` - Options to enable preemption for various schedulers.
  - `system_scheduler_enabled` `(bool: true)` - Specifies whether preemption for system jobs is enabled. Note that if this is set to true, then system jobs can preempt any other jobs.
  - `batch_scheduler_enabled` `(bool: false")` - Specifies whether preemption for batch jobs is enabled. Note that if this is set to true, then batch jobs can preempt any other jobs.
  - `service_scheduler_enabled` `(bool: false)` - Specifies whether preemption for service jobs is enabled. Note that if this is set to true, then service jobs can preempt any other jobs.
