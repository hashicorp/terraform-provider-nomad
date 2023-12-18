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

	"github.com/dustin/go-humanize"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-nomad/nomad/helper"
)

func resourceCSIVolume() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCSIVolumeCreate,
		UpdateContext: resourceCSIVolumeCreate,
		DeleteContext: resourceCSIVolumeDelete,
		Read:          resourceCSIVolumeRead,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Importer: &schema.ResourceImporter{
			StateContext: helper.NamespacedImporterContext,
		},

		Schema: map[string]*schema.Schema{
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

			"snapshot_id": {
				ForceNew:      true,
				Description:   "The snapshot ID to restore when creating this volume. Storage provider must support snapshots. Conflicts with 'clone_id'.",
				Optional:      true,
				Type:          schema.TypeString,
				ConflictsWith: []string{"clone_id"},
			},

			"clone_id": {
				ForceNew:      true,
				Description:   "The volume ID to clone when creating this volume. Storage provider must support cloning. Conflicts with 'snapshot_id'.",
				Optional:      true,
				Type:          schema.TypeString,
				ConflictsWith: []string{"snapshot_id"},
			},

			"capacity_min": {
				Description:      "Defines how small the volume can be. The storage provider may return a volume that is larger than this value.",
				Optional:         true,
				Type:             schema.TypeString,
				StateFunc:        capacityStateFunc,
				ValidateDiagFunc: capacityValidate,
			},

			"capacity_max": {
				Description:      "Defines how large the volume can be. The storage provider may return a volume that is smaller than this value.",
				Optional:         true,
				Type:             schema.TypeString,
				StateFunc:        capacityStateFunc,
				ValidateDiagFunc: capacityValidate,
			},

			"capability": {
				ForceNew:    true,
				Description: "Capabilities intended to be used in a job. At least one capability must be provided.",
				Required:    true,
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
													ForceNew:    true,
													Description: "Define the attributes for the topology request.",
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
						"preferred": {
							ForceNew:    true,
							Description: "Preferred topologies indicate that the volume should be created in a location accessible from some of the listed topologies.",
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
													Description: "Define the attributes for the topology request.",
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
					},
				},
			},

			"capacity": {
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
			"external_id": {
				Description: "The ID of the physical volume from the storage provider.",
				Computed:    true,
				Type:        schema.TypeString,
			},
		},
	}
}

// resourceCSIVolumeRead is shared between nomad_csi_volume and
// nomad_csi_volume_registration because once a volume is stored in Nomad it
// is read from the same endpoint, regardless of how it was created.
func resourceCSIVolumeRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	id := d.Id()
	opts := &api.QueryOptions{
		Namespace: d.Get("namespace").(string),
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
	d.Set("namespace", volume.Namespace)
	d.Set("volume_id", volume.ID)
	d.Set("external_id", volume.ExternalID)

	d.Set("capacity", int(volume.Capacity))
	d.Set("capacity_min_bytes", volume.RequestedCapacityMin)
	d.Set("capacity_max_bytes", volume.RequestedCapacityMax)
	// only save capacity min/max in state if the user has set it/them
	if cMin := d.Get("capacity_min").(string); cMin != "" {
		d.Set("capacity_min", humanize.IBytes(uint64(volume.RequestedCapacityMin)))
	}
	if cMax := d.Get("capacity_max").(string); cMax != "" {
		d.Set("capacity_max", humanize.IBytes(uint64(volume.RequestedCapacityMax)))
	}

	d.Set("controller_required", volume.ControllerRequired)
	d.Set("controllers_expected", volume.ControllersExpected)
	d.Set("controllers_healthy", volume.ControllersHealthy)
	d.Set("controllers_healthy", volume.ControllersHealthy)
	d.Set("plugin_id", volume.PluginID)
	d.Set("plugin_provider", volume.Provider)
	d.Set("plugin_provider_version", volume.ProviderVersion)
	d.Set("nodes_healthy", volume.NodesHealthy)
	d.Set("nodes_expected", volume.NodesExpected)
	d.Set("schedulable", volume.Schedulable)
	d.Set("capability", flattenCSIVolumeCapabilities(volume.RequestedCapabilities))
	d.Set("topologies", flattenCSIVolumeTopologies(volume.Topologies))
	d.Set("topology_request", flattenCSIVolumeTopologyRequests(volume.RequestedTopologies))
	// The Nomad API redacts `mount_options` and `secrets`, so we don't update them
	// with the response payload; they will remain as is.

	return nil
}

func resourceCSIVolumeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	var parsingDiags diag.Diagnostics

	capacityMin, capacityMax, capacityDiags := parseCapacity(d)
	parsingDiags = append(parsingDiags, capacityDiags...)

	// Parse capabilities set.
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
		PluginID:              d.Get("plugin_id").(string),
		Name:                  d.Get("name").(string),
		SnapshotID:            d.Get("snapshot_id").(string),
		CloneID:               d.Get("clone_id").(string),
		RequestedCapacityMin:  int64(capacityMin),
		RequestedCapacityMax:  int64(capacityMax),
		RequestedCapabilities: capabilities,
		RequestedTopologies:   topologyRequest,
		Secrets:               helper.ToMapStringString(d.Get("secrets")),
		Parameters:            helper.ToMapStringString(d.Get("parameters")),
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
		if mountFlagsList, ok := mountOptsMap["mount_flags"].([]interface{}); ok {
			volume.MountOptions.MountFlags = []string{}
			for _, rawflag := range mountFlagsList {
				volume.MountOptions.MountFlags = append(volume.MountOptions.MountFlags, rawflag.(string))
			}
		}
	}

	// Create the volume.
	log.Printf("[DEBUG] creating CSI volume %q in namespace %q", volume.ID, volume.Namespace)
	opts := &api.WriteOptions{
		Namespace: d.Get("namespace").(string),
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate)-time.Minute, func() *retry.RetryError {
		_, _, err = client.CSIVolumes().Create(volume, opts)
		if err != nil {
			err = fmt.Errorf("error creating CSI volume: %s", err)
			if csiErrIsRetryable(err) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		log.Printf("[DEBUG] CSI volume %q created in namespace %q", volume.ID, volume.Namespace)
		d.SetId(volume.ID)

		err := resourceCSIVolumeRead(d, meta) // populate other computed attributes
		if err != nil {
			return retry.NonRetryableError(err)
		}

		return nil
	})

	if err != nil {
		return diag.FromErr(err)
	}

	var warnings diag.Diagnostics

	capacityAfter := uint64(d.Get("capacity").(int))
	warnings = append(warnings, checkCapacity(capacityAfter, capacityMin)...)

	return warnings
}

func resourceCSIVolumeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	id := d.Id()
	log.Printf("[DEBUG] deleting CSI volume: %q", id)
	opts := &api.WriteOptions{
		Namespace: d.Get("namespace").(string),
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}

	return diag.FromErr(retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete)-time.Minute, func() *retry.RetryError {
		err := client.CSIVolumes().Delete(id, opts)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("error deleting CSI volume: %s", err))
		}
		return nil
	}))
}
