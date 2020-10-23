package nomad

import (
	"fmt"
	"log"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceSchedulerConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceSchedulerConfigurationWrite,
		Update: resourceSchedulerConfigurationUpdate,
		Delete: resourceSchedulerConfigurationDelete,
		Read:   resourceSchedulerConfigurationRead,

		Schema: map[string]*schema.Schema{
			"algorithm": {
				Description: "Scheduler algorithm",
				Type:        schema.TypeString,
				Default:     "binpack",
				Optional:    true,
				ValidateFunc: validation.StringInSlice([]string{
					"binpack",
					"spread",
				}, false),
			},
			"preemption": {
				Required:    true,
				Description: "Preemption for various schedulers",
				Type:        schema.TypeList,
				Elem:        resourcePreemptionConfig(),
			},
		},
	}
}

func resourcePreemptionConfig() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"system_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"service_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"batch_enabled": {
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

	config := api.SchedulerConfiguration{
		SchedulerAlgorithm: api.SchedulerAlgorithm(d.Get("algorithm").(string)),
	}

	pcfg, err := expandPreemptionConfig(d)
	if err != nil {
		return err
	}
	config.PreemptionConfig = *pcfg

	log.Printf("[DEBUG] Upserting Scheduler configuration")
	_, _, err = operator.SchedulerSetConfiguration(&config, nil)
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

	reg, err := client.Agent().Region()
	if err != nil {
		return fmt.Errorf("error getting region: %s", err.Error())
	}
	// retrieve the config
	log.Printf("[DEBUG] Reading scheduler configuration")
	config, _, err := operator.SchedulerGetConfiguration(nil)
	if err != nil {
		return fmt.Errorf("error reading scheduler configuration: %s", err.Error())
	}
	log.Printf("[DEBUG] Read scheduler configuration")

	d.SetId(fmt.Sprintf("nomad-scheduler-configuration-%s", reg))
	d.Set("algorithm", config.SchedulerConfig.SchedulerAlgorithm)
	err = d.Set("preemption", flattenPreeption(config.SchedulerConfig.PreemptionConfig))
	if err != nil {
		return fmt.Errorf("error setting preemption: %s", err.Error())
	}

	return nil
}

func flattenPreeption(preemption api.PreemptionConfig) []interface{} {
	result := map[string]interface{}{}
	result["system_enabled"] = preemption.SystemSchedulerEnabled
	result["service_enabled"] = preemption.ServiceSchedulerEnabled
	result["batch_enabled"] = preemption.BatchSchedulerEnabled
	return []interface{}{result}
}

func expandPreemptionConfig(d *schema.ResourceData) (*api.PreemptionConfig, error) {
	configs := d.Get("preemption")
	cfgs := configs.([]interface{})

	if len(cfgs) < 1 {
		return nil, nil
	}

	cfg, ok := cfgs[0].(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("expected map[string]interface{} for region limit, got %T", cfgs[0])
	}

	results := &api.PreemptionConfig{
		BatchSchedulerEnabled:   cfg["batch_enabled"].(bool),
		ServiceSchedulerEnabled: cfg["service_enabled"].(bool),
		SystemSchedulerEnabled:  cfg["system_enabled"].(bool),
	}

	return results, nil
}
