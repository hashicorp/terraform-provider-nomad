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
(`pem_keys`), which may be more convenient for use.

## Example Usage

```hcl
data "nomad_jwks" "example" {}
```

## Attribute Reference

The following attributes are exported:
* `keys`: `list of maps` a list of JWK keys in structured format: see [RFC7517](https://datatracker.ietf.org/doc/html/rfc7517) for the
JWK field meanings.
  * `key_use` `(string)` - JWK field `use`
  * `key_type` `(string)` - JWK field `kty` (e.g. `RSA`, `OKP`)
  * `key_id` `(string)` - JWK field `kid`
  * `algorithm` `(string)` - JWK field `alg` (e.g. `RS256`, `EdDSA`)
  * `modulus` `(string)` - JWK field `n` (RSA only)
  * `exponent` `(string)` - JWK field `e` (RSA only)
  * `curve` `(string)` - JWK field `crv` (EdDSA only, e.g. `Ed25519`)
  * `x` `(string)` - JWK field `x` (EdDSA only, the public key)
* `pem_keys`: `list of strings` a list JWK keys rendered as PEM-encoded X.509 keys
