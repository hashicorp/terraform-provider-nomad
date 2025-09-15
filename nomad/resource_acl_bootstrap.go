// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceACLBootstrap() *schema.Resource {
	return &schema.Resource{
		Create: resourceACLBootstrapCreate,
		Delete: resourceACLBootstrapDelete,
		Read:   resourceACLBootstrapRead,
		Exists: resourceACLBootstrapExists,

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
		},
	}
}

func resourceACLBootstrapCreate(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	token := api.BootstrapRequest{
		BootstrapSecret: d.Get("bootstrap_token").(string),
	}
	// create our token
	log.Println("[DEBUG] Creating ACL Bootstrap token")
	resp, _, err := client.ACLTokens().BootstrapOpts(token.BootstrapSecret, nil)
	if err != nil {
		return fmt.Errorf("error bootstrapping the cluster: %w", err)
	}
	log.Printf("[DEBUG] Created ACL token AccessorID %q", resp.AccessorID)
	d.SetId(resp.AccessorID)

	return resourceACLBootstrapRead(d, meta)
}

// not implemented as a cluster bootstrap can't be reverted
func resourceACLBootstrapDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceACLBootstrapRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client
	accessor := d.Id()

	// retrieve the token
	log.Printf("[DEBUG] Reading ACL bootstrap token %q", accessor)
	token, _, err := client.ACLTokens().Info(accessor, nil)
	if err != nil {
		// we have Exists, so no need to handle 404
		return fmt.Errorf("error reading ACL token %q: %s", accessor, err.Error())
	}
	log.Printf("[DEBUG] Read ACL bootstrap token %q", accessor)

	d.Set("accessor_id", token.AccessorID)
	d.Set("bootstrap_token", token.SecretID)

	return nil
}

func resourceACLBootstrapExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	accessor := d.Id()
	log.Printf("[DEBUG] Checking if ACL token %q exists", accessor)
	_, _, err := client.ACLTokens().Info(accessor, nil)
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}

		return true, fmt.Errorf("error checking for ACL token %q: %#v", accessor, err)
	}

	return true, nil
}
