package nomad

import (
	"fmt"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceACLRoles() *schema.Resource {
	return &schema.Resource{
		Read: aclRolesDataSourceRead,

		Schema: map[string]*schema.Schema{
			"prefix": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"acl_roles": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "The ACL Role unique identifier.",
							Computed:    true,
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
							Description: "The ACL policies applied to the role.",
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
				},
			},
		},
	}
}

func aclRolesDataSourceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(ProviderConfig).client

	qOpts := &api.QueryOptions{
		Prefix: d.Get("prefix").(string),
	}
	aclRoles, _, err := client.ACLRoles().List(qOpts)
	if err != nil {
		return fmt.Errorf("failed to list ACL Roles: %v", err)
	}

	result := make([]map[string]interface{}, len(aclRoles))
	for i, aclRole := range aclRoles {

		policies := make([]map[string]interface{}, len(aclRole.Policies))
		for i, policyLink := range aclRole.Policies {
			policies[i] = map[string]interface{}{"name": policyLink.Name}
		}

		result[i] = map[string]interface{}{
			"id":          aclRole.ID,
			"name":        aclRole.Name,
			"description": aclRole.Description,
			"policies":    policies,
		}
	}

	d.SetId("nomad-roles")
	return d.Set("acl_roles", result)
}
