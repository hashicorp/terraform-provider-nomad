---
layout: "nomad"
page_title: "Nomad: nomad_namespace"
sidebar_current: "docs-nomad-resource-namespace"
description: |-
  Provisions a namespace within a Nomad cluster.
---

# nomad_namespace

Provisions a namespace within a Nomad cluster.

~> **Enterprise Only!** This API endpoint and functionality only exists in
Nomad Enterprise. This is not present in the open source version of Nomad.

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
  quota = "dev"
}
```

Registering a namespace with a quota

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
  name = "web"
  description = "Web team production environment."
  quota = "${nomad_quota_specification.web_team.name}"
}
```

## Argument Reference

The following arguments are supported:

- `name` `(string: <required>)` - A unique name for the namespace.
- `description` `(string: "")` - A description of the namespace.
- `quota` `(string: "")` - A resource quota to attach to the namespace.
