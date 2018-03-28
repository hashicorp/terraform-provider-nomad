package nomad

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceNomadJob_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testProviders,
		CheckDestroy: testAccCheckNomadJobDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceNomadJobConfig,
				Check:  testAccCheckDataSourceNomadJobExists("data.nomad_job.foobaz"),
			},
			{
				Config: testAccCheckDataSourceNomadJobConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.nomad_job.foobaz", "name", "foo"),
					resource.TestCheckResourceAttr(
						"data.nomad_job.foobaz", "type", "service"),
					resource.TestCheckResourceAttr(
						"data.nomad_job.foobaz", "status", "running"),
					resource.TestCheckResourceAttr(
						"data.nomad_job.foobaz", "priority", "50"),
				),
			},
			{
				Config:      testAccCheckDataSourceNomadJobConfigErr,
				Destroy:     false,
				ExpectError: regexp.MustCompile(`.*job not found`),
			},
		},
	})
}

func testAccCheckDataSourceNomadJobExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Job ID is set")
		}

		providerConfig := testProvider.Meta().(ProviderConfig)
		client := providerConfig.client

		id := rs.Primary.ID

		// Try to find the job
		test_job, _, err := client.Jobs().Info(id, nil)

		if err != nil {
			return err
		}

		if *test_job.ID != rs.Primary.ID {
			return fmt.Errorf("Job not found")
		}

		return nil
	}
}

func testAccCheckNomadJobDestroy(s *terraform.State) error {
	providerConfig := testProvider.Meta().(ProviderConfig)
	client := providerConfig.client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "nomad_job" {
			continue
		}

		id := rs.Primary.ID

		_, _, err := client.Jobs().Deregister(id, false, nil)
		if err != nil {
			return fmt.Errorf("error deregistering job: %s", err)
		}
	}

	return nil
}

var testAccCheckDataSourceNomadJobConfig = `
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

data "nomad_job" "foobaz" {
  job_id               = "foo"
}
`

var testAccCheckDataSourceNomadJobConfigErr = `
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

data "nomad_job" "foobar" {
  job_id               = "foo-mia"
}
`
