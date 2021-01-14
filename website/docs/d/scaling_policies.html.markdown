---
layout: "nomad"
page_title: "Nomad: nomad_scaling_policies"
sidebar_current: "docs-nomad-datasource-scaling-policies"
description: |-
  Retrieve a list of Scaling Policies.
---

# nomad_scaling_policies

Retrieve a list of Scaling Policies.

## Example Usage

```hcl
data "nomad_scaling_policies" "example" {
  job_id = "webapp"
  type   = "horizontal"
}
```

## Argument Reference

The following arguments are supported:

* `job_id` `(string)` - An optional string to filter scaling policies based on the target job. If not provided, policies for all jobs are returned.
* `type` `(string)` - An optional string to filter scaling policies based on policy type. If not provided, policies of all types are returned.

## Attribute Reference

The following attributes are exported:

* `policies` `list of maps` - A list of scaling policies.
  * `id` `(string)` - The scaling policy ID.
  * `enabled` `(boolean)` - Whether or not the scaling policy is enabled.
  * `type` `(string)` - The scaling policy type.
  * `target` `(map[string]string)` - The scaling policy target.
