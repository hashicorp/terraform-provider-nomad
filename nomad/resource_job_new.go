package nomad

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceJobNew() *schema.Resource {
	return &schema.Resource{

		Create: resourceJobNewRegister,
		Read:   resourceJobNewRead,
		Update: resourceJobNewRegister,
		Delete: resourceJobNewDeregister,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "global",
			},
			"datacenters": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"constraint": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attribute": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"operator": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "=",
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"group": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"count": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  1,
						},
						"task": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"driver": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"config": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceJobNewRegister(d *schema.ResourceData, m interface{}) error {
	name := d.Get("name").(string)
	d.SetId(name)
	return resourceJobNewRead(d, m)
}

func resourceJobNewRead(d *schema.ResourceData, m interface{}) error {
	client := m.(ProviderConfig).client
	id := d.Id()

	job, _, err := client.Jobs().Info(id, nil)
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			log.Printf("[DEBUG] job %q does not exist, so removing", id)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("error checking for job: %s", err)
	}

	log.Printf("read job: %+v", job)

	d.Set("name", job.ID)
	d.Set("region", job.Region)
	d.Set("datacenters", job.Datacenters)
	d.Set("type", job.Type)
	d.Set("group", flattenTaskGroups(job.TaskGroups))

	return nil
}

func resourceJobNewDeregister(d *schema.ResourceData, m interface{}) error {
	return nil
}
