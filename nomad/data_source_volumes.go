package nomad

import (
	"fmt"
	"log"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func dataSourceVolumes() *schema.Resource {
	return &schema.Resource{
		Read: volumesDataSourceRead,

		Schema: map[string]*schema.Schema{
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Volume type",
				Default:     "csi",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{"csi"}, false),
				},
			},
			"node_id": {
				Type:        schema.TypeString,
				Description: "Volume node filter",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"plugin_id": {
				Type:        schema.TypeString,
				Description: "Plugin ID filter",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"namespace": {
				Description: "Volume namespace filter",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "default",
			},

			"volumes": {
				Description: "Volumes",
				Computed:    true,
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeMap},
			},
		},
	}
}

func volumesDataSourceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(ProviderConfig).client

	ns := d.Get("namespace").(string)
	if ns == "" {
		ns = "default"
	}
	q := &api.QueryOptions{
		Namespace: ns,
	}
	if v, ok := d.GetOk("node_id"); ok && v != "" {
		q.Params["node_id"] = v.(string)
	}
	if v, ok := d.GetOk("plugin_id"); ok && v != "" {
		q.Params["plugin_id"] = v.(string)
	}

	log.Printf("[DEBUG] Reading list of volumes from Nomad")
	resp, _, err := client.CSIVolumes().List(q)
	if err != nil {
		return fmt.Errorf("error reading volumes from Nomad: %s", err)
	}
	volumes := make([]map[string]string, 0, len(resp))
	for _, v := range resp {
		volume := map[string]string{
			"ID":             v.ID,
			"ExternalID":     v.ExternalID,
			"Namespace":      v.Namespace,
			"Name":           v.Name,
			"AccessMode":     string(v.AccessMode),
			"AttachmentMode": string(v.AttachmentMode),
		}
		volumes = append(volumes, volume)
	}
	log.Printf("[DEBUG] Finished reading volumes from Nomad")
	d.SetId(client.Address() + "/v1/volumes")

	return d.Set("volumes", volumes)
}
