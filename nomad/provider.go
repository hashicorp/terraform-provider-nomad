package nomad

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/vault/command/config"
)

type ProviderConfig struct {
	client      *api.Client
	vaultToken  *string
	consulToken *string
	config      *api.Config
}

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"address": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_ADDR", nil),
				Description: "URL of the root of the target Nomad agent.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Region of the target Nomad agent.",
			},
			"http_auth": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_HTTP_AUTH", ""),
				Description: "HTTP basic auth configuration.",
			},
			"ca_file": {
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("NOMAD_CACERT", nil),
				Description:   "A path to a PEM-encoded certificate authority used to verify the remote agent's certificate.",
				ConflictsWith: []string{"ca_pem"},
			},
			"ca_pem": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "PEM-encoded certificate authority used to verify the remote agent's certificate.",
				ConflictsWith: []string{"ca_file"},
			},
			"cert_file": {
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("NOMAD_CLIENT_CERT", nil),
				Description:   "A path to a PEM-encoded certificate provided to the remote agent; requires use of key_file or key_pem.",
				ConflictsWith: []string{"cert_pem"},
			},
			"cert_pem": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "PEM-encoded certificate provided to the remote agent; requires use of key_file or key_pem.",
				ConflictsWith: []string{"cert_file"},
			},
			"key_file": {
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("NOMAD_CLIENT_KEY", nil),
				Description:   "A path to a PEM-encoded private key, required if cert_file or cert_pem is specified.",
				ConflictsWith: []string{"key_pem"},
			},
			"key_pem": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "PEM-encoded private key, required if cert_file or cert_pem is specified.",
				ConflictsWith: []string{"key_file"},
			},
			"consul_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("CONSUL_HTTP_TOKEN", ""),
				Description: "Consul token to validate Consul Connect Service Identity policies specified in the job file.",
			},
			"vault_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("VAULT_TOKEN", ""),
				Description: "Vault token if policies are specified in the job file.",
			},
			"secret_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_TOKEN", ""),
				Description: "ACL token secret for API requests.",
			},
			"headers": {
				Type:        schema.TypeList,
				Optional:    true,
				Sensitive:   true,
				Description: "The headers to send with each Nomad request.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The header name",
						},
						"value": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The header value",
						},
					},
				},
			},
			"ignore_env_vars": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A set of environment variables that are ignored by the provider when configuring the Nomad API client.",
				Elem:        &schema.Schema{Type: schema.TypeBool},
			},
		},

		ConfigureFunc: providerConfigure,

		DataSourcesMap: map[string]*schema.Resource{
			"nomad_acl_policies":     dataSourceAclPolicies(),
			"nomad_acl_policy":       dataSourceAclPolicy(),
			"nomad_acl_role":         dataSourceACLRole(),
			"nomad_acl_roles":        dataSourceACLRoles(),
			"nomad_acl_token":        dataSourceACLToken(),
			"nomad_acl_tokens":       dataSourceACLTokens(),
			"nomad_datacenters":      dataSourceDatacenters(),
			"nomad_deployments":      dataSourceDeployments(),
			"nomad_job":              dataSourceJob(),
			"nomad_job_parser":       dataSourceJobParser(),
			"nomad_namespace":        dataSourceNamespace(),
			"nomad_namespaces":       dataSourceNamespaces(),
			"nomad_plugin":           dataSourcePlugin(),
			"nomad_plugins":          dataSourcePlugins(),
			"nomad_scaling_policies": dataSourceScalingPolicies(),
			"nomad_scaling_policy":   dataSourceScalingPolicy(),
			"nomad_scheduler_config": dataSourceSchedulerConfig(),
			"nomad_regions":          dataSourceRegions(),
			"nomad_volumes":          dataSourceVolumes(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"nomad_acl_policy":          resourceACLPolicy(),
			"nomad_acl_role":            resourceACLRole(),
			"nomad_acl_token":           resourceACLToken(),
			"nomad_external_volume":     resourceExternalVolume(),
			"nomad_job":                 resourceJob(),
			"nomad_namespace":           resourceNamespace(),
			"nomad_quota_specification": resourceQuotaSpecification(),
			"nomad_sentinel_policy":     resourceSentinelPolicy(),
			"nomad_volume":              resourceVolume(),
			"nomad_scheduler_config":    resourceSchedulerConfig(),
		},
	}
}

