---
layout: "nomad"
page_title: "Nomad: nomad_plugins"
sidebar_current: "docs-nomad-datasource-plugins"
description: |-
  Retrieve a list of plugins.
---

# nomad_plugins

Retrieve a list of dynamic plugins in Nomad.

## Example Usage

```hcl
data "nomad_plugins" "example" {}
```

## Attribute Reference

The following attributes are exported:

* `plugins`: `(list of maps)` a list of dynamic plugins registered in the cluster.
  * `id`: `(string)` ID for the plugin.
  * `provider`: `(string)` Plugin provider vendor.
  * `controller_required`: `(boolean)` Whether a controller is required.
  * `controllers_healthy`: `(integer)` Number of healthy controllers.
  * `controllers_expected`: `(integer)` Number of expected controllers.
  * `nodes_healthy`: `(integer)` Number of nodes with a healthy client.
  * `nodes_expected`: `(integer)` Expectec number of nodes with a client.
