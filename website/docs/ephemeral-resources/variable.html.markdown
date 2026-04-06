---
layout: "nomad"
page_title: "Nomad: nomad_variable"
sidebar_current: "docs-nomad-ephemeral-resource-variable"
description: |-
  Reads a Nomad variable without storing its items in Terraform state.
---

# nomad_variable

Reads a Nomad variable during a Terraform run without storing its items in
state.

Use the stateful `nomad_variable` resource to manage the variable lifecycle.
Use this ephemeral resource whenever variable items need to be read during a
run. Values returned by this resource are never persisted in state.

## Example Usage

```hcl
resource "nomad_variable" "example" {
  path = "some/path/of/your/choosing"

  items_wo = jsonencode({
    example_key = "example_value"
  })

  items_wo_version = 1
}

ephemeral "nomad_variable" "example" {
  path = nomad_variable.example.path
}

resource "some_resource" "example" {
  secret_value_wo         = ephemeral.nomad_variable.example.items.example_key
  secret_value_wo_version = 1
}
```

## Argument Reference

- `path` `(string: <required>)` - The path of the variable.
- `namespace` `(string: "default")` - The namespace in which the variable exists.

## Attribute Reference

The following attributes are exported:

- `path` `(string)` - The path at which the variable exists.
- `namespace` `(string)` - The namespace in which the variable exists.
- `items` `(map[string]string)` - Map of items in the variable.