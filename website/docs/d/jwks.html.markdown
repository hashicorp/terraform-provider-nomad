---
layout: "nomad"
page_title: "Nomad: nomad_jwks"
sidebar_current: "docs-nomad-datasource-jwks"
description: |-
  Retrieve the cluster JWKS public keys.
---

# nomad_jwks

Retrieve the cluster JWKS public keys.

The keys are returned both as a list of maps (`keys`), and as a list of PEM-encoded strings
(`pem_keys`), which may be more convenient for use with other providers (eg
`terraform-provider-vault`).

## Example Usage

```hcl
data "nomad_jwks" "example" {}
```

## Attribute Reference

The following attributes are exported:
* `keys`: `list of maps` a list of JWK keys in structured format: see [RFC7517](https://datatracker.ietf.org/doc/html/rfc7517) for the
JWK field meanings.
  * `key_use` `(string)` - JWK field `use`
  * `key_type` `(string)` - JWK field `kty`
  * `key_id` `(string)` - JWK field `kid`
  * `algorithm` `(string)` - JWK field `alg`
  * `modulus` `(string)` - JWK field `n`
  * `exponent` `(string)` - JWK field `e`
* `pem_keys`: `list of strings` a list JWK keys rendered as PEM-encoded X.509 keys
