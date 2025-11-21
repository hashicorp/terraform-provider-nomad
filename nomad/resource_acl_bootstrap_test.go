// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceACLBootstrap_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceACLBootstrap_initialConfig(),
				Check:  testResourceACLBootstrap_initialCheck(),
			},
		},
		// Note: We don't include CheckDestroy because bootstrap cannot be reverted
	})
}

func testResourceACLBootstrap_initialConfig() string {
	return `
resource "nomad_acl_bootstrap" "test" {
}
`
}

func testResourceACLBootstrap_initialCheck() resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		testResourceACLBootstrapExists("nomad_acl_bootstrap.test"),
		resource.TestCheckResourceAttrSet("nomad_acl_bootstrap.test", "accessor_id"),
		resource.TestCheckResourceAttrSet("nomad_acl_bootstrap.test", "bootstrap_token"),
	)
}

func testResourceACLBootstrapExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		providerConfig := testProvider.Meta().(ProviderConfig)
		client := providerConfig.client

		_, _, err := client.ACLTokens().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("ACL bootstrap token doesn't exist: %s", err)
		}

		return nil
	}
}
