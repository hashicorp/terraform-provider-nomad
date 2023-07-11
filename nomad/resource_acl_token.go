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

func resourceACLToken() *schema.Resource {
	return &schema.Resource{
		Create: resourceACLTokenCreate,
		Update: resourceACLTokenUpdate,
		Delete: resourceACLTokenDelete,
		Read:   resourceACLTokenRead,
		Exists: resourceACLTokenExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"accessor_id": {
				Description: "Nomad-generated ID for this token.",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"secret_id": {
				Description: "The value that grants access to Nomad.",
				Computed:    true,
				Sensitive:   true,
				Type:        schema.TypeString,
			},
			"name": {
				Description: "Human-readable name for this token.",
				Optional:    true,
				Type:        schema.TypeString,
			},
			"type": {
				Description: "The type of token to create, 'client' or 'management'.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"policies": {
				Description: "The ACL policies to associate with the token, if it's a 'client' type.",
				Optional:    true,
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"role": {
				Description: "The roles that should be applied to the token. It may be used multiple times.",
				Optional:    true,
				Type:        schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The ID of the ACL role to link.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the ACL role linked.",
						},
					},
				},
			},
			"global": {
				Description: "Whether the token should be replicated to all regions or not.",
				Optional:    true,
				Type:        schema.TypeBool,
				ForceNew:    true,
				Default:     false,
			},
			"create_time": {
				Description: "The timestamp the token was created.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"expiration_ttl": {
				Description: `Provides a TTL for the token in the form of a time duration such as "5m" or "1h".`,
				Optional:    true,
				Default:     "0s",
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"expiration_time": {
				Description: "The point after which a token is considered expired and eligible for destruction.",
				Computed:    true,
				Type:        schema.TypeString,
			},
		},
	}
}

func resourceACLTokenCreate(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	token, err := resourceACLTokenGenerate(d)
	if err != nil {
		return err
	}

	// create our token
	log.Println("[DEBUG] Creating ACL token")
	resp, _, err := client.ACLTokens().Create(token, nil)
	if err != nil {
		return fmt.Errorf("error creating ACL token: %s", err.Error())
	}
	log.Printf("[DEBUG] Created ACL token %q", resp.AccessorID)
	d.SetId(resp.AccessorID)

	return resourceACLTokenRead(d, meta)
}

func resourceACLTokenUpdate(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	token, err := resourceACLTokenGenerate(d)
	if err != nil {
		return err
	}

	// update the token
	log.Printf("[DEBUG] Updating ACL token %q", d.Id())
	_, _, err = client.ACLTokens().Update(token, nil)
	if err != nil {
		return fmt.Errorf("error updating ACL token %q: %s", d.Id(), err.Error())
	}
	log.Printf("[DEBUG] Updated ACL token %q", d.Id())

	return resourceACLTokenRead(d, meta)
}

func resourceACLTokenDelete(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client
	accessor := d.Id()

	// delete the token
	log.Printf("[DEBUG] Deleting ACL token %q", accessor)
	_, err := client.ACLTokens().Delete(accessor, nil)
	if err != nil {
		return fmt.Errorf("error deleting ACL token %q: %s", accessor, err.Error())
	}
	log.Printf("[DEBUG] Deleted ACL token %q", accessor)

	return nil
}

func resourceACLTokenRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client
	accessor := d.Id()

	// retrieve the token
	log.Printf("[DEBUG] Reading ACL token %q", accessor)
	token, _, err := client.ACLTokens().Info(accessor, nil)
	if err != nil {
		// we have Exists, so no need to handle 404
		return fmt.Errorf("error reading ACL token %q: %s", accessor, err.Error())
	}
	log.Printf("[DEBUG] Read ACL token %q", accessor)

	var expirationTime string
	if token.ExpirationTime != nil {
		expirationTime = token.ExpirationTime.Format(time.RFC3339)
	}

	roles := make([]map[string]interface{}, len(token.Roles))
	for i, roleLink := range token.Roles {
		roles[i] = map[string]interface{}{"id": roleLink.ID, "name": roleLink.Name}
	}

	d.Set("accessor_id", token.AccessorID)
	d.Set("secret_id", token.SecretID)
	d.Set("name", token.Name)
	d.Set("type", token.Type)
	d.Set("policies", token.Policies)
	d.Set("role", roles)
	d.Set("global", token.Global)
	d.Set("create_time", token.CreateTime.UTC().String())
	d.Set("expiration_ttl", token.ExpirationTTL.String())
	d.Set("expiration_time", expirationTime)

	return nil
}

func resourceACLTokenExists(d *schema.ResourceData, meta interface{}) (bool, error) {
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

// resourceACLTokenGenerate takes the resource data and converts this into a
// valid ACL Token object. Any error returned is fatal to this run of Terraform
// and indicates a user error when configuring certain schema values.
func resourceACLTokenGenerate(d *schema.ResourceData) (*api.ACLToken, error) {

	policies := make([]string, 0, len(d.Get("policies").(*schema.Set).List()))
	for _, pol := range d.Get("policies").(*schema.Set).List() {
		policies = append(policies, pol.(string))
	}

	roles := make([]*api.ACLTokenRoleLink, 0, len(d.Get("role").(*schema.Set).List()))
	for _, raw := range d.Get("role").(*schema.Set).List() {
		role := raw.(map[string]interface{})
		roles = append(roles, &api.ACLTokenRoleLink{ID: role["id"].(string)})
	}

	token := api.ACLToken{
		AccessorID: d.Id(),
		Name:       d.Get("name").(string),
		Type:       d.Get("type").(string),
		Policies:   policies,
		Roles:      roles,
		Global:     d.Get("global").(bool),
	}

	// Identify and parse the expiration TTL if this has been set by the user.
	if ttlString := d.Get("expiration_ttl").(string); ttlString != "" {
		ttl, err := time.ParseDuration(ttlString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expiration_ttl: %v", err)
		}
		token.ExpirationTTL = ttl
	}

	return &token, nil
}
