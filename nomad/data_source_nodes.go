// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNodes() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNodesRead,

		Schema: map[string]*schema.Schema{
			"prefix": {
				Description: "Specifies a string to filter nodes based on an ID prefix. Must have an even number of hexadecimal characters (0-9a-f).",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"filter": {
				Description: "Specifies the expression used to filter the results.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"os": {
				Description: "If true, include special attributes such as operating system name in the response.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"resources": {
				Description: "If true, include NodeResources and ReservedResources in the response.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"nodes": {
				Description: "List of nodes returned.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "The ID of the node.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"name": {
							Description: "The name of the node.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"datacenter": {
							Description: "The datacenter of the node.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"node_class": {
							Description: "The node class of the node.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"node_pool": {
							Description: "The node pool of the node.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"address": {
							Description: "The address of the node.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"version": {
							Description: "The Nomad version of the node.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"drain": {
							Description: "Whether the node is in drain mode.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"status": {
							Description: "The status of the node.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"status_description": {
							Description: "The status description of the node.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"scheduling_eligibility": {
							Description: "The scheduling eligibility of the node.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"attributes": {
							Description: "A map of attributes for the node. OS-related attributes are only included when the os parameter is set to true.",
							Type:        schema.TypeMap,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Computed: true,
						},
						"drivers": {
							Description: "A list of driver information for the node.",
							Type:        schema.TypeList,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Description: "The driver name.",
										Type:        schema.TypeString,
										Computed:    true,
									},
									"detected": {
										Description: "Whether the driver is detected.",
										Type:        schema.TypeBool,
										Computed:    true,
									},
									"healthy": {
										Description: "Whether the driver is healthy.",
										Type:        schema.TypeBool,
										Computed:    true,
									},
									"attributes": {
										Description: "Driver-specific attributes.",
										Type:        schema.TypeMap,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
										Computed: true,
									},
								},
							},
						},
						"node_resources": {
							Description: "Resources available on the node. Only populated when resources parameter is true.",
							Type:        schema.TypeList,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cpu": {
										Description: "CPU resources on the node.",
										Type:        schema.TypeList,
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"cpu_shares": {
													Description: "Total CPU shares available.",
													Type:        schema.TypeInt,
													Computed:    true,
												},
												"total_cpu_cores": {
													Description: "Total number of CPU cores.",
													Type:        schema.TypeInt,
													Computed:    true,
												},
												"reservable_cpu_cores": {
													Description: "List of reservable CPU core IDs.",
													Type:        schema.TypeList,
													Computed:    true,
													Elem: &schema.Schema{
														Type: schema.TypeInt,
													},
												},
											},
										},
									},
									"memory": {
										Description: "Memory resources on the node.",
										Type:        schema.TypeList,
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"memory_mb": {
													Description: "Total memory in MB.",
													Type:        schema.TypeInt,
													Computed:    true,
												},
											},
										},
									},
									"disk": {
										Description: "Disk resources on the node.",
										Type:        schema.TypeList,
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"disk_mb": {
													Description: "Total disk space in MB.",
													Type:        schema.TypeInt,
													Computed:    true,
												},
											},
										},
									},
									"networks": {
										Description: "Network resources on the node.",
										Type:        schema.TypeList,
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"device": {
													Description: "The network device.",
													Type:        schema.TypeString,
													Computed:    true,
												},
												"cidr": {
													Description: "The CIDR of the network.",
													Type:        schema.TypeString,
													Computed:    true,
												},
												"ip": {
													Description: "The IP address of the network.",
													Type:        schema.TypeString,
													Computed:    true,
												},
												"mode": {
													Description: "The network mode.",
													Type:        schema.TypeString,
													Computed:    true,
												},
											},
										},
									},
									"devices": {
										Description: "Device resources on the node (GPUs, etc.).",
										Type:        schema.TypeList,
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"vendor": {
													Description: "The device vendor.",
													Type:        schema.TypeString,
													Computed:    true,
												},
												"type": {
													Description: "The device type.",
													Type:        schema.TypeString,
													Computed:    true,
												},
												"name": {
													Description: "The device name.",
													Type:        schema.TypeString,
													Computed:    true,
												},
												"count": {
													Description: "The number of device instances.",
													Type:        schema.TypeInt,
													Computed:    true,
												},
											},
										},
									},
									"min_dynamic_port": {
										Description: "Minimum dynamic port for this node.",
										Type:        schema.TypeInt,
										Computed:    true,
									},
									"max_dynamic_port": {
										Description: "Maximum dynamic port for this node.",
										Type:        schema.TypeInt,
										Computed:    true,
									},
								},
							},
						},
						"reserved_resources": {
							Description: "Resources reserved on the node. Only populated when resources parameter is true.",
							Type:        schema.TypeList,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cpu": {
										Description: "Reserved CPU resources.",
										Type:        schema.TypeList,
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"cpu_shares": {
													Description: "Reserved CPU shares.",
													Type:        schema.TypeInt,
													Computed:    true,
												},
											},
										},
									},
									"memory": {
										Description: "Reserved memory resources.",
										Type:        schema.TypeList,
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"memory_mb": {
													Description: "Reserved memory in MB.",
													Type:        schema.TypeInt,
													Computed:    true,
												},
											},
										},
									},
									"disk": {
										Description: "Reserved disk resources.",
										Type:        schema.TypeList,
										Computed:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"disk_mb": {
													Description: "Reserved disk space in MB.",
													Type:        schema.TypeInt,
													Computed:    true,
												},
											},
										},
									},
									"networks": {
										Description: "Reserved network resources.",
										Type:        schema.TypeMap,
										Computed:    true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
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

func dataSourceNodesRead(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	prefix := d.Get("prefix").(string)
	filter := d.Get("filter").(string)
	osParam := d.Get("os").(bool)
	resourcesParam := d.Get("resources").(bool)
	id := strconv.Itoa(schema.HashString(prefix + filter + strconv.FormatBool(osParam) + strconv.FormatBool(resourcesParam)))

	log.Printf("[DEBUG] Reading nodes list")

	queryOptions := &api.QueryOptions{
		Prefix: prefix,
		Filter: filter,
		Params: make(map[string]string),
	}

	// Add os parameter if enabled
	if osParam {
		queryOptions.Params["os"] = "true"
	}

	// Add resources parameter if enabled
	if resourcesParam {
		queryOptions.Params["resources"] = "true"
	}

	resp, _, err := client.Nodes().List(queryOptions)
	if err != nil {
		return fmt.Errorf("error reading nodes: %w", err)
	}

	nodes := make([]map[string]any, len(resp))
	for i, node := range resp {
		// Flatten drivers to a list with name, detected, healthy, health_description, and attributes
		drivers := make([]map[string]any, 0, len(node.Drivers))
		for name, info := range node.Drivers {
			if info != nil {
				drivers = append(drivers, map[string]any{
					"name":       name,
					"detected":   info.Detected,
					"healthy":    info.Healthy,
					"attributes": info.Attributes,
				})
			}
		}

		nodes[i] = map[string]any{
			"id":                     node.ID,
			"name":                   node.Name,
			"datacenter":             node.Datacenter,
			"node_class":             node.NodeClass,
			"node_pool":              node.NodePool,
			"address":                node.Address,
			"version":                node.Version,
			"drain":                  node.Drain,
			"status":                 node.Status,
			"status_description":     node.StatusDescription,
			"scheduling_eligibility": node.SchedulingEligibility,
			"attributes":             node.Attributes,
			"drivers":                drivers,
			"node_resources":         flattenNodeResources(node.NodeResources),
			"reserved_resources":     flattenReservedResources(node.ReservedResources),
		}
	}
	log.Printf("[DEBUG] Read %d nodes", len(nodes))

	d.SetId(id)
	return d.Set("nodes", nodes)
}
