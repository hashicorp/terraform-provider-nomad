// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"log"
	"testing"

	"github.com/hashicorp/nomad/api"
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
						"preemption_config.sysbatch_scheduler_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"nomad_scheduler_config.config",
						"preemption_config.system_scheduler_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"nomad_scheduler_config.config",
						"memory_oversubscription_enabled",
						"false",
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
					resource.TestCheckResourceAttr(
						"nomad_scheduler_config.config",
						"memory_oversubscription_enabled",
						"false",
					),
				),
			},
			{
				Config: testAccNomadSchedulerConfigMemoryOversubscription,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"nomad_scheduler_config.config",
						"memory_oversubscription_enabled",
						"true",
					),
				),
			},
			{
				Config: testAccNomadSchedulerConfigDataSource,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.nomad_scheduler_config.config",
						"scheduler_algorithm",
						"binpack",
					),
					resource.TestCheckResourceAttr(
						"data.nomad_scheduler_config.config",
						"preemption_config.batch_scheduler_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"data.nomad_scheduler_config.config",
						"preemption_config.service_scheduler_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"data.nomad_scheduler_config.config",
						"preemption_config.system_scheduler_enabled",
						"false",
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

func TestSchedulerConfig_memoryOversubscriptionEnabledOutsideTest(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testProviders,
		CheckDestroy: testFinalConfiguration,
		Steps: []resource.TestStep{
			{
				Config: testAccNomadSchedulerConfigMemoryOversubscriptionOff,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"nomad_scheduler_config.config",
						"memory_oversubscription_enabled",
						"false",
					),
				),
			},
			{
				PreConfig: func() {
					providerConfig := testProvider.Meta().(ProviderConfig)
					client := providerConfig.client
					operator := client.Operator()

					config := api.SchedulerConfiguration{
						MemoryOversubscriptionEnabled: true,
						PreemptionConfig: api.PreemptionConfig{
							SysBatchSchedulerEnabled: true,
							SystemSchedulerEnabled:   false,
							BatchSchedulerEnabled:    false,
							ServiceSchedulerEnabled:  true,
						},
					}
					if _, _, err := operator.SchedulerSetConfiguration(&config, nil); err != nil {
						log.Printf("[ERROR] upserting scheduler configuration: %s", err.Error())
					}
					log.Printf("[DEBUG] Upserted scheduler configuration")
				},
				Config:             testAccNomadSchedulerConfigMemoryOversubscription,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

const testAccNomadSchedulerConfigSpread = `
resource "nomad_scheduler_config" "config" {
	scheduler_algorithm = "spread"
	preemption_config = {
		sysbatch_scheduler_enabled = true
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
		sysbatch_scheduler_enabled = true
		system_scheduler_enabled = false
		batch_scheduler_enabled = false
		service_scheduler_enabled = true
	}
}
`

const testAccNomadSchedulerConfigMemoryOversubscriptionOff = `
resource "nomad_scheduler_config" "config" {
	memory_oversubscription_enabled = false
	scheduler_algorithm = "binpack"
	preemption_config = {
		sysbatch_scheduler_enabled = true
		system_scheduler_enabled = false
		batch_scheduler_enabled = false
		service_scheduler_enabled = true
	}
}
`
const testAccNomadSchedulerConfigMemoryOversubscription = `
resource "nomad_scheduler_config" "config" {
	memory_oversubscription_enabled = true
	scheduler_algorithm = "binpack"
	preemption_config = {
		sysbatch_scheduler_enabled = true
		system_scheduler_enabled = false
		batch_scheduler_enabled = false
		service_scheduler_enabled = true
	}
}
`

const testAccNomadSchedulerConfigDataSource = `
data "nomad_scheduler_config" "config" {}

resource "nomad_scheduler_config" "config" {
	memory_oversubscription_enabled = true
	scheduler_algorithm = "binpack"
	preemption_config = {
		sysbatch_scheduler_enabled = true
		system_scheduler_enabled = false
		batch_scheduler_enabled = false
		service_scheduler_enabled = true
	}
}
`

// for details on why this is the way it is, checkout the comments on
// resourceSchedulerConfigurationDelete.
func testFinalConfiguration(_ *terraform.State) error { return nil }
