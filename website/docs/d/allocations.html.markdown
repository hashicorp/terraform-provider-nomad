---
layout: "nomad"
page_title: "Nomad: nomad_allocations"
sidebar_current: "docs-nomad-datasource-allocations"
description: |-
  Retrieve a list of allocations from Nomad.
---

# nomad_allocations

Retrieve a list of allocations from Nomad.

## Example Usage

```hcl
data "nomad_allocations" "example" {
  filter = "JobID == \"example\""
}
```

## Argument Reference

The following arguments are supported:

- `prefix` `(string: <optional>)` - Specifies a string to filter allocations
  based on an ID prefix.
- `filter` `(string: <optional>)` - Specifies the
  [expression][nomad_api_filter] used to filter the results.

## Attribute Reference

The following attributes are exported:

- `allocations` `(list of allocations)` - A list of allocations matching the
  search criteria.
  - `id` `(string)` - The ID of the allocation.
  - `eval_id` `(string)` - The ID of the evaluation that generated the allocation.
  - `name` `(string)` - The name of the allocation.
  - `namespace` `(string)` - The namespace the allocation belongs to.
  - `node_id` `(string)` - The ID of the node to which the allocation was scheduled.
  - `node_name` `(string)` - The ID of the node to which the allocation was scheduled.
  - `job_id` `(string)` - The ID of the job related to the allocation.
  - `job_type` `(string)` - The type of the job related to the allocation.
  - `job_version` `(int)` - The version of the job that generated the allocation.
  - `task_group` `(string)` - The job task group related to the allocation.
  - `desired_status` `(string)` - The current desired status of the allocation.
  - `client_status` `(string)` - The current client status of the allocation.
  - `followup_eval_id` `(string)` - The ID of the evaluation that succeeds the allocation evaluation.
  - `next_allocation` `(string)` - The ID of the allocation that succeeds the allocation.
  - `preempted_by_allocation` `(string)` - The ID of the allocation that preempted the allocation.
  - `create_index` `(int)` - The Raft index in which the allocation was created.
  - `modify_index` `(int)` - The Raft index in which the allocation was last modified.
  - `create_time` `(int)` - The timestamp of when the allocation was created.
  - `modify_time` `(int)` - The timestamp of when the allocation was last modified.

[nomad_api_filter]: https://developer.hashicorp.com/nomad/api-docs/v1.6.x#filtering
