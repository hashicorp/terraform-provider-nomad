package nomad

import (
	"fmt"
	"testing"

	nomadapi "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestSchedulerConfig_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testProviders,
		CheckDestroy: testFinalConfiguration,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNomadSchedulerConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_scheduler_config.config", "algorithm", "spread"),
					resource.TestCheckResourceAttr("nomad_scheduler_config.config.preemption", "system_enabled", "true"),
					resource.TestCheckResourceAttr("nomad_scheduler_config.config.preemption", "batch_enabled", "true"),
					resource.TestCheckResourceAttr("nomad_scheduler_config.config.preemption", "service_enabled", "true"),
				),
			},
			resource.TestStep{
				Config: testAccNomadSchedulerConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_scheduler_config.config", "algorithm", "binpack"),
					resource.TestCheckResourceAttr("nomad_scheduler_config.config.preemption", "system_enabled", "false"),
					resource.TestCheckResourceAttr("nomad_scheduler_config.config.preemption", "batch_enabled", "false"),
					resource.TestCheckResourceAttr("nomad_scheduler_config.config.preemption", "service_enabled", "false"),
				),
			},
		},
	})
}

const testAccNomadSchedulerConfigBasic = `
resource "nomad_scheduler_config" "config" {}
`

const testAccNomadSchedulerConfig = `
resource "nomad_scheduler_config" "config" {
	algorithm = "binpack"
	preemption {
		system_enabled = true
		batch_enabled = true
		service_enabled = true
	}
}`

// when destroying the nomad scheduler config resource, the configuration
// should not be changed
func testFinalConfiguration(s *terraform.State) error {
	client := testProvider.Meta().(ProviderConfig).client
	operator := client.Operator()
	qOpts := &nomadapi.QueryOptions{}
	config, _, err := operator.SchedulerGetConfiguration(qOpts)
	if err != nil {
		return fmt.Errorf("err: %v", err)
	}
	if config.SchedulerConfig.SchedulerAlgorithm != "binpack" {
		return fmt.Errorf("err: scheduler_algorithm during destroy: %v", config.SchedulerConfig.SchedulerAlgorithm)
	}
	if config.SchedulerConfig.PreemptionConfig.BatchSchedulerEnabled != false {
		return fmt.Errorf("err: scheduler_algorithm_preemtion_batch during destroy: %v", config.SchedulerConfig.PreemptionConfig.BatchSchedulerEnabled)
	}
	if config.SchedulerConfig.PreemptionConfig.SystemSchedulerEnabled != false {
		return fmt.Errorf("err: scheduler_algorithm_preemtion_system during destroy: %v", config.SchedulerConfig.PreemptionConfig.SystemSchedulerEnabled)
	}
	if config.SchedulerConfig.PreemptionConfig.ServiceSchedulerEnabled != false {
		return fmt.Errorf("err: scheduler_algorithm_preemtion_system during destroy: %v", config.SchedulerConfig.PreemptionConfig.ServiceSchedulerEnabled)
	}
	return nil
}
