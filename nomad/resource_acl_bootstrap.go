package nomad

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceACLBootStrap() *schema.Resource {
	return &schema.Resource{
		Create: resourceACLBootStrapCreate,
		Delete: resourceACLBootStrapDelete,
		Read:   resourceACLBootStrapRead,
		Exists: resourceACLBootStrapExists,

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

func resourceACLBootStrapCreate(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	token := api.BootstrapRequest{
		BootstrapSecret: d.Get("bootstrap_token").(string),
	}
	// create our token
	log.Println("[DEBUG] Creating ACL Bootstrap token")
	resp, _, err := client.ACLTokens().BootstrapOpts(token.BootstrapSecret, nil)
	if err != nil {
		return fmt.Errorf("error bootstrapping the cluster: %s", err.Error())
	}
	log.Printf("[DEBUG] Created ACL token AccessorID %q", resp.AccessorID)
	d.SetId(resp.AccessorID)

	return resourceACLBootStrapRead(d, meta)
}

// not implemented as a cluster bootstrap can't be reverted
func resourceACLBootStrapDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceACLBootStrapRead(d *schema.ResourceData, meta interface{}) error {
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

	var expirationTime string
	if token.ExpirationTime != nil {
		expirationTime = token.ExpirationTime.Format(time.RFC3339)
	}
	token.SecretID = "00000000-0000-0000-0000-000000000000"

	d.Set("accessor_id", token.AccessorID)
	d.Set("secret_id", token.SecretID)
	log.Printf("[DEBUG] TOKENID", token.SecretID)
	d.Set("create_time", token.CreateTime.UTC().String())
	d.Set("expiration_tll", token.ExpirationTTL.String())
	d.Set("expiration_time", expirationTime)

	return nil
}

func resourceACLBootStrapExists(d *schema.ResourceData, meta interface{}) (bool, error) {
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
