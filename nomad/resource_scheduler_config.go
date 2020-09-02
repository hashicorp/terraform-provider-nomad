package nomad

import (
	"fmt"
	"log"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceSchedulerConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceSchedulerConfigurationWrite,
		Update: resourceSchedulerConfigurationUpdate,
		Delete: resourceSchedulerConfigurationDelete,
		Read:   resourceSchedulerConfigurationRead,

		Schema: map[string]*schema.Schema{
			"scheduler_config": {
				Description: "Scheduler configuration settings",
				Required:    true,
				Type:        schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"scheduler_algorithm": {
							Description: "Scheduler algorithm",
							Type:        schema.TypeString,
							Default:     "binpack",
							Optional:    true,
						},
						"preemption_config": {
							Required:    true,
							Description: "Preemption for various schedulers",
							Type:        schema.TypeSet,
							Elem:        resourcePreemptionConfigSchema(),
						},
					},
				},
			},
		},
	}
}

func resourcePreemptionConfigSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"system_scheduler_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"service_scheduler_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"batch_scheduler_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceSchedulerConfigurationWrite(d *schema.ResourceData, meta interface{}) error {
	return resourceSchedulerConfigurationUpdate(d, meta)
}
func resourceSchedulerConfigurationUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(ProviderConfig).client
	operator := client.Operator()

	preemptionConfig := api.PreemptionConfig{
		BatchSchedulerEnabled:   d.Get("batch_scheduler_enabled").(bool),
		SystemSchedulerEnabled:  d.Get("system_scheduler_enabled").(bool),
		ServiceSchedulerEnabled: d.Get("service_scheduler_enabled").(bool),
	}

	config := api.SchedulerConfiguration{
		SchedulerAlgorithm: api.SchedulerAlgorithm(d.Get("scheduler_algorithm").(string)),
		PreemptionConfig:   preemptionConfig,
	}

	log.Printf("[DEBUG] Upserting Scheduler configuration")
	_, _, err := operator.SchedulerSetConfiguration(&config, nil)
	if err != nil {
		return fmt.Errorf("error upserting scheduler configuration: %s", err.Error())
	}
	log.Printf("[DEBUG] Upserted scheduler configuration")

	return resourceSchedulerConfigurationRead(d, meta)
}

func resourceSchedulerConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceSchedulerConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(ProviderConfig).client
	operator := client.Operator()

	// retrieve the config
	log.Printf("[DEBUG] Reading scheduler configuration")
	config, _, err := operator.SchedulerGetConfiguration(nil)
	if err != nil {
		return fmt.Errorf("error reading scheduler configuration: %s", err.Error())
	}
	log.Printf("[DEBUG] Read scheduler configuration")

	pcfg := api.PreemptionConfig{
		BatchSchedulerEnabled:   config.SchedulerConfig.PreemptionConfig.BatchSchedulerEnabled,
		SystemSchedulerEnabled:  config.SchedulerConfig.PreemptionConfig.SystemSchedulerEnabled,
		ServiceSchedulerEnabled: config.SchedulerConfig.PreemptionConfig.ServiceSchedulerEnabled,
	}

	scfg := api.SchedulerConfiguration{
		SchedulerAlgorithm: config.SchedulerConfig.SchedulerAlgorithm,
		PreemptionConfig:   pcfg,
	}

	d.Set("scheduler_config", scfg)

	return nil
}
