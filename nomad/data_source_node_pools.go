// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNodePools() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNodePoolsRead,

		Schema: map[string]*schema.Schema{
			"prefix": {
				Description: "Specifies a string to filter node pools based on a name prefix.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"filter": {
				Description: "Specifies the expression used to filter the results.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"node_pools": {
				Description: "List of node pools returned",
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Unique name for this node pool.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"description": {
							Description: "Description for this node pool.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"meta": {
							Description: "Metadata associated with the node pool",
							Type:        schema.TypeMap,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Computed: true,
						},
						"scheduler_config": {
							Description: "Scheduler configuration for the node pool.",
							Type:        schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"scheduler_algorithm": {
										Description: "The scheduler algorithm to use in the node pool.",
										Type:        schema.TypeString,
										Computed:    true,
									},

									// This field must be a string instead of a bool (and differ from
									// Nomad) in order to represent a tristate.
									// - "enabled": memory oversubscription is enabled in the pool.
									// - "disabled": memory oversubscription is disabled in the pool.
									// - "": the global memory oversubscription value is used.
									"memory_oversubscription": {
										Description: "If true, the node pool will have memory oversubscription enabled.",
										Type:        schema.TypeString,
										Computed:    true,
									},
								},
							},
							Computed: true,
						},
					},
				},
				Computed: true,
			},
		},
	}
}

func dataSourceNodePoolsRead(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	prefix := d.Get("prefix").(string)
	filter := d.Get("filter").(string)
	id := strconv.Itoa(schema.HashString(prefix + filter))

	log.Printf("[DEBUG] Reading node pool list")
	resp, _, err := client.NodePools().List(&api.QueryOptions{
		Prefix: prefix,
		Filter: filter,
	})
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading node pools: %w", err)
	}

	pools := make([]map[string]any, len(resp))
	for i, p := range resp {
		pools[i] = map[string]any{
			"name":             p.Name,
			"description":      p.Description,
			"meta":             p.Meta,
			"scheduler_config": flattenNodePoolSchedulerConfiguration(p.SchedulerConfiguration),
		}
	}
	log.Printf("[DEBUG] Read node pool list")

	d.SetId(id)
	return d.Set("node_pools", pools)
}
