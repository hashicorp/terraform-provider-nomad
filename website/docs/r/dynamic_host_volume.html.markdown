---
---
layout: "nomad"
page_title: "Nomad: nomad_dynamic_host_volume"
sidebar_current: "docs-nomad-resource-dynamic-host-volume"
description: |-
  Manages the lifecycle of creating and deleting dynamic host volumes.
---

# nomad_dynamic_host_volume

Creates and registers a dynamic host volume in Nomad. Note that Nomad supports
two workflows for dynamic host volumes: create and register. Both resources
result in the same data source with the same outputs.

~> **Warning:** Destroying this resource **will result in data loss**. Use the
  [`prevent_destroy`][tf_docs_prevent_destroy] directive to avoid accidental
  deletions.


## Example Usage

Creating a dynamic host volume:

```hcl
resource nomad_dynamic_host_volume "example" {
  name      = "example"
  namespace = "prod"
  plugin_id = "mkdir"

  capacity_max = "12 GiB"
  capacity_min = "1.0 GiB"

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
  }

  constraint {
    attribute = "$${attr.kernel.name}"
    value     = "linux"
  }

  parameters = {
    some_key = "some_value"
  }
}
```

## Argument Reference

The following arguments are supported:

- `capacity_min` `(string: <optional>)` - Option for requesting a minimum
  capacity, in bytes. The capacity of a volume may be the physical size of a
  disk, or a quota, depending on the plugin. The specific size of the resulting
  volume is somewhere between `capacity_min` and `capacity_max`; the exact
  behavior is up to the plugin. If you want to specify an exact size, set
  `capacity_min` and `capacity_max` to the same value. Accepts human-friendly
  suffixes such as `"100GiB"`. Plugins that cannot restrict the size of volumes
  may ignore this field.

- `capacity_max` `(string: <optional>)` - Option for requesting a maximum
  capacity, in bytes. The capacity of a volume may be the physical size of a
  disk, or a quota, depending on the plugin. The specific size of the resulting
  volume is somewhere between `capacity_min` and `capacity_max`; the exact
  behavior is up to the plugin. If you want to specify an exact size, set
  `capacity_min` and `capacity_max` to the same value. Accepts human-friendly
  suffixes such as `"100GiB"`. Plugins that cannot restrict the size of volumes
  may ignore this field.

- `capability` `(block: <optional>)` - Option for validating the capability of a
  volume. Each capability block has the following attributes:
  * `access_mode` `(string)` - How the volume can be mounted by
    allocations. Refer to the [`access_mode`][] documentation for details.
  * `attachment_mode` `(string)` - The storage API that will be used by the
    volume. Refer to the [`attachment_mode`][] documentation.

- `constraint` `(block: <optional>)` - A restriction on the eligible nodes where
  a volume can be created, similar to the [`constraint`][] block on a Nomad job
  specification.. You can provide multiple `constraint` blocks to add more
  constraints. Each constraint block has the following attributes.
  * `attribute` `(string)` - The [node attribute][] to check for the constraint.
  * `value` `(string)` - The value of the attribute to compare against.
  * `operator` `(string)`- The operator to use in the comparison.

- `name` `(string: <required>)` - The name of the volume, which is used as the
  [`volume.source`][volume_source] field in job specifications that claim this
  volume. Host volume names must be unique per node. Names are visible to any
  user with `node:read` ACL, even across namespaces, so they should not be
  treated as sensitive values.

- `namespace` `(string: <optional>)` - The namespace of the volume. This field
  overrides the namespace provided by the `-namespace` flag or `NOMAD_NAMESPACE`
  environment variable. Defaults to `"default"` if unset.

- `node_id` `(string: <optional>)` - A specific node where you would like the
  volume to be created.

- `node_pool` `(string: <optional>)` - A specific node pool where you would like
  the volume to be created. If you also provide `node_id`, the node must be in the
  provided `node_pool`.

- `parameters` `(map<string|string>: <optional>)` - A key-value map of strings
  passed directly to the plugin to configure the volume. The details of these
  parameters are specific to the plugin.

- `plugin_id` `(string: <required>)` - The ID of the [dynamic host volume
  plugin][dhv_plugin] that manages this volume.


[tf_docs_prevent_destroy]: https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle#prevent_destroy
[`constraint`]: /nomad/docs/job-specification/constraint
[node attribute]: /nomad/docs/runtime/interpolation#interpreted_node_vars
[`access_mode`]: /nomad/docs/other-specifications/volume/capability#access_mode
[`attachment_mode`]: /nomad/docs/other-specifications/volume/capability#attachment_mode
[volume_source]: /nomad/docs/job-specification/volume#source
# nomad_dynamic_host_volume

Creates and registers a dynamic host volume in Nomad.

~> **Warning:** Destroying this resource **will result in data loss**. Use the
  [`prevent_destroy`][tf_docs_prevent_destroy] directive to avoid accidental
  deletions.



[tf_docs_prevent_destroy]: https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle#prevent_destroy
