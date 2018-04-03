---
layout: "nomad"
page_title: "Nomad: nomad_deployment"
sidebar_current: "docs-nomad-datasource-deployment"
description: |-
  Get information on an deployment.
---

# nomad_deployment

Get information on an deployment ID. The aim of this datasource is to enable
you to act on various settings and states of a particular deployment.

An error is triggered if zero or more than one result is returned by the query.

## Example Usage

Get the data about a deployment:

```hcl
data "nomad_deployment" "example1" {
  deployment_id = "70638f62-5c19-193e-30d6-f9d6e689ab8e"
}
```

## Argument Reference

The following arguments are supported:

* `deployment_id`: `string` The ID of the deployment.

## Attributes Reference

The following attributes are exported:

* `namespace`: `string` Namespace of the deployment.
* `job_id`: `string` ID of the job.
* `job_version`: `string` Job version.
* `job_create_index`: `integer` Job creation index.
* `job_modify_index`: `integer` Job modification index.
* `task_groups`: `map` Task Groups.
* `status`: `string` Deployment status.
* `status_description`: `string` Deployment Status Description.
* `create_index`: `integer` Deployment creation date.
* `modify_index`: `integer` Deployment modification date.
