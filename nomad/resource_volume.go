// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"log"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceVolume() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: "nomad_volume is deprecated and may be removed in a future release. Use nomad_csi_volume_registration instead.",

		Create: resourceVolumeCreate,
		Update: resourceVolumeCreate,
		Delete: resourceVolumeDelete,
		Read:   resourceVolumeRead,

		Schema: map[string]*schema.Schema{
			// the following cannot be updated without destroying:
			// - Namespace/ID
			// - PluginID
			// - ExternalID
			// - Type

			"type": {
				ForceNew:    true,
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The type of the volume. Currently, only 'csi' is supported.",
				Default:     "csi",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{"csi"}, false),
				},
			},

			"namespace": {
				ForceNew:    true,
				Description: "The namespace in which to create the volume.",
				Optional:    true,
				Default:     "default",
				Type:        schema.TypeString,
			},

			"volume_id": {
				ForceNew:    true,
				Description: "The unique ID of the volume, how jobs will refer to the volume.",
				Required:    true,
				Type:        schema.TypeString,
			},

			"name": {
				Description: "The display name of the volume.",
				Required:    true,
				Type:        schema.TypeString,
			},

			"plugin_id": {
				ForceNew:    true,
				Description: "The ID of the CSI plugin that manages this volume.",
				Required:    true,
				Type:        schema.TypeString,
			},

			"external_id": {
				ForceNew:    true,
				Description: "The ID of the physical volume from the storage provider.",
				Required:    true,
				Type:        schema.TypeString,
			},

			"capability": {
				Description: "Capabilities intended to be used in a job. At least one capability must be provided.",
				// COMPAT(1.5.0)
				// Update once `access_mode` and `attachment_mode` are removed.
				ForceNew:      false,
				Optional:      true,
				ConflictsWith: []string{"access_mode", "attachment_mode"},
				Type:          schema.TypeSet,
				MinItems:      1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_mode": {
							Description: "Defines whether a volume should be available concurrently.",
							Type:        schema.TypeString,
							Required:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"single-node-reader-only",
									"single-node-writer",
									"multi-node-reader-only",
									"multi-node-single-writer",
									"multi-node-multi-writer",
								}, false),
							},
						},
						"attachment_mode": {
							Description: "The storage API that will be used by the volume.",
							Required:    true,
							Type:        schema.TypeString,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"block-device",
									"file-system",
								}, false),
							},
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["access_mode"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["attachment_mode"].(string)))

					i := int(crc32.ChecksumIEEE(buf.Bytes()))
					if i >= 0 {
						return i
					}
					if -i >= 0 {
						return -i
					}
					// i == MinInt
					return 0
				},
			},

			"mount_options": {
				Description: "Options for mounting 'block-device' volumes without a pre-formatted file system.",
				Optional:    true,
				Type:        schema.TypeList,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"fs_type": {
							Description: "The file system type.",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"mount_flags": {
							Description: "The flags passed to mount.",
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
						},
					},
				},
			},

			"secrets": {
				Description: "An optional key-value map of strings used as credentials for publishing and unpublishing volumes.",
				Optional:    true,
				Type:        schema.TypeMap,
				Sensitive:   true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"parameters": {
				Description: "An optional key-value map of strings passed directly to the CSI plugin to configure the volume.",
				Optional:    true,
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"topology_request": {
				ForceNew:    true,
				Description: "Specify locations (region, zone, rack, etc.) where the provisioned volume is accessible from.",
				Optional:    true,
				Type:        schema.TypeList,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"required": {
							ForceNew:    true,
							Description: "Required topologies indicate that the volume must be created in a location accessible from all the listed topologies.",
							Optional:    true,
							Type:        schema.TypeList,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"topology": {
										ForceNew:    true,
										Description: "Defines the location for the volume.",
										Required:    true,
										Type:        schema.TypeList,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"segments": {
													ForceNew:    true,
													Description: "Define attributes for the topology request.",
													Required:    true,
													Type:        schema.TypeMap,
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
						// Volume registration does not support preferred topologies.
						// https://www.nomadproject.io/docs/commands/volume/register#topology_request-parameters
					},
				},
			},

			"context": {
				Description: "An optional key-value map of strings passed directly to the CSI plugin to validate the volume.",
				Optional:    true,
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"deregister_on_destroy": {
				Description: "If true, the volume will be deregistered on destroy.",
				Optional:    true,
				Default:     true,
				Type:        schema.TypeBool,
			},

			"controller_required": {
				Computed: true,
				Type:     schema.TypeBool,
			},

			"controllers_expected": {
				Computed: true,
				Type:     schema.TypeInt,
			},

			"controllers_healthy": {
				Computed: true,
				Type:     schema.TypeInt,
			},

			"plugin_provider": {
				Computed: true,
				Type:     schema.TypeString,
			},

			"plugin_provider_version": {
				Computed: true,
				Type:     schema.TypeString,
			},

			"nodes_healthy": {
				Computed: true,
				Type:     schema.TypeInt,
			},

			"nodes_expected": {
				Computed: true,
				Type:     schema.TypeInt,
			},

			"schedulable": {
				Computed: true,
				Type:     schema.TypeBool,
			},
			"topologies": {
				Computed: true,
				Type:     schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"segments": {
							Computed: true,
							Type:     schema.TypeMap,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},

			// COMPAT(1.5.0)
			// Deprecated fields.
			"access_mode": {
				Description:   "Defines whether a volume should be available concurrently.",
				Optional:      true,
				Deprecated:    "use capability instead",
				ConflictsWith: []string{"capability"},
				Type:          schema.TypeString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"single-node-reader-only",
						"single-node-writer",
						"multi-node-reader-only",
						"multi-node-single-writer",
						"multi-node-multi-writer",
					}, false),
				},
			},

			"attachment_mode": {
				Description:   "The storage API that will be used by the volume.",
				Optional:      true,
				Deprecated:    "use capability instead",
				ConflictsWith: []string{"capability"},
				Type:          schema.TypeString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"block-device",
						"file-system",
					}, false),
				},
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceVolumeResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceVolumeStateUpgradeV0,
				Version: 0,
			},
		},
	}
}

