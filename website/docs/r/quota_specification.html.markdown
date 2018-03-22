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
      cpu       = 2400
      memory_mb = 1200
    }
  }
}
```

## Argument Reference

The following arguments are supported:

- `name` `(string: <required>)` - A unique name for the quota specification.
- `description` `(string: "")` - A description of the quota specification.
- `limits` `(block: <required>)` - A block of quota limits to enforce. Can
  be repeated. See below for the structure of this block.


### `limits` blocks

The `limits` block describes the quota limits to be enforced. It supports
the following arguments:

- `region` `(string: <required>)` - The region these limits should apply to.
- `region_limit` `(block: <required>)` - The limits to enforce. This block
  may only be specified once in the `limits` block. Its structure is
  documented below.

### `region_limit` blocks

The `region_limit` block describes the quota limits to be enforced on a region.
It supports the following arguments:

- `cpu` `(int: 0)` - The amount of CPU to limit allocations to. A value of zero
  is treated as unlimited, and a negative value is treated as fully disallowed.
- `memory_mb` `(int: 0)` - The amount of memory (in megabytes) to limit
  allocations to. A value of zero is treated as unlimited, and a negative value
  is treated as fully disallowed.
