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
  * `ID`: `string` Volume ID.
  * `ExternalID`: `string` The native ID for the volume.
  * `Namespace`: `string` Volume namespace.
  * `Name`: `string` User-friendly name.
  * `AccessMode`: `string` Describes write-access and concurrent usage for the volume.
  * `AttachmentMode`: `string` Describes the storage API used to interact with the device.
