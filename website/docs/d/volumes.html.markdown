---
layout: "nomad"
page_title: "Nomad: nomad_volumes"
sidebar_current: "docs-nomad-datasource-volumes"
description: |-
  Retrieve a list of volumes.
---

# nomad_volumes

Retrieve a list of volumes in Nomad.

## Example Usage

```hcl
data "nomad_volumes" "example" {}
```

## Argument Reference

The following arguments are supported:

* `type`: `(string: "csi")` Volume type (currently only supports `csi`)
* `node_id`: `(string: optional)` Volume node filter. 
* `plugin_id`: `(string: optional)` Plugin ID filter. 
* `namespace`: `(string: "default")` Nomad namespace.

## Attribute Reference

The following attributes are exported:

* `volumes`: `list of maps` a list of volumes in the cluster.
  * `namespace`: `string` Volume namespace.
  * `id`: `string` Volume ID.
  * `name`: `string` User-friendly name.
  * `external_id`: `string` The native ID for the volume.
  * `access_mode`: `string` Describes write-access and concurrent usage for the volume.
  * `attachment_mode`: `string` Describes the storage API used to interact with the device.
