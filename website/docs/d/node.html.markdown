---
layout: "nomad"
page_title: "Nomad: nomad_node"
sidebar_current: "docs-nomad-datasource-node"
description: |-
  Get information about a specific Nomad node.
---

# nomad_node

Get information about a specific Nomad node by its ID.

~> **Note:** All node attribute values can change if the node is restarted and
its fingerprint changes. In particular, the `drain`, `status`, `status_description`,
and `scheduling_eligibility` fields are ephemeral and can change at any time
without an agent restart.

## Example Usage

```hcl
data "nomad_node" "my_node" {
  node_id = "abc123"
}
```

## Argument Reference

The following arguments are supported:

- `node_id` `(string: <required>)` - The ID of the node to look up.

## Attribute Reference

The following attributes are exported:

- `name` `(string)` - The name of the node.
- `datacenter` `(string)` - The datacenter of the node.
- `node_class` `(string)` - The node class of the node.
- `node_pool` `(string)` - The node pool of the node.
- `http_addr` `(string)` - The HTTP address of the node.
- `drain` `(bool)` - Whether the node is in drain mode. This value is ephemeral
  and can change without an agent restart.
- `status` `(string)` - The status of the node. This value is ephemeral and
  can change without an agent restart.
- `status_description` `(string)` - The status description of the node. This
  value is ephemeral and can change without an agent restart.
- `scheduling_eligibility` `(string)` - The scheduling eligibility of the node.
  This value is ephemeral and can change without an agent restart.
- `attributes` `(map of string)` - A map of attributes for the node, including
  OS and hardware fingerprint information.
- `meta` `(map of string)` - A map of metadata for the node.
- `drivers` `(list)` - A list of driver information for the node.
  - `name` `(string)` - The driver name.
  - `detected` `(bool)` - Whether the driver is detected.
  - `healthy` `(bool)` - Whether the driver is healthy.
  - `attributes` `(map of string)` - Driver-specific attributes.
- `host_volumes` `(list)` - A list of host volumes on the node.
  - `name` `(string)` - The name of the host volume.
  - `path` `(string)` - The path of the host volume.
  - `read_only` `(bool)` - Whether the host volume is read-only.
  - `id` `(string)` - The ID of the host volume (set for dynamic host volumes only).
- `node_resources` `(list)` - Resources available on the node.
  - `cpu` `(list)` - CPU resources on the node.
    - `cpu_shares` `(int)` - Total CPU shares available.
    - `total_cpu_cores` `(int)` - Total number of CPU cores.
    - `reservable_cpu_cores` `(list of int)` - List of reservable CPU core IDs.
  - `memory` `(list)` - Memory resources on the node.
    - `memory_mb` `(int)` - Total memory in MB.
  - `disk` `(list)` - Disk resources on the node.
    - `disk_mb` `(int)` - Total disk space in MB.
  - `networks` `(list)` - Network resources on the node.
    - `device` `(string)` - The network device.
    - `cidr` `(string)` - The CIDR of the network.
    - `ip` `(string)` - The IP address of the network.
    - `mode` `(string)` - The network mode.
  - `devices` `(list)` - Device resources on the node (GPUs, etc.).
    - `vendor` `(string)` - The device vendor.
    - `type` `(string)` - The device type.
    - `name` `(string)` - The device name.
    - `count` `(int)` - The number of device instances.
  - `min_dynamic_port` `(int)` - Minimum dynamic port for this node.
  - `max_dynamic_port` `(int)` - Maximum dynamic port for this node.
- `reserved_resources` `(list)` - Resources reserved on the node.
  - `cpu` `(list)` - Reserved CPU resources.
    - `cpu_shares` `(int)` - Reserved CPU shares.
  - `memory` `(list)` - Reserved memory resources.
    - `memory_mb` `(int)` - Reserved memory in MB.
  - `disk` `(list)` - Reserved disk resources.
    - `disk_mb` `(int)` - Reserved disk space in MB.
  - `networks` `(map of string)` - Reserved network resources.
