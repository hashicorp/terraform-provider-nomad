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

- `load_namespace_env_var` `(bool: false)` - If true, the `NOMAD_NAMESPACE`
  environment variable will be loaded into the provider configuration.

    ~> **Warning:** This value should not be set (or set to `false`) when
      running Terraform in environments where it runs within a Nomad
      allocation, such as in Terraform Cloud.

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

- `headers` - (Optional) A configuration block, described below, that provides headers
  to be sent along with all requests to Nomad.  This block can be specified
  multiple times.

- `vault_token` `(string: "")` - A Vault token used when [submitting the job](https://www.nomadproject.io/docs/job-specification/job#vault_token).
  This can also be specified as the `VAULT_TOKEN` environment variable or using a
  Vault token helper (see [Vault's documentation](https://www.vaultproject.io/docs/commands/token-helper.html)
  for more details). See [below](#configuring-multiple-tokens) for strategies when
  multiple Vault tokens are required.

- `consul_token` `(string: "")` - A Consul token used when [submitting the job](https://www.nomadproject.io/docs/job-specification/job#consul_token).
  This can also be specified as the `CONSUL_HTTP_TOKEN` environment variable.
  See [below](#configuring-multiple-tokens) for strategies when multiple Consul tokens are required.

- `secret_id` `(string: "")` - The Secret ID of an ACL token to make requests with,
  for ACL-enabled clusters. This can also be specified via the `NOMAD_TOKEN`
  environment variable.

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

## Configuring Multiple Tokens

Nomad supports passing a Vault or Consul token during job registration; this token is used
only to verify that the submitter has permissions to access the Vault or Consul SI policies
used in the Nomad job. When running the Nomad CLI, this token can be provided in
a number of ways:

For Vault:
- in the job spec using the [`vault_token`](https://www.nomadproject.io/docs/job-specification/job#vault_token) parameter
- using the [`-vault-token`](https://www.nomadproject.io/docs/commands/job/run#vault-token) command line flag
- setting the `VAULT_TOKEN` environment variable.

For Consul:
- in the job spec using the [`consul_token`](https://www.nomadproject.io/docs/job-specification/job#consul_token) parameter
- using the [`-consul-token`](https://www.nomadproject.io/docs/commands/job/run#consul-token) command line flag
- setting the `CONSUL_HTTP_TOKEN` environment variable

When using the Nomad Provider to register Nomad jobs, the options are similar:
- the tokens can be placed in the job spec in the [`nomad_job`](./resources/job) resource
- the tokens can be [configured](#argument-reference) on the Nomad Provider.
- the tokens can be set in the environment variables when running `terraform apply`

There are two problems that arise. The first is that we likely want to avoid putting
these tokens into Terraform files where they may be inadvertently distributed. The second
is that different Nomad jobs might require different tokens, each with access to a
different set of policies. In this case, there are a few different strategies for
managing the tokens and ensuring that the correct token is used for a particular
job.

One approach is to use provider aliases, creating a Nomad Provider for each token:
```hcl
provider "nomad" {
  alias = "a"
  vault_token = var.vault_token_a
  consul_token = var.consul_token_a
}

provider "nomad" {
  alias = "b"
  vault_token = var.vault_token_b
  consul_token = var.consul_token_b
}

resource "nomad_job" "job_a" {
  provider = nomad.a
  jobspec = file("${path.module}/job_a.hcl")
}

resource "nomad_job" "job_b" {
  provider = nomad.b
  jobspec = file("${path.module}/job_b.hcl")
}
```

The tokens can be passed from the command lines as variables:
```bash
$ terraform apply  -var vault_token_a=s.lraLq3axH9mkbdVRkWS6H06Q  \
                   -var vault_token_b=s.koqvVqdAkG8yt7irxDdmIQiC  \
                   -var consul_token_a=fc0ff975-e845-4140-804c-c348e9414ff8 \
                   -var consul_token_b=aed18d86-dd9d-4029-8a7a-906f6bba640a
```

The downside here is that it requires creating multiple Nomad provider aliases
and specifying the desired alias for every job resource. Another approach is to inject
the tokens into the jobspec using `templatefile`:
```hcl
resource "nomad_job" "job_a" {
  jobspec = templatefile(
    "${path.module}/job_a.hcl.tmpl",
    {
      vault_token = "s.lraLq3axH9mkbdVRkWS6H06Q"
      consul_token = "fc0ff975-e845-4140-804c-c348e9414ff8"
    }
  )
}

resource "nomad_job" "job_b" {
  jobspec = templatefile(
    "${path.module}/job_b.hcl.tmpl",
    {
      vault_token = "s.koqvVqdAkG8yt7irxDdmIQiC"
      consul_token = "aed18d86-dd9d-4029-8a7a-906f6bba640a"
    }
  )
}
```

This approach has the benefit that only jobs requiring a token need to be modified.
