package nomad

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
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

	d.SetId(client.Address() + "/namespace/" + name)
	return nil
}
