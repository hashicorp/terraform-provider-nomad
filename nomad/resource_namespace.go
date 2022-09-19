package nomad

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
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
				Type:        schema.TypeSet,
				Elem:        resourceNamespaceCapabilities(),
				MaxItems:    1,
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

	namespace := api.Namespace{
		Name:         d.Get("name").(string),
		Description:  d.Get("description").(string),
		Quota:        d.Get("quota").(string),
		Meta:         m,
		Capabilities: capabilities,
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
			_, err = client.Namespaces().Delete(name, nil)
		}

		if err == nil {
			break
		} else if retries < 10 {
			if strings.Contains(err.Error(), "has non-terminal jobs") {
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

func flattenNamespaceCapabilities(capabilities *api.NamespaceCapabilities) *schema.Set {
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

	result := []interface{}{rawCapabilities}
	return schema.NewSet(schema.HashResource(resourceNamespaceCapabilities()), result)
}

func expandNamespaceCapabilities(d *schema.ResourceData) (*api.NamespaceCapabilities, error) {
	capabilitiesI := d.Get("capabilities").(*schema.Set).List()
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
	return &res, nil
}
