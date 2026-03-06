---
layout: "nomad"
page_title: "Nomad: nomad_namespace"
sidebar_current: "docs-nomad-datasource-namespace"
description: |-
  Get information about a namespace in Nomad.
---

# nomad_namespace

Get information about a namespace in Nomad.

## Example Usage

```hcl
data "nomad_namespace" "namespaces" {
  name = "default"
}
```

## Argument Reference

- `name` `(string)` - The name of the namespace.

## Attribute Reference

The following attributes are exported:

* `description` `(string)` - The description of the namespace.
* `quota` `(string)` - The quota associated with the namespace.
* `meta` `(map[string]string)` -  Arbitrary KV metadata associated with the namespace.
* `capabilities` `(block)` - Capabilities of the namespace
  * `enabled_task_drivers` `([]string)` - Task drivers enabled for the namespace.
  * `disabled_task_drivers` `([]string)` - Task drivers disabled for the namespace.
  * `enabled_network_modes` `([]string)` - Network modes enabled for the namespace.
  * `disabled_network_modes` `([]string)` - Network modes disabled for the namespace.
* `node_pool_config` `(block)` - Node pool configuration for the namespace.
  * `default` `(string)` - The default node pool for jobs that don't define one.
  * `allowed` `([]string)` - The list of node pools allowed to be used in this namespace.
  * `denied` `([]string)` - The list of node pools not allowed to be used in this namespace.
* `vault_config` `(block)` - Vault configuration for the namespace.
  * `default` `(string)` - The Vault cluster to use when none is specified in the job.
  * `allowed` `([]string)` - The list of Vault clusters allowed to be used in this namespace.
  * `denied` `([]string)` - The list of Vault clusters not allowed to be used in this namespace.
* `consul_config` `(block)` - Consul configuration for the namespace.
  * `default` `(string)` - The Consul cluster to use when none is specified in the job.
  * `allowed` `([]string)` - The list of Consul clusters allowed to be used in this namespace.
  * `denied` `([]string)` - The list of Consul clusters not allowed to be used in this namespace.
