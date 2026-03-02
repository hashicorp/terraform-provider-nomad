// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-nomad/nomad/helper"
)

func dataSourceNode() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNodeRead,

		Schema: map[string]*schema.Schema{
			"node_id": {
				Description: "The ID of the node.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"datacenter": {
				Description: "The datacenter of the node.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"name": {
				Description: "The name of the node.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"http_addr": {
				Description: "The HTTP address of the node.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"attributes": {
				Description: "A map of attributes for the node.",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			"meta": {
				Description: "A map of metadata for the node.",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
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

			"drivers": {
				Description: "A map of driver information for the node.",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeBool,
				},
				Computed: true,
			},
			"host_volumes": {
				Description: "A map of host volumes on the node.",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			"node_resources": {
				Description: "Resources available on the node.",
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
				Description: "Resources reserved on the node.",
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
	}
}

func dataSourceNodeRead(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	nodeID := d.Get("node_id").(string)
	log.Printf("[DEBUG] Reading node %q", nodeID)

	node, _, err := client.Nodes().Info(nodeID, nil)
	if err != nil {
		return fmt.Errorf("error reading node %q: %w", nodeID, err)
	}
	if node == nil {
		return fmt.Errorf("node %q not found", nodeID)
	}
	log.Printf("[DEBUG] Read node %q", nodeID)

	sw := helper.NewStateWriter(d)
	sw.Set("name", node.Name)
	sw.Set("datacenter", node.Datacenter)
	sw.Set("node_class", node.NodeClass)
	sw.Set("node_pool", node.NodePool)
	sw.Set("http_addr", node.HTTPAddr)
	sw.Set("drain", node.Drain)
	sw.Set("status", node.Status)
	sw.Set("status_description", node.StatusDescription)
	sw.Set("scheduling_eligibility", node.SchedulingEligibility)
	sw.Set("attributes", node.Attributes)
	sw.Set("meta", node.Meta)

	// Flatten drivers to a simple map of driver name -> detected (bool)
	drivers := make(map[string]bool)
	for name, info := range node.Drivers {
		if info != nil {
			drivers[name] = info.Detected
		}
	}
	sw.Set("drivers", drivers)

	// Flatten host volumes to a simple map of volume name -> path
	hostVolumes := make(map[string]string)
	for name, info := range node.HostVolumes {
		if info != nil {
			hostVolumes[name] = info.Path
		}
	}
	sw.Set("host_volumes", hostVolumes)

	// Set node resources
	sw.Set("node_resources", flattenNodeResources(node.NodeResources))
	sw.Set("reserved_resources", flattenReservedResources(node.ReservedResources))

	if err := sw.Error(); err != nil {
		return err
	}

	d.SetId(nodeID)
	return nil
}

func flattenNodeResources(r *api.NodeResources) []map[string]any {
	if r == nil {
		return nil
	}

	// Flatten CPU
	cpu := []map[string]any{{
		"cpu_shares":      r.Cpu.CpuShares,
		"total_cpu_cores": int(r.Cpu.TotalCpuCores),
		"reservable_cpu_cores": func() []int {
			cores := make([]int, len(r.Cpu.ReservableCpuCores))
			for i, c := range r.Cpu.ReservableCpuCores {
				cores[i] = int(c)
			}
			return cores
		}(),
	}}

	// Flatten Memory
	memory := []map[string]any{{
		"memory_mb": r.Memory.MemoryMB,
	}}

	// Flatten Disk
	disk := []map[string]any{{
		"disk_mb": r.Disk.DiskMB,
	}}

	// Flatten Networks
	networks := make([]map[string]any, len(r.Networks))
	for i, n := range r.Networks {
		networks[i] = map[string]any{
			"device": n.Device,
			"cidr":   n.CIDR,
			"ip":     n.IP,
			"mode":   n.Mode,
		}
	}

	// Flatten Devices
	devices := make([]map[string]any, len(r.Devices))
	for i, d := range r.Devices {
		instanceCount := 0
		if d.Instances != nil {
			instanceCount = len(d.Instances)
		}
		devices[i] = map[string]any{
			"vendor": d.Vendor,
			"type":   d.Type,
			"name":   d.Name,
			"count":  instanceCount,
		}
	}

	return []map[string]any{{
		"cpu":              cpu,
		"memory":           memory,
		"disk":             disk,
		"networks":         networks,
		"devices":          devices,
		"min_dynamic_port": r.MinDynamicPort,
		"max_dynamic_port": r.MaxDynamicPort,
	}}
}

func flattenReservedResources(r *api.NodeReservedResources) []map[string]any {
	if r == nil {
		return nil
	}

	// Flatten reserved CPU
	cpu := []map[string]any{{
		"cpu_shares": int(r.Cpu.CpuShares),
	}}

	// Flatten reserved Memory
	memory := []map[string]any{{
		"memory_mb": int(r.Memory.MemoryMB),
	}}

	// Flatten reserved Disk
	disk := []map[string]any{{
		"disk_mb": int(r.Disk.DiskMB),
	}}

	// Flatten reserved Networks as a map
	networks := map[string]string{
		"reserved_host_ports": r.Networks.ReservedHostPorts,
	}

	return []map[string]any{{
		"cpu":      cpu,
		"memory":   memory,
		"disk":     disk,
		"networks": networks,
	}}
}
