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

## Attribute Reference

The following attributes are exported:

* `volumes`: `list of maps` a list of volumes in the cluster.
  * `namespace`: `string` Volume namespace.
  * `id`: `string` Volume ID.
  * `name`: `string` User-friendly name.
  * `external_id`: `string` The native ID for the volume.
  * `access_mode`: `string` Describes write-access and concurrent usage for the volume.
  * `attachment_mode`: `string` Describes the storage API used to interact with the device.