func toMapStringString(m interface{}) map[string]string {
	mss := map[string]string{}
	for k, v := range m.(map[string]interface{}) {
		mss[k] = v.(string)
	}
	return mss
}

func resourceVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	// Read capabilities maintaining backwards compatibility with the previous
	// fields attachment_mode and access_mode.
	capabilitySet, capabilitiesOk := d.GetOk("capability")
	attachmentMode, attachmentModeOk := d.GetOk("attachment_mode")
	accessMode, accessModeOk := d.GetOk("access_mode")

	if !capabilitiesOk && !attachmentModeOk && !accessModeOk {
		return errors.New(`one of "capability" or "access_mode" and "attachment_mode" must be configured`)
	}

	var capabilities []*api.CSIVolumeCapability

	if capabilitiesOk {
		var err error
		capabilities, err = parseVolumeCapabilities(capabilitySet)
		if err != nil {
			return fmt.Errorf("failed to unpack capabilities: %v", err)
		}
	} else {
		capabilities = []*api.CSIVolumeCapability{
			{
				AccessMode:     api.CSIVolumeAccessMode(accessMode.(string)),
				AttachmentMode: api.CSIVolumeAttachmentMode(attachmentMode.(string)),
			},
		}
	}

	topologyRequest, err := parseVolumeTopologyRequest(d.Get("topology_request"))
	if err != nil {
		return fmt.Errorf("failed to unpack topology request: %v", err)
	}

	volume := &api.CSIVolume{
		ID:                    d.Get("volume_id").(string),
		Name:                  d.Get("name").(string),
		ExternalID:            d.Get("external_id").(string),
		RequestedCapabilities: capabilities,
		RequestedTopologies:   topologyRequest,
		Secrets:               toMapStringString(d.Get("secrets")),
		Parameters:            toMapStringString(d.Get("parameters")),
		Context:               toMapStringString(d.Get("context")),
		PluginID:              d.Get("plugin_id").(string),

		// COMPAT(1.5.0)
		// Maintain backwards compatibility.
		AccessMode:     capabilities[0].AccessMode,
		AttachmentMode: capabilities[0].AttachmentMode,
	}

	// Unpack the mount_options if we have any and configure the volume struct.
	mountOpts, ok := d.GetOk("mount_options")
	if ok {
		mountOptsList, ok := mountOpts.([]interface{})
		if !ok || len(mountOptsList) != 1 {
			return errors.New("failed to unpack mount_options configuration block")
		}

		mountOptsMap, ok := mountOptsList[0].(map[string]interface{})
		if !ok {
			return errors.New("failed to unpack mount_options configuration block")
		}
		volume.MountOptions = &api.CSIMountOptions{}

		if val, ok := mountOptsMap["fs_type"].(string); ok {
			volume.MountOptions.FSType = val
		}
		rawMountFlags := mountOptsMap["mount_flags"].([]interface{})
		volume.MountOptions.MountFlags = make([]string, len(rawMountFlags))
		for index, value := range rawMountFlags {
			if val, ok := value.(string); ok {
				volume.MountOptions.MountFlags[index] = val
			}
		}
	}

	// Register the volume
	log.Printf("[DEBUG] registering volume %q in namespace %q", volume.ID, volume.Namespace)
	opts := &api.WriteOptions{
		Namespace: d.Get("namespace").(string),
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}
	_, err = client.CSIVolumes().Register(volume, opts)
	if err != nil {
		return fmt.Errorf("error registering volume: %s", err)
	}

	log.Printf("[DEBUG] volume %q registered in namespace %q", volume.ID, volume.Namespace)
	d.SetId(volume.ID)

	return resourceVolumeRead(d, meta) // populate other computed attributes
}

func resourceVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	// If deregistration is disabled, then do nothing
	deregister_on_destroy := d.Get("deregister_on_destroy").(bool)
	if !deregister_on_destroy {
		log.Printf(
			"[WARN] volume %q will not deregister since "+
				"'deregister_on_destroy' is %t", d.Id(), deregister_on_destroy)
		return nil
	}

	id := d.Id()
	log.Printf("[DEBUG] deregistering volume: %q", id)
	opts := &api.WriteOptions{
		Namespace: d.Get("namespace").(string),
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}
	err := client.CSIVolumes().Deregister(id, true, opts)
	if err != nil {
		return fmt.Errorf("error deregistering volume: %s", err)
	}

	return nil
}

func resourceVolumeRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	id := d.Id()
	opts := &api.QueryOptions{
		Namespace: d.Get("namespace").(string),
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}
	log.Printf("[DEBUG] reading information for volume %q in namespace %q", id, opts.Namespace)
	volume, _, err := client.CSIVolumes().Info(id, opts)
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			log.Printf("[DEBUG] volume %q does not exist, so removing", id)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("error checking for volume: %s", err)
	}
	log.Printf("[DEBUG] found volume %q in namespace %q", volume.Name, volume.Namespace)

	d.Set("name", volume.Name)
	d.Set("controller_required", volume.ControllerRequired)
	d.Set("controllers_expected", volume.ControllersExpected)
	d.Set("controllers_healthy", volume.ControllersHealthy)
	d.Set("controllers_healthy", volume.ControllersHealthy)
	d.Set("plugin_provider", volume.Provider)
	d.Set("plugin_provider_version", volume.ProviderVersion)
	d.Set("nodes_healthy", volume.NodesHealthy)
	d.Set("nodes_expected", volume.NodesExpected)
	d.Set("schedulable", volume.Schedulable)
	d.Set("topologies", flattenVolumeTopologies(volume.Topologies))
	d.Set("topology_request", flattenVolumeTopologyRequests(volume.RequestedTopologies))
	// The Nomad API redacts `mount_options` and `secrets`, so we don't update them
	// with the response payload; they will remain as is.

	return nil
}

