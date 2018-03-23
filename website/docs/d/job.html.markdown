---
layout: "nomad"
page_title: "Nomad: nomad_job"
sidebar_current: "docs-nomad-datasource-job"
description: |-
  Get information on an job.
---

# nomad_job

Get information on an job ID. The aim of this datasource is to enable
you to act on various settings and states of a particular job.

An error is triggered if zero or more than one result is returned by the query.

## Example Usage

Get the data about a snapshot:

```hcl
data "nomad_job" "example1" {
  id = "example_job"
}
```

## Argument Reference

The following arguments are supported:

* `id` - The ID of the job.

## Attributes Reference

The following attributes are exported:

* `name`: Name of the job.
* `type`: Job type.
* `version`: Job version.
* `namespace`: Namespace of the job.
* `region`: Job region.
* `datacenters`: Allowed datacenters for job.
* `status`: Job status.
* `status_description`: Job Status Description.
* `submit_time`: Job submission date.
* `create_index`: Creation Index.
* `modify_index`: Modification Index.
* `job_modify_index`: Job modification index.
* `stop`: Job enabled status.
* `priority`: Job priority.
* `parent_id`: Job's parent ID.
* `task_groups`: Job's Task Groups.
* `stable`: Job stability status.
* `all_at_once`: All-at-once setting status.
* `contraints`: Job constraints.
* `update_strategy`: Job's update strategy.
* `periodic_config`: Job's periodic configuration.
* `vault_token`: Job Vault token.
