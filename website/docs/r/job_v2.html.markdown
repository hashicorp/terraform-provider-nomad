---
layout: "nomad"
page_title: "Nomad: nomad_job_v2"
sidebar_current: "docs-nomad-resource-job-v2"
description: |-
  Manages the lifecycle of registering and deregistering Nomad jobs
  (applications).
---

# nomad_job_v2

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
resource "nomad_job_v2" "app" {
  job {
		name        = "example"
		datacenters = ["dc1"]

		group {
			name = "cache"

			task {
				name   = "redis"
				driver = "docker"
				config = jsonencode({
					image = "redis:3.2"
				})
			}
		}
	}
}
```

## Argument Reference

The following arguments are supported:

- `purge_on_delete` `(boolean: true)` - Whether the job will be purged when
deleted.

- `job` - The [Nomad Job specification](https://www.nomadproject.io/docs/job-specification)
to register.

## Import

`nomad_job_v2` can be imported:

```
$ terraform import nomad_job_v2.app example
```
