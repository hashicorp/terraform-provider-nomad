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

~> **Note:** Use `items_wo` with `items_wo_version` when you want Terraform to
  write variable items without storing those values in the state file.

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

Creating a variable with write-only items:

```hcl
resource "nomad_variable" "example" {
  path = "some/path/of/your/choosing"

  items_wo = jsonencode({
    example_key = "example_value"
  })

  items_wo_version = 1
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
- `items` `(map[string]string)` - An arbitrary map of items to create in the variable. Conflicts with `items_wo` and `items_wo_version`.
- `items_wo` `(string)` - A JSON-encoded map of variable items to write without storing those values in Terraform state. Conflicts with `items` and requires `items_wo_version`.
- `items_wo_version` `(number)` - A version marker for `items_wo`. Required when using `items_wo`, conflicts with `items`, and should be incremented to apply a new write-only payload.
