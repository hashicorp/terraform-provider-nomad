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

## HCL2 jobspec

The input jobspec can also be provided in the [HCL2 format](https://www.nomadproject.io/docs/job-specification/hcl2)
by enabling `hcl2` parsing:

```hcl
resource "nomad_job" "app" {
  jobspec = file("${path.module}/jobspec.hcl")

  hcl2 {
    enabled = true
  }
}
```

~> **Note:** This option should be enabled whenever possible as it includes
   support for newer jobspec entries.

### Variables

HCL2 variables can be passed from Terraform to the jobspec parser through the
`vars` attribute inside the `hcl2` block. The variable must also be declared
inside the jobspec as an [input variable](https://www.nomadproject.io/docs/job-specification/hcl2/variables#declaring-an-input-variable).

Due to the way resource attributes are stored in the Terraform state, the
values must be provided as strings.

```hcl
resource "nomad_job" "app" {
  hcl2 {
    enabled  = true
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
    enabled  = true
    allow_fs = true
  }
}
```

If you do need to track changes to external files, you can use the
[`local_file`](https://registry.terraform.io/providers/hashicorp/local/latest/docs/data-sources/file)
data source and the
[`templatefile`](https://www.terraform.io/docs/configuration/functions/templatefile.html)
function to load the local file into Terraform and then render its content
into the jobspec:

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

## Argument Reference

The following arguments are supported:

- `jobspec` `(string: <required>)` - The contents of the jobspec to register.

- `deregister_on_destroy` `(boolean: true)` - Determines if the job will be
  deregistered when this resource is destroyed in Terraform.

- `purge_on_destroy` `(boolean: false)` - Set this to true if you want the job to
  be purged when the resource is destroyed.

- `deregister_on_id_change` `(boolean: true)` - Determines if the job will be
  deregistered if the ID of the job in the jobspec changes.

- `detach` `(boolean: true)` - If true, the provider will return immediately
  after creating or updating, instead of monitoring.

- `policy_override` `(boolean: false)` - Determines if the job will override any
  soft-mandatory Sentinel policies and register even if they fail.

- `json` `(boolean: false)` - Set this to `true` if your jobspec is structured with
  JSON instead of the default HCL.

- `hcl2` `(block: optional)` - Options for the HCL2 jobspec parser.
  - `enabled` `(boolean: false)` - Set this to `true` if your jobspec uses the HCL2
    format instead of the default HCL.
  - `allow_fs` `(boolean: false)` - Set this to `true` to be able to use
    [HCL2 filesystem functions](#filesystem-functions)

### Timeouts

`nomad_job` provides the following [`Timeouts`][tf_docs_timeouts] configuration
options when [`detach`](#detach) is set to `false`:

- `create` `(string: "5m")` - Timeout when registering a new job.
- `update` `(string: "5m")` - Timeout when updating an existing job.

[tf_docs_timeouts]: https://www.terraform.io/docs/configuration/blocks/resources/syntax.html#operation-timeouts
