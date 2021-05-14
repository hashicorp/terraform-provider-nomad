---
layout: "nomad"
page_title: "Nomad: nomad_scheduler_config"
sidebar_current: "docs-nomad-datasource-scheduler-config"
description: |-
  Retrieve the cluster's scheduler configuration.
---

# nomad_scheduler_config

Retrieve the cluster's [scheduler configuration](https://www.nomadproject.io/api-docs/operator#sample-response-3).

## Example Usage

```hcl
data "nomad_scheduler_config" "global" {}
```

## Attribute Reference

The following attributes are exported:

* `memory_oversubscription_enabled` `(bool: false)` - When `true`, tasks may exceed their reserved memory limit.
* `scheduler_algorithm` `(string)` - Specifies whether scheduler binpacks or spreads allocations on available nodes.
* `preemption_config` `(map[string]bool)` - Options to enable preemption for various schedulers.
