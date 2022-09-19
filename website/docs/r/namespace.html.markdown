---
layout: "nomad"
page_title: "Nomad: nomad_namespace"
sidebar_current: "docs-nomad-resource-namespace"
description: |-
  Provisions a namespace within a Nomad cluster.
---

# nomad_namespace

Provisions a namespace within a Nomad cluster.

Nomad auto-generates a default namespace called `default`. This namespace
cannot be removed, so destroying a `nomad_namespace` resource where
`name = "default"` will cause the namespace to be reset to its default
configuration.

## Example Usage

Registering a namespace:

```hcl
resource "nomad_namespace" "dev" {
  name        = "dev"
  description = "Shared development environment."
  quota       = "dev"
  meta        = {
    owner = "John Doe"
    foo   = "bar"
  }
}
```

Registering a namespace with a quota:

```hcl
resource "nomad_quota_specification" "web_team" {
  name        = "web-team"
  description = "web team quota"

  limits {
    region = "global"

    region_limit {
      cpu       = 1000
      memory_mb = 256
    }
  }
}

resource "nomad_namespace" "web" {
  name        = "web"
  description = "Web team production environment."
  quota       = nomad_quota_specification.web_team.name
}
```

## Argument Reference

The following arguments are supported:

- `name` `(string: <required>)` - A unique name for the namespace.
- `description` `(string: "")` - A description of the namespace.
- `quota` `(string: "")` - A resource quota to attach to the namespace.
- `meta` `(map[string]string: <optional>)` -  Specifies arbitrary KV metadata to associate with the namespace.
- `capabilities` `(block: <optional>)` - A block of capabilities for the namespace. Can't 
  be repeated. See below for the structure of this block.


### `capabilities` blocks

The `capabilities` block describes the capabilities of the namespace. It supports
the following arguments:

- `enabled_task_drivers` `([]string: <optional>)` - Task drivers enabled for the namespace.
- `disabled_task_drivers` `([]string: <optional>)` - Task drivers disabled for the namespace.