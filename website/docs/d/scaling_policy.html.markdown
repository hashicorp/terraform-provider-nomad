---
layout: "nomad"
page_title: "Nomad: nomad_scaling_policy"
sidebar_current: "docs-nomad-datasource-scaling-policy"
description: |-
  Retrieve a Scaling Policy.
---

# nomad_scaling_policy

Retrieve a Scaling Policy.

## Example Usage

```hcl
data "nomad_scaling_policy" "example" {
  id = "ad19848d-1921-179c-affa-244a3543be88"
}
```

## Argument Reference

The following arguments are supported:

* `id` `(string: <required>)` - The  ID of the scaling policy.

## Attribute Reference

The following attributes are exported:

* `enabled` `(boolean)` - Whether or not the scaling policy is enabled.
* `type` `(string)` - The scaling policy type.
* `min` `(integer)` - The minimum value set in the scaling policy.
* `max` `(integer)` - The maximum value set in the scaling policy.
* `policy` `(string)` - The policy inside the scaling policy.
* `target` `(map[string]string)` - The scaling policy target.
