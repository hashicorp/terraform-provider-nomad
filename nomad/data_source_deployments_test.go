package nomad

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceDeployments(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceNomadDeploymentsConfig,
				Check:  testAccCheckDataSourceNomadDeploymentsExist,
			},
			{
				Config:      testAccCheckDataSourceNomadDeploymentsConfigErr,
				Destroy:     false,
				ExpectError: regexp.MustCompile(`.*No deployments found`),
			},
		},
	})
}

func testAccCheckDataSourceNomadDeploymentsExist(s *terraform.State) error {
	providerConfig := testProvider.Meta().(ProviderConfig)
	client := providerConfig.client

	// Try to find deployments
	_, _, err := client.Deployments().List(nil)

	if err != nil {
		return fmt.Errorf("Deployments not found")
	}

	return nil
}

var testAccCheckDataSourceNomadDeploymentsConfig = `
resource "nomad_job" "foobar" {
	jobspec = <<EOT
		job "foo" {
			datacenters = ["dc1"]
			type = "service"
			group "foo" {
				task "foo" {
					leader = true ## new in Nomad 0.5.6
					driver = "raw_exec"
					config {
						command = "/bin/sleep"
						args = ["1"]
					}
					resources {
						cpu = 100
						memory = 10
					}
					logs {
						max_files = 3
						max_file_size = 10
					}
				}
			}
		}
	EOT
}

data "nomad_deployments" "foobar" {}

`

var testAccCheckDataSourceNomadDeploymentsConfigErr = `
data "nomad_deployments" "foobar" {}

`
