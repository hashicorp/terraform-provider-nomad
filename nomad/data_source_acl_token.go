// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceACLToken() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceACLTokenRead,
		Schema: map[string]*schema.Schema{
			"accessor_id": {
				Description: "Non-sensitive identifier for this token.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"secret_id": {
				Description: "The token value itself.",
				Computed:    true,
				Sensitive:   true,
				Type:        schema.TypeString,
			},
			"name": {
				Description: "Human-friendly name of the ACL token.",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"type": {
				Description: "The type of the token.",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"policies": {
				Description: "List of policy names associated with this token.",
				Computed:    true,
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"roles": {
				Description: "The roles that are applied to the token.",
				Computed:    true,
				Type:        schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the ACL role.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the ACL role.",
						},
					},
				},
			},
			"global": {
				Description: "Whether the token is replicated to all regions, or if it will only be used in the region it was created.",
				Computed:    true,
				Type:        schema.TypeBool,
			},
			"create_time": {
				Description: "Date and time the token was created.",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"expiration_ttl": {
				Description: "The expiration TTL for the token.",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"expiration_time": {
				Description: "The point after which a token is considered revoked and eligible for destruction.",
				Computed:    true,
				Type:        schema.TypeString,
			},
		},
	}
}

func dataSourceACLTokenRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client
	accessor := d.Get("accessor_id").(string)

	// retrieve the token
	log.Printf("[DEBUG] Reading ACL Token %q", accessor)
	token, _, err := client.ACLTokens().Info(accessor, nil)
	if err != nil {

		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}

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

	d.SetId(accessor)
	d.Set("name", token.Name)
	d.Set("type", token.Type)
	d.Set("policies", token.Policies)
	d.Set("roles", token.Roles)
	d.Set("secret_id", token.SecretID)
	d.Set("global", token.Global)
	d.Set("create_time", token.CreateTime.UTC().String())
	d.Set("expiration_ttl", token.ExpirationTTL.String())
	d.Set("expiration_time", expirationTime)

	return nil
}
