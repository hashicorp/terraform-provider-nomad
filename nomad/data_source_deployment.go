package nomad

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceDeployment() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDeploymentRead,
		Schema: map[string]*schema.Schema{

			"id": {
				Description: "Deployment ID",
				Type:        schema.TypeString,
				Required:    true,
			},
			// computed attributes
			"namepsace": {
				Description: "Deployment Namespace",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"job_id": {
				Description: "Job ID",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"job_version": {
				Description: "Job Version",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"job_create_index": {
				Description: "Job Create Index",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"job_modify_index": {
				Description: "Job Modify Index",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"task_groups": {
				Description: "Task Groups",
				Computed:    true,
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"placed_canaries": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"auto_revert": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"promoted": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"desired_canaries": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"desired_total": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"placed_alloc": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"healthy_alloc": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"unhealthy_alloc": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
			"status": {
				Description: "Deployment Status",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"status_description": {
				Description: "Deployment Status Description",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"create_index": {
				Description: "Create Index",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"modify_index": {
				Description: "Modify Index",
				Type:        schema.TypeInt,
				Computed:    true,
			},
		},
	}
}

func dataSourceDeploymentRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	id := d.Get("id").(string)
	log.Printf("[DEBUG] Getting deployment status: %q", id)
	deployment, _, err := client.Deployments().Info(id, nil)
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			return err
		}

		return fmt.Errorf("error checking for deployment: %#v", err)
	}

	d.SetId(deployment.ID)
	d.Set("namespace", deployment.Namespace)
	d.Set("job_id", deployment.JobID)
	d.Set("job_version", deployment.JobVersion)
	d.Set("job_create_index", deployment.JobCreateIndex)
	d.Set("job_modify_index", deployment.JobModifyIndex)
	d.Set("task_groups", deployment.TaskGroups)
	d.Set("status", deployment.Status)
	d.Set("status_description", deployment.StatusDescription)
	d.Set("create_index", deployment.CreateIndex)
	d.Set("modify_index", deployment.ModifyIndex)

	return nil
}
