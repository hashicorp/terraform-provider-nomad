---
layout: "nomad"
page_title: "Nomad: nomad_datacenters"
sidebar_current: "docs-nomad-datasource-datacenters"
description: |-
  Retrieve a list of datacenters.
---

# nomad_datacenters

Retrieve a list of datacenters.

## Example Usage

```hcl
data "nomad_datacenters" "datacenters" {
  prefix            = "prod"
  ignore_down_nodes = true
}
```

## Argument Reference

The following arguments are supported:

* `prefix` `(string)`: An optional string to filter datacenters based on name prefix. If not provided, all datacenters are returned.
* `ignore_down_nodes` `(bool: false)`: An optional flag that, if set to `true` will ignore down nodes when compiling the list of datacenters.

## Attribute Reference

The following attributes are exported:

* `datacenters`: `list(string)` a list of datacenters.
