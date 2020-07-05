---
layout: "nomad"
page_title: "Nomad: nomad_volume"
sidebar_current: "docs-nomad-resource-volume"
description: |-
  Manages the lifecycle of registering and deregistering Nomad volumes.
---

# nomad_job

Manages an external volume in Nomad.

This can be used to register external volumes in a Nomad cluster. 

## Example Usage

Registering a volume:

```hcl
resource "nomad_volume" "vol1" {
}
```

## Argument Reference

The following arguments are supported:

- `jobspec` `(string: <required>)` - The contents of the jobspec to register.

- `deregister_on_destroy` `(bool: true)` - Determines if the job will be
  deregistered when this resource is destroyed in Terraform.

- `deregister_on_id_change` `(bool: true)` - Determines if the job will be
  deregistered if the ID of the job in the jobspec changes.

- `detach` `(bool: true)` - If true, the provider will return immediately
  after creating or updating, instead of monitoring.

- `policy_override` `(bool: false)` - Determines if the job will override any
  soft-mandatory Sentinel policies and register even if they fail.

- `json` `(bool: false)` - Set this to true if your jobspec is structured with
  JSON instead of the default HCL.
