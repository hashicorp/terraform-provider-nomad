---
layout: "nomad"
page_title: "Nomad: nomad_node_pools"
sidebar_current: "docs-nomad-datasource-node-pools"
description: |-
  Retrieve a list of node pools available in Nomad.
---

# nomad_node_pools

Retrieve a list of node pools available in Nomad.

## Example Usage

```hcl
data "nomad_node_pools" "prod" {
  filter = "Meta.env == \"prod\""
}
```

## Argument Reference

The following arguments are supported:

- `prefix` `(string)` - Specifies a string to filter node pools based on a name
  prefix.
- `filter` `(string)` - Specifies the [expression][nomad_api_filter] used to
  filter the results.

## Attribute Reference

The following attributes are exported:

- `node_pools` `(list of node pools)` - A list of node pools matching the
  search criteria.
  - `name` `(string)` - The name of the node pool.
  - `description` `(string)` - The description of the node pool.
  - `meta` `(map[string]string)` - Arbitrary KV metadata associated with the
    node pool.
  - `node_identity_ttl` `(string)` - The TTL applied to node identities issued to
    nodes in this pool.
  - `scheduler_config` `(block)` - Scheduler configuration for the node pool.
    - `scheduler_algorithm` `(string)` - The scheduler algorithm used in the node
      pool. If empty or not defined the global cluster configuration is used.
    - `memory_oversubscription` `(string)` - Whether or not memory
      oversubscription is enabled in the node pool. If empty or not defined the
      global cluster configuration is used.

      -> This option differs from Nomad, where it's represented as a boolean, to
      allow distinguishing between memory oversubscription being disabled in the
      node pool and this property not being set.

[nomad_api_filter]: https://developer.hashicorp.com/nomad/api-docs/v1.6.x#filtering
