// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-nomad/nomad/helper"
)

func dataSourceNodePool() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNodePoolRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Unique name for this node pool.",
				Type:        schema.TypeString,
				Required:    true,
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
			"node_identity_ttl": {
				Description: "The TTL applied to node identities issued to nodes in this pool.",
				Type:        schema.TypeString,
				Computed:    true,
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
	}
}

func dataSourceNodePoolRead(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	name := d.Get("name").(string)
	log.Printf("[DEBUG] Reading node pool %q", name)
	pool, _, err := client.NodePools().Info(name, nil)
	if err != nil {
		return fmt.Errorf("error reading node pool %q: %w", name, err)
	}
	log.Printf("[DEBUG] Read node pool %q", name)

	sw := helper.NewStateWriter(d)
	sw.Set("name", pool.Name)
	sw.Set("description", pool.Description)
	sw.Set("meta", pool.Meta)
	sw.Set("scheduler_config", flattenNodePoolSchedulerConfiguration(pool.SchedulerConfiguration))

	// The identity TTL was introduced in 1.11.0, so only set it if it's
	// non-zero to ensure this provider does not panic with older servers.
	if pool.NodeIdentityTTL != 0 {
		sw.Set("node_identity_ttl", pool.NodeIdentityTTL.String())
	}

	if err := sw.Error(); err != nil {
		return err
	}

	d.SetId(name)
	return nil
}
