package nomad

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceACLRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceACLRoleCreate,
		Update: resourceACLRoleUpdate,
		Delete: resourceACLRoleDelete,
		Read:   resourceACLRoleRead,
		Exists: resourceACLRoleExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Unique name for this ACL role.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"description": {
				Description: "Description for this ACL role.",
				Optional:    true,
				Type:        schema.TypeString,
			},
			"policies": {
				Description: "The policies that should be applied to the role.",
				Required:    true,
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

func resourceACLRoleCreate(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	role := generateNomadACLRole(d)

	// Create our ACL role.
	log.Printf("[DEBUG] Creating ACL role")
	aclRoleCreateResp, _, err := client.ACLRoles().Create(role, nil)
	if err != nil {
		return fmt.Errorf("error creating ACL role: %s", err.Error())
	}

	d.SetId(aclRoleCreateResp.ID)
	log.Printf("[DEBUG] Created ACL role %q", aclRoleCreateResp.ID)

	return resourceACLRoleRead(d, meta)
}

func resourceACLRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	role := generateNomadACLRole(d)

	// Perform the in-place update of the ACL role.
	log.Printf("[DEBUG] Updating ACL Role %q", role.ID)
	_, _, err := client.ACLRoles().Update(role, nil)
	if err != nil {
		return fmt.Errorf("error updating ACL Role %q: %s", role.ID, err.Error())
	}
	log.Printf("[DEBUG] Updated ACL Role %q", role.ID)

	return resourceACLRoleRead(d, meta)
}

func resourceACLRoleDelete(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	roleID := d.Id()

	// Delete the ACL role.
	log.Printf("[DEBUG] Deleting ACL Role %q", roleID)
	_, err := client.ACLRoles().Delete(roleID, nil)
	if err != nil {
		return fmt.Errorf("error deleting ACL Role %q: %s", roleID, err.Error())
	}
	log.Printf("[DEBUG] Deleted ACL Role %q", roleID)

	d.SetId("")

	return nil
}

func resourceACLRoleRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client
	roleID := d.Id()

	// If the role has not been created, the ID will be an empty string which
	// means we can skip attempting to perform the lookup.
	if roleID == "" {
		return nil
	}

	log.Printf("[DEBUG] Reading ACL Role %q", roleID)
	role, _, err := client.ACLRoles().Get(roleID, nil)
	if err != nil {
		return fmt.Errorf("error reading ACL Role %q: %s", roleID, err.Error())
	}
	log.Printf("[DEBUG] Read ACL Role %q", roleID)

	policies := make([]map[string]interface{}, len(role.Policies))
	for i, policyLink := range role.Policies {
		policies[i] = map[string]interface{}{"name": policyLink.Name}
	}

	d.Set("name", role.Name)
	d.Set("description", role.Description)
	d.Set("policies", policies)

	return nil
}

func resourceACLRoleExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	roleID := d.Id()
	log.Printf("[DEBUG] Checking if ACL Role %q exists", roleID)
	_, _, err := client.ACLRoles().Get(roleID, nil)
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}

		return true, fmt.Errorf("error checking for ACL Role %q: %#v", roleID, err)
	}

	return true, nil
}

func generateNomadACLRole(d *schema.ResourceData) *api.ACLRole {

	policies := make([]*api.ACLRolePolicyLink, 0)

	for _, raw := range d.Get("policies").(*schema.Set).List() {
		s := raw.(map[string]interface{})
		policies = append(policies, &api.ACLRolePolicyLink{Name: s["name"].(string)})
	}

	return &api.ACLRole{
		ID:          d.Id(),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Policies:    policies,
	}
}
