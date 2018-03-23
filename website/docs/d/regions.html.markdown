---
layout: "nomad"
page_title: "Nomad: nomad_regions"
sidebar_current: "docs-nomad-datasource-regions"
description: |-
  Retrieve a list of regions available in Nomad.
---

# nomad_regions

Retrieve a list of regions available in Nomad.

## Example Usage

```hcl
data "nomad_regions" "regions" {
}

data "template_file" "jobs" {
  count = "${length(data.nomad_regions.regions.regions)}"
  template = <<EOT
job "foo" {
  datacenters = ["dc1"]
  type = "service"
  region = "$$region"
  # ... rest of your job here
}
EOT
  vars {
    region = "${data.nomad_regions.regions[count.index]}"
  }
}

resource "nomad_job" "app" {
  count = "${length(data.nomad_regions.regions.regions)}"
  jobspec = "${data.template_file.jobs[count.index].rendered}"
}
```

## Attribute Reference

The following attributes are exported:

- `regions` `(list of strings)` - a list of regions available in the cluster.
