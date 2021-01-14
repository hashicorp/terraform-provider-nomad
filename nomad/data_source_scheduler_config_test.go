package nomad

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDataSourceSchedulerConfig_basic(t *testing.T) {
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
		},
	})
}

const testAccNomadDataSourceSchedulerConfigS = `
resource "nomad_scheduler_config" "config" {
	scheduler_algorithm = "spread"
	preemption_config = {
		system_scheduler_enabled = true
		batch_scheduler_enabled = true
		service_scheduler_enabled = true
	}
}

data "nomad_scheduler_config" "config" {}
`
