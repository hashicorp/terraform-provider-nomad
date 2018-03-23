package nomad

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestDataSourceDeployments(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testDataSourceDeployments_config,
				Check:  testDataSourceDeployments_check,
			},
		},
	})
}

func testDataSourceDeployments_check(s *terraform.State) error {
	providerConfig := testProvider.Meta().(ProviderConfig)
	client := providerConfig.client

	// Try to find deployments
	_, _, err := client.Deployments().List(nil)

	if err != nil {
		return fmt.Errorf("Deployments not found")
	}

	return nil
}

var testDataSourceDeployments_config = `

data "nomad_deployments" "foobar" {}

`
