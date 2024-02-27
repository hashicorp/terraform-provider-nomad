// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAllocations() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAllocationsRead,

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
			"namespace": {
				Description: "Specifies the namespace to search for allocation in.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"allocations": {
				Description: "List of node pools returned",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"eval_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"namespace": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"node_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"node_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"job_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"job_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"job_version": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"task_group": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"desired_status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"client_status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"followup_eval_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"next_allocation": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"preempted_by_allocation": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"create_index": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"modify_index": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"create_time": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"modify_time": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAllocationsRead(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	prefix := d.Get("prefix").(string)
	filter := d.Get("filter").(string)
	id := strconv.Itoa(schema.HashString(prefix + filter))

	log.Printf("[DEBUG] Reading allocation list")

	queryOptions := api.QueryOptions{
		Prefix: prefix,
		Filter: filter,
	}
	namespace := d.Get("namespace").(string)
	if namespace != "" {
		queryOptions.Namespace = namespace
	}
	resp, _, err := client.Allocations().List(&queryOptions)
	if err != nil {
		return fmt.Errorf("error reading allocations: %w", err)
	}

	allocs := make([]map[string]any, len(resp))
	for i, alloc := range resp {
		allocs[i] = map[string]any{
			"id":                      alloc.ID,
			"eval_id":                 alloc.EvalID,
			"name":                    alloc.Name,
			"namespace":               alloc.Namespace,
			"node_id":                 alloc.NodeID,
			"node_name":               alloc.NodeName,
			"job_id":                  alloc.JobID,
			"job_type":                alloc.JobType,
			"job_version":             alloc.JobVersion,
			"task_group":              alloc.TaskGroup,
			"desired_status":          alloc.DesiredStatus,
			"client_status":           alloc.ClientStatus,
			"followup_eval_id":        alloc.FollowupEvalID,
			"next_allocation":         alloc.NextAllocation,
			"preempted_by_allocation": alloc.PreemptedByAllocation,
			"create_index":            alloc.CreateIndex,
			"modify_index":            alloc.ModifyIndex,
			"create_time":             alloc.CreateTime,
			"modify_time":             alloc.ModifyTime,
		}
	}
	log.Printf("[DEBUG] Read allocations")

	d.SetId(id)
	return d.Set("allocations", allocs)
}
