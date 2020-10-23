---
layout: "nomad"
page_title: "Nomad: nomad_scheduler_config"
sidebar_current: "docs-nomad-scheduler-config"
description: |-
  Manages scheduler configuration on the Nomad server.
---

# nomad_scheduler_config

Manages scheduler configuration on Nomad.

## Example Usage

Modify scheduler config:

```hcl
resource "nomad_scheduler_config" "global" {
  algorithm = "spread"
  preemption {
      system_enabled = true
      batch_enabled = false
      service_enabled = true
  }
}
```

## Argument Reference

The following arguments are supported:

- `algorithm` `(string: "binpack")` - Specifies whether scheduler binpacks or spreads allocations on available nodes.
- `system_enabled` `(bool: true)` - Specifies whether preemption for system jobs is enabled. Note that this defaults to true.
- `batch_enabled` `(bool: false")` - Specifies whether preemption for batch jobs is enabled. Note that this defaults to false and must be explicitly enabled.
- `service_enabled` `(book: true)` - Specifies whether preemption for service jobs is enabled. Note that this defaults to false and must be explicitly enabled.
