// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-nomad/nomad/helper"
)

func dataSourceDynamicHostVolume() *schema.Resource {
	return &schema.Resource{
		Read: dynamicHostVolumeReadWithCapacity,
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "Volume ID",
				Type:        schema.TypeString,
				Required:    true,
			},
			"namespace": {
				Description: "Volume namespace",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "default",
			},

			// computed attributes
			"name": {
				Description: "Volume name",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"capability": {
				Description: "Capability",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_mode": {
							Description: "An access mode available for the volume.",
							Type:        schema.TypeString,
							Optional:    false,
							Computed:    true,
						},

						"attachment_mode": {
							Description: "An attachment mode available for the volume.",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
			"capacity": {
				Description: "Provisioned capacity",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"capacity_max": {
				Description: "Requested maximum capacity",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"capacity_min": {
				Description: "Requested minimum capacity",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"constraint": {
				Description: "Constraints",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attribute": {
							Description: "An attribute to check to constrain volume placement",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"value": {
							Description: "The requested value of the attribute",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"operator": {
							Description: "The operator to use for comparison",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
			"host_path": {
				Description: "Host path",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"node_id": {
				Description: "Node ID",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"node_pool": {
				Description: "Node pool",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"parameters": {
				Description: "Parameters",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed: true,
			},
			"plugin_id": {
				Description: "Plugin ID",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"state": {
				Description: "State",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"capacity_bytes": {
				Computed: true,
				Type:     schema.TypeInt,
			},
			"capacity_min_bytes": {
				Computed: true,
				Type:     schema.TypeInt,
			},
			"capacity_max_bytes": {
				Computed: true,
				Type:     schema.TypeInt,
			},
		},
	}
}

// dynamicHostVolumeRead gets a dynamic host volume from Nomad. Used by the
// registration resource and as the base for dynamicHostVolumeReadWithCapacity.
func dynamicHostVolumeRead(d *schema.ResourceData, meta any) error {
	return dynamicHostVolumeReadImpl(d, meta, false)
}

// dynamicHostVolumeReadWithCapacity reads a dynamic host volume and also sets
// the capacity_min and capacity_max string fields. Used by the create resource
// and the data source which have these fields in their schema.
func dynamicHostVolumeReadWithCapacity(d *schema.ResourceData, meta any) error {
	return dynamicHostVolumeReadImpl(d, meta, true)
}

func dynamicHostVolumeReadImpl(d *schema.ResourceData, meta any, setCapacityStrings bool) error {
	client := meta.(ProviderConfig).client

	ns, id := getDynamicHostVolumeNamespacedID(d)
	vol, _, err := client.HostVolumes().Get(id, &api.QueryOptions{Namespace: ns})
	if err != nil {
		return fmt.Errorf("Failed to get information about %q: %v", id, err)
	}

	sw := helper.NewStateWriter(d)
	mapping := map[string]any{
		"capability":         flattenDynamicHostVolumeCapabilities(vol.RequestedCapabilities),
		"capacity_bytes":     vol.CapacityBytes,
		"capacity_max_bytes": vol.RequestedCapacityMaxBytes,
		"capacity_min_bytes": vol.RequestedCapacityMinBytes,
		"constraint":         flattenConstraints(vol.Constraints),
		"host_path":          vol.HostPath,
		"id":                 vol.ID,
		"name":               vol.Name,
		"namespace":          vol.Namespace,
		"node_id":            vol.NodeID,
		"node_pool":          vol.NodePool,
		"parameters":         vol.Parameters,
		"plugin_id":          vol.PluginID,
		"state":              vol.State,
	}
	for k, v := range mapping {
		sw.Set(k, v)
	}

	// Always set capacity from the API so state reflects reality.
	sw.Set("capacity", humanize.IBytes(uint64(vol.CapacityBytes)))

	// capacity_min and capacity_max only exist on the create resource and data
	// source schemas, not on the registration resource.
	if setCapacityStrings {
		sw.Set("capacity_min", humanize.IBytes(uint64(vol.RequestedCapacityMinBytes)))
		sw.Set("capacity_max", humanize.IBytes(uint64(vol.RequestedCapacityMaxBytes)))
	}

	d.SetId(id)
	return nil
}

// getDynamicHostVolumeNamespacedID returns the namespace and ID for a dynamic
// host volume from the resource data
func getDynamicHostVolumeNamespacedID(d *schema.ResourceData) (string, string) {
	id := d.Get("id").(string)
	ns := d.Get("namespace").(string)
	if ns == "" {
		ns = "default"
	}

	return ns, id
}
