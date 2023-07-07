// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-nomad/nomad/helper"
	"github.com/hashicorp/terraform-provider-nomad/nomad/helper/pointer"
)

const (
	nodePoolMemoryOversubscriptionEnabled  = "enabled"
	nodePoolMemoryOversubscriptionDisabled = "disabled"
)

func resourceNodePool() *schema.Resource {
	return &schema.Resource{
		Create: resourceNodePoolWrite,
		Update: resourceNodePoolWrite,
		Delete: resourceNodePoolDelete,
		Read:   resourceNodePoolRead,
		Exists: resourceNodePoolExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Unique name for this node pool.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"description": {
				Description: "Description for this node pool.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"meta": {
				Description: "Metadata associated with the node pool",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"scheduler_config": {
				Description: "Scheduler configuration for the node pool.",
				Type:        schema.TypeList,
				Elem:        resourceNodePoolSchedulerConfiguration(),
				MaxItems:    1,
				Optional:    true,
			},
		},
	}
}

func resourceNodePoolSchedulerConfiguration() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"scheduler_algorithm": {
				Description: "The scheduler algorithm to use in the node pool.",
				Type:        schema.TypeString,
				Optional:    true,
				ValidateFunc: validation.StringInSlice([]string{
					string(api.SchedulerAlgorithmBinpack),
					string(api.SchedulerAlgorithmSpread),
				}, false),
			},

			// This field must be a string instead of a bool (and differ from
			// Nomad) in order to represent a tristate.
			// - "enabled": memory oversubscription is enabled in the pool.
			// - "disabled": memory oversubscription is disabled in the pool.
			// - "": the global memory oversubscription value is used.
			"memory_oversubscription": {
				Description: "If true, the node pool will have memory oversubscription enabled.",
				Type:        schema.TypeString,
				Optional:    true,
				ValidateFunc: validation.StringInSlice([]string{
					nodePoolMemoryOversubscriptionEnabled,
					nodePoolMemoryOversubscriptionDisabled,
				}, false),
			},
		},
	}

}

func resourceNodePoolRead(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client
	name := d.Id()

	log.Printf("[DEBUG] Reading node pool %q", name)
	pool, _, err := client.NodePools().Info(name, nil)
	if err != nil {
		// we have Exists, so no need to handle 404
		return fmt.Errorf("error reading node pool %q: %w", name, err)
	}
	log.Printf("[DEBUG] Read node pool %q", name)

	sw := helper.NewStateWriter(d)
	sw.Set("name", pool.Name)
	sw.Set("description", pool.Description)
	sw.Set("meta", pool.Meta)
	sw.Set("scheduler_config", flattenNodePoolSchedulerConfiguration(pool.SchedulerConfiguration))

	return sw.Error()
}

func resourceNodePoolWrite(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	m := make(map[string]string)
	for name, value := range d.Get("meta").(map[string]any) {
		m[name] = value.(string)
	}

	schedConfig, err := expandNodePoolSchedulerConfiguration(d)
	if err != nil {
		return fmt.Errorf("failed to parse node pool scheduler configuration: %w", err)
	}

	pool := &api.NodePool{
		Name:                   d.Get("name").(string),
		Description:            d.Get("description").(string),
		Meta:                   m,
		SchedulerConfiguration: schedConfig,
	}

	log.Printf("[DEBUG] Upserting node pool %q", pool.Name)
	if _, err := client.NodePools().Register(pool, nil); err != nil {
		return fmt.Errorf("error upserting node pool %q: %w", pool.Name, err)
	}
	log.Printf("[DEBUG] Upserted node pool %q", pool.Name)
	d.SetId(pool.Name)

	return resourceNodePoolRead(d, meta)
}

func resourceNodePoolDelete(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client
	name := d.Id()

	log.Printf("[DEBUG] Deleting node pool %q", name)
	retries := 0
	for {
		_, err := client.NodePools().Delete(name, nil)
		if err == nil {
			break
		}

		retry := strings.Contains(err.Error(), "has non-terminal jobs") ||
			strings.Contains(err.Error(), "has nodes")
		if retries < 10 && retry {
			log.Printf("[INFO] could not delete node pool %q, retrying: %v", name, err)
			time.Sleep(5 * time.Second)
			retries++
			continue
		}

		return fmt.Errorf("failed to delete node pool %q: %w", name, err)
	}
	log.Printf("[DEBUG] Deleted node pool %q", name)

	return nil
}

func resourceNodePoolExists(d *schema.ResourceData, meta any) (bool, error) {
	client := meta.(ProviderConfig).client

	name := d.Id()
	log.Printf("[DEBUG] Checking if node pool %q exists", name)
	resp, _, err := client.NodePools().Info(name, nil)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}

		return true, fmt.Errorf("error checking for node pool %q: %#v", name, err)
	}
	if resp == nil {
		// just to be sure
		log.Printf("[DEBUG] Response was nil, node pool %q doesn't exist", name)
		return false, nil
	}

	return true, nil
}

func expandNodePoolSchedulerConfiguration(d *schema.ResourceData) (*api.NodePoolSchedulerConfiguration, error) {
	rawConfig := d.Get("scheduler_config").([]any)
	if len(rawConfig) < 1 {
		return nil, nil
	}

	config, ok := rawConfig[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T for node pool scheduler configuration", rawConfig[0])
	}

	result := &api.NodePoolSchedulerConfiguration{}

	if rawAlgo, ok := config["scheduler_algorithm"]; ok {
		algo, ok := rawAlgo.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected type %T for scheduler algorithm", rawAlgo)
		}
		result.SchedulerAlgorithm = api.SchedulerAlgorithm(algo)
	}

	if rawMemOverSub, ok := config["memory_oversubscription"]; ok {
		memOverSub, ok := rawMemOverSub.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected type %T for memory oversubcription enabled", rawMemOverSub)
		}
		switch memOverSub {
		case nodePoolMemoryOversubscriptionEnabled:
			result.MemoryOversubscriptionEnabled = pointer.Of(true)
		case nodePoolMemoryOversubscriptionDisabled:
			result.MemoryOversubscriptionEnabled = pointer.Of(false)
		}
	}
	return result, nil
}

func flattenNodePoolSchedulerConfiguration(config *api.NodePoolSchedulerConfiguration) []any {
	if config == nil {
		return nil
	}

	rawConfig := map[string]any{
		"scheduler_algorithm": string(config.SchedulerAlgorithm),
	}

	if config.MemoryOversubscriptionEnabled != nil {
		if *config.MemoryOversubscriptionEnabled {
			rawConfig["memory_oversubscription"] = nodePoolMemoryOversubscriptionEnabled
		} else {
			rawConfig["memory_oversubscription"] = nodePoolMemoryOversubscriptionDisabled
		}
	}

	return []any{rawConfig}
}
