// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	// maxPathLength is defined by Nomad here:
	// https://github.com/hashicorp/nomad/blob/c7c8bd4ce2fcd9f2cd13278043b7baf1f8dd4f47/nomad/structs/variables.go#LL158
	maxPathLength = 128
)

var (
	validVariablePath = regexp.MustCompile("^[a-zA-Z0-9-_~/]+$")
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
				Description:  "The path at which the variable items will be stored, must be between 1 and 128 characters in length, be URL safe, and not include '@' or '.' characters",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: pathValidation(),
			},
			"namespace": {
				Description: "Variable namespace",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     api.DefaultNamespace,
			},
			"items": {
				Description: "A map of strings to be added as items in the variable",
				Type:        schema.TypeMap,
				Required:    true,
				Sensitive:   true,
			},
		},
	}
}

func resourceVariableWrite(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	variable := &api.Variable{
		Namespace: d.Get("namespace").(string),
		Path:      d.Get("path").(string),
		Items:     make(map[string]string),
	}

	for name, value := range d.Get("items").(map[string]any) {
		variable.Items[name] = value.(string)
	}

	log.Printf("[DEBUG] Upserting Variable %s@%s", variable.Path, variable.Namespace)
	if _, _, err := client.Variables().Create(variable, nil); err != nil {
		return fmt.Errorf("error creating Variable %s@%s: %s", variable.Path, variable.Namespace, err.Error())
	}

	log.Printf("[DEBUG] Created variable %s@%s", variable.Path, variable.Namespace)
	d.SetId(variable.Path + "@" + variable.Namespace)

	return resourceVariableRead(d, meta)
}

func resourceVariableDelete(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	variableID := d.Id()
	path := d.Get("path").(string)
	ns := d.Get("namespace").(string)

	log.Printf("[DEBUG] Deleting variable %q", variableID)
	if _, err := client.Variables().Delete(path, &api.WriteOptions{Namespace: ns}); err != nil {
		return fmt.Errorf("Failed to delete Variable %s: %v", variableID, err)
	}

	log.Printf("[DEBUG] Deleted Variable %q", d.Id())
	d.SetId("")

	return nil
}

func resourceVariableRead(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	variableID := d.Id()
	// If the variable has not been created, the ID will be an empty string
	// which means we can skip attempting to perform the lookup.
	if variableID == "" {
		return nil
	}

	path := d.Get("path").(string)
	ns := d.Get("namespace").(string)

	log.Printf("[DEBUG] Reading variable %s", variableID)
	variable, _, err := client.Variables().Read(path, &api.QueryOptions{Namespace: ns})
	if err != nil {
		return fmt.Errorf("Failed to get information about %s: %v", variableID, err)
	}

	return d.Set("items", variable.Items)
}

func resourceVariableExists(d *schema.ResourceData, meta any) (bool, error) {
	client := meta.(ProviderConfig).client
	variableID := d.Id()

	log.Printf("[DEBUG] Checking if Variable %q exists", variableID)

	path := d.Get("path").(string)
	ns := d.Get("namespace").(string)

	_, _, err := client.Variables().Read(path, &api.QueryOptions{Namespace: ns})
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}

		return true, fmt.Errorf("error checking for Variable %s: %#v", variableID, err)
	}

	return true, nil
}

func pathValidation() schema.SchemaValidateFunc {
	return func(i interface{}, k string) ([]string, []error) {
		// Verify path actually is a string
		path, ok := i.(string)
		if !ok {
			return nil, []error{fmt.Errorf("expected type of %s to be string", k)}
		}

		// Limit length to 128 characters
		if len(path) > maxPathLength {
			return nil, []error{fmt.Errorf("expecnted path legnth to be less than %v but got a path legnth of %v", maxPathLength, len(path))}
		}

		// Limit to RFC3986 URL-safe characters, minus '@' and '.' as they conflict with template blocks
		if !validVariablePath.MatchString(path) {
			return nil, []error{fmt.Errorf("the path %s contains invalid characters", path)}
		}

		// Validate paths that have 'nomad' as the first directory in the path
		// The following validation is taken from:
		// https://github.com/hashicorp/nomad/blob/c7c8bd4ce2fcd9f2cd13278043b7baf1f8dd4f47/nomad/structs/variables.go#L191
		parts := strings.Split(path, "/")

		if parts[0] != "nomad" {
			return nil, nil
		}

		if len(parts) == 1 {
			return nil, []error{fmt.Errorf("path of 'nomad' is a reserved top-level directory path")}
		}

		switch {
		case parts[1] == "jobs":
			return nil, nil
		case parts[1] == "job-templates" && len(parts) == 3:
			return nil, nil
		case parts[1] == "job-templates":
			return nil, []error{fmt.Errorf("the path 'nomad/job-templates' is reserved, you may write variables at the level below it, for example, 'nomad/job-templates/template-name'")}
		default:
			return nil, []error{fmt.Errorf("only paths at 'nomad/jobs' or 'nomad/job-templates' and below are valid paths under the top-level 'nomad' directory")}
		}

		return nil, nil
	}
}
