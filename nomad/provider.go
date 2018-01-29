package nomad

import (
	"fmt"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

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
			"nomad_job":        resourceJob(),
		},
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := api.DefaultConfig()
	config.Address = d.Get("address").(string)
	config.Region = d.Get("region").(string)
	config.TLSConfig.CACert = d.Get("ca_file").(string)
	config.TLSConfig.ClientCert = d.Get("cert_file").(string)
	config.TLSConfig.ClientKey = d.Get("key_file").(string)
	config.SecretID = d.Get("secret_id").(string)

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to configure Nomad API: %s", err)
	}

	return client, nil
}
