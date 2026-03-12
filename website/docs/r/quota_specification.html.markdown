---
layout: "nomad"
page_title: "Nomad: nomad_quota_specification"
sidebar_current: "docs-nomad-resource-quota-specification"
description: |-
  Manages a quota specification in a Nomad cluster.
---

# nomad_quota_specification

Manages a quota specification in a Nomad cluster.

## Example Usage

Registering a quota specification:

```hcl
resource "nomad_quota_specification" "prod_api" {
  name        = "prod-api"
  description = "Production instances of backend API servers"

  limits {
    region = "global"

    region_limit {
      cpu          = 2400
      cores        = 4
      memory_mb    = 1200
      memory_max_mb = 2400
      secrets_mb   = 100

      devices {
        name  = "nvidia/gpu"
        count = 2
      }

      node_pools {
        node_pool     = "batch"
        cpu           = 800
        cores         = 2
        memory_mb     = 1024
        memory_max_mb = 2048
        secrets_mb    = 64

        devices {
          name  = "fpga"
          count = 1
        }

        storage {
          variables_mb    = 25
          host_volumes_mb = 50
        }
      }

      storage {
        variables_mb    = 500
        host_volumes_mb = 1000
      }
    }
  }
}
```

## Argument Reference

The following arguments are supported:

- `name` `(string: <required>)` - A unique name for the quota specification.
- `description` `(string: "")` - A description of the quota specification.
- [`limits`](#limits-blocks) `(block: <required>)` - A block of quota limits to enforce. Can
  be repeated. See below for the structure of this block.

### `limits` blocks

The `limits` block describes the quota limits to be enforced. It supports
the following arguments:

- `region` `(string: <required>)` - The region these limits should apply to.
- [`region_limit`](#region_limit-blocks) `(block: <required>)` - The limits to enforce. This block
  may only be specified once in the `limits` block. Its structure is
  documented below.

### `region_limit` blocks

The `region_limit` block describes the quota limits to be enforced on a region.
It supports the following arguments:

- `cpu` `(int: 0)` - The amount of CPU to limit allocations to. A value of zero
  is treated as unlimited, and a negative value is treated as fully disallowed.
- `cores` `(int: 0)` - The number of CPU cores to limit allocations to. A value
  of zero is treated as unlimited, and a negative value is treated as fully
  disallowed.
- `memory_mb` `(int: 0)` - The amount of memory (in megabytes) to limit
  allocations to. A value of zero is treated as unlimited, and a negative value
  is treated as fully disallowed.
- `memory_max_mb` `(int: 0)` - The maximum amount of memory (in megabytes) to
  limit allocations to. A value of zero is treated as unlimited, and a negative
  value is treated as fully disallowed.
- `secrets_mb` `(int: 0)` - The amount of secrets storage (in megabytes) to
  limit allocations to. A value of zero is treated as unlimited, and a negative
  value is treated as fully disallowed.
- [`devices`](#devices-blocks) `(block: optional)` - A list of device quotas to enforce. Can be
  repeated. See below for the structure of this block.
- [`node_pools`](#node_pools-blocks) `(block: optional)` - Per-node-pool quota limits. Can be
  repeated. See below for the structure of this block.
- [`storage`](#storage-blocks) `(block: optional)` - Storage resource quota configuration. May only
  be specified once. See below for the structure of this block.

### `devices` blocks

The `devices` block describes a device quota to enforce. It supports the
following arguments:

- `name` `(string: <required>)` - The name of the device, e.g.
  `"nvidia/gpu"`.
- `count` `(int: 0)` - The number of device instances to limit allocations to.

### `storage` blocks

The `storage` block describes storage resource quota settings. It supports the
following arguments:

- `variables_mb` `(int: 0)` - The amount of storage (in megabytes) to limit
  Nomad variables to.
- `host_volumes_mb` `(int: 0)` - The amount of storage (in megabytes) to limit
  host volumes to.

### `node_pools` blocks

The `node_pools` block describes per-node-pool quota limits. It supports the
following arguments:

- `node_pool` `(string: <required>)` - The node pool name to apply limits to.
- `cpu` `(int: 0)` - The amount of CPU to limit allocations to. A value of zero
  is treated as unlimited, and a negative value is treated as fully disallowed.
- `cores` `(int: 0)` - The number of CPU cores to limit allocations to. A value
  of zero is treated as unlimited, and a negative value is treated as fully
  disallowed.
- `memory_mb` `(int: 0)` - The amount of memory (in megabytes) to limit
  allocations to. A value of zero is treated as unlimited, and a negative value
  is treated as fully disallowed.
- `memory_max_mb` `(int: 0)` - The maximum amount of memory (in megabytes) to
  limit allocations to. A value of zero is treated as unlimited, and a negative
  value is treated as fully disallowed.
- `secrets_mb` `(int: 0)` - The amount of secrets storage (in megabytes) to
  limit allocations to. A value of zero is treated as unlimited, and a negative
  value is treated as fully disallowed.
- [`devices`](#devices-blocks) `(block: optional)` - A list of device quotas to
  enforce for the node pool. Can be repeated.
- [`storage`](#storage-blocks) `(block: optional)` - Storage resource quota
  configuration for the node pool. May only be specified once.
