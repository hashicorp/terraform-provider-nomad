package nomad

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

// Test will
// * Create job with update stanza
// * Pass config with same job spec and the deployments data source
// * Check there is 1 deployment and Destroy after this step
// * Pass just the data source config
// * Verify deployment is cancelled
func TestAccDataSourceDeployments(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceNomadDeploymentsJobCfg,
			},
			{
				Config:  testAccCheckDataSourceNomadDeploymentsCfgWithJob,
				Check:   resource.TestCheckResourceAttr("data.nomad_deployments.foobar", "deployments.#", "1"),
				Destroy: true,
			},
			{
				Config: testAccCheckDataSourceNomadDeploymentsCfg,
				Check:  resource.TestCheckResourceAttr("data.nomad_deployments.foobar", "deployments.0.Status", "cancelled"),
			},
		},
	})
}

var testAccCheckDataSourceNomadDeploymentsJobCfg = `
resource "nomad_job" "foobar" {
	jobspec = <<EOT
		job "foo" {
			update {} ## creates deployment
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
`
var testAccCheckDataSourceNomadDeploymentsCfg = `

data "nomad_deployments" "foobar" {}

`

var testAccCheckDataSourceNomadDeploymentsCfgWithJob = testAccCheckDataSourceNomadDeploymentsJobCfg + testAccCheckDataSourceNomadDeploymentsCfg
