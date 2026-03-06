---
layout: "nomad"
page_title: "Nomad: nomad_nodes"
sidebar_current: "docs-nomad-datasource-nodes"
description: |-
  Retrieve a list of nodes from Nomad.
---

# nomad_nodes

Retrieve a list of nodes from Nomad.

~> **Note:** All node attribute values can change if the node is restarted and
its fingerprint changes. In particular, the `drain`, `status`, `status_description`,
and `scheduling_eligibility` fields are ephemeral and can change at any time
without an agent restart.

## Example Usage

```hcl
data "nomad_nodes" "all" {}
```

### Filtering by status

```hcl
data "nomad_nodes" "ready" {
  filter = "Status == \"ready\""
}
```

### Including OS attributes and resources

```hcl
data "nomad_nodes" "with_details" {
  os        = true
  resources = true
}
```

## Argument Reference

The following arguments are supported:

- `prefix` `(string: <optional>)` - Specifies a string to filter nodes based on
  an ID prefix. Must have an even number of hexadecimal characters (0-9a-f).
- `filter` `(string: <optional>)` - Specifies the [expression][nomad_api_filter]
  used to filter the results.
- `os` `(bool: false)` - If true, include special attributes such as operating
  system name in the response. When false, the `attributes` map will not contain
  OS-related attributes.
- `resources` `(bool: false)` - If true, include `node_resources` and
  `reserved_resources` in the response.

## Attribute Reference

The following attributes are exported:

- `nodes` `(list of nodes)` - A list of nodes matching the search criteria.
  - `id` `(string)` - The ID of the node.
  - `name` `(string)` - The name of the node.
  - `datacenter` `(string)` - The datacenter of the node.
  - `node_class` `(string)` - The node class of the node.
  - `node_pool` `(string)` - The node pool of the node.
  - `address` `(string)` - The address of the node.
  - `version` `(string)` - The Nomad version of the node.
  - `drain` `(bool)` - Whether the node is in drain mode. This value is ephemeral
    and can change without an agent restart.
  - `status` `(string)` - The status of the node. This value is ephemeral and
    can change without an agent restart.
  - `status_description` `(string)` - The status description of the node. This
    value is ephemeral and can change without an agent restart.
  - `scheduling_eligibility` `(string)` - The scheduling eligibility of the node.
    This value is ephemeral and can change without an agent restart.
  - `attributes` `(map of string)` - A map of attributes for the node. OS-related
    attributes are only included when the `os` parameter is set to true.
  - `drivers` `(list of drivers)` - A list of driver information for the node.
    - `name` `(string)` - The driver name.
    - `detected` `(bool)` - Whether the driver is detected.
    - `healthy` `(bool)` - Whether the driver is healthy.
    - `attributes` `(map of string)` - Driver-specific attributes.
  - `node_resources` `(list)` - Resources available on the node. Only populated
    when the `resources` parameter is set to true.
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
  - `reserved_resources` `(list)` - Resources reserved on the node. Only populated
    when the `resources` parameter is set to true.
    - `cpu` `(list)` - Reserved CPU resources.
      - `cpu_shares` `(int)` - Reserved CPU shares.
    - `memory` `(list)` - Reserved memory resources.
      - `memory_mb` `(int)` - Reserved memory in MB.
    - `disk` `(list)` - Reserved disk resources.
      - `disk_mb` `(int)` - Reserved disk space in MB.
    - `networks` `(map of string)` - Reserved network resources.

[nomad_api_filter]: https://developer.hashicorp.com/nomad/api-docs#filtering
