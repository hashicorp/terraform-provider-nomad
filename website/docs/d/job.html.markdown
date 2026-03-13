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
  job_id    = "example"
  namespace = "dev"
}
```

## Argument Reference

The following arguments are supported:

* `job_id`: `(string)` ID of the job.
* `namespace`: `(string)` Namespace of the specified job.

## Attributes Reference

The following attributes are exported:

* `name`: `(string)` Name of the job.
* `type`: `(string)` Scheduler type used during job creation.
* `version`: `(integer)` Version of the specified job.
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
* `task_groups`: `(list of maps)` A list of the job's task groups.
  * `name`: `(string)` Task group name.
  * `count`: `(integer)` Task group count.
  * `update_strategy`: `(list of maps)` Effective update strategy for the task group.
    * `stagger`: `(string)` Delay between migrating job allocations off cluster nodes marked for draining.
    * `max_parallel`: `(integer)` Number of task groups that can be updated at the same time.
    * `health_check`: `(string)` Type of mechanism in which allocations health is determined.
    * `min_healthy_time`: `(string)` Minimum time the allocation must be in the healthy state.
    * `healthy_deadline`: `(string)` Deadline in which the allocation must be marked as healthy.
    * `auto_revert`: `(boolean)` Whether the job should auto-revert to the last stable job on deployment failure.
    * `canary`: `(integer)` Number of canary jobs that need to reach healthy status before unblocking rolling updates.
  * `placed_canaries`: `(list of strings)` Allocations placed as canaries for the task group.
  * `auto_revert`: `(boolean)` Whether the latest deployment for the task group is marked for auto-revert.
  * `promoted`: `(boolean)` Whether the canary deployment has been promoted.
  * `desired_canaries`: `(integer)` Desired number of canaries.
  * `desired_total`: `(integer)` Desired total number of allocations.
  * `placed_allocs`: `(integer)` Number of placed allocations.
  * `healthy_allocs`: `(integer)` Number of healthy allocations.
  * `unhealthy_allocs`: `(integer)` Number of unhealthy allocations.
  * `task`: `(list of maps)` Tasks in the task group.
    * `name`: `(string)` Task name.
    * `driver`: `(string)` Task driver.
    * `meta`: `(map of strings)` Task metadata.
    * `volume_mounts`: `(list of maps)` Task volume mounts.
      * `volume`: `(string)` Volume name.
      * `destination`: `(string)` Destination path inside the task.
      * `read_only`: `(boolean)` Whether the volume mount is read-only.
  * `volumes`: `(list of maps)` Volume requests for the task group.
    * `name`: `(string)` Volume name.
    * `type`: `(string)` Volume type.
    * `read_only`: `(boolean)` Whether the volume is read-only.
    * `source`: `(string)` Volume source.
  * `meta`: `(map of strings)` Task group metadata.
* `stable`: `(boolean)` Job stability status.
* `all_at_once`: `(boolean)`  If the scheduler can make partial placements on oversubscribed nodes.
* `constraints`: `(list of maps)` Job constraints.
  * `ltarget`: `(string)` Attribute being constrained.
  * `rtarget`: `(string)` Constraint value.
  * `operand`: `(string)` Operator used to compare the attribute to the constraint.
* `update_strategy`: `(list of maps)` Job-level update strategy returned by Nomad.
  * `stagger`: `(string)` Delay between migrating job allocations off cluster nodes marked for draining.
  * `max_parallel`: `(integer)` Number of task groups that can be updated at the same time.
  * `health_check`: `(string)` Type of mechanism in which allocations health is determined.
  * `min_healthy_time`: `(string)` Minimum time the allocation must be in the healthy state.
  * `healthy_deadline`: `(string)` Deadline in which the allocation must be marked as healthy.
  * `auto_revert`: `(boolean)` Whether the job should auto-revert to the last stable job on deployment failure.
  * `canary`: `(integer)` Number of canary jobs that need to reach healthy status before unblocking rolling updates.
* `periodic_config`: `(list of maps)` Job's periodic configuration.
  * `enabled`: `(boolean)` If periodic scheduling is enabled for the specified job.
  * `spec`: `(string)` Cron specification for the periodic job.
  * `spec_type`: `(string)` Type of periodic specification.
  * `prohibit_overlap`: `(boolean)` If the specified job should wait until previous instances of the job have completed.
  * `timezone`: `(string)` Time zone to evaluate the next launch interval against.