func parseVolumeCapabilities(i interface{}) ([]*api.CSIVolumeCapability, error) {
	capabilities := []*api.CSIVolumeCapability{}

	capabilitySet, ok := i.(*schema.Set)
	if !ok {
		return nil, fmt.Errorf("invalid type %T, expected *schema.Set", i)
	}

	for _, capabilitySetItem := range capabilitySet.List() {
		capabilityMap, ok := capabilitySetItem.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid type %T, expected map[string]interface{}", capabilitySetItem)
		}

		c := &api.CSIVolumeCapability{}
		for k, v := range capabilityMap {
			switch k {
			case "access_mode":
				c.AccessMode = api.CSIVolumeAccessMode(v.(string))
			case "attachment_mode":
				c.AttachmentMode = api.CSIVolumeAttachmentMode(v.(string))
			}
		}
		capabilities = append(capabilities, c)
	}

	return capabilities, nil
}

// parseVolumeTopologyRequest parses a Terraform state representation of volume
// topology request into its Nomad API representation.
func parseVolumeTopologyRequest(i interface{}) (*api.CSITopologyRequest, error) {
	req := &api.CSITopologyRequest{}
	var err error

	topologyRequestList, ok := i.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type %T for topology_request, expected []interface{}", i)
	}

	if len(topologyRequestList) == 0 {
		return nil, nil
	}

	topologyMap, ok := topologyRequestList[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type %T for topology_request item, expected map[string]interface{}", topologyRequestList[0])
	}

	if required, ok := topologyMap["required"]; ok {
		req.Required, err = parseVolumeTopologies("required", required)
		if err != nil {
			return nil, fmt.Errorf("failed to parse required CSI topology: %v", err)
		}
	}

	if preferred, ok := topologyMap["preferred"]; ok {
		req.Preferred, err = parseVolumeTopologies("preferred", preferred)
		if err != nil {
			return nil, fmt.Errorf("failed to parse preferred CSI topology: %v", err)
		}
	}

	return req, nil
}

// parseVolumeTopologis parses a Terraform state representation of volume
// topology into its Nomad API representation.
func parseVolumeTopologies(prefix string, i interface{}) ([]*api.CSITopology, error) {
	var topologies []*api.CSITopology

	topologiesList, ok := i.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type %T for %s.topology, expected []interface{}", i, prefix)
	}

	if len(topologiesList) == 0 {
		return topologies, nil
	}

	topologiesListMap, ok := topologiesList[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type %T for %s.topology, expected map[string]interface{}", topologiesList[0], prefix)
	}

	topologiesListMapList, ok := topologiesListMap["topology"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type %T for %s.topology, expected []interface{}", topologiesListMap["topology"], prefix)
	}

	for j, topologyItem := range topologiesListMapList {
		topologyItemMap, ok := topologyItem.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf(
				"invalid type %T for %s.topology.%d, expected map[string]interface{}",
				topologyItem, prefix, j)
		}
		segmentsMap, ok := topologyItemMap["segments"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf(
				"invalid type %T for %s.topology.%d.segments, expected map[string]interface{}",
				topologyItemMap["segments"], prefix, j)
		}

		segmentsMapString := make(map[string]string, len(segmentsMap))
		for k, v := range segmentsMap {
			segmentsMapString[k] = v.(string)
		}
		topologies = append(topologies, &api.CSITopology{
			Segments: segmentsMapString,
		})
	}

	return topologies, nil
}

// flattenVolumeTopologies turns a list of Nomad API CSITopology structs into
// the flat representation used by Terraform.
func flattenVolumeTopologies(topologies []*api.CSITopology) []interface{} {
	topologiesList := []interface{}{}

	for _, topo := range topologies {
		if topo == nil {
			continue
		}
		topoItem := make(map[string]interface{})
		topoItem["segments"] = topo.Segments
		topologiesList = append(topologiesList, topoItem)
	}

	return topologiesList
}

// flattenVolumeTopologyRequests turns a list of Nomad API CSITopologyRequest structs into
// the flat representation used by Terraform.
func flattenVolumeTopologyRequests(topologyReqs *api.CSITopologyRequest) []interface{} {
	if topologyReqs == nil {
		return nil
	}

	topologyRequestList := make([]interface{}, 1)
	topologyMap := make(map[string]interface{}, 1)
	topologyRequestList[0] = topologyMap

	if topologyReqs.Required != nil {
		topologyMap["required"] = []any{
			map[string]any{
				"topology": flattenVolumeTopologies(topologyReqs.Required),
			},
		}
	}
	if topologyReqs.Preferred != nil {
		topologyMap["preferred"] = []any{
			map[string]any{
				"topology": flattenVolumeTopologies(topologyReqs.Preferred),
			},
		}
	}

	return topologyRequestList
}

