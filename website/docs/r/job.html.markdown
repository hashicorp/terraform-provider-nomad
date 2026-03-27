---
layout: "nomad"
page_title: "Nomad: nomad_job"
sidebar_current: "docs-nomad-resource-job"
description: |-
  Manages the lifecycle of registering and deregistering Nomad jobs
  (applications).
---

# nomad_job

Manages a job registered in Nomad.

This can be used to initialize your cluster with system jobs, common services,
and more. In day to day Nomad use it is common for developers to submit jobs to
Nomad directly, such as for general app deployment. In addition to these apps, a
Nomad cluster often runs core system services that are ideally setup during
infrastructure creation. This resource is ideal for the latter type of job, but
can be used to manage any job within Nomad.

## Example Usage

Registering a job from a jobspec file:

```hcl
resource "nomad_job" "app" {
  jobspec = file("${path.module}/jobspec.hcl")
}
```

Registering a job from an inline jobspec. This is less realistic but
is an example of how it is possible. More likely, the contents will
be paired with something such as the
[template_file](https://www.terraform.io/docs/providers/template/d/file.html)
resource to render parameterized jobspecs.

```hcl
resource "nomad_job" "app" {
  jobspec = <<EOT
job "foo" {
  datacenters = ["dc1"]
  type        = "service"
  group "foo" {
    task "foo" {
      driver = "raw_exec"
      config {
        command = "/bin/sleep"
        args    = ["1"]
      }

      resources {
        cpu    = 20
        memory = 10
      }

      logs {
        max_files     = 3
        max_file_size = 10
      }
    }
  }
}
EOT
}
```

## JSON jobspec

The input jobspec can also be provided as JSON instead of HCL by setting the
argument `json` to `true`:

```hcl
resource "nomad_job" "app" {
  jobspec = file("${path.module}/jobspec.json")
  json    = true
}
```

When using JSON, the input jobspec should have the same structured used by the
[Nomad API](https://www.nomadproject.io/api-docs/json-jobs/). The Nomad CLI
can translate HCL jobs to JSON:

```shellsession
nomad job run -output my-job.nomad > my-job.json
```

Or you can also use the [`/v1/jobs/parse`](https://www.nomadproject.io/api-docs/jobs/#parse-job)
API endpoint.

### Variables

~> **Warning:** mixing Nomad HCL2 variables and Terraform values may result in
  plan failures. It's best to avoid using variables in job files and instead
  use [string templates][tf_docs_string_template] or render a file with the
  [`templatefile`][tf_docs_templatefile] Terraform function.

HCL2 variables can be passed from Terraform to the jobspec parser through the
`vars` attribute inside the `hcl2` block. The variable must also be declared
inside the jobspec as an [input variable](https://www.nomadproject.io/docs/job-specification/hcl2/variables#declaring-an-input-variable).

Due to the way resource attributes are stored in the Terraform state, the
values must be provided as strings.

```hcl
resource "nomad_job" "app" {
  hcl2 {
    vars = {
      "restart_attempts" = "5",
      "datacenters"      = "[\"dc1\", \"dc2\"]",
    }
  }

  jobspec = <<EOT
variable "datacenters" {
  type = list(string)
}

variable "restart_attempts" {
  type = number
}

job "foo-hcl2" {
  datacenters = var.datacenters

  restart {
    attempts = var.restart_attempts
    ...
  }
  ...
}
```

Variables must have known-values at plan time. This means that you will not be
able to reference values from resources that don't exist in the Terraform state
yet. Instead, use [string templates][tf_docs_string_template] or the
[`templatefile`][tf_docs_templatefile] Terraform function to provide a fully
rendered jobspec.

```hcl
resource "random_pet" "random_dc" {}

# This resource will fail to plan because random_pet.random_dc.id is unknown.
resource "nomad_job" "job_with_hcl2" {
  jobspec = <<EOT
variable "datacenter" {
  type = string
}

job "example" {
  datacenters = [var.datacenter]
  ...
}
EOT

  hcl2 {
    vars = {
      datacenter = random_pet.random_dc.id
    }
  }
}

# This will work since Terraform will provide a fully rendered jobspec once it
# knows the value of random_pet.random_dc.id.
resource "nomad_job" "job_with_hcl2" {
  jobspec = <<EOT
job "example" {
  datacenters = ["${random_pet.random_dc.id}"]
  ...
}
EOT
}
```

### Filesystem functions

Please note that [filesystem functions](https://www.nomadproject.io/docs/job-specification/hcl2/functions/file/abspath)
will create an implicit dependency in your Terraform configuration. For
example, Terraform will not be able to detect changes to files loaded using the
[`file`](https://www.nomadproject.io/docs/job-specification/hcl2/functions/file/file)
function inside a jobspec.

To avoid confusion, these functions are disabled by default. To enable them
set `allow_fs` to `true`:

```hcl
resource "nomad_job" "app" {
  jobspec = file("${path.module}/jobspec.hcl")

  hcl2 {
    allow_fs = true
  }
}
```

If you do need to track changes to external files, you can use the
[`local_file`](https://registry.terraform.io/providers/hashicorp/local/latest/docs/data-sources/file)
data source and the [`templatefile`][tf_docs_templatefile] function to load the
local file into Terraform and then render its content into the jobspec:

```hcl
# main.tf

data "local_file" "index_html" {
  filename = "${path.module}/index.html"
}

resource "nomad_job" "nginx" {
  jobspec = templatefile("${path.module}/nginx.nomad.tpl", {
    index_html = data.local_file.index_html.content
  })
}
```

```hcl
# nginx.nomad.tpl

job "nginx" {
...
      template {
        data        = <<EOF
${index_html}
EOF
        destination = "local/www/index.html"
      }
...
}
```

## Tracking Jobspec Changes

The Nomad API allows [submitting the raw jobspec when registering and updating
jobs](https://developer.hashicorp.com/nomad/api-docs/jobs#submission). If
available, the job submission source is used to detect changes to the `jobspec`
and `hcl2.vars` arguments.

## Argument Reference

The following arguments are supported:

- `jobspec` `(string: <required>)` - The contents of the jobspec to register.

- `deregister_on_destroy` `(boolean: true)` - Determines if the job will be
  deregistered when this resource is destroyed in Terraform.

- `purge_on_destroy` `(boolean: false)` - Set this to true if you want the job to
  be purged when the resource is destroyed.

- `deregister_on_id_change` `(boolean: true)` - Determines if the job will be
  deregistered if the ID of the job in the jobspec changes.

- `rerun_if_dead` `(boolean: false)` - Set this to true to force the job to run
  again if its status is `dead`.

- `detach` `(boolean: true)` - If true, the provider will return immediately
  after creating or updating, instead of monitoring.

- `policy_override` `(boolean: false)` - Determines if the job will override any
  soft-mandatory Sentinel policies and register even if they fail.

- `preserve_counts` `(boolean: false)` - If true, preserves the current task
  group counts already stored in Nomad during job registration instead of
  applying the counts from the submitted jobspec.

- `json` `(boolean: false)` - Set this to `true` if your jobspec is structured with
  JSON instead of the default HCL.

- `hcl2` `(block: optional)` - Options for the HCL2 jobspec parser.
  - `allow_fs` `(boolean: false)` - Set this to `true` to be able to use
    [HCL2 filesystem functions](#filesystem-functions)

## Attributes Reference

The following attributes are exported:

- `name` `(string)` - The job name, as derived from the jobspec.
- `namespace` `(string)` - The namespace of the job, as derived from the jobspec.
- `type` `(string)` - The type of the job, as derived from the jobspec.
- `region` `(string)` - The target region for the job.
- `datacenters` `(set of strings)` - The target datacenters for the job.
- `modify_index` `(string)` - Integer that increments for each change. Used to detect any changes between plan and apply.
- `status` `(string)` - The current status of the job.
- `status_description` `(string)` - Additional status information returned by Nomad.
- `version` `(integer)` - The current job version.
- `submit_time` `(integer)` - The Unix timestamp when the job was submitted.
- `create_index` `(integer)` - The job creation index.
- `stop` `(boolean)` - Whether the job is stopped.
- `priority` `(integer)` - The job priority for scheduling and resource access.
- `parent_id` `(string)` - The parent job ID, if applicable.
- `stable` `(boolean)` - Whether the job is stable.
- `all_at_once` `(boolean)` - Whether the scheduler can make partial placements on oversubscribed nodes.
- `deployment_id` `(string)` - If `detach = false`, the deployment associated with the last create or update, if one exists.
- `deployment_status` `(string)` - If `detach = false`, the status for the deployment associated with the last create or update, if one exists.
- `allocation_ids` `(list of strings)` - Allocation IDs associated with the job when `read_allocation_ids = true`.
- `constraints` `(list of maps)` - Job constraints.
  - `ltarget` `(string)` - Attribute being constrained.
  - `rtarget` `(string)` - Constraint value.
  - `operand` `(string)` - Operator used to compare the attribute to the constraint.
- `update_strategy` `(list of maps)` - Job-level update strategy returned by Nomad.
  - `stagger` `(string)` - Delay between each set of `max_parallel` updates when updating system jobs.
  - `max_parallel` `(integer)` - Number of allocations within a task group that can be destructively updated at the same time. Setting `0` forces updates instead of deployments.
  - `health_check` `(string)` - Mechanism used to determine allocation health: `checks`, `task_states`, or `manual`.
  - `min_healthy_time` `(string)` - Minimum time the allocation must be in the healthy state before further updates can proceed.
  - `healthy_deadline` `(string)` - Deadline by which the allocation must become healthy before it is marked unhealthy.
  - `auto_revert` `(boolean)` - Whether the job should automatically revert to the last stable job on deployment failure.
  - `canary` `(integer)` - Number of canary allocations created before destructive updates continue.
- `periodic_config` `(list of maps)` - The job's periodic configuration.
  - `enabled` `(boolean)` - Whether the periodic job is enabled. When disabled, scheduled runs and force launches are prevented.
  - `spec` `(string)` - Cron expression configuring the interval at which the job is launched.
  - `spec_type` `(string)` - Type of periodic specification, such as `cron`.
  - `prohibit_overlap` `(boolean)` - Whether this job should wait until previous instances of the same job have completed before launching again.
  - `timezone` `(string)` - Time zone used to evaluate the next launch interval.
- `task_groups` `(list of maps)` - A list of the job's task groups.
  - `name` `(string)` - Task group name.
  - `count` `(integer)` - Task group count.
  - `update_strategy` `(list of maps)` - Effective update strategy for the task group.
    - `stagger` `(string)` - Delay between each set of `max_parallel` updates when updating system jobs.
    - `max_parallel` `(integer)` - Number of allocations within a task group that can be destructively updated at the same time. Setting `0` forces updates instead of deployments.
    - `health_check` `(string)` - Mechanism used to determine allocation health: `checks`, `task_states`, or `manual`.
    - `min_healthy_time` `(string)` - Minimum time the allocation must be in the healthy state before further updates can proceed.
    - `healthy_deadline` `(string)` - Deadline by which the allocation must become healthy before it is marked unhealthy.
    - `auto_revert` `(boolean)` - Whether the job should automatically revert to the last stable job on deployment failure.
    - `canary` `(integer)` - Number of canary allocations created before destructive updates continue.
  - `task` `(list of maps)` - Tasks in the task group.
    - `name` `(string)` - Task name.
    - `driver` `(string)` - Task driver.
    - `meta` `(map of strings)` - Task metadata.
    - `volume_mounts` `(list of maps)` - Task volume mounts.
      - `volume` `(string)` - Volume name.
      - `destination` `(string)` - Destination path inside the task.
      - `read_only` `(boolean)` - Whether the volume mount is read-only.
  - `volumes` `(list of maps)` - Volume requests for the task group.
    - `name` `(string)` - Volume name.
    - `type` `(string)` - Volume type.
    - `read_only` `(boolean)` - Whether the volume is read-only.
    - `source` `(string)` - Volume source.
  - `meta` `(map of strings)` - Task group metadata.
### Timeouts

`nomad_job` provides the following [`Timeouts`][tf_docs_timeouts] configuration
options when `detach` is set to `false`:

- `create` `(string: "5m")` - Timeout when registering a new job.
- `update` `(string: "5m")` - Timeout when updating an existing job.

## Importing Jobs

Jobs are imported using the pattern `<job ID>@<namespace>`.

```console
$ terraform import nomad_job.example example@my-namespace
nomad_job.example: Importing from ID "example@my-namespace"...
nomad_job.example: Import prepared!
  Prepared nomad_job for import
nomad_job.example: Refreshing state... [id=example@my-namespace]

Import successful!

The resources that were imported are shown above. These resources are now in
your Terraform state and will henceforth be managed by Terraform.
```

[tf_docs_timeouts]: https://www.terraform.io/docs/configuration/blocks/resources/syntax.html#operation-timeouts
[tf_docs_templatefile]: https://www.terraform.io/docs/configuration/functions/templatefile.html
[tf_docs_string_template]: https://www.terraform.io/language/expressions/strings#string-templates
