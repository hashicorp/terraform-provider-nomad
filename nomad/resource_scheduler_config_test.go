package nomad

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestSchedulerConfig_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testProviders,
		CheckDestroy: testFinalConfiguration,
		Steps: []resource.TestStep{
			{
				Config: testAccNomadSchedulerConfigSpread,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"nomad_scheduler_config.config",
						"scheduler_algorithm",
						"spread",
					),
					resource.TestCheckResourceAttr(
						"nomad_scheduler_config.config",
						"preemption_config.batch_scheduler_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"nomad_scheduler_config.config",
						"preemption_config.service_scheduler_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"nomad_scheduler_config.config",
						"preemption_config.system_scheduler_enabled",
						"true",
					),
				),
			},
			{
				Config: testAccNomadSchedulerConfigBinpack,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"nomad_scheduler_config.config",
						"scheduler_algorithm",
						"binpack",
					),
					resource.TestCheckResourceAttr(
						"nomad_scheduler_config.config",
						"preemption_config.batch_scheduler_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"nomad_scheduler_config.config",
						"preemption_config.service_scheduler_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"nomad_scheduler_config.config",
						"preemption_config.system_scheduler_enabled",
						"false",
					),
				),
			},
		},
	})
}

const testAccNomadSchedulerConfigSpread = `
resource "nomad_scheduler_config" "config" {
	scheduler_algorithm = "spread"
	preemption_config = {
		system_scheduler_enabled = true
		batch_scheduler_enabled = true
		service_scheduler_enabled = true
	}
}
`

const testAccNomadSchedulerConfigBinpack = `
resource "nomad_scheduler_config" "config" {
	scheduler_algorithm = "binpack"
	preemption_config = {
		system_scheduler_enabled = false
		batch_scheduler_enabled = false
		service_scheduler_enabled = true
	}
}
`

// for details on why this is the way it is, checkout the comments on
// resourceSchedulerConfigurationDelete.
func testFinalConfiguration(_ *terraform.State) error { return nil }
