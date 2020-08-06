---
layout: "nomad"
page_title: "Nomad: nomad_job_parser"
sidebar_current: "docs-nomad-datasource-job-parser"
description: |-
  Parse a HCL jobspec and produce the equivalent JSON encoded job.
---

# nomad_acl_policy

Parse a HCL jobspec and produce the equivalent JSON encoded job.

## Example Usage

```hcl
data "nomad_job_parser" "my_job" {
  hcl = file("${path.module}/jobpec.hcl")
  canonicalize = false
}
```

## Attribute Reference

The following attributes are exported:

- `hcl` `(string)` - the HCL definition of the job.
- `canonicalize` `(boolean: true)` - flag to enable setting any unset fields to their default values.
- `json` `(string)` - the parsed job as JSON string.
