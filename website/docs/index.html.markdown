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

- `vault_token` `(string: "")` - A Vault token used when [submitting the job](https://www.nomadproject.io/docs/job-specification/job#vault_token).
  This can also be specified as the `VAULT_TOKEN` environment variable or using a
  Vault token helper (see [Vault's documentation](https://www.vaultproject.io/docs/commands/token-helper.html)
  for more details). See [below](#multiple-vault-tokens) for strategies when 
  multiple Vault tokens are required.

- `secret_id` `(string: "")` - The Secret ID of an ACL token to make requests with,
  for ACL-enabled clusters. This can also be specified via the `NOMAD_TOKEN`
  environment variable.

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

## Multiple Vault Tokens

Nomad supports passing a Vault token during job registration; this token is used
only to verify that the submitter has permissions to access the Vault policies
used in the Nomad job. When running the Nomad CLI, this token can be provided in
a number of ways:
* in the job spec using the [`vault_token`](https://www.nomadproject.io/docs/job-specification/job#vault_token) parameter
* using the [`-vault-token`](https://www.nomadproject.io/docs/commands/job/run#vault-token) command line flag
* setting the `VAULT_TOKEN` environment variable.

When using the Nomad Provider to register Nomad jobs, the options are similar:
* the token can be placed in the job spec in the [`nomad_job`](./resources/job) resource
* the token can be [configured](#vault_token) on the Nomad Provider.
* the token can be set in the `VAULT_TOKEN` environment variable when running `terraform apply`

There are two problems that arise. The first is that we likely want to avoid putting
Vault tokens into Terraform files where they may be inadvertently distributed. The second
is that Nomad jobs might require different Vault tokens, each with access to a
different set of policies. In this case, there are a few different strategies for
managing the tokens and ensuring that the correct token is used for a particular
job.

One approach is to provider aliases, creating a Nomad Provider for each Vault token:
```hcl
provider "nomad" {
  alias = "a"
  vault_token = var.vault_token_a
}

provider "nomad" {
  alias = "b"
  vault_token = var.vault_token_b
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
                   -var vault_token_b=s.koqvVqdAkG8yt7irxDdmIQiC
```

The downside here is that it requires creating multiple Nomad provider aliases
and specifying the desired alias for every job resource. Another approach is inject
the Vault token into the jobspec using `templatefile`:
```hcl
resource "nomad_job" "job_a" {
  jobspec = templatefile(
    "${path.module}/job_a.hcl.tmpl",
    { vault_token = "s.lraLq3axH9mkbdVRkWS6H06Q" }
  )
}

resource "nomad_job" "job_b" {
  jobspec = templatefile(
    "${path.module}/job_b.hcl.tmpl",
    { vault_token = "s.koqvVqdAkG8yt7irxDdmIQiC" }
  )
}
```

This approach has the benefit that only jobs requiring a Vault token need to be modified.