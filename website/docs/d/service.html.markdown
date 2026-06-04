---
layout: "nomad"
page_title: "Nomad: nomad_service"
sidebar_current: "docs-nomad-datasource-service"
description: |-
  Retrieve information about a specific Nomad service and its registrations.
---

# nomad_service

Retrieve information about a specific Nomad service and its registrations.

## Example Usage

```hcl
data "nomad_service" "example" {
  service_name = "my-webapp"
}

output "service_addresses" {
  value = [for r in data.nomad_service.example.registrations : "${r.address}:${r.port}"]
}
```

### With rendezvous hashing

```hcl
data "nomad_service" "example" {
  service_name = "my-webapp"
  choose       = "2|my-hash-key"
}
```

### With filter expression

```hcl
data "nomad_service" "example" {
  service_name = "my-webapp"
  filter       = "\"canary\" in Tags"
}
```

## Argument Reference

The following arguments are supported:

- `service_name` `(string: <required>)` - The name of the service to look up.
- `namespace` `(string: <optional>)` - The namespace of the service. Defaults
  to `"default"`.
- `filter` `(string: <optional>)` - Specifies the
  [expression][nomad_api_filter] used to filter the results.
- `choose` `(string: <optional>)` - Specifies the number of services to return
  and a hash key for rendezvous hashing. Must be in the form
  `"<number>|<key>"`.

## Attribute Reference

The following attributes are exported:

- `id` `(string)` - The service name used as the data source ID.
- `registrations` `(list of registrations)` - A list of service registrations
  matching the query. Each registration has the following attributes:
  - `id` `(string)` - The unique identifier of the service registration.
  - `address` `(string)` - The IP address of the service registration.
  - `port` `(int)` - The port number of the service registration.
  - `node_id` `(string)` - The ID of the node where the service is running.
  - `datacenter` `(string)` - The datacenter of the node where the service is running.
  - `alloc_id` `(string)` - The ID of the allocation providing this service.
  - `job_id` `(string)` - The ID of the job that registered this service.
  - `namespace` `(string)` - The namespace of the service registration.
  - `tags` `(list of string)` - The tags associated with this service registration.
  - `create_index` `(int)` - The Raft index when this registration was created.
  - `modify_index` `(int)` - The Raft index when this registration was last modified.

[nomad_api_filter]: https://developer.hashicorp.com/nomad/api-docs#filtering
