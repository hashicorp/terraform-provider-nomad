// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package acl

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	ephemeralschema "github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

var (
	_ ephemeral.EphemeralResource              = (*IntroTokenEphemeralResource)(nil)
	_ ephemeral.EphemeralResourceWithConfigure = (*IntroTokenEphemeralResource)(nil)
)

type IntroTokenEphemeralResource struct {
	SDKv2Meta func() any
}

type introTokenModel struct {
	NodeName types.String `tfsdk:"node_name"`
	NodePool types.String `tfsdk:"node_pool"`
	TTL      types.String `tfsdk:"ttl"`
	JWT      types.String `tfsdk:"jwt"`
}

func NewIntroTokenEphemeralResource() ephemeral.EphemeralResource {
	return &IntroTokenEphemeralResource{}
}

func (r *IntroTokenEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	sdkv2Meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Ephemeral Resource Configure Type",
			fmt.Sprintf("Expected provider data of type func() any, got %T.", req.ProviderData),
		)
		return
	}

	r.SDKv2Meta = sdkv2Meta
}

func (r *IntroTokenEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node_intro_token"
}

func (r *IntroTokenEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = ephemeralschema.Schema{
		Attributes: map[string]ephemeralschema.Attribute{
			"node_name": ephemeralschema.StringAttribute{
				Optional:    true,
				Description: "The node name to scope the introduction token to.",
			},
			"node_pool": ephemeralschema.StringAttribute{
				Optional:    true,
				Description: "The node pool to scope the introduction token to.",
			},
			"ttl": ephemeralschema.StringAttribute{
				Optional:    true,
				Description: "Requested introduction token TTL, such as 5m or 1h.",
			},
			"jwt": ephemeralschema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The signed JWT node introduction token.",
			},
		},
	}
}

func (r *IntroTokenEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data introTokenModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		log.Printf("[DEBUG] nomad_node_intro_token: config decoding returned diagnostics")
		return
	}

	if r.SDKv2Meta == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Nomad Provider",
			"The provider has not been configured. Configure the nomad provider before using nomad_node_intro_token.",
		)
		return
	}

	providerData := r.SDKv2Meta()
	providerConfig, ok := providerData.(nomad.ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Metadata Type",
			fmt.Sprintf("Expected nomad.ProviderConfig, got %T.", providerData),
		)
		return
	}

	request := &api.ACLIdentityClientIntroductionTokenRequest{}
	if !data.NodeName.IsNull() && !data.NodeName.IsUnknown() {
		request.NodeName = data.NodeName.ValueString()
	}
	if !data.NodePool.IsNull() && !data.NodePool.IsUnknown() {
		request.NodePool = data.NodePool.ValueString()
	}
	if !data.TTL.IsNull() && !data.TTL.IsUnknown() {
		ttl, err := time.ParseDuration(data.TTL.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid TTL", fmt.Sprintf("Failed to parse ttl %q: %v", data.TTL.ValueString(), err))
			return
		}
		request.TTL = ttl
	}

	client := providerConfig.Client()
	if client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Nomad Client",
			"The provider did not expose a configured Nomad API client.",
		)
		return
	}

	log.Printf("[DEBUG] nomad_node_intro_token: creating client introduction token")
	introToken, _, err := client.ACLIdentity().CreateClientIntroductionToken(request, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Create Node Introduction Token",
			err.Error(),
		)
		return
	}

	log.Printf("[DEBUG] nomad_node_intro_token: successfully created introduction token")
	data.JWT = types.StringValue(introToken.JWT)

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
