// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func jobSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Version:     1,
		Description: "Registers a job in Nomad from a jobspec.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The resource ID, equal to the job name. Exists for compatibility with configurations that reference .id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"jobspec": schema.StringAttribute{
				Required:    true,
				Description: "Job specification. If you want to point to a file use the file() function.",
			},
			"policy_override":         optionalBoolAttribute("Override any soft-mandatory Sentinel policies that fail.", false),
			"preserve_counts":         optionalBoolAttribute("If true, preserve the current task group counts during job registration instead of using the counts from the jobspec.", false),
			"preserve_resources":      optionalBoolAttribute("If true, preserve the current task resources during job registration instead of using the resources from the jobspec.", false),
			"deregister_on_destroy":   optionalBoolAttribute("If true, the job will be deregistered on destroy.", true),
			"deregister_on_id_change": optionalBoolAttribute("If true, the job will be deregistered when the job ID changes.", true),
			"detach":                  optionalBoolAttribute("If true, the provider will return immediately after creating or updating, instead of monitoring.", true),
			"deployment_id": schema.StringAttribute{
				Computed:    true,
				Description: "If detach = false, the ID for the deployment associated with the last job create/update, if one exists.",
			},
			"deployment_status": schema.StringAttribute{
				Computed:    true,
				Description: "If detach = false, the status for the deployment associated with the last job create/update, if one exists.",
			},
			"json": optionalBoolAttribute("If true, the jobspec will be parsed as JSON instead of HCL.", false),
			"modify_index": schema.StringAttribute{
				Computed:    true,
				Description: "Integer that increments for each change. Used to detect any changes between plan and apply.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the job, as derived from the jobspec.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"namespace": schema.StringAttribute{
				Computed:    true,
				Description: "The namespace of the job, as derived from the jobspec.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "The type of the job, as derived from the jobspec.",
			},
			"rerun_if_dead": optionalBoolAttribute("If true, forces the job to run again on apply if it is currently dead.", false),
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The status of the job.",
			},
			"status_description": schema.StringAttribute{
				Computed:    true,
				Description: "The status description of the job.",
			},
			"version": schema.Int64Attribute{
				Computed:    true,
				Description: "The version of the job.",
			},
			"submit_time": schema.StringAttribute{
				Computed:    true,
				Description: "The time the job was submitted.",
			},
			"create_index": schema.Int64Attribute{
				Computed:    true,
				Description: "The creation index of the job.",
			},
			"stop": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the job is stopped.",
			},
			"priority": schema.Int64Attribute{
				Computed:    true,
				Description: "The priority of the job for scheduling and resource access.",
			},
			"parent_id": schema.StringAttribute{
				Computed:    true,
				Description: "The parent job ID, if applicable.",
			},
			"stable": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the job is stable.",
			},
			"all_at_once": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the scheduler can make partial placements on oversubscribed nodes.",
			},
			"constraints":     constraintsAttribute(),
			"update_strategy": updateStrategyAttribute(),
			"periodic_config": periodicConfigAttribute(),
			"region": schema.StringAttribute{
				Computed:    true,
				Description: "The target region for the job, as derived from the jobspec.",
			},
			"datacenters": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "The target datacenters for the job, as derived from the jobspec.",
			},
			"read_allocation_ids": schema.BoolAttribute{
				Optional:           true,
				Computed:           true,
				Default:            booldefault.StaticBool(false),
				Description:        "Whether to retrieve allocation IDs associated with this job.",
				DeprecationMessage: "Retrieving allocation IDs from the job resource is deprecated and will be removed in a future release. Use the nomad_allocations data source instead.",
			},
			"allocation_ids": schema.ListAttribute{
				Computed:           true,
				ElementType:        types.StringType,
				Description:        "The IDs for allocations associated with this job.",
				DeprecationMessage: "Retrieving allocation IDs from the job resource is deprecated and will be removed in a future release. Use the nomad_allocations data source instead.",
			},
			"task_groups":      taskGroupsAttribute(),
			"purge_on_destroy": optionalBoolAttribute("Whether to purge the job when the resource is destroyed.", false),
		},
		Blocks: map[string]schema.Block{
			"hcl2": schema.ListNestedBlock{
				Description: "Configuration for the HCL2 jobspec parser.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"allow_fs": optionalBoolAttribute("If true, HCL2 file system functions will be enabled when parsing the jobspec.", false),
						"vars": schema.MapAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: "Additional variables to use when templating the job with HCL2.",
						},
					},
				},
			},
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
			}),
		},
	}
}

func optionalBoolAttribute(description string, defaultValue bool) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Default:     booldefault.StaticBool(defaultValue),
		Description: description,
	}
}

func constraintsAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed:    true,
		Description: "The job constraints.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"ltarget": schema.StringAttribute{Computed: true, Description: "The attribute being constrained."},
				"rtarget": schema.StringAttribute{Computed: true, Description: "The constraint value."},
				"operand": schema.StringAttribute{Computed: true, Description: "The operator used to compare the attribute to the constraint."},
			},
		},
	}
}

func updateStrategyAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed:    true,
		Description: "The update strategy for rolling updates and canary deployments.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"stagger":          schema.StringAttribute{Computed: true, Description: "Delay between each set of max_parallel updates when updating system jobs."},
				"max_parallel":     schema.Int64Attribute{Computed: true, Description: "Number of allocations within a task group that can be destructively updated at the same time. Setting 0 forces updates instead of deployments."},
				"health_check":     schema.StringAttribute{Computed: true, Description: "Mechanism used to determine allocation health: checks, task_states, or manual."},
				"min_healthy_time": schema.StringAttribute{Computed: true, Description: "Minimum time the allocation must be in the healthy state before further updates can proceed."},
				"healthy_deadline": schema.StringAttribute{Computed: true, Description: "Deadline by which the allocation must become healthy before it is marked unhealthy."},
				"auto_revert":      schema.BoolAttribute{Computed: true, Description: "Whether the job should automatically revert to the last stable job on deployment failure."},
				"canary":           schema.Int64Attribute{Computed: true, Description: "Number of canary allocations created before destructive updates continue."},
			},
		},
	}
}

func periodicConfigAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed:    true,
		Description: "The job's periodic configuration for time-based scheduling.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"enabled":          schema.BoolAttribute{Computed: true, Description: "Whether the periodic job is enabled."},
				"spec":             schema.StringAttribute{Computed: true, Description: "Cron expression configuring the interval at which the job is launched."},
				"spec_type":        schema.StringAttribute{Computed: true, Description: "Type of periodic specification, such as cron."},
				"prohibit_overlap": schema.BoolAttribute{Computed: true, Description: "Whether this job should wait until previous instances have completed before launching again."},
				"timezone":         schema.StringAttribute{Computed: true, Description: "Time zone used to evaluate the next launch interval."},
			},
		},
	}
}

func taskGroupsAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"name":            schema.StringAttribute{Computed: true},
				"count":           schema.Int64Attribute{Computed: true},
				"update_strategy": updateStrategyAttribute(),
				"task": schema.ListNestedAttribute{
					Computed: true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"name":      schema.StringAttribute{Computed: true},
							"driver":    schema.StringAttribute{Computed: true},
							"meta":      schema.MapAttribute{Computed: true, ElementType: types.StringType},
							"resources": resourcesAttribute(),
							"volume_mounts": schema.ListNestedAttribute{
								Computed: true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"volume":      schema.StringAttribute{Computed: true},
										"destination": schema.StringAttribute{Computed: true},
										"read_only":   schema.BoolAttribute{Computed: true},
									},
								},
							},
						},
					},
				},
				"volumes": schema.ListNestedAttribute{
					Computed: true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"name":      schema.StringAttribute{Computed: true},
							"type":      schema.StringAttribute{Computed: true},
							"read_only": schema.BoolAttribute{Computed: true},
							"source":    schema.StringAttribute{Computed: true},
						},
					},
				},
				"meta": schema.MapAttribute{Computed: true, ElementType: types.StringType},
			},
		},
	}
}

func resourcesAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Computed:    true,
		Description: "The effective resources requested by the task.",
		Attributes: map[string]schema.Attribute{
			"cpu":           schema.Int64Attribute{Computed: true},
			"cores":         schema.Int64Attribute{Computed: true},
			"memory_mb":     schema.Int64Attribute{Computed: true},
			"memory_max_mb": schema.Int64Attribute{Computed: true},
			"disk_mb":       schema.Int64Attribute{Computed: true},
			"secrets_mb":    schema.Int64Attribute{Computed: true},
			"iops": schema.Int64Attribute{
				Computed:           true,
				DeprecationMessage: "The Nomad resources IOPS field is deprecated.",
			},
			"numa": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"affinity": schema.StringAttribute{Computed: true},
					"devices":  schema.ListAttribute{Computed: true, ElementType: types.StringType},
				},
			},
			"networks": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"mode":     schema.StringAttribute{Computed: true},
						"device":   schema.StringAttribute{Computed: true},
						"cidr":     schema.StringAttribute{Computed: true},
						"ip":       schema.StringAttribute{Computed: true},
						"hostname": schema.StringAttribute{Computed: true},
						"mbits": schema.Int64Attribute{
							Computed:           true,
							DeprecationMessage: "The Nomad network MBits field is deprecated.",
						},
						"dns": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"servers":  schema.ListAttribute{Computed: true, ElementType: types.StringType},
								"searches": schema.ListAttribute{Computed: true, ElementType: types.StringType},
								"options":  schema.ListAttribute{Computed: true, ElementType: types.StringType},
							},
						},
						"reserved_ports": portsAttribute(),
						"dynamic_ports":  portsAttribute(),
						"cni": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"args": schema.MapAttribute{Computed: true, ElementType: types.StringType},
							},
						},
					},
				},
			},
			"devices": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":  schema.StringAttribute{Computed: true},
						"count": schema.Int64Attribute{Computed: true},
						"constraints": schema.ListNestedAttribute{
							Computed:     true,
							NestedObject: constraintsAttribute().NestedObject,
						},
						"affinities": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"ltarget": schema.StringAttribute{Computed: true},
									"rtarget": schema.StringAttribute{Computed: true},
									"operand": schema.StringAttribute{Computed: true},
									"weight":  schema.Int64Attribute{Computed: true},
								},
							},
						},
					},
				},
			},
		},
	}
}

func portsAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"label":            schema.StringAttribute{Computed: true},
				"value":            schema.Int64Attribute{Computed: true},
				"to":               schema.Int64Attribute{Computed: true},
				"host_network":     schema.StringAttribute{Computed: true},
				"ignore_collision": schema.BoolAttribute{Computed: true},
			},
		},
	}
}
