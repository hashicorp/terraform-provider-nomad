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
  address = "http://nomad.mycompany.com:4646"
  region  = "us-east-2"
}

# Register a job
resource "nomad_job" "monitoring" {
  jobspec = file("${path.module}/jobspec.hcl")
}
```

## Argument Reference

The following arguments are supported:

- `address` `(string: "http://127.0.0.1:4646")` - The HTTP(S) API address of the
  Nomad agent. This must include the leading protocol (e.g. `https://`). This
  can also be specified as the `NOMAD_ADDR` environment variable.

- `region` `(string: "")` - The Nomad region to target. This can also be
  specified as the `NOMAD_REGION` environment variable.

- `http_auth` `(string: "")` - HTTP Basic Authentication credentials to be used
  when communicating with Nomad, in the format of either `user` or `user:pass`.
  This can also be specified using the `NOMAD_HTTP_AUTH` environment variable.

- `ca_file` `(string: "")` - A local file path to a PEM-encoded certificate
  authority used to verify the remote agent's certificate. This can also be
  specified as the `NOMAD_CACERT` environment variable.

- `ca_pem` `(string: "")` - PEM-encoded certificate authority used to verify
  the remote agent's certificate.

- `cert_file` `(string: "")` - A local file path to a PEM-encoded certificate
  provided to the remote agent. If this is specified, `key_file` or `key_pem`
  is also required. This can also be specified as the `NOMAD_CLIENT_CERT`
  environment variable.

- `cert_pem` `(string: "")` - PEM-encoded certificate provided to the remote
  agent. If this is specified, `key_file` or `key_pem` is also required.

- `key_file` `(string: "")` - A local file path to a PEM-encoded private key.
  This is required if `cert_file` or `cert_pem` is specified. This can also be
  specified via the `NOMAD_CLIENT_KEY` environment variable.

- `key_pem` `(string: "")` - PEM-encoded private key. This is required if
  `cert_file` or `cert_pem` is specified.

- `skip_verify` `(boolean: false)` - Set this to true if you want to skip TLS verification on the client side.
  This can also be specified via the `NOMAD_SKIP_VERIFY` environment variable.

- `headers` - (Optional) A configuration block, described below, that provides headers
  to be sent along with all requests to Nomad.  This block can be specified
  multiple times.

- `secret_id` `(string: "")` - The Secret ID of an ACL token to make requests with,
  for ACL-enabled clusters. This can also be specified via the `NOMAD_TOKEN`
  environment variable.

- `ignore_env_vars` `(map[string]bool: {})` - A map of environment variables
  that are ignored by the provider when configuring the Nomad API client.
  Supported keys are: `NOMAD_NAMESPACE` and `NOMAD_REGION`. When using the
  provider within Terraform Cloud, the default value is set to
    ```
    {
      NOMAD_NAMESPACE: true,
      NOMAD_REGION:    true,
    }
    ```.
  Set these values to `false` if you need to load these environment variables.

- `auth_jwt` `(block)` - Authenticates to Nomad using a JWT authentication method, described below.
  This block can only be specified one time.

The `headers` configuration block accepts the following arguments:
* `name` - (Required) The name of the header.
* `value` - (Required) The value of the header.

An example using the `headers` configuration block with repeated blocks and
headers:
```hcl
provider "nomad" {
  headers {
    name = "Test-Header-1"
    value = "a"
  }
  headers {
    name = "Test-header-1"
    value = "b"
  }
  headers {
    name = "test-header-2"
    value = "c"
  }
}
```

The `auth_jwt` configuration block accepts the following arguments:
* `auth_method` - (Required) The name of the auth method.
* `login_token` - (Required) The value of the jwt token.

An example using the `auth_jwt` configuration block :
```hcl
provider "nomad" {
  auth_jwt {
    auth_method = "ci"
    login_token = var.jwt_token
  }
}
```

## Multi-Region Deployments

Each instance of the `nomad` provider is associated with a single region. Use
[`alias`](https://www.terraform.io/docs/configuration/providers.html#alias-multiple-provider-instances)
to specify multiple providers for multi-region clusters:

```hcl
provider "nomad" {
  address = "http://nomad.mycompany.com:4646"
  region  = "us"
  alias   = "us"
}

provider "nomad" {
  address = "http://nomad.mycompany.com:4646"
  region  = "eu"
  alias   = "eu"
}

resource "nomad_job" "nomad_us" {
  provider = nomad.us
  jobspec  = file("${path.module}/jobspec-us.nomad")
}

resource "nomad_job" "nomad_eu" {
  provider = nomad.eu
  jobspec  = file("${path.module}/jobspec-eu.nomad")
}
```
