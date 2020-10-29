package nomad

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceJob() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceJobRead,
		Schema: map[string]*schema.Schema{

			"job_id": {
				Description: "Job ID",
				Type:        schema.TypeString,
				Required:    true,
			},
			"namespace": {
				Description: "Job Namespace",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "default",
			},
			// computed attributes
			"name": {
				Description: "Job Name",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"type": {
				Description: "Job Type",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"version": {
				Description: "Job Version",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"region": {
				Description: "Job Region",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"datacenters": {
				Description: "Job Datacenters",
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
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
			"submit_time": {
				Description: "Job Submit Time",
				Type:        schema.TypeInt,
				Computed:    true,
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
			"job_modify_index": {
				Description: "Job Modify Index",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"stop": {
				Description: "Job Stopped",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"priority": {
				Description: "Job Priority",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"parent_id": {
				Description: "Job Parent ID",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"task_groups": {
				Description: "Job Task Groups",
				Computed:    true,
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"count": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"constraints": {
							Type:     schema.TypeMap,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ltarget": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"rtarget": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"operand": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						//"tasks": {
						//	Type:     schema.TypeInt,
						//	Computed: true,
						//	Type:        schema.TypeList,
						//	Elem: &schema.Resource{
						//		Schema: map[string]*schema.Schema{
						//		}
						//},
						"restart_policy": {
							Type:     schema.TypeMap,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"interval": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"attempts": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"delay": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"mode": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"reschedule_policy": {
							Type:     schema.TypeMap,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"attempts": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"interval": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"delay": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"delay_function": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"max_delay": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"unlimited": {
										Type:     schema.TypeBool,
										Computed: true,
									},
								},
							},
						},
						"ephemeral_disk": {
							Type:     schema.TypeMap,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"sticky": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"migrate": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"size_mb": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"update_strategy": {
							Computed: true,
							Type:     schema.TypeMap,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"stagger": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"max_parallel": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"health_check": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"min_healthy_time": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"healthy_deadline": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"auto_revert": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"canary": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"migrate_strategy": {
							Type:     schema.TypeMap,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"max_parallel": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"health_check": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"min_healthy_time": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"healthy_deadline": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
			"stable": {
				Description: "Job Stable",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"all_at_once": {
				Description: "Job All At Once",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"constraints": {
				Description: "Job Constraints",
				Computed:    true,
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ltarget": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"rtarget": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"operand": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"update_strategy": {
				Description: "Job Update Policy",
				Computed:    true,
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"stagger": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"max_parallel": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"health_check": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"min_healthy_time": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"healthy_deadline": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"auto_revert": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"canary": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"periodic_config": {
				Description: "Job Periodic Configuration",
				Computed:    true,
				Type:        schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"spec": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"spec_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"prohibit_overlap": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"timezone": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceJobRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	id := d.Get("job_id").(string)
	ns := d.Get("namespace").(string)
	if ns == "" {
		ns = "default"
	}
	log.Printf("[DEBUG] Getting job status: %q/%q", ns, id)
	job, _, err := client.Jobs().Info(id, &api.QueryOptions{
		Namespace: ns,
	})
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			return err
		}

		return fmt.Errorf("error checking for job: %#v", err)
	}

	d.SetId(*job.ID)
	d.Set("name", job.Name)
	d.Set("type", job.Type)
	d.Set("version", job.Version)
	d.Set("namespace", job.Namespace)
	d.Set("region", job.Region)
	d.Set("datacenters", job.Datacenters)
	d.Set("status", job.Status)
	d.Set("status_description", job.StatusDescription)
	d.Set("submit_time", job.SubmitTime)
	d.Set("create_index", job.CreateIndex)
	d.Set("modify_index", job.ModifyIndex)
	d.Set("job_modify_index", job.JobModifyIndex)
	d.Set("stop", job.Stop)
	d.Set("priority", job.Priority)
	d.Set("parent_id", job.ParentID)
	d.Set("task_groups", job.TaskGroups)
	d.Set("stable", job.Stable)
	d.Set("all_at_once", job.AllAtOnce)
	d.Set("contraints", job.Constraints)
	d.Set("update_strategy", job.Update)
	d.Set("periodic_config", job.Periodic)

	return nil
}
