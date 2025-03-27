---
layout: "nomad"
page_title: "Nomad: nomad_dynamic_host_volume"
sidebar_current: "docs-nomad-datasource-dynamic-host-volume"
description: |-
  Retrieve a dynamic host volumes.
---

# nomad_dynamic_host_volume

Get information on a dynamic host volume from Nomad.

## Example Usage

Check for the existing of a host volume:

```hcl
data "nomad_dynamic_host_volume" "example" {
  namespace = "prod"
  id        = "d688ff7a-d299-11ef-ae3c-6f2400953c18"
}
```

This will check for a dynamic host volume with the ID
`d688ff7a-d299-11ef-ae3c-6f2400953c18`.

## Argument Reference

The following arguments are supported:

* `id`: `(string)` - the ID of the volume
* `namespace`: `(string)` - the namespace of the volume. Defaults to `"default"`

## Attributes Reference

The following attributes are exported:

- `capacity` `(string)` - The size of the volume, in human-friendly format
  (ex. 10 GiB)

- `capacity_bytes` `(int)` - The size of the volume, in bytes.

- `capacity_max` `(string)` - The requested maximum capacity of the volume, in
  human-friendly format (ex. 10 GiB).

- `capacity_max_bytes` `(string)` - The requested maximum capacity of the
  volume, in bytes.

- `capacity_min` `(string)` - The requested minimum capacity of the volume, in
  human-friendly format (ex. 10 GiB).

- `capacity_min_bytes` `(string)` - The requested minimum capacity of the
  volume, in bytes.

- `capability` `(block)` - Option for validating the capability of a
  volume. Each capability block has the following attributes:
  * `access_mode` `(string)` - How the volume can be mounted by
    allocations. Refer to the [`access_mode`][] documentation for details.
  * `attachment_mode` `(string)` - The storage API that will be used by the
    volume. Refer to the [`attachment_mode`][] documentation.

- `constraint` `(block)` - The restrictions used to place the volume on a node,
  similar to the [`constraint`][] block on a Nomad job specification. A volume
  may have multiple `constraint` blocks. Each constraint block has the following
  attributes.
  * `attribute` `(string)` - The [node attribute][] to check for the constraint.
  * `value` `(string)` - The value of the attribute to compare against.
  * `operator` `(string)`- The operator to use in the comparison.

- `host_path` `(string)` - The path on disk where the volume exists.

- `name` `(string)` - The name of the volume, which is used as the
  [`volume.source`][volume_source] field in job specifications that claim this
  volume. Host volume names are be unique per node. Names are visible to any
  user with `node:read` ACL, even across namespaces, so they should not be
  treated as sensitive values.

- `namespace` `(string)` - The namespace of the volume.

- `node_id` `(string)` - A specific node where the volume is mounted.

- `node_pool` `(string: <optional>)` - The node pool of the node where the
  volume is mounted.

- `parameters` `(map<string|string>)` - A key-value map of strings
  passed directly to the plugin to configure the volume. The details of these
  parameters are specific to the plugin.

- `plugin_id` `(string: <required>)` - The ID of the [dynamic host volume
  plugin][dhv_plugin] that manages this volume.

[`constraint`]: /nomad/docs/job-specification/constraint
[node attribute]: /nomad/docs/runtime/interpolation#interpreted_node_vars
[`access_mode`]: /nomad/docs/other-specifications/volume/capability#access_mode
[`attachment_mode`]: /nomad/docs/other-specifications/volume/capability#attachment_mode
[volume_source]: /nomad/docs/job-specification/volume#source
