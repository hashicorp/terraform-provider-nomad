// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNamespace() *schema.Resource {
	return &schema.Resource{
		Read: namespaceDataSourceRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"quota": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"meta": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"capabilities": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     resourceNamespaceCapabilities(),
			},
			"node_pool_config": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"default": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"allowed": {
							Computed: true,
							Type:     schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"denied": {
							Computed: true,
							Type:     schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func namespaceDataSourceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(ProviderConfig).client

	name := d.Get("name").(string)
	ns, _, err := client.Namespaces().Info(name, nil)
	if err != nil {
		return fmt.Errorf("Failed to get information about %q: %v", name, err)
	}

	if err = d.Set("description", ns.Description); err != nil {
		return fmt.Errorf("Failed to set 'description': %v", err)
	}
	if err = d.Set("quota", ns.Quota); err != nil {
		return fmt.Errorf("Failed to set 'quota': %v", err)
	}
	if err = d.Set("meta", ns.Meta); err != nil {
		return fmt.Errorf("Failed to set 'meta': %v", err)
	}
	if err = d.Set("capabilities", flattenNamespaceCapabilities(ns.Capabilities)); err != nil {
		return fmt.Errorf("Failed to set 'capabilities': %v", err)
	}
	if err = d.Set("node_pool_config", flattenNamespaceNodePoolConfig(ns.NodePoolConfiguration)); err != nil {
		return fmt.Errorf("Failed to set 'node_pool_config': %v", err)
	}

	d.SetId(client.Address() + "/namespace/" + name)
	return nil
}
