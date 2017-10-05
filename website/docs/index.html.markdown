---
layout: "nomad"
page_title: "Provider: Nomad"
sidebar_current: "docs-nomad-index"
description: |-
  HashiCorp Nomad is an application scheduler. The Nomad provider exposes
  resources for interacting with a HashiCorp Nomad cluster.
---

# Nomad Provider

[HashiCorp Nomad](https://www.nomadproject.io) is an application scheduler. The
Nomad provider exposes resources to interact with a Nomad cluster.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Nomad provider
provider "nomad" {
  address = "nomad.mycompany.com"
  region  = "us-east-2"
}

# Register a job
resource "nomad_job" "monitoring" {
  jobspec = "${file("${path.module}/jobspec.hcl")}"
}
```

## Argument Reference

The following arguments are supported:

- `address` `(string: "http://127.0.0.1:4646")` - The HTTP(S) API address of the
  Nomad agent. This must include the leading protocol (e.g. `https://`). This
  can also be specified as the `NOMAD_ADDR` environment variable.

- `region` `(string: "")` - The Nomad region to target. This can also be
  specified as the `NOMAD_REGION` environment variable.

- `ca_file` `(string: "")` - A local file path to a PEM-encoded certificate
  authority used to verify the remote agent's certificate. This can also be
  specified as the `NOMAD_CACERT` environment variable.

- `cert_file` `(string: "")` - A local file path to a PEM-encoded certificate
  provided to the remote agent. If this is specified, `key_file` is also
  required. This can also be specified as the `NOMAD_CLIENT_CERT` environment
  variable.

- `key_file` `(string: "")` - A local file path to a PEM-encoded private key.
  This is required if `cert_file` is specified. This can also be specified via
  the `NOMAD_CLIENT_KEY` environment variable.

- `vault_token` `(string: "")` - A vault token to be inserted in the job file.
  This can also be specified as the `VAULT_TOKEN` environment variable or in
  the `$HOME/.vault-token` file.