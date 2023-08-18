// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"log"

	"github.com/dustin/go-humanize"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceExternalVolume() *schema.Resource {
	return &schema.Resource{
		DeprecationMessage: "nomad_external_volume is deprecated and may be removed in a future release. Use nomad_csi_volume instead.",

		Create: resourceExternalVolumeCreate,
		Update: resourceExternalVolumeCreate,
		Delete: resourceExternalVolumeDelete,

		// Once created, external volumes are automatically registered as a
		// normal volume.
		Read: resourceVolumeRead,

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
				ForceNew:    true,
				Description: "Defines how small the volume can be. The storage provider may return a volume that is larger than this value.",
				Optional:    true,
				Type:        schema.TypeString,
			},

			"capacity_max": {
				ForceNew:    true,
				Description: "Defines how large the volume can be. The storage provider may return a volume that is smaller than this value.",
				Optional:    true,
				Type:        schema.TypeString,
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

func resourceExternalVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	// Parse capacities from human-friendly string to number.
	var capacityMin uint64
	if capacityMinStr := d.Get("capacity_min").(string); capacityMinStr != "" {
		var err error
		capacityMin, err = humanize.ParseBytes(capacityMinStr)
		if err != nil {
			return fmt.Errorf("invalid value 'capacity_min': %v", err)
		}
	}

	var capacityMax uint64
	if capacityMaxStr := d.Get("capacity_max").(string); capacityMaxStr != "" {
		var err error
		capacityMax, err = humanize.ParseBytes(capacityMaxStr)
		if err != nil {
			return fmt.Errorf("invalid value 'capacity_max': %v", err)
		}
	}

	// Parse capabilities set.
	capabilities, err := parseVolumeCapabilities(d.Get("capability"))
	if err != nil {
		return fmt.Errorf("failed to unpack capabilities: %v", err)
	}

	topologyRequest, err := parseVolumeTopologyRequest(d.Get("topology_request"))
	if err != nil {
		return fmt.Errorf("failed to unpack topology request: %v", err)
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
		Secrets:               toMapStringString(d.Get("secrets")),
		Parameters:            toMapStringString(d.Get("parameters")),
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
		if mountFlagsList, ok := mountOptsMap["mount_flags"].([]interface{}); ok {
			volume.MountOptions.MountFlags = []string{}
			for _, rawflag := range mountFlagsList {
				volume.MountOptions.MountFlags = append(volume.MountOptions.MountFlags, rawflag.(string))
			}
		}
	}

	// Create the volume.
	log.Printf("[DEBUG] creating volume %q in namespace %q", volume.ID, volume.Namespace)
	opts := &api.WriteOptions{
		Namespace: d.Get("namespace").(string),
	}
	if opts.Namespace == "" {
		opts.Namespace = providerConfig.config.Namespace
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}
	_, _, err = client.CSIVolumes().Create(volume, opts)
	if err != nil {
		return fmt.Errorf("error creating volume: %s", err)
	}

	log.Printf("[DEBUG] volume %q created in namespace %q", volume.ID, volume.Namespace)
	d.SetId(volume.ID)

	return resourceVolumeRead(d, meta) // populate other computed attributes
}

func resourceExternalVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	id := d.Id()
	log.Printf("[DEBUG] deleting volume: %q", id)
	opts := &api.WriteOptions{
		Namespace: d.Get("namespace").(string),
	}
	if opts.Namespace == "" {
		opts.Namespace = providerConfig.config.Namespace
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}
	err := client.CSIVolumes().Delete(id, opts)
	if err != nil {
		return fmt.Errorf("error deleting volume: %s", err)
	}

	return nil
}
