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
				Config: testAccNomadDataSourceSchedulerConfigS,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.nomad_scheduler_config.config",
						"scheduler_algorithm",
						"spread",
					),
					resource.TestCheckResourceAttr(
						"data.nomad_scheduler_config.config",
						"preemption_config.batch_scheduler_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"data.nomad_scheduler_config.config",
						"preemption_config.service_scheduler_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"data.nomad_scheduler_config.config",
						"preemption_config.system_scheduler_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"data.nomad_scheduler_config.config",
						"memory_oversubscription_enabled",
						"true",
					),
				),
			},
		},
	})
}

const testAccNomadDataSourceSchedulerConfigS = `
resource "nomad_scheduler_config" "config" {
	memory_oversubscription_enabled = true
	scheduler_algorithm = "spread"
	preemption_config = {
		system_scheduler_enabled = true
		batch_scheduler_enabled = true
		service_scheduler_enabled = true
	}
}

data "nomad_scheduler_config" "config" {}
`
