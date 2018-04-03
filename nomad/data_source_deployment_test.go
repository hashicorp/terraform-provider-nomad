package nomad

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceDeployment(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceNomadDeploymentConfig,
				Check:  testAccCheckDataSourceNomadDeploymentExist("data.nomad_deployment.foobaz"),
			},
			{
				Config:      testAccCheckDataSourceNomadDeploymentConfigErr,
				Destroy:     false,
				ExpectError: regexp.MustCompile(`.*deployment not found`),
			},
		},
	})
}

func testAccCheckDataSourceNomadDeploymentExist(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Deployment ID is set")
		}

		providerConfig := testProvider.Meta().(ProviderConfig)
		client := providerConfig.client

		id := rs.Primary.ID

		// Try to find the job
		test_deployment, _, err := client.Deployments().Info(id, nil)

		if err != nil {
			return err
		}

		if test_deployment.ID != rs.Primary.ID {
			return fmt.Errorf("Deployment not found")
		}

		return nil
	}
}

var testAccCheckDataSourceNomadDeploymentConfig = `
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

data "nomad_deployment" "foobaz" {
	deployment_id			=	""
}

`

var testAccCheckDataSourceNomadDeploymentConfigErr = `
data "nomad_deployment" "foobaz" {
	deployment_id			= ""
}

`
