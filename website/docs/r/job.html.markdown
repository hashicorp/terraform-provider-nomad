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
  jobspec = file("${path.module}/jobpec.hcl")
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

- `json` `(boolean: false)` - Set this to true if your jobspec is structured with
  JSON instead of the default HCL.
