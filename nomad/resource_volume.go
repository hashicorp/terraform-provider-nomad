package nomad

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceVolume() *schema.Resource {
	return &schema.Resource{
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

			// "mount_options": {
			// 	Description: "Options for mounting 'block-device' volumes without a pre-formatted file system.",
			// 	Optional:    true,
			// 	Type:        schema.TypeMap,
			// 	Elem: &schema.Schema{
			// 		Type: schema.TypeString,
			// 	},
			// },

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

	ns := d.Get("namespace").(string)
	if ns == "" {
		ns = "default"
	}

	// TODO: mountOptions := d.Get("mount_options")

	volume := &api.CSIVolume{
		ID:             d.Get("volume_id").(string),
		Name:           d.Get("name").(string),
		ExternalID:     d.Get("external_id").(string),
		AccessMode:     api.CSIVolumeAccessMode(d.Get("access_mode").(string)),
		AttachmentMode: api.CSIVolumeAttachmentMode(d.Get("attachment_mode").(string)),
		Secrets:        toMapStringString(d.Get("secrets")),
		Parameters:     toMapStringString(d.Get("parameters")),
		Context:        toMapStringString(d.Get("context")),
		PluginID:       d.Get("plugin_id").(string),
	}

	// Register the volume
	log.Printf("[DEBUG] registering volume %q in namespace %q", volume.ID, volume.Namespace)
	_, err := client.CSIVolumes().Register(volume, nil)
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

	return nil
}
