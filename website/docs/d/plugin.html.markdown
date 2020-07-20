---
layout: "nomad"
page_title: "Nomad: nomad_plugin"
sidebar_current: "docs-nomad-datasource-plugin"
description: |-
  Get information on a specific CSI plugin.
---

# nomad_plugin

Lookup a plugin by ID. The aim of this datasource is to determine whether
a particular plugin exists on the cluster, to find information on the health
and availability of the plugin, and to optionally wait for the plugin
before performing actions the require an available plugin controller.

If a plugin with the specified ID does not exist and the datasource is not
configured to wait, it will result in an error. For simple existence checks,
use the `nomad_plugins` listing datasource.

## Example Usage

Check for the existence of a plugin:

```hcl
data "nomad_plugin" "ebs" {
  plugin_id        = "aws-ebs0"
  wait_for_healthy = true
}
```

This will check for a plugin with the ID `aws-ebs0`, waiting until the plugin
is healthy before returning.

## Argument Reference

The following arguments are supported:

* `plugin_id`: `(string)` ID of the plugin.
* `wait_for_registration`: `(boolean)` if the plugin doesn't exist, retry until it does
* `wait_for_healthy`: `(boolean)` retry until the plugin exists and all controllers are healthy

## Attributes Reference

The following attributes are exported:

* `plugin_id`: `(string)` ID of the plugin
* `plugin_provider`: `(string)` Plugin provider name
* `plugin_provider_version`: `(string)` Plugin provider version
* `controller_required`: `(boolean)` Whether a controller is required.
* `controllers_expected`: `(integer)` The number of registered controllers.
* `controllers_healthy`: `(integer)` The number of healthy controllers.
* `nodes_expected`: `(integer)` The number of registered nodes.
* `nodes_healthy`: `(integer)` The number of healthy nodes.
