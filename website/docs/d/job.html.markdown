---
layout: "nomad"
page_title: "Nomad: nomad_job"
sidebar_current: "docs-nomad-datasource-job"
description: |-
  Get information on an job.
---

# nomad_job

Get information on a job ID. The aim of this datasource is to enable
you to act on various settings and states of a particular job.

An error is triggered if zero or more than one result is returned by the query.

## Example Usage

Get the data about a snapshot:

```hcl
data "nomad_job" "example" {
  job_id = "example"
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
* `datacenters`: `(list of strings)` Datacenters allowed to run the specified job.
* `status`: `(string)` Execution status of the specified job.
* `status_description`: `(string)` Status description of the specified job.
* `submit_time`: `(integer)` Job submission date.
* `create_index`: `(integer)` Creation Index.
* `modify_index`: `(integer)` Modification Index.
* `job_modify_index`: `(integer)` Job modify index (used for version verification).
* `stop`: `(boolean)` Job enabled status.
* `priority`: `(integer)` Used for the prioritization of scheduling and resource access.
* `parent_id`: `(string)` Job's parent ID.
* `task_groups`: `(list of maps)` A list of of the job's task groups.
  * `placed_canaries`: `(string)`
  * `auto_revert`: `(boolean)`
  * `promoted`: `(boolean)`
  * `desired_canaries`: `(integer)`
  * `desired_total`: `(integer)`
  * `placed_alloc`: `(integer)`
  * `healthy_alloc`: `(integer)`
  * `unhealthy_alloc`: `(integer)`
* `stable`: `(boolean)` Job stability status.
* `all_at_once`: `(boolean)`  If the scheduler can make partial placements on oversubscribed nodes.
* `contraints`: `(list of maps)` Job constraints.
  * `ltarget`: `(string)` Attribute being constrained.
  * `rtarget`: `(string)` Constraint value.
  * `operand`: `(string)` Operator used to compare the attribute to the constraint.
* `update_strategy`: `(list of maps)` Job's update strategy which controls rolling updates and canary deployments.
  * `stagger`: `(string)` Delay between migrating job allocations off cluster nodes marked for draining.
  * `max_parallel`: `(integer)` Number of task groups that can be updated at the same time.
  * `health_check`: `(string)` Type of mechanism in which allocations health is determined.
  * `min_healthy_time`: `(string)` Minimum time the job allocation must be in the healthy state.
  * `healthy_deadline`: `(string)` Deadline in which the allocation must be marked as healthy after which the allocation is automatically transitioned to unhealthy.
  * `auto_revert`: `(boolean)` Specifies if the job should auto-revert to the last stable job on deployment failure.
  * `canary`: `(integer)` Number of canary jobs that need to reach healthy status before unblocking rolling updates.
* `periodic_config`: `(list of maps)` Job's periodic configuration (time based scheduling).
  * `enabled`: `(boolean)` If periodic scheduling is enabled for the specified job.
  * `spec`: `(string)`
  * `spec_type`: `(string)`
  * `prohibit_overlap`: `(boolean)`  If the specified job should wait until previous instances of the job have completed.
  * `timezone`: `(string)` Time zone to evaluate the next launch interval against.
