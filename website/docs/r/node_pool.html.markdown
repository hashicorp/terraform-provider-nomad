---
layout: "nomad"
page_title: "Nomad: nomad_node_pool"
sidebar_current: "docs-nomad-resource-node-pool"
description: |-
  Provisions a node pool within a Nomad cluster.
---

# nomad_node_pool

Provisions a node pool within a Nomad cluster.

## Example Usage

Registering a node pool:

```hcl
resource "nomad_node_pool" "dev" {
  name        = "dev"
  description = "Nodes for the development environment."

  meta = {
    department = "Engineering"
    env        = "dev"
  }
}
```

## Argument Reference

The following arguments are supported:

- `name` `(string)` - The name of the node pool.
- `description` `(string)` - The description of the node pool.
- `meta` `(map[string]string)` - Arbitrary KV metadata associated with the
  node pool.
- `scheduler_config` `(block)` - Scheduler configuration for the node pool.
  - `scheduler_algorithm` `(string)` - The scheduler algorithm used in the node
    pool. Possible values are `binpack` or `spread`. If not defined the global
    cluster configuration is used.
  - `memory_oversubscription` `(string)` - Whether or not memory
    oversubscription is enabled in the node pool. Possible values are
    `"enabled"` or `"disabled"`. If not defined the global cluster
    configuration is used.

    -> This option differs from Nomad, where it's represented as a boolean, to
    allow distinguishing between memory oversubscription being disabled in the
    node pool and this property not being set.