// Get gets the value of the stored token, if any
func getToken() (string, error) {
	helper, err := config.DefaultTokenHelper()
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
	ignoreEnvVars := d.Get("ignore_env_vars").(map[string]interface{})
	if len(ignoreEnvVars) == 0 {
		// The Terraform SDK doesn't support DefaultFunc for complex types yet,
		// so implement the default value logic here for now.
		// https://github.com/hashicorp/terraform-plugin-sdk/issues/142
		if os.Getenv("TFC_RUN_ID") != "" {
			ignoreEnvVars = map[string]interface{}{
				"NOMAD_NAMESPACE": true,
				"NOMAD_REGION":    true,
			}
		}
	}

	conf := api.DefaultConfig()
	conf.Address = d.Get("address").(string)
	conf.SecretID = d.Get("secret_id").(string)

	if region, ok := d.GetOk("region"); ok {
		conf.Region = region.(string)
	} else if ignore, ok := ignoreEnvVars["NOMAD_REGION"]; ok && ignore.(bool) {
		conf.Region = ""
	}

	// The namespace is set per-resource but `DefaultConfig` loads it from the
	// NOMAD_NAMESPACE env var automatically. This will cause problems when
	// Terraform is running within a Nomad job (such as in Terraform Cloud) so
	// we need to unset it unless the provider is configured to load it.
	if ignore, ok := ignoreEnvVars["NOMAD_NAMESPACE"]; ok && ignore.(bool) {
		conf.Namespace = ""
	}

	// HTTP basic auth configuration.
	httpAuth := d.Get("http_auth").(string)
	if httpAuth != "" {
		var username, password string
		if strings.Contains(httpAuth, ":") {
			split := strings.SplitN(httpAuth, ":", 2)
			username = split[0]
			password = split[1]
		} else {
			username = httpAuth
		}
		conf.HttpAuth = &api.HttpBasicAuth{Username: username, Password: password}
	}

	// TLS configuration items.
	conf.TLSConfig.CACert = d.Get("ca_file").(string)
	conf.TLSConfig.ClientCert = d.Get("cert_file").(string)
	conf.TLSConfig.ClientKey = d.Get("key_file").(string)
	conf.TLSConfig.CACertPEM = []byte(d.Get("ca_pem").(string))
	conf.TLSConfig.ClientCertPEM = []byte(d.Get("cert_pem").(string))
	conf.TLSConfig.ClientKeyPEM = []byte(d.Get("key_pem").(string))

	if _, ok := os.LookupEnv("TF_ACC"); ok {
		// Revert the Nomad API client to non-pooled to avoid EOF errors when
		// running the test suite since it instantiates the provider multiple
		// times, creating several clients in parallel.
		// https://github.com/hashicorp/nomad/pull/12492
		conf.HttpClient = nonPooledHttpClient()
	}

	// Set headers if provided
	headers := d.Get("headers").([]interface{})
	parsedHeaders := make(http.Header)

	for _, h := range headers {
		header := h.(map[string]interface{})
		if name, ok := header["name"]; ok {
			parsedHeaders.Add(name.(string), header["value"].(string))
		}
	}
	conf.Headers = parsedHeaders

	// Get the vault token from the conf, VAULT_TOKEN
	// or ~/.vault-token (in that order)
	var err error
	vaultToken := d.Get("vault_token").(string)
	if vaultToken == "" {
		vaultToken, err = getToken()
		if err != nil {
			return nil, err
		}
	}

	consulToken := d.Get("consul_token").(string)

	client, err := api.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to configure Nomad API: %s", err)
	}

	res := ProviderConfig{
		config:      conf,
		client:      client,
		vaultToken:  &vaultToken,
		consulToken: &consulToken,
	}

	return res, nil
}

func nonPooledHttpClient() *http.Client {
	httpClient := cleanhttp.DefaultClient()
	transport := httpClient.Transport.(*http.Transport)
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Default to http/1: alloc exec/websocket aren't supported in http/2
	// well yet: https://github.com/gorilla/websocket/issues/417
	transport.ForceAttemptHTTP2 = false

	return httpClient
}
