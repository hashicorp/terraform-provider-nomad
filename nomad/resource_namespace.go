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
)

func resourceNamespace() *schema.Resource {
	return &schema.Resource{
		Create: resourceNamespaceWrite,
		Update: resourceNamespaceWrite,
		Delete: resourceNamespaceDelete,
		Read:   resourceNamespaceRead,
		Exists: resourceNamespaceExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Unique name for this namespace.",
				Required:    true,
				Type:        schema.TypeString,
				ForceNew:    true,
			},

			"description": {
				Description: "Description for this namespace.",
				Optional:    true,
				Type:        schema.TypeString,
			},

			"quota": {
				Description: "Quota to set for this namespace.",
				Optional:    true,
				Type:        schema.TypeString,
			},

			"meta": {
				Description: "Metadata associated with the namespace.",
				Optional:    true,
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"capabilities": {
				Description: "Capabilities of the namespace.",
				Optional:    true,
				Type:        schema.TypeList,
				Elem:        resourceNamespaceCapabilities(),
				MaxItems:    1,
			},
			"node_pool_config": {
				Description: "Node pool configuration",
				Optional:    true,
				Type:        schema.TypeList,
				Elem:        resourceNamespaceNodePoolConfig(),
				MaxItems:    1,

				// Set as computed because in Nomad Enterprise the default node
				// pool is set to `default` if not set.
				Computed: true,
			},
		},
	}
}

func resourceNamespaceCapabilities() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"enabled_task_drivers": {
				Description: "Enabled task drivers for the namespace.",
				Optional:    true,
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"disabled_task_drivers": {
				Description: "Disabled task drivers for the namespace.",
				Optional:    true,
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"enabled_network_modes": {
				Description: "Enabled network modes for the namespace.",
				Optional:    true,
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"disabled_network_modes": {
				Description: "Disabled network modes for the namespace.",
				Optional:    true,
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceNamespaceNodePoolConfig() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"default": {
				Description: "The node pool to use when none are specified in the job.",
				Optional:    true,
				Computed:    true,
				Type:        schema.TypeString,
			},
			"allowed": {
				Description: "The list of node pools allowed to be used in this namespace. Cannot be used with denied.",
				Optional:    true,
				Type:        schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"node_pool_config.0.denied"},
			},
			"denied": {
				Description: "The list of node pools not allowed to be used in this namespace. Cannot be used with allowed.",
				Optional:    true,
				Type:        schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"node_pool_config.0.allowed"},
			},
		},
	}
}

func resourceNamespaceWrite(d *schema.ResourceData, meta interface{}) error {
	client := meta.(ProviderConfig).client

	m := make(map[string]string)
	for name, value := range d.Get("meta").(map[string]interface{}) {
		m[name] = value.(string)
	}

	capabilities, err := expandNamespaceCapabilities(d)
	if err != nil {
		return err
	}

	npConfig, err := expandNamespaceNodePoolConfig(d)
	if err != nil {
		return err
	}

	namespace := api.Namespace{
		Name:                  d.Get("name").(string),
		Description:           d.Get("description").(string),
		Quota:                 d.Get("quota").(string),
		Meta:                  m,
		Capabilities:          capabilities,
		NodePoolConfiguration: npConfig,
	}

	log.Printf("[DEBUG] Upserting namespace %q", namespace.Name)
	if _, err := client.Namespaces().Register(&namespace, nil); err != nil {
		return fmt.Errorf("error inserting namespace %q: %s", namespace.Name, err.Error())
	}
	log.Printf("[DEBUG] Created namespace %q", namespace.Name)
	d.SetId(namespace.Name)

	return resourceNamespaceRead(d, meta)
}

func resourceNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(ProviderConfig).client
	name := d.Id()

	log.Printf("[DEBUG] Deleting namespace %q", name)
	retries := 0
	for {
		var err error
		if name == api.DefaultNamespace {
			log.Printf("[DEBUG] Can't delete default namespace, clearing attributes instead")
			d.Set("description", "Default shared namespace")
			d.Set("quota", "")
			err = resourceNamespaceWrite(d, meta)
		} else {
			// make sure there are no quota specs associated with that namespace
			if d.Get("quota") != "" {
				d.Set("quota", "")
				err = resourceNamespaceWrite(d, meta)
				if err != nil {
					return err
				}
			}
			_, err = client.Namespaces().Delete(name, nil)
		}

		if err == nil {
			break
		} else if retries < 10 {
			if strings.Contains(err.Error(), "has non-terminal jobs") || strings.Contains(err.Error(), "has non-terminal allocations") {
				log.Printf("[WARN] could not delete namespace %q because of non-terminal jobs, will pause and retry", name)
				time.Sleep(5 * time.Second)
				retries++
				continue
			}
			return fmt.Errorf("error deleting namespace %q: %s", name, err.Error())
		} else {
			return fmt.Errorf("too many failures attempting to delete namespace %q: %s", name, err.Error())
		}
	}

	if name == api.DefaultNamespace {
		log.Printf("[DEBUG] %s namespace reset", name)
	} else {
		log.Printf("[DEBUG] Deleted namespace %q", name)
	}

	return nil
}

func resourceNamespaceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(ProviderConfig).client
	name := d.Id()

	log.Printf("[DEBUG] Reading namespace %q", name)
	namespace, _, err := client.Namespaces().Info(name, nil)
	if err != nil {
		// we have Exists, so no need to handle 404
		return fmt.Errorf("error reading namespace %q: %s", name, err.Error())
	}
	log.Printf("[DEBUG] Read namespace %q", name)

	d.Set("name", namespace.Name)
	d.Set("description", namespace.Description)
	d.Set("quota", namespace.Quota)
	d.Set("meta", namespace.Meta)
	d.Set("capabilities", flattenNamespaceCapabilities(namespace.Capabilities))
	d.Set("node_pool_config", flattenNamespaceNodePoolConfig(namespace.NodePoolConfiguration))

	return nil
}

func resourceNamespaceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(ProviderConfig).client

	name := d.Id()
	log.Printf("[DEBUG] Checking if namespace %q exists", name)
	resp, _, err := client.Namespaces().Info(name, nil)
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		// there's an open issue to resolve this situation:
		// https://github.com/hashicorp/nomad/issues/1849
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}

		return true, fmt.Errorf("error checking for namespace %q: %#v", name, err)
	}
	if resp == nil {
		// just to be sure
		log.Printf("[DEBUG] Response was nil, namespace %q doesn't exist", name)
		return false, nil
	}

	return true, nil
}

func flattenNamespaceCapabilities(capabilities *api.NamespaceCapabilities) []any {
	if capabilities == nil {
		return nil
	}
	rawCapabilities := map[string]interface{}{}
	if capabilities.EnabledTaskDrivers != nil {
		enabledI := make([]interface{}, len(capabilities.EnabledTaskDrivers))
		for i, v := range capabilities.EnabledTaskDrivers {
			enabledI[i] = v
		}
		rawCapabilities["enabled_task_drivers"] = enabledI
	}
	if capabilities.DisabledTaskDrivers != nil {
		disabledI := make([]interface{}, len(capabilities.DisabledTaskDrivers))
		for i, v := range capabilities.DisabledTaskDrivers {
			disabledI[i] = v
		}
		rawCapabilities["disabled_task_drivers"] = disabledI
	}
	if capabilities.EnabledNetworkModes != nil {
		enabledI := make([]interface{}, len(capabilities.EnabledNetworkModes))
		for i, v := range capabilities.EnabledNetworkModes {
			enabledI[i] = v
		}
		rawCapabilities["enabled_network_modes"] = enabledI
	}
	if capabilities.DisabledNetworkModes != nil {
		disabledI := make([]interface{}, len(capabilities.DisabledNetworkModes))
		for i, v := range capabilities.DisabledNetworkModes {
			disabledI[i] = v
		}
		rawCapabilities["disabled_network_modes"] = disabledI
	}

	return []any{rawCapabilities}
}

