package nomad

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceACLRole() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceACLRoleRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The ACL Role unique identifier.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"name": {
				Description: "Unique name of the ACL role.",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"description": {
				Description: "Description for the ACL role.",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"policies": {
				Description: "The list of policies applied to the role.",
				Computed:    true,
				Type:        schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the ACL policy to link.",
						},
					},
				},
			},
		},
	}
}

func dataSourceACLRoleRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	roleID := d.Get("id").(string)

	log.Printf("[DEBUG] Reading ACL Role %q", roleID)
	aclRole, _, err := client.ACLRoles().Get(roleID, nil)
	if err != nil {

		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading ACL Role %q: %s", roleID, err.Error())
	}
	log.Printf("[DEBUG] Read ACL Role %q", roleID)

	policies := make([]map[string]interface{}, len(aclRole.Policies))
	for i, policyLink := range aclRole.Policies {
		policies[i] = map[string]interface{}{"name": policyLink.Name}
	}

	d.SetId(aclRole.ID)
	d.Set("name", aclRole.Name)
	d.Set("description", aclRole.Description)
	d.Set("policies", policies)

	return nil
}
