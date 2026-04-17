// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"path": {
				Description:      "The path at which the variable items will be stored, must be between 1 and 128 characters in length, be URL safe, and not include '@' or '.' characters",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: pathValidation(),
			},
			"namespace": {
				Description: "Variable namespace",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     api.DefaultNamespace,
			},
			"items": {
				Description:  "A map of strings to be added as items in the variable",
				Type:         schema.TypeMap,
				Optional:     true,
				Sensitive:    true,
				ExactlyOneOf: []string{"items", "items_wo"},
			},
			"items_wo": {
				Description:  "JSON-encoded variable items to write without storing them in Terraform state. Requires items_wo_version.",
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				WriteOnly:    true,
				ExactlyOneOf: []string{"items", "items_wo"},
				RequiredWith: []string{"items_wo_version"},
			},
			"items_wo_version": {
				Description:   "Version marker for items_wo updates. Required when using items_wo, and increment this value to apply a new write-only variable payload.",
				Type:          schema.TypeInt,
				Optional:      true,
				ConflictsWith: []string{"items"},
				ValidateFunc:  validation.IntAtLeast(1),
			},
		},
	}
}

func resourceVariableWrite(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	variable := &api.Variable{
		Namespace: d.Get("namespace").(string),
		Path:      d.Get("path").(string),
	}

	if rawItems, ok := d.GetOk("items"); ok {
		variable.Items = expandVariableItems(rawItems.(map[string]any))
	} else if d.IsNewResource() || d.HasChange("items_wo_version") {
		rawItems, diags := d.GetRawConfigAt(cty.GetAttrPath("items_wo"))
		if diags.HasError() {
			return fmt.Errorf("error reading items_wo config: %v", diags)
		}
		if rawItems.IsNull() || !rawItems.IsKnown() {
			return fmt.Errorf("items_wo must be provided when using items_wo_version")
		}

		items := map[string]string{}
		if err := json.Unmarshal([]byte(rawItems.AsString()), &items); err != nil {
			return fmt.Errorf("error decoding items_wo: %w", err)
		}
		variable.Items = items
	}

	log.Printf("[DEBUG] Upserting variable %s@%s", variable.Path, variable.Namespace)
	if _, _, err := client.Variables().Create(variable, nil); err != nil {
		return fmt.Errorf("error creating variable %s@%s: %s", variable.Path, variable.Namespace, err.Error())
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
		return fmt.Errorf("error deleting variable %s: %v", variableID, err)
	}

	log.Printf("[DEBUG] Deleted variable %q", d.Id())
	d.SetId("")

	return nil
}

func resourceVariableRead(d *schema.ResourceData, meta any) error {
	client := meta.(ProviderConfig).client

	path := d.Get("path").(string)
	ns := d.Get("namespace").(string)
	variableID := path + "@" + ns

	log.Printf("[DEBUG] Reading variable %s", variableID)
	variable, _, err := client.Variables().Read(path, &api.QueryOptions{Namespace: ns})
	if err != nil {
		return fmt.Errorf("error getting information about %s: %v", variableID, err)
	}

	d.SetId(variableID)

	if d.Get("items_wo_version").(int) > 0 {
		return d.Set("items", nil)
	}

	return d.Set("items", variable.Items)
}

func resourceVariableExists(d *schema.ResourceData, meta any) (bool, error) {
	client := meta.(ProviderConfig).client
	variableID := d.Id()

	log.Printf("[DEBUG] Checking if variable %q exists", variableID)

	path := d.Get("path").(string)
	ns := d.Get("namespace").(string)

	_, _, err := client.Variables().Read(path, &api.QueryOptions{Namespace: ns})
	if err != nil {
		if strings.Contains(err.Error(), "404") || errors.Is(err, api.ErrVariablePathNotFound) {
			return false, nil
		}
		return true, fmt.Errorf("error checking for variable %s: %#v", variableID, err)
	}

	return true, nil
}

func expandVariableItems(rawItems map[string]any) map[string]string {
	items := make(map[string]string, len(rawItems))
	for name, value := range rawItems {
		items[name] = value.(string)
	}

	return items
}

func pathValidation() schema.SchemaValidateDiagFunc {
	return func(i any, k cty.Path) diag.Diagnostics {
		// Verify path actually is a string
		path, ok := i.(string)
		if !ok {
			return diag.Errorf("expected type of %s to be string", k)
		}

		var diags diag.Diagnostics

		// Limit length to 128 characters
		if len(path) > maxPathLength {
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       fmt.Sprintf("expected path legnth to be less than %v but got a path legnth of %v", maxPathLength, len(path)),
				AttributePath: k,
			})
		}

		// Limit to RFC3986 URL-safe characters, minus '@' and '.' as they conflict with template blocks
		if !validVariablePath.MatchString(path) {
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       fmt.Sprintf("path %s contains invalid characters", path),
				AttributePath: k,
			})
		}

		// Validate paths that have 'nomad' as the first directory in the path
		// The following validation is taken from:
		// https://github.com/hashicorp/nomad/blob/c7c8bd4ce2fcd9f2cd13278043b7baf1f8dd4f47/nomad/structs/variables.go#L191
		parts := strings.Split(path, "/")

		if parts[0] != "nomad" {
			return diags
		}

		if len(parts) == 1 {
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "path of 'nomad' is a reserved top-level directory path",
				AttributePath: k,
			})
			return diags
		}

		switch {
		case parts[1] == "jobs":
			return diags
		case parts[1] == "job-templates" && len(parts) == 3:
			return diags
		case parts[1] == "job-templates":
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "the path 'nomad/job-templates' is reserved, you may write variables at the level below it, for example, 'nomad/job-templates/template-name'",
				AttributePath: k,
			})
		default:
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "only paths at 'nomad/jobs' or 'nomad/job-templates' and below are valid paths under the top-level 'nomad' directory",
				AttributePath: k,
			})
		}

		return diags
	}
}
