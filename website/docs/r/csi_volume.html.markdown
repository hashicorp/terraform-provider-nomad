---
layout: "nomad"
page_title: "Nomad: nomad_csi_volume"
sidebar_current: "docs-nomad-resource-csi-volume"
description: |-
  Manages the lifecycle of creating and deleting CSI volumes.
---

# nomad_csi_volume

Creates and registers a CSI volume in Nomad.

This can be used to create and register CSI volumes in a Nomad cluster.

~> **Warning:** this resource will store any sensitive values placed in
  `secrets` or `mount_options` in the Terraform's state file. Take care to
  [protect your state file](/docs/state/sensitive-data.html).

## Example Usage

Creating a volume:

```hcl
# It can sometimes be helpful to wait for a particular plugin to be available
data "nomad_plugin" "ebs" {
  plugin_id        = "aws-ebs0"
  wait_for_healthy = true
}

resource "nomad_csi_volume" "mysql_volume" {
  depends_on = [data.nomad_plugin.ebs]

  plugin_id    = "aws-ebs0"
  volume_id    = "mysql_volume"
  name         = "mysql_volume"
  capacity_min = "10GiB"
  capacity_max = "20GiB"

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
  }

  mount_options {
    fs_type = "ext4"
  }

  topology_request {
    required {
      topology {
        segments = {
          rack = "R1"
          zone = "us-east-1a"
        }
      }

      topology {
        segments = {
          rack = "R2"
        }
      }
    }
  }
}
```

## Argument Reference

The following arguments are supported:

- `namespace`: `(string: "default")` - The namespace in which to register the volume.
- `volume_id`: `(string: <required>)` - The unique ID of the volume.
- `name`: `(string: <required>)` - The display name for the volume.
- `plugin_id`: `(string: <required>)` - The ID of the Nomad plugin for registering this volume.
- `snapshot_id`: `(string: <optional>)` - The external ID of a snapshot to restore. If ommited, the volume will be created from scratch. Conflicts with `clone_id`.
- `clone_id`: `(string: <optional>)` - The external ID of an existing volume to restore. If ommited, the volume will be created from scratch. Conflicts with `snapshot_id`.
- `capacity_min`: `(string: <optional>)` - Option to signal a minimum volume size. This may not be supported by all storage providers.
- `capacity_max`: `(string: <optional>)` - Option to signal a maximum volume size. This may not be supported by all storage providers.
- `capability`: `(`[`Capability`](#capability-1)`: <required>)` - Options for validating the capability of a volume.
- `topology_request`: `(`[`TopologyRequest`](#topology-request)`: <optional>)` - Specify locations (region, zone, rack, etc.) where the provisioned volume is accessible from.
- `mount_options`: `(block: optional)` Options for mounting `block-device` volumes without a pre-formatted file system.
  - `fs_type`: `(string: optional)` - The file system type.
  - `mount_flags`: `[]string: optional` - The flags passed to `mount`.
- `secrets`: `(map[string]string: optional)` An optional key-value map of strings used as credentials for publishing and unpublishing volumes.
- `parameters`: `(map[string]string: optional)` An optional key-value map of strings passed directly to the CSI plugin to configure the volume.

### Capability

- `access_mode`: `(string: <required>)` - Defines whether a volume should be available concurrently. Possible values are:
  - `single-node-reader-only`
  - `single-node-writer`
  - `multi-node-reader-only`
  - `multi-node-single-writer`
  - `multi-node-multi-writer`
- `attachment_mode`: `(string: <required>)` - The storage API that will be used by the volume. Possible values are:
  - `block-device`
  - `file-system`

### Topology Request

- `required`: `(`[`Topology`](#topology)`: <optional>)` - Required topologies indicate that the volume must be created in a location accessible from all the listed topologies.
- `preferred`: `(`[`Topology`](#topology)`: <optional>)` - Preferred topologies indicate that the volume should be created in a location accessible from some of the listed topologies.

### Topology

- `topology`: `(List of segments: <required>)` - Defines the location for the volume.
  - `segments`: `(map[string]string)` - Define the attributes for the topology request.

In addition to the above arguments, the following attributes are exported and
can be referenced:

- `access_mode`: `(string)`
- `attachment_mode`: `(string)`
- `controller_required`: `(boolean)`
- `controllers_expected`: `(integer)`
- `controllers_healthy`: `(integer)`
- `plugin_provider`: `(string)`
- `plugin_provider_version`: `(string)`
- `nodes_healthy`: `(integer)`
- `nodes_expected`: `(integer)`
- `schedulable`: `(boolean)`
- `topologies`: `(List of topologies)`

### Timeouts

`nomad_csi_volume` provides the following [`Timeouts`][tf_docs_timeouts]
configuration options.

- `create` `(string: "15m")` - Timeout when creating or updating a new CSI volume.
- `delete` `(string: "15m")` - Timeout when deleting a CSI volume .

[tf_docs_timeouts]: https://www.terraform.io/docs/configuration/blocks/resources/syntax.html#operation-timeouts
