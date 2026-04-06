---
layout: "nomad"
page_title: "Nomad: nomad_node_intro_token"
sidebar_current: "docs-nomad-ephemeral-node-intro-token"
description: |-
  Creates a short-lived Nomad client introduction token as an ephemeral resource.
---

# nomad_node_intro_token

Creates a short-lived Nomad client introduction token using an `ephemeral` block.

This ephemeral resource is useful when you need a temporary JWT for Nomad client
bootstrap or enrollment flows without persisting that token in Terraform state.

## Example Usage

```hcl
ephemeral "nomad_node_intro_token" "client" {
  node_name = "bootstrap-client"
  node_pool = "default"
  ttl       = "15m"
}

# Reference the signed JWT as:
# ephemeral.nomad_node_intro_token.client.jwt
```

## Argument Reference

The following arguments are supported:

- `node_name` `(string: "")` - The node name to scope the introduction token to.

- `node_pool` `(string: "")` - The node pool to scope the introduction token to.

- `ttl` `(string: "")` - The requested token TTL as a duration such as `"5m"`
  or `"1h"`.

## Attributes Reference

In addition to the above arguments, the following attribute is exported:

- `jwt` `(string, sensitive)` - The signed JWT node introduction token returned
  by Nomad.

## Notes

- This ephemeral resource requires a configured `nomad` provider with access to
  the ACL identity API.

- `ttl` must be a valid Go duration string accepted by Nomad, such as `"30s"`,
  `"5m"`, or `"1h"`.

- If `node_name` or `node_pool` are omitted, the token is created without those
  optional request fields.
