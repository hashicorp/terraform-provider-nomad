package nomad

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccDataSourceDeployment(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceNomadDeploymentConfig,
				Check:  testAccCheckDataSourceNomadDeploymentExist(),
			},
			{
				Config:      testAccCheckDataSourceNomadDeploymentConfigErr,
				Destroy:     false,
				ExpectError: regexp.MustCompile(`missing deployment ID`),
			},
		},
	})
}

func testAccCheckDataSourceNomadDeploymentExist() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceName := "data.nomad_deployment.foobaz"

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
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
  detach = false
  jobspec = <<EOT
job "foo-deployment-datasource" {
  update {
    min_healthy_time = "100ms"
  }

  datacenters = ["dc1"]
  type        = "service"
  group "foo" {
    task "foo" {
      driver = "raw_exec"
      config {
        command = "/bin/sleep"
        args    = ["10"]
      }
      resources {
        cpu    = 1
        memory = 10
      }
      logs {
        max_files     = 1
        max_file_size = 1
      }
    }
  }
}
EOT
}

data "nomad_deployment" "foobaz" {
  deployment_id = nomad_job.foobar.deployment_id
}
`

var testAccCheckDataSourceNomadDeploymentConfigErr = `
data "nomad_deployment" "foobaz" {
  deployment_id = ""
}`
