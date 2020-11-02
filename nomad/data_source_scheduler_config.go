package nomad

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceSchedulerConfig() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceSchedulerConfigRead,

		Schema: map[string]*schema.Schema{
			"scheduler_algorithm": {
				Description: "Specifies whether scheduler binpacks or spreads allocations on available nodes.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"preemption_config": {
				Description: "Options to enable preemption for various schedulers.",
				Computed:    true,
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeBool},
			},
		},
	}
}

func dataSourceSchedulerConfigRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(ProviderConfig).client

	schedCfg, _, err := client.Operator().SchedulerGetConfiguration(nil)
	if err != nil {
		return fmt.Errorf("failed to query scheduler config: %v", err)
	}

	// Set a unique ID, as we have nothing else to go on.
	d.SetId(resource.UniqueId())

	if err := d.Set("scheduler_algorithm", schedCfg.SchedulerConfig.SchedulerAlgorithm); err != nil {
		return fmt.Errorf("failed to set scheduler_algorithm: %v", err)
	}
	premptMap := map[string]bool{
		"batch_scheduler_enabled":   schedCfg.SchedulerConfig.PreemptionConfig.BatchSchedulerEnabled,
		"service_scheduler_enabled": schedCfg.SchedulerConfig.PreemptionConfig.ServiceSchedulerEnabled,
		"system_scheduler_enabled":  schedCfg.SchedulerConfig.PreemptionConfig.SystemSchedulerEnabled,
	}
	if err := d.Set("preemption_config", premptMap); err != nil {
		return fmt.Errorf("failed to set preemption_config: %v", err)
	}
	return nil
}