func expandNamespaceCapabilities(d *schema.ResourceData) (*api.NamespaceCapabilities, error) {
	capabilitiesI := d.Get("capabilities").([]any)
	if len(capabilitiesI) < 1 {
		return nil, nil
	}
	capabilities, ok := capabilitiesI[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected map[string]interface{} for region capabilities, got %T", capabilitiesI[0])
	}
	var res api.NamespaceCapabilities
	if enaI, ok := capabilities["enabled_task_drivers"]; ok {
		enaS, ok := enaI.([]interface{})
		if !ok {
			return nil, fmt.Errorf("expected enabled_task_drivers to be []string, got %T", enaS)
		}
		res.EnabledTaskDrivers = make([]string, len(enaS))
		for index, value := range enaS {
			if val, ok := value.(string); ok {
				res.EnabledTaskDrivers[index] = val
			}
		}
	}
	if disI, ok := capabilities["disabled_task_drivers"]; ok {
		disS, ok := disI.([]interface{})
		if !ok {
			return nil, fmt.Errorf("expected disabled_task_drivers to be []string, got %T", disS)
		}
		res.DisabledTaskDrivers = make([]string, len(disS))
		for index, value := range disS {
			if val, ok := value.(string); ok {
				res.DisabledTaskDrivers[index] = val
			}
		}
	}
	if enaI, ok := capabilities["enabled_network_modes"]; ok {
		enaS, ok := enaI.([]interface{})
		if !ok {
			return nil, fmt.Errorf("expected enabled_network_modes to be []string, got %T", enaS)
		}
		res.EnabledNetworkModes = make([]string, len(enaS))
		for index, value := range enaS {
			if val, ok := value.(string); ok {
				res.EnabledNetworkModes[index] = val
			}
		}
	}
	if disI, ok := capabilities["disabled_network_modes"]; ok {
		disS, ok := disI.([]interface{})
		if !ok {
			return nil, fmt.Errorf("expected disabled_network_modes to be []string, got %T", disS)
		}
		res.DisabledNetworkModes = make([]string, len(disS))
		for index, value := range disS {
			if val, ok := value.(string); ok {
				res.DisabledNetworkModes[index] = val
			}
		}
	}
	return &res, nil
}

func expandNamespaceNodePoolConfig(d *schema.ResourceData) (*api.NamespaceNodePoolConfiguration, error) {
	npConfigI := d.Get("node_pool_config").([]any)
	if len(npConfigI) < 1 {
		return nil, nil
	}

	npConfig, ok := npConfigI[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected map[string]interface{} for namespace node pool configuration, got %T", npConfigI[0])
	}

	var res api.NamespaceNodePoolConfiguration

	if defaultI, ok := npConfig["default"]; ok {
		defaultStr, ok := defaultI.(string)
		if !ok {
			return nil, fmt.Errorf("expected default to be a string, got %T", defaultI)
		}
		res.Default = defaultStr

	}

	if allowedI, ok := npConfig["allowed"]; ok {
		allowedSet, ok := allowedI.(*schema.Set)
		if !ok {
			return nil, fmt.Errorf("expected allowed to be a *schema.Set, got %T", allowedI)
		}
		if allowedSet.Len() > 0 {
			res.Allowed = make([]string, allowedSet.Len())
			for i, v := range allowedSet.List() {
				res.Allowed[i] = v.(string)
			}
		}
	}

	if deniedI, ok := npConfig["denied"]; ok {
		deniedSet, ok := deniedI.(*schema.Set)
		if !ok {
			return nil, fmt.Errorf("expected denied to be a *schema.Set, got %T", deniedI)
		}
		if deniedSet.Len() > 0 {
			res.Denied = make([]string, deniedSet.Len())
			for i, v := range deniedSet.List() {
				res.Denied[i] = v.(string)
			}
		}
	}

	return &res, nil
}
func flattenNamespaceNodePoolConfig(npConfig *api.NamespaceNodePoolConfiguration) []any {
	if npConfig == nil {
		return nil
	}

	rawNpConfig := map[string]any{
		"default": npConfig.Default,
	}

	if npConfig.Allowed != nil {
		allowed := make([]any, len(npConfig.Allowed))
		for i, v := range npConfig.Allowed {
			allowed[i] = v
		}
		rawNpConfig["allowed"] = allowed
	}

	if npConfig.Denied != nil {
		denied := make([]any, len(npConfig.Denied))
		for i, v := range npConfig.Denied {
			denied[i] = v
		}
		rawNpConfig["denied"] = denied
	}

	return []any{rawNpConfig}
}
