---
layout: "nomad"
page_title: "Nomad: nomad_deployments"
sidebar_current: "docs-nomad-datasource-deployments"
description: |-
  Retrieve a list of deployments.
---

# nomad_regions

Retrieve a list of deployments in Nomad.

## Example Usage

```hcl
data "nomad_deployments" "example" {}
```

## Attribute Reference

The following attributes are exported:

- `deployments`: a list of deployments in the cluster.
