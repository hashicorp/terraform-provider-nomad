package nomad

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceACLTokens() *schema.Resource {
	return &schema.Resource{
		Read: aclTokensDataSourceRead,

		Schema: map[string]*schema.Schema{
			"prefix": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"acl_tokens": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"accessor_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"policies": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"global": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"create_time": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func aclTokensDataSourceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(ProviderConfig).client

	tokens, _, err := client.ACLTokens().List(nil)
	if err != nil {
		return fmt.Errorf("error while getting the list of tokens: %v", err)
	}

	prefix := d.Get("prefix").(string)
	result := make([]map[string]interface{}, 0)
	for _, t := range tokens {
		if !strings.HasPrefix(t.AccessorID, prefix) {
			continue
		}
		result = append(result, map[string]interface{}{
			"accessor_id": t.AccessorID,
			"name":        t.Name,
			"type":        t.Type,
			"policies":    t.Policies,
			"global":      t.Global,
			"create_time": t.CreateTime.String(),
		})
	}

	d.SetId("nomad-tokens")
	return d.Set("acl_tokens", result)
}
