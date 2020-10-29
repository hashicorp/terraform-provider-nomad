---
layout: "nomad"
page_title: "Nomad: nomad_acl_policies"
sidebar_current: "docs-nomad-datasource-acl_policies"
description: |-
  Retrieve a list of ACL Policies.
---

# nomad_acl_policies

Retrieve a list of ACL Policies.

## Example Usage

```hcl
data "nomad_acl_policies" "example" {
  prefix = "prod"
}
```

## Argument Reference

The following arguments are supported:

* `prefix`: `(string)` An optional string to filter ACL policies based on name prefix. If not provided, all policies are returned. 

## Attribute Reference

The following attributes are exported:

* `policies`: `list of maps` a list of ACL policies.
  * `name` `(string)` - the name of the ACL Policy.
  * `description` `(string)` - the description of the ACL Policy.

