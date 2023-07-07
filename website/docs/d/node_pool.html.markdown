---
layout: "nomad"
page_title: "Nomad: nomad_node_pool"
sidebar_current: "docs-nomad-datasource-node-pool"
description: |-
  Get information about a node pool in Nomad.
---

# nomad_node_pool

Get information about a node pool in Nomad.

## Example Usage

```hcl
data "nomad_node_pool" "dev" {
  name = "dev"
}
```

## Argument Reference

- `name` `(string)` - The name of the node pool to fetch.

## Attribute Reference

The following attributes are exported:

- `description` `(string)` - The description of the node pool.
- `meta` `(map[string]string)` - Arbitrary KV metadata associated with the
  node pool.
- `scheduler_config` `(block)` - Scheduler configuration for the node pool.
  - `scheduler_algorithm` `(string)` - The scheduler algorithm used in the node
    pool. If empty or not defined the global cluster configuration is used.
  - `memory_oversubscription` `(string)` - Whether or not memory
    oversubscription is enabled in the node pool. If empty or not defined the
    global cluster configuration is used.

    -> This option differs from Nomad, where it's represented as a boolean, to
    allow distinguishing between memory oversubscription being disabled in the
    node pool and this property not being set.
