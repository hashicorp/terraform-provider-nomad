// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceDynamicHostVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceDynamicHostVolumeWrite,
		Update: resourceDynamicHostVolumeWrite,
		Delete: resourceDynamicHostVolumeDelete,
		Read:   dynamicHostVolumeRead,
		Exists: resourceDynamicHostVolumeExists,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
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
			"plugin_id": {
				Description: "Plugin ID",
				Type:        schema.TypeString,
				Required:    true,
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
			"capacity_max": {
				Description: "Requested maximum capacity",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"capacity_min": {
				Description: "Requested minimum capacity",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"constraint": {
				Description: "Constraints",
				Type:        schema.TypeList,
				Optional:    true,
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
			"node_id": {
				Description: "Node ID",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ValidateFunc: validation.All(
					validation.IsUUID,
				),
			},
			"node_pool": {
				Description: "Node pool",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			"parameters": {
				Description: "Parameters",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},

			// computed attributes
			"id": {
				Description: "Volume ID",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"capacity": {
				Description: "Provisioned capacity",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"host_path": {
				Description: "Host path",
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

func resourceDynamicHostVolumeWrite(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	parameters := make(map[string]string)
	for name, value := range d.Get("parameters").(map[string]any) {
		parameters[name] = value.(string)
	}

	constraints, err := expandDynamicHostVolumeConstraints(d)
	if err != nil {
		return err
	}

	caps, err := expandDynamicHostVolumeCapabilities(d)
	if err != nil {
		return err
	}

	var capacityMin, capacityMax uint64
	min := d.Get("capacity_min").(string)
	if min != "" {
		capacityMin, err = humanize.ParseBytes(min)
		if err != nil {
			return fmt.Errorf("could not parse capacity_min value as bytes: %w", err)
		}
	}
	max := d.Get("capacity_max").(string)
	if max != "" {
		capacityMax, err = humanize.ParseBytes(max)
		if err != nil {
			return fmt.Errorf("could not parse capacity_max value as bytes: %w", err)
		}
	}

	req := &api.HostVolumeCreateRequest{
		Volume: &api.HostVolume{
			Namespace:                 d.Get("namespace").(string),
			Name:                      d.Get("name").(string),
			ID:                        d.Get("id").(string),
			Constraints:               constraints,
			NodeID:                    d.Get("node_id").(string),
			NodePool:                  d.Get("node_pool").(string),
			Parameters:                parameters,
			PluginID:                  d.Get("plugin_id").(string),
			RequestedCapabilities:     caps,
			RequestedCapacityMaxBytes: int64(capacityMax),
			RequestedCapacityMinBytes: int64(capacityMin),
		},
	}

	log.Printf("[DEBUG] Upserting dynamic host volume %q", req.Volume.Name)
	resp, _, err := client.HostVolumes().Create(req, nil)
	if err != nil {
		return fmt.Errorf("error upserting node pool %q: %w", req.Volume.Name, err)
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

func dynamicHostVolumeWaitForReady(client *api.Client, ns, id string) error {
	opts := &api.QueryOptions{Namespace: ns}
	for {
		vol, qm, err := client.HostVolumes().Get(id, opts)
		if err != nil {
			return err
		}
		if vol == nil {
			return fmt.Errorf("volume not found")
		}
		if vol.State == api.HostVolumeStateReady {
			return nil
		}
		opts.WaitIndex = qm.LastIndex
		log.Printf("[DEBUG] Waiting for dynamic host volume %q to be ready", id)
	}
}

// resourceDynamicHostVolumeDelete is shared between the Create and Register workflows
func resourceDynamicHostVolumeDelete(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client
	ns, id := getDynamicHostVolumeNamespacedID(d)
	log.Printf("[DEBUG] Deleting dynamic host volume %q", id)

	_, _, err := client.HostVolumes().Delete(
		&api.HostVolumeDeleteRequest{ID: id},
		&api.WriteOptions{Namespace: ns},
	)
	if err != nil {
		return fmt.Errorf("could not delete dynamic host volume %q: %w", id, err)
	}
	return nil
}

// resourceDynamicHostVolumeExists is shared between the Create and Register workflows
func resourceDynamicHostVolumeExists(d *schema.ResourceData, meta any) (bool, error) {
	client := meta.(ProviderConfig).client
	ns, id := getDynamicHostVolumeNamespacedID(d)
	log.Printf("[DEBUG] Checking if dynamic host volume %q exists", id)

	vol, err := getDynamicHostVolume(client, ns, id)
	if err != nil {
		return false, fmt.Errorf("error checking for dynamic host volume %q: %w", id, err)
	}
	return vol != nil, nil
}

// getDynamicHostVolume is a helper function for the main resource methods
func getDynamicHostVolume(client *api.Client, ns, id string) (*api.HostVolume, error) {
	vol, _, err := client.HostVolumes().Get(id, &api.QueryOptions{Namespace: ns})
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}

		return nil, err
	}
	return vol, nil
}

func flattenConstraints(constraints []*api.Constraint) []map[string]string {
	constraintList := []map[string]string{}
	for _, c := range constraints {
		item := make(map[string]string)
		item["attribute"] = c.LTarget
		item["value"] = c.RTarget
		item["operator"] = c.Operand
		constraintList = append(constraintList, item)
	}
	return constraintList
}

func expandDynamicHostVolumeConstraints(d *schema.ResourceData) ([]*api.Constraint, error) {
	result := []*api.Constraint{}
	raw, ok := d.GetOk("constraint")
	if !ok {
		return result, nil
	}

	constraintList, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("error reading constraints: unexpected type %T", raw)
	}

	for _, rawItem := range constraintList {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("error reading constraint item: unexpected type %T", rawItem)
		}
		constraint := &api.Constraint{}
		if attr, ok := item["attribute"].(string); ok {
			constraint.LTarget = attr
		}
		if val, ok := item["value"].(string); ok {
			constraint.RTarget = val
		}
		if op, ok := item["operator"].(string); ok {
			constraint.Operand = op
		}
		result = append(result, constraint)
	}

	return result, nil
}

// flattenDynamicHostVolumeCapabilities turns a list of Nomad API
// HostVolumeCapability structs into the flat representation used by Terraform.
func flattenDynamicHostVolumeCapabilities(capabilities []*api.HostVolumeCapability) []map[string]string {
	capList := []map[string]string{}
	for _, c := range capabilities {
		if c == nil {
			continue
		}
		item := make(map[string]string)
		item["access_mode"] = string(c.AccessMode)
		item["attachment_mode"] = string(c.AttachmentMode)
		capList = append(capList, item)
	}

	return capList
}

func expandDynamicHostVolumeCapabilities(d *schema.ResourceData) ([]*api.HostVolumeCapability, error) {
	result := []*api.HostVolumeCapability{}
	raw, ok := d.GetOk("capability")
	if !ok {
		return result, nil
	}

	capList, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("error reading capabilities: unexpected type %T", raw)
	}
	for _, rawItem := range capList {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("error reading capability item: unexpected type %T", rawItem)
		}

		cap := &api.HostVolumeCapability{}
		if attach, ok := item["attachment_mode"].(string); ok {
			cap.AttachmentMode = api.HostVolumeAttachmentMode(attach)
		}
		if access, ok := item["access_mode"].(string); ok {
			cap.AccessMode = api.HostVolumeAccessMode(access)
		}

		result = append(result, cap)
	}

	return result, nil
}
