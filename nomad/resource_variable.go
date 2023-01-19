// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceVariable() *schema.Resource {
	return &schema.Resource{
		Create: resourceVariableWrite,
		Update: resourceVariableWrite,
		Delete: resourceVariableDelete,
		Read:   resourceVariableRead,
		Exists: resourceVariableExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"path": {
				Type:     schema.TypeString,
				Required: true,
			},
			"namespace": {
				Description: "Variable namespace filter",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     api.DefaultNamespace,
			},
			"items": {
				Type:      schema.TypeMap,
				Required:  true,
				Sensitive: true,
			},
		},
	}
}

func resourceVariableWrite(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	variable := api.Variable{
		Namespace: d.Get("namespace").(string),
		Path:      d.Get("path").(string),
		Items:     make(map[string]string),
	}
	for name, value := range d.Get("items").(map[string]any) {
		variable.Items[name] = value.(string)
	}

	log.Printf("[DEBUG] Upserting Variable %s@%s", variable.Path, variable.Namespace)
	if _, _, err := client.Variables().Create(&variable, nil); err != nil {
		return fmt.Errorf("error inserting variable %s@%s: %s", variable.Path, variable.Namespace, err.Error())
	}
	log.Printf("[DEBUG] Created variable %s@%s", variable.Path, variable.Namespace)

	return resourceVariableRead(d, meta)
}

func resourceVariableDelete(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	path := d.Get("path").(string)
	ns := d.Get("namespace").(string)
	d.SetId(path + "@" + ns)

	log.Printf("[DEBUG] Deleting variable %q", d.Id())
	if _, err := client.Variables().Delete(path, &api.WriteOptions{Namespace: ns}); err != nil {
		return fmt.Errorf("Failed to delete secret %s@%s: %v", path, ns, err)
	}
	log.Printf("[DEBUG] Deleted variable %q", d.Id())

	return nil
}

func resourceVariableRead(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	path := d.Get("path").(string)
	ns := d.Get("namespace").(string)

	log.Printf("[DEBUG] Reading variable %q", d.Id())
	variable, _, err := client.Variables().Read(path, &api.QueryOptions{Namespace: ns})
	if err != nil {
		return fmt.Errorf("Failed to get information about %s@%s: %v", path, ns, err)
	}
	d.SetId(path + "@" + ns)
	return d.Set("items", variable.Items)
}

func resourceVariableExists(d *schema.ResourceData, meta any) (bool, error) {
	err := resourceVariableRead(d, meta)
	return err == nil, nil
}
