---
layout: "nomad"
page_title: "Nomad: nomad_dynamic_host_volume_registration"
sidebar_current: "docs-nomad-resource-dynamic-host-volume-registration"
description: |-
  Manages the lifecycle of registering and deregistering host volumes.
---

# nomad_dynamic_host_volume_registration

Registers a dynamic host volume in Nomad that has already been created. Note
that Nomad supports two workflows for dynamic host volumes: create and
register. Both resources result in the same data source with the same outputs.

## Argument Reference

The following arguments are supported:

- `capacity` `(string: <optional>)` - The size of a volume in bytes. Either the
  physical size of a disk or a quota, depending on the plugin. This field must
  be between the `capacity_min` and `capacity_max` values unless they are
  omitted. Accepts human-friendly suffixes such as `"100GiB"`.

- `capability` `(block: <optional>)` - Option for validating the capability of a
  volume. Each capability block has the following attributes:
  * `access_mode` `(string)` - How the volume can be mounted by
    allocations. Refer to the [`access_mode`][] documentation for details.
  * `attachment_mode` `(string)` - The storage API that will be used by the
    volume. Refer to the [`attachment_mode`][] documentation.

- `host_path` `(string)` - The path on disk where the volume exists.

- `name` `(string: <required>)` - The name of the volume, which is used as the
  [`volume.source`][volume_source] field in job specifications that claim this
  volume. Host volume names must be unique per node. Names are visible to any
  user with `node:read` ACL, even across namespaces, so they should not be
  treated as sensitive values.

- `namespace` `(string: <optional>)` - The namespace of the volume. This field
  overrides the namespace provided by the `-namespace` flag or `NOMAD_NAMESPACE`
  environment variable. Defaults to `"default"` if unset.

- `node_id` `(string: <required>)` - A specific node where the volume is
  mounted.

- `parameters` `(map<string|string>: <optional>)` - A key-value map of strings
  passed directly to the plugin to configure the volume. The details of these
  parameters are specific to the plugin.


[`access_mode`]: /nomad/docs/other-specifications/volume/capability#access_mode
[`attachment_mode`]: /nomad/docs/other-specifications/volume/capability#attachment_mode
[volume_source]: /nomad/docs/job-specification/volume#source
