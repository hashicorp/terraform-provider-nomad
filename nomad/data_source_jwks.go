// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"
	"strings"

	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceJWKS() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceJWKSRead,
		Schema: map[string]*schema.Schema{
			"keys": {
				Description: "JSON Web Key Set (JWKS) public keys for validating workload identity JWTs",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key_use": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"key_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"key_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"algorithm": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"modulus": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"exponent": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"pem_keys": {
				Description: "JWKS as a list of PEM keys",
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
			},
		},
	}
}

type Key struct {
	KeyUse    string `json:"use"`
	KeyType   string `json:"kty"`
	KeyId     string `json:"kid"`
	Algorithm string `json:"alg"`
	Modulus   string `json:"n"`
	Exponent  string `json:"e"`
}

func dataSourceJWKSRead(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client
	operator := client.Raw()
	queryOpts := &api.QueryOptions{}

	jwks := struct {
		Keys []Key `json:"keys"`
	}{}

	log.Printf("[DEBUG] Reading JWKS from Nomad")
	_, err := operator.Query("/.well-known/jwks.json", &jwks, queryOpts)

	if err != nil {
		return fmt.Errorf("error reading JWKS from Nomad: %s", err)
	}

	if len(jwks.Keys) == 0 {
		return fmt.Errorf("no keys found")
	}

	d.SetId(id.UniqueId())
	if err := d.Set("keys", fromKeys(jwks.Keys)); err != nil {
		return fmt.Errorf("error setting JWKS: %#v", err)
	}

	pemKeys := make([]string, 0, len(jwks.Keys))

	for _, key := range jwks.Keys {
		pemKey, err := keyToPem(key)
		if err != nil {
			return fmt.Errorf("Could not encode JWK as PEM: %s", err)
		}
		pemKeys = append(pemKeys, pemKey)
	}

	if err := d.Set("pem_keys", pemKeys); err != nil {
		return fmt.Errorf("error setting JWKS pemKeys: %s", err)
	}

	return nil
}

func keyToPem(key Key) (string, error) {

	// FIXME does Nomad always use RSA keys for JWKS?
	if key.KeyType != "RSA" {
		return "", fmt.Errorf("Key type not supported: %s", key.Algorithm)
	}
	modulus, err := base64.RawURLEncoding.DecodeString(key.Modulus)

	if err != nil {
		return "", fmt.Errorf("Could not decode modulus as base64 from JWK: %s", err)
	}

	exponent, err := base64.RawURLEncoding.DecodeString(key.Exponent)

	if err != nil {
		return "", fmt.Errorf("Could not decode exponent as base64 from JWK: %s", err)
	}

	modulusInt := new(big.Int)
	modulusInt.SetBytes(modulus)

	exponentInt := new(big.Int)
	exponentInt.SetBytes(exponent)

	publicKey := rsa.PublicKey{N: modulusInt, E: int(exponentInt.Uint64())}

	x509Cert, err := x509.MarshalPKIXPublicKey(&publicKey)

	if err != nil {
		return "", fmt.Errorf("Could not marshal JWK public key to X509 PKIX: %s", err)
	}

	x509CertEncoded := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: x509Cert,
		})

	// Vault renders the keys without a trailing newline; strip this out to avoid unneccesary updates
	// FIXME this might be more appropriate to handle as a `chomp` on the terraform side when
	// interacting with Vault. If so, document this.
	return strings.TrimSpace(string(x509CertEncoded)), nil
}

func fromKeys(keys []Key) []interface{} {
	output := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		p := map[string]interface{}{
			"key_use":   key.KeyUse,
			"key_type":  key.KeyType,
			"key_id":    key.KeyId,
			"algorithm": key.Algorithm,
			"modulus":   key.Modulus,
			"exponent":  key.Exponent,
		}
		output = append(output, p)
	}
	return output
}
