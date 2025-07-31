---
layout: "nomad"
page_title: "Nomad: nomad_namespace"
sidebar_current: "docs-nomad-datasource-namespace"
description: |-
  Get information about a namespace in Nomad.
---

# nomad_namespace

Get information about a namespace in Nomad.

## Example Usage

```hcl
data "nomad_namespace" "namespaces" {
  name = "default"
}
```

## Argument Reference

- `name` `(string)` - The name of the namespace.

## Attribute Reference

The following attributes are exported:

* `description` `(string)` - The description of the namespace.
* `quota` `(string)` - The quota associated with the namespace.
* `meta` `(map[string]string)` -  Arbitrary KV metadata associated with the namespace.
* `capabilities` `(block)` - Capabilities of the namespace
  * `enabled_task_drivers` `([]string)` - Task drivers enabled for the namespace.
  * `disabled_task_drivers` `([]string)` - Task drivers disabled for the namespace.
  * `enabled_network_modes` `([]string)` - Network modes enabled for the namespace.
  * `disabled_network_modes` `([]string)` - Network modes disabled for the namespace.
