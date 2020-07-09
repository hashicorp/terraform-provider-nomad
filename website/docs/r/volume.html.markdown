---
layout: "nomad"
page_title: "Nomad: nomad_volume"
sidebar_current: "docs-nomad-resource-volume"
description: |-
  Manages the lifecycle of registering and deregistering Nomad volumes.
---

# nomad_volume

Manages an external volume in Nomad.

This can be used to register external volumes in a Nomad cluster. 

## Example Usage

Registering a volume:

```hcl
// it can sometimes be helpful to wait for a particular plugin to be available
data "nomad_plugin" "ebs" {
  plugin_id        = "aws-ebs0"
  wait_for_healthy = true
}
resource "nomad_volume" "mysql_volume" {
  depends_on      = [data.nomad_plugin.ebs]
  type            = "csi"
  plugin_id       = "aws-ebs0"
  volume_id       = "mysql_volume"
  name            = "mysql_volume"
  external_id     = module.hashistack.ebs_test_volume_id
  access_mode     = "single-node-writer"
  attachment_mode = "file-system"
}

```

## Argument Reference

The following arguments are supported:

- `type`: `(string: <required>)` The type of the volume. Currently, only `csi` is supported.
- `namespace`: `(string: "default")` The namespace in which to register the volume.
- `volume_id`: `(string: <required>)` The unique ID of the volume.
- `name`: `(string: <required>)` The display name for the volume.
- `plugin_id`: `(string: <required>)` The ID of the Nomad plugin for registering this volume.
- `external_id`: `(string: <required>)` The ID of the physical volume from the storage provider.
- `access_mode`: `(string: <required>)` Defines whether a volume should be available concurrently. Possible values are:
  - `single-node-reader-only`
  - `single-node-writer`
  - `multi-node-reader-only`
  - `multi-node-single-writer`
  - `multi-node-multi-writer`
- `attachment_mode`: `(string: <required>)` The storage API that will be used by the volume.
- `secrets`: `(map[string]string: optional)` An optional key-value map of strings used as credentials for publishing and unpublishing volumes.
- `parameters`: `(map[string]string: optional)` An optional key-value map of strings passed directly to the CSI plugin to configure the volume.
- `context`: `(map[string]string: optional)` An optional key-value map of strings passed directly to the CSI plugin to validate the volume.
- `deregister_on_destroy`: `(boolean: false)` If true, the volume will be deregistered on destroy.

In addition to the above arguments, the following attributes are exported and
can be referenced:

- `controller_required`: `(boolean)` 
- `controllers_expected`: `(integer)`
- `controllers_healthy`: `(integer)`
- `plugin_provider`: `(string)`
- `plugin_provider_version`: `(string)`
- `nodes_healthy`: `(integer)`
- `nodes_expected`: `(integer)`
- `schedulable`: `(boolean)`
