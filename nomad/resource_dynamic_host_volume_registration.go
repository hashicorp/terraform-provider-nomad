// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"

	"github.com/dustin/go-humanize"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-nomad/nomad/helper"
)

func resourceDynamicHostVolumeRegistration() *schema.Resource {
	return &schema.Resource{
		Create: resourceDynamicHostVolumeRegistrationWrite,
		Update: resourceDynamicHostVolumeRegistrationWrite,
		Delete: resourceDynamicHostVolumeDelete,
		Read:   dynamicHostVolumeRead,
		Exists: resourceDynamicHostVolumeExists,

		Importer: &schema.ResourceImporter{
			StateContext: helper.NamespacedImporterContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Volume name",
				Type:        schema.TypeString,
				Required:    true,
			},
			"namespace": {
				Description: "Volume namespace",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "default",
			},
			"capacity": {
				Description: "Provisioned capacity",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"capability": {
				Description: "Capability",
				Type:        schema.TypeList,
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_mode": {
							Description: "An access mode available for the volume.",
							Type:        schema.TypeString,
							Required:    true,
							ValidateFunc: validation.StringInSlice([]string{
								string(api.HostVolumeAccessModeSingleNodeReader),
								string(api.HostVolumeAccessModeSingleNodeWriter),
								string(api.HostVolumeAccessModeSingleNodeSingleWriter),
								string(api.HostVolumeAccessModeSingleNodeMultiWriter),
							}, false),
						},

						"attachment_mode": {
							Description: "An attachment mode available for the volume.",
							Type:        schema.TypeString,
							Required:    true,
							ValidateFunc: validation.StringInSlice([]string{
								string(api.HostVolumeAttachmentModeFilesystem),
								string(api.HostVolumeAttachmentModeBlockDevice),
							}, false),
						},
					},
				},
			},
			"host_path": {
				Description: "Host path",
				Type:        schema.TypeString,
				Required:    true,
			},
			"node_id": {
				Description: "Node ID",
				Type:        schema.TypeString,
				Required:    true,
				ValidateFunc: validation.All(
					validation.IsUUID,
				),
			},
			"parameters": {
				Description: "Parameters",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},

			// computed attributes. some of these will always be empty for
			// register, but this allows import
			"id": {
				Description: "Volume ID",
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
			"plugin_id": {
				Description: "Plugin ID",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"node_pool": {
				Description: "Node pool",
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
							Required:    true,
						},
						"value": {
							Description: "The requested value of the attribute",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"operator": {
							Description: "The operator to use for comparison",
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "=",
						},
					},
				},
			},
		},
	}
}

func resourceDynamicHostVolumeRegistrationWrite(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	parameters := make(map[string]string)
	for name, value := range d.Get("parameters").(map[string]any) {
		parameters[name] = value.(string)
	}

	var capacity uint64
	var err error
	capacityStr := d.Get("capacity").(string)
	if capacityStr != "" {
		capacity, err = humanize.ParseBytes(capacityStr)
		if err != nil {
			return fmt.Errorf("could not parse capacity value as bytes: %w", err)
		}
	}

	caps, err := expandDynamicHostVolumeCapabilities(d)
	if err != nil {
		return err
	}

	req := &api.HostVolumeRegisterRequest{
		Volume: &api.HostVolume{
			Name:                  d.Get("name").(string),
			Namespace:             d.Get("namespace").(string),
			ID:                    d.Get("id").(string),
			RequestedCapabilities: caps,
			CapacityBytes:         int64(capacity),
			HostPath:              d.Get("host_path").(string),
			NodeID:                d.Get("node_id").(string),
			NodePool:              d.Get("node_pool").(string),
			Parameters:            parameters,
		},
	}

	log.Printf("[DEBUG] Upserting dynamic host volume %q", req.Volume.Name)
	resp, _, err := client.HostVolumes().Register(req, nil)
	if err != nil {
		return fmt.Errorf("error upserting dynamic host volume %q: %w", req.Volume.Name, err)
	}
	log.Printf("[DEBUG] Upserted dynamic host volume %q", resp.Volume.ID)
	d.SetId(resp.Volume.ID)
	d.Set("namespace", resp.Volume.Namespace)

	err = dynamicHostVolumeWaitForReady(client, resp.Volume.Namespace, resp.Volume.ID)
	if err != nil {
		return fmt.Errorf("error polling for dynamic host volume readiness: %w", err)
	}

	return dynamicHostVolumeRead(d, meta)
}
