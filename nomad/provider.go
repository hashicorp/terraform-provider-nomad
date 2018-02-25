package nomad

import (
	"fmt"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/command"
)

type ProviderConfig struct {
	client     *api.Client
	vaultToken *string
}

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"address": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_ADDR", nil),
				Description: "URL of the root of the target Nomad agent.",
			},

			"region": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_REGION", ""),
				Description: "Region of the target Nomad agent.",
			},
			"ca_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_CACERT", ""),
				Description: "A path to a PEM-encoded certificate authority used to verify the remote agent's certificate.",
			},
			"cert_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_CLIENT_CERT", ""),
				Description: "A path to a PEM-encoded certificate provided to the remote agent; requires use of key_file.",
			},
			"key_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_CLIENT_KEY", ""),
				Description: "A path to a PEM-encoded private key, required if cert_file is specified.",
			},
			"vault_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("VAULT_TOKEN", ""),
				Description: "Vault token if policies are specified in the job file.",
			},
			"secret_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_TOKEN", ""),
				Description: "ACL token secret for API requests.",
			},
		},

		ConfigureFunc: providerConfigure,

		ResourcesMap: map[string]*schema.Resource{
			"nomad_acl_policy": resourceACLPolicy(),
			"nomad_acl_token":  resourceACLToken(),
			"nomad_job":        resourceJob(),
		},
	}
}

// Get gets the value of the stored token, if any
func getToken() (string, error) {
	helper, err := command.DefaultTokenHelper()
	if err != nil {
		return "", fmt.Errorf("Error getting token helper: %s", err)
	}
	token, err := helper.Get()
	if err != nil {
		return "", fmt.Errorf("Error getting token: %s", err)
	}
	return token, nil
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := api.DefaultConfig()
	config.Address = d.Get("address").(string)
	config.Region = d.Get("region").(string)
	config.TLSConfig.CACert = d.Get("ca_file").(string)
	config.TLSConfig.ClientCert = d.Get("cert_file").(string)
	config.TLSConfig.ClientKey = d.Get("key_file").(string)
	config.SecretID = d.Get("secret_id").(string)

	// Get the vault token from the config, VAULT_TOKEN
	// or ~/.vault-token (in that order)
	var err error
	vaultToken := d.Get("vault_token").(string)
	if vaultToken == "" {
		vaultToken, err = getToken()
		if err != nil {
			return nil, err
		}
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to configure Nomad API: %s", err)
	}

	res := ProviderConfig{
		client:     client,
		vaultToken: &vaultToken,
	}

	return res, nil
}