// resourceVolumeStateUpgradeV0 migrates a nomad_volume resource schema from v0 to v1.
func resourceVolumeStateUpgradeV0(_ context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	if val, ok := rawState["mount_options"]; ok {
		rawState["mount_options"] = []interface{}{val}
	}
	return rawState, nil
}

// resourceVolumeResourceV0 returns the v0 schema for a nomad_volume.
func resourceVolumeResourceV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"type": {
				ForceNew:    true,
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The type of the volume. Currently, only 'csi' is supported.",
				Default:     "csi",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{"csi"}, false),
				},
			},

			"namespace": {
				ForceNew:    true,
				Description: "The namespace in which to create the volume.",
				Optional:    true,
				Default:     "default",
				Type:        schema.TypeString,
			},

			"volume_id": {
				ForceNew:    true,
				Description: "The unique ID of the volume, how jobs will refer to the volume.",
				Required:    true,
				Type:        schema.TypeString,
			},

			"name": {
				Description: "The display name of the volume.",
				Required:    true,
				Type:        schema.TypeString,
			},

			"plugin_id": {
				ForceNew:    true,
				Description: "The ID of the CSI plugin that manages this volume.",
				Required:    true,
				Type:        schema.TypeString,
			},

			"external_id": {
				ForceNew:    true,
				Description: "The ID of the physical volume from the storage provider.",
				Required:    true,
				Type:        schema.TypeString,
			},

			"access_mode": {
				Description: "Defines whether a volume should be available concurrently.",
				Required:    true,
				Type:        schema.TypeString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"single-node-reader-only",
						"single-node-writer",
						"multi-node-reader-only",
						"multi-node-single-writer",
						"multi-node-multi-writer",
					}, false),
				},
			},

			"attachment_mode": {
				Description: "The storage API that will be used by the volume.",
				Required:    true,
				Type:        schema.TypeString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"block-device",
						"file-system",
					}, false),
				},
			},

			"mount_options": {
				Description: "Options for mounting 'block-device' volumes without a pre-formatted file system.",
				Optional:    true,
				Type:        schema.TypeMap,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"fs_type": {
							Description: "The file system type.",
							Type:        schema.TypeString,
						},
						"mount_flags": {
							Description: "The flags passed to mount.",
							Type:        schema.TypeList,
						},
					},
				},
			},

			"secrets": {
				Description: "An optional key-value map of strings used as credentials for publishing and unpublishing volumes.",
				Optional:    true,
				Type:        schema.TypeMap,
				Sensitive:   true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"parameters": {
				Description: "An optional key-value map of strings passed directly to the CSI plugin to configure the volume.",
				Optional:    true,
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"context": {
				Description: "An optional key-value map of strings passed directly to the CSI plugin to validate the volume.",
				Optional:    true,
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"deregister_on_destroy": {
				Description: "If true, the volume will be deregistered on destroy.",
				Optional:    true,
				Default:     true,
				Type:        schema.TypeBool,
			},

			"controller_required": {
				Computed: true,
				Type:     schema.TypeBool,
			},

			"controllers_expected": {
				Computed: true,
				Type:     schema.TypeInt,
			},

			"controllers_healthy": {
				Computed: true,
				Type:     schema.TypeInt,
			},

			"plugin_provider": {
				Computed: true,
				Type:     schema.TypeString,
			},

			"plugin_provider_version": {
				Computed: true,
				Type:     schema.TypeString,
			},

			"nodes_healthy": {
				Computed: true,
				Type:     schema.TypeInt,
			},

			"nodes_expected": {
				Computed: true,
				Type:     schema.TypeInt,
			},

			"schedulable": {
				Computed: true,
				Type:     schema.TypeBool,
			},
		},
	}
}
