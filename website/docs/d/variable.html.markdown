---
layout: "nomad"
page_title: "Nomad: nomad_variable"
sidebar_current: "docs-nomad-datasource-variable"
description: |-
  Get the information about a Nomad variable.
---

# nomad_variable

Get the information about a Nomad variable.

~> **Warning:** this data source will store the sensitive values from `items`
  in the Terraform's state file. Take care to
  [protect your state file](/docs/state/sensitive-data.html).

## Example Usage

```hcl
resource "nomad_variable" "example" {
  path  = "path/of/existing/variable"
}
```

## Argument Reference

- `path` `(string)` - Path to the existing variable.
- `namespace` `(string: "default")` - The namepsace in which the variable exists.

## Attribute Reference

The following attributes are exported:
- `path` `(string)` - The path at which the variable exists.
- `namespace` `(string)` - The namespace in which the variable exists.
- `items` `(map[string]string)` - Map of items in the variable.
