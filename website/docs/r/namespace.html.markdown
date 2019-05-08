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

## Example Usage

Registering a namespace:

```hcl
resource "nomad_namespace" "dev" {
  name        = "dev"
  description = "Shared development environment."
  quota = "dev"
}
```

## Argument Reference

The following arguments are supported:

- `name` `(string: <required>)` - A unique name for the namespace.
- `description` `(string: "")` - A description of the namespace.
- `quota` `(string: "")` - a resource quota to attach to the namespace.
