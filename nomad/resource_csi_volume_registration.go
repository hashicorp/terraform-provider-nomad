// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-nomad/nomad/helper"
)

func resourceCSIVolumeRegistration() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCSIVolumeRegistrationCreate,
		UpdateContext: resourceCSIVolumeRegistrationCreate,
		DeleteContext: resourceCSIVolumeRegistrationDelete,
		Read:          resourceCSIVolumeRead,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// the following cannot be updated without destroying:
			// - Namespace/ID
			// - PluginID
			// - ExternalID

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
				Optional:    true,
				Type:        schema.TypeSet,
				MinItems:    1,
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
						// https://developer.hashicorp.com/nomad/docs/other-specifications/volume/topology_request#preferred
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
		},
	}
}

func resourceCSIVolumeRegistrationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	var parsingDiags diag.Diagnostics

	capabilities, err := parseCSIVolumeCapabilities(d.Get("capability"))
	if err != nil {
		parsingDiags = append(parsingDiags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to unpack capabilities: %v", err),
		})
	}

	topologyRequest, err := parseCSIVolumeTopologyRequest(d.Get("topology_request"))
	if err != nil {
		parsingDiags = append(parsingDiags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to unpack topology request: %v", err),
		})
	}

	// Check for parsing errors before creating the resource.
	if parsingDiags.HasError() {
		return parsingDiags
	}

	volume := &api.CSIVolume{
		ID:                    d.Get("volume_id").(string),
		Name:                  d.Get("name").(string),
		ExternalID:            d.Get("external_id").(string),
		RequestedCapabilities: capabilities,
		RequestedTopologies:   topologyRequest,
		Secrets:               helper.ToMapStringString(d.Get("secrets")),
		Parameters:            helper.ToMapStringString(d.Get("parameters")),
		Context:               helper.ToMapStringString(d.Get("context")),
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
			return diag.Errorf("failed to unpack mount_options configuration block")
		}

		mountOptsMap, ok := mountOptsList[0].(map[string]interface{})
		if !ok {
			return diag.Errorf("failed to unpack mount_options configuration block")
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
	log.Printf("[DEBUG] registering CSI volume %q in namespace %q", volume.ID, volume.Namespace)
	opts := &api.WriteOptions{
		Namespace: d.Get("namespace").(string),
	}
	if opts.Namespace == "" {
		opts.Namespace = providerConfig.config.Namespace
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}

	return diag.FromErr(retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate)-time.Minute, func() *retry.RetryError {
		_, err = client.CSIVolumes().Register(volume, opts)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("error registering CSI volume: %s", err))
		}

		log.Printf("[DEBUG] CSI volume %q registered in namespace %q", volume.ID, volume.Namespace)
		d.SetId(volume.ID)

		err := resourceCSIVolumeRead(d, meta) // populate other computed attributes
		if err != nil {
			return retry.NonRetryableError(err)
		}

		return nil
	}))
}

func resourceCSIVolumeRegistrationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
	log.Printf("[DEBUG] deregistering CSI volume: %q", id)
	opts := &api.WriteOptions{
		Namespace: d.Get("namespace").(string),
	}
	if opts.Namespace == "" {
		opts.Namespace = providerConfig.config.Namespace
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}

	return diag.FromErr(retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete)-time.Minute, func() *retry.RetryError {
		err := client.CSIVolumes().Deregister(id, true, opts)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("error deregistering CSI volume: %s", err))
		}

		return nil
	}))
}

func resourceCSIVolumeRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	id := d.Id()
	opts := &api.QueryOptions{
		Namespace: d.Get("namespace").(string),
	}
	if opts.Namespace == "" {
		opts.Namespace = providerConfig.config.Namespace
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}
	log.Printf("[DEBUG] reading information for CSI volume %q in namespace %q", id, opts.Namespace)
	volume, _, err := client.CSIVolumes().Info(id, opts)
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			log.Printf("[DEBUG] CSI volume %q does not exist, so removing", id)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("error checking for CSI volume: %s", err)
	}
	log.Printf("[DEBUG] found CSI volume %q in namespace %q", volume.Name, volume.Namespace)

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
	d.Set("topologies", flattenCSIVolumeTopologies(volume.Topologies))
	d.Set("topology_request", flattenCSIVolumeTopologyRequests(volume.RequestedTopologies))
	// The Nomad API redacts `mount_options` and `secrets`, so we don't update them
	// with the response payload; they will remain as is.

	return nil
}

func parseCSIVolumeCapabilities(i interface{}) ([]*api.CSIVolumeCapability, error) {
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
func parseCSIVolumeTopologyRequest(i interface{}) (*api.CSITopologyRequest, error) {
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
		req.Required, err = parseCSIVolumeTopologies("required", required)
		if err != nil {
			return nil, fmt.Errorf("failed to parse required CSI topology: %v", err)
		}
	}

	if preferred, ok := topologyMap["preferred"]; ok {
		req.Preferred, err = parseCSIVolumeTopologies("preferred", preferred)
		if err != nil {
			return nil, fmt.Errorf("failed to parse preferred CSI topology: %v", err)
		}
	}

	return req, nil
}

// parseVolumeTopologis parses a Terraform state representation of volume
// topology into its Nomad API representation.
func parseCSIVolumeTopologies(prefix string, i interface{}) ([]*api.CSITopology, error) {
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
func flattenCSIVolumeTopologies(topologies []*api.CSITopology) []interface{} {
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

// flattenCSIVolumeTopologyRequests turns a list of Nomad API CSITopologyRequest structs into
// the flat representation used by Terraform.
func flattenCSIVolumeTopologyRequests(topologyReqs *api.CSITopologyRequest) []interface{} {
	if topologyReqs == nil {
		return nil
	}

	topologyRequestList := make([]interface{}, 1)
	topologyMap := make(map[string]interface{}, 1)
	topologyRequestList[0] = topologyMap

	if topologyReqs.Required != nil {
		topologyMap["required"] = []any{
			map[string]any{
				"topology": flattenCSIVolumeTopologies(topologyReqs.Required),
			},
		}
	}
	if topologyReqs.Preferred != nil {
		topologyMap["preferred"] = []any{
			map[string]any{
				"topology": flattenCSIVolumeTopologies(topologyReqs.Preferred),
			},
		}
	}

	return topologyRequestList
}
