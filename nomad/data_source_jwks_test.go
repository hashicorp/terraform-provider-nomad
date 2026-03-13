// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testAccNomadJWKSConfig = `data "nomad_jwks" "test" {}`

func TestAccDataSourceNomadJWKS_Basic(t *testing.T) {
	dataSourceName := "data.nomad_jwks.test"
	expectedKeyCount := "1"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNomadJWKSConfig,
			},
			{
				Config: testAccNomadJWKSConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "keys.#", expectedKeyCount),
					resource.TestMatchResourceAttr(dataSourceName, "keys.0.key_type", regexp.MustCompile("RSA")),
					resource.TestCheckResourceAttr(dataSourceName, "pem_keys.#", expectedKeyCount),
					resource.TestCheckResourceAttrWith(dataSourceName, "pem_keys.0", validateKeyPEM),
				),
			},
		},
	})
}

func validateKeyPEM(keyPEM string) error {
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return fmt.Errorf("failed to parse key PEM")
	}
	_, err := x509.ParsePKIXPublicKey(block.Bytes)
	return err
}

func TestKeyToPem(t *testing.T) {
	tests := []struct {
		name       string
		key        Key
		wantErr    string
		assertFunc func(*testing.T, string)
	}{
		{
			name: "okp ed25519",
			key: Key{
				KeyType: "OKP",
				Curve:   "Ed25519",
				X:       "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8",
			},
			assertFunc: func(t *testing.T, pemKey string) {
				t.Helper()

				block, _ := pem.Decode([]byte(pemKey))
				if block == nil {
					t.Fatal("expected PEM block")
				}

				publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
				if err != nil {
					t.Fatalf("parse public key: %v", err)
				}

				if _, ok := publicKey.(ed25519.PublicKey); !ok {
					t.Fatalf("expected Ed25519 public key, got %T", publicKey)
				}
			},
		},
		{
			name: "okp unsupported curve",
			key: Key{
				KeyType: "OKP",
				Curve:   "X25519",
				X:       "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8",
			},
			wantErr: "Unsupported curve: X25519",
		},
		{
			name: "okp invalid key size",
			key: Key{
				KeyType: "OKP",
				Curve:   "Ed25519",
				X:       "AQID",
			},
			wantErr: "Invalid Ed25519 key size",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pemKey, err := keyToPem(test.key)
			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", test.wantErr)
				}
				if !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("expected error containing %q, got %q", test.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if test.assertFunc != nil {
				test.assertFunc(t, pemKey)
			}
		})
	}
}
