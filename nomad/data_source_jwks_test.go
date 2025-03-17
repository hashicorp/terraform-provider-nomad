// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"regexp"
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
