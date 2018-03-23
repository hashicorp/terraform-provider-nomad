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

Get the data about a snapshot:

```hcl
data "nomad_deployment" "example1" {
  id = "example_deployment"
}
```

## Argument Reference

The following arguments are supported:

* `id` - The ID of the deployment.

## Attributes Reference

The following attributes are exported:

* `namespace`: Namespace of the deployment.
* `job_id`: ID of the job.
* `job_version`: Job version.
* `job_create_index`: Job creation index.
* `job_modify_index`: Job modification index.
* `task_groups`: Task Groups.
* `status`: Deployment status.
* `status_description`: Deployment Status Description.
* `create_index`: Deployment creation date.
* `modify_index`: Deployment modification date.
