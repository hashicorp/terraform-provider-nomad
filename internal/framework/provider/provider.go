// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure NomadProvider satisfies various provider interfaces.
var (
	_ frameworkprovider.Provider = (*NomadProvider)(nil)
)

type NomadProvider struct {
	// SDKv2Meta is a function which returns provider meta struct from the SDKv2 code
	SDKv2Meta func() any
}

func New(sdkv2Meta func() any) frameworkprovider.Provider {
	return &NomadProvider{
		SDKv2Meta: sdkv2Meta,
	}
}

func (p *NomadProvider) Metadata(_ context.Context, _ frameworkprovider.MetadataRequest, resp *frameworkprovider.MetadataResponse) {
	resp.TypeName = "nomad"
}

func (p *NomadProvider) Schema(_ context.Context, _ frameworkprovider.SchemaRequest, resp *frameworkprovider.SchemaResponse) {
	resp.Schema = providerschema.Schema{
		Attributes: map[string]providerschema.Attribute{
			"address": providerschema.StringAttribute{
				Optional:    true,
				Description: "URL of the root of the target Nomad agent.",
			},
			"region": providerschema.StringAttribute{
				Optional:    true,
				Description: "Region of the target Nomad agent.",
			},
			"http_auth": providerschema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "HTTP basic auth configuration.",
			},
			"ca_file": providerschema.StringAttribute{
				Optional:    true,
				Description: "A path to a PEM-encoded certificate authority used to verify the remote agent's certificate.",
			},
			"ca_pem": providerschema.StringAttribute{
				Optional:    true,
				Description: "PEM-encoded certificate authority used to verify the remote agent's certificate.",
			},
			"cert_file": providerschema.StringAttribute{
				Optional:    true,
				Description: "A path to a PEM-encoded certificate provided to the remote agent; requires use of key_file or key_pem.",
			},
			"cert_pem": providerschema.StringAttribute{
				Optional:    true,
				Description: "PEM-encoded certificate provided to the remote agent; requires use of key_file or key_pem.",
			},
			"key_file": providerschema.StringAttribute{
				Optional:    true,
				Description: "A path to a PEM-encoded private key, required if cert_file or cert_pem is specified.",
			},
			"key_pem": providerschema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "PEM-encoded private key, required if cert_file or cert_pem is specified.",
			},
			"secret_id": providerschema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "ACL token secret for API requests.",
			},
			"ignore_env_vars": providerschema.MapAttribute{
				Optional:    true,
				ElementType: types.BoolType,
				Description: "A set of environment variables that are ignored by the provider when configuring the Nomad API client.",
			},
			"skip_verify": providerschema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS verification on client side.",
			},
		},
		Blocks: map[string]providerschema.Block{
			"headers": providerschema.ListNestedBlock{
				Description: "The headers to send with each Nomad request.",
				NestedObject: providerschema.NestedBlockObject{
					Attributes: map[string]providerschema.Attribute{
						"name": providerschema.StringAttribute{
							Required:    true,
							Description: "The header name",
						},
						"value": providerschema.StringAttribute{
							Required:    true,
							Sensitive:   true,
							Description: "The header value",
						},
					},
				},
			},
			"auth_jwt": providerschema.ListNestedBlock{
				Description: "Authenticates to Nomad using a JWT authentication method.",
				Validators:  nil,
				NestedObject: providerschema.NestedBlockObject{
					Attributes: map[string]providerschema.Attribute{
						"auth_method": providerschema.StringAttribute{
							Required:    true,
							Description: "The name of the auth method to use for login.",
						},
						"login_token": providerschema.StringAttribute{
							Required:    true,
							Sensitive:   true,
							Description: "The externally issued authentication token to be exchanged for a Nomad ACL Token.",
						},
					},
				},
			},
		},
	}
}

func (p *NomadProvider) Configure(_ context.Context, _ frameworkprovider.ConfigureRequest, resp *frameworkprovider.ConfigureResponse) {
	resp.ResourceData = p.SDKv2Meta
	resp.DataSourceData = p.SDKv2Meta
	resp.EphemeralResourceData = p.SDKv2Meta
}

func (p *NomadProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

func (p *NomadProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *NomadProvider) EphemeralResources(_ context.Context) []func() ephemeral.EphemeralResource {
	return nil
}
