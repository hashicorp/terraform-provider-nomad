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

func resourceACLBootstrap() *schema.Resource {
	return &schema.Resource{
		Create: resourceACLBootstrapCreate,
		Delete: resourceACLBootstrapDelete,
		Read:   resourceACLBootstrapRead,
		// Exists: resourceACLBootstrapExists,

		Schema: map[string]*schema.Schema{
			"accessor_id": {
				Description: "Nomad-generated ID for this token.",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"bootstrap_token": {
				Description: "The value that grants access to Nomad.",
				Computed:    true,
				Optional:    true,
				Sensitive:   true,
				Type:        schema.TypeString,
			},
			"type": {
				Description: "The type of token. This will always be 'management' for this resource.",
				Computed:    true,
				Optional:    true,
				Type:        schema.TypeString,
			},
			"roles": {
				Description: "The roles associated with this token.",
				Computed:    true,
				Optional:    true,
				Type:        schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"global": {
				Description: "Whether this token is global.",
				Computed:    true,
				Optional:    true,
				Type:        schema.TypeBool,
			},
			"create_time": {
				Description: "The time this token was created in RFC3339 formate",
				Computed:    true,
				Optional:    true,
				Type:        schema.TypeString,
			},
		},
	}
}

func resourceACLBootstrapCreate(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	token := api.BootstrapRequest{}
	if v, ok := d.GetOk("bootstrap_token"); ok {
		token.BootstrapSecret = v.(string)
	}
	// create our token
	log.Println("[DEBUG] Creating ACL Bootstrap token with %v", token.BootstrapSecret)
	resp, _, err := client.ACLTokens().BootstrapOpts(token.BootstrapSecret, nil)
	if err != nil {
		// Check if bootstrap was already done
		if strings.Contains(err.Error(), "ACL bootstrap already done") {
			return fmt.Errorf("ACL bootstrap has already been performed on this cluster. Cannot bootstrap again")
		}
		return fmt.Errorf("error bootstrapping the cluster: %w", err)
	}
	log.Printf("[DEBUG] Created ACL token AccessorID %q", resp.AccessorID)
	log.Printf("[DEBUG] Created ACL token %q", resp)
	d.SetId(resp.AccessorID)
	d.Set("accessor_id", resp.AccessorID)
	d.Set("bootstrap_token", resp.SecretID)
	d.Set("name", resp.Name)
	d.Set("type", resp.Type)
	d.Set("roles", resp.Roles)
	d.Set("global", resp.Global)
	d.Set("create_time", resp.CreateTime.Format(time.RFC3339))

	return nil
}

// not implemented as a cluster bootstrap can't be reverted
func resourceACLBootstrapDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceACLBootstrapRead(d *schema.ResourceData, meta interface{}) error {
	// providerConfig := meta.(ProviderConfig)
	// client := providerConfig.client
	// accessor := d.Id()

	// log.Printf("[DEBUG] Reading ACL bootstrap token %q", accessor)
	// token, _, err := client.ACLTokens().Info(accessor, nil)
	// if err != nil {
	// 	if strings.Contains(err.Error(), "404") {
	// 	}
	// 	return fmt.Errorf("error reading ACL token %q: %s", accessor, err.Error())
	// }
	// log.Printf("[DEBUG] Read ACL bootstrap token %q", accessor)
	//
	// d.Set("accessor_id", token.AccessorID)
	// d.Set("bootstrap_token", token.SecretID)

	return nil
}

// func resourceACLBootstrapExists(d *schema.ResourceData, meta interface{}) (bool, error) {
// 	providerConfig := meta.(ProviderConfig)
// 	client := providerConfig.client
//
// 	accessor := d.Id()
// 	log.Printf("[DEBUG] Checking if ACL token %q exists", accessor)
// 	_, _, err := client.ACLTokens().Info(accessor, nil)
// 	if err != nil {
// 		return true, fmt.Errorf("error checking for ACL token %q: %#v", accessor, err)
// 	}
//
// 	return true, nil
// }
