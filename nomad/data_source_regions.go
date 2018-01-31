package nomad

import (
	"fmt"
	"log"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceRegions() *schema.Resource {
	return &schema.Resource{
		Read: genericSecretDataSourceRead,

		Schema: map[string]*schema.Schema{
			"regions": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func genericSecretDataSourceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	log.Printf("[DEBUG] Reading regions from Nomad")
	resp, err := client.Regions().List()
	if err != nil {
		return fmt.Errorf("error reading regions from Nomad: %s", err)
	}
	log.Printf("[DEBUG] Read regions from Nomad")
	d.SetId(client.Address() + "/regions")

	return d.Set("regions", resp)
}
