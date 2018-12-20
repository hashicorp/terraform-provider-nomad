---
layout: "nomad"
page_title: "Nomad: nomad_namespaces"
sidebar_current: "docs-nomad-datasource-namespaces"
description: |-
  Retrieve a list of namespaces available in Nomad.
---

# nomad_namespaces

Retrieve a list of namespaces available in Nomad.

## Example Usage

```hcl
data "nomad_namespaces" "namespaces" {
}

resource "nomad_acl_policy" "namespace" {
  count = "${length(data.nomad_namespaces.namespaces.namespaces)}"
  name = "namespace-${data.nomad_namespaces.namespaces[count.index]}"
  description = "Write to the namespace ${data.nomad_namespaces.namespaces[count.index]}"
  rules_hcl = <<EOT
namespace "${data.nomad_namespaces.namespaces[count.index]}" {
  policy = "write"
}
EOT
}

```

## Attribute Reference

The following attributes are exported:

- `namespaces` `(list of strings)` - a list of namespaces available in the cluster.
