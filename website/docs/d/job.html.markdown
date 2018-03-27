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
  job_id = "example_job"
}
```

## Argument Reference

The following arguments are supported:

* `job_id`: `(string)` ID of the job.

## Attributes Reference

The following attributes are exported:

* `name`: `(string)` Name of the job.
* `type`: `(string)` Scheduler type used during job creation.
* `version`: `(integer)` Version of the specified job.
* `namespace`: `(string)` Namespace of the specified job.
* `region`: `(string)` Region where the Nomad cluster resides.
* `datacenters`: `(list of strings)` Allowed datacenters that can run the specified job.
* `status`: `(string)` Execution status of the specified job.
* `status_description`: `(string)` Status description of the specified job.
* `submit_time`: `(integer)` Job submission date.
* `create_index`: `(integer)` Creation Index.
* `modify_index`: `(integer)` Modification Index.
* `job_modify_index`: `(integer)` Job modify index (used for version verification).
* `stop`: `(boolean)` Job enabled status.
* `priority`: `(integer)` Job priority.
* `parent_id`: `(string)` Job's parent ID.
* `task_groups`: `(list of maps)` Job's Task Groups.
* `stable`: `(boolean)` Job stability status.
* `all_at_once`: `(boolean)` All-at-once setting status.
* `contraints`: `(list of maps)` Job constraints.
* `update_strategy`: `(list of maps)` Job's update strategy.
* `periodic_config`: `(list of maps)` Job's periodic configuration.
