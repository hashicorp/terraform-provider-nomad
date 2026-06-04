---
layout: "nomad"
page_title: "Nomad: nomad_services"
sidebar_current: "docs-nomad-datasource-services"
description: |-
  Retrieve the list of all registered Nomad services.
---

# nomad_services

Retrieve the list of all registered Nomad services.

## Example Usage

```hcl
data "nomad_services" "all" {}

output "service_names" {
  value = [for s in data.nomad_services.all.services : s.name]
}
```

### With namespace filter

```hcl
data "nomad_services" "production" {
  namespace = "production"
}
```

### All namespaces

```hcl
data "nomad_services" "everything" {
  namespace = "*"
}
```

## Argument Reference

The following arguments are supported:

- `namespace` `(string: <optional>)` - The namespace to filter services. Use
  `"*"` to list services across all namespaces.

## Attribute Reference

The following attributes are exported:

- `id` `(string)` - A static ID for this data source.
- `services` `(list of services)` - A list of services. Each service has the
  following attributes:
  - `namespace` `(string)` - The namespace in which the service is registered.
  - `name` `(string)` - The name of the service.
  - `tags` `(list of string)` - The tags associated with the service.
