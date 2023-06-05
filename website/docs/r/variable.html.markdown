---
layout: "nomad"
page_title: "Nomad: nomad_variable"
sidebar_current: "docs-nomad-resource-variable"
description: |-
  Manages the lifecycle of Nomad variables.
---

# nomad_variable

Creates and manages a variable, including it's contents, within a
Nomad cluster.

~> **Warning:** this resource will store the sensitive values placed in
  `items` in the Terraform's state file. Take care to
  [protect your state file](/docs/state/sensitive-data.html).

## Example Usage

Creating a variable in the default namespace:

```hcl
resource "nomad_variable" "example" {
  path  = "some/path/of/your/choosing"
  items = {
    example_key = "example_value"
  }
}
```

Creating a variable in a custom namespace:

```hcl
resource "nomad_namespace" "example" {
  name        = "example"
  description = "Example namespace."
}

resource "nomad_variable" "example" {
  path      = "some/path/of/your/choosing"
  namespace = nomad_namespace.example.name
  items     = {
    example_key = "example_value"
  }
}
```

## Argument Reference

- `path` `(string: <required>)` - A unique path to create the variable at.
- `namespace` `(string: "default")` - The namepsace to create the variable in.
- `items` `(map[string]string: <required>)` - An arbitrary map of items to create in the variable.
