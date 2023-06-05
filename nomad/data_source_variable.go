// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceVariable() *schema.Resource {
	return &schema.Resource{
		Read: resourceVariableRead,

		Schema: map[string]*schema.Schema{
			"path": {
				Description: "The path of the variable",
				Type:        schema.TypeString,
				Required:    true,
			},
			"namespace": {
				Description: "Variable namespace",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     api.DefaultNamespace,
			},
			"items": {
				Description: "A map of values from the stored variable",
				Type:        schema.TypeMap,
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}
