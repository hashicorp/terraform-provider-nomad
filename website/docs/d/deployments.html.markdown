---
layout: "nomad"
page_title: "Nomad: nomad_deployments"
sidebar_current: "docs-nomad-datasource-deployments"
description: |-
  Retrieve a list of deployments and a summary of their attributes.
---

# nomad_deployments

Retrieve a list of deployments in Nomad.

## Example Usage

```hcl
data "nomad_deployments" "example" {}
```

## Attribute Reference

The following attributes are exported:

* `deployments`: `list of maps` a list of deployments in the cluster.
  * `ID`: `string` Deployment ID.
  * `JobID`: `string` Job ID associated with the deployment.
  * `JobVersion`: `string` Job version.
  * `Status`: `string` Deployment status.
  * `StatusDescription`: `string` Detailed description of the deployment's status. 
