---
layout: "nomad"
page_title: "Nomad: nomad_job_parser"
sidebar_current: "docs-nomad-datasource-job-parser"
description: |-
  Parse a HCL jobspec and produce the equivalent JSON encoded job.
---

# nomad_job_parser

Parse a HCL jobspec and produce the equivalent JSON encoded job.

## Example Usage

```hcl
data "nomad_job_parser" "my_job" {
  hcl = file("${path.module}/jobspec.hcl")
  canonicalize = false
}
```

### With Variables

```hcl
data "nomad_job_parser" "my_job" {
  hcl = file("${path.module}/jobspec.hcl")

  variables = <<EOT
datacenter = "dc1"
image      = "nginx:latest"
EOT
}
```

## Argument Reference

The following arguments are supported:

- `hcl` `(string: <required>)` - The HCL definition of the job.
- `canonicalize` `(boolean: false)` - Flag to enable setting any unset fields to their default values.
- `variables` `(string: "")` - HCL2 variables to pass to the job parser. Interpreted as the content of a variables file.

## Attribute Reference

The following attributes are exported:

- `json` `(string)` - The parsed job as JSON string.
