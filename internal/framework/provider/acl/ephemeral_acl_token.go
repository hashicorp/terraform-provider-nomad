// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package acl

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	ephemeralschema "github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

var _ ephemeral.EphemeralResource = &ACLTokenEphemeralResource{}
var _ ephemeral.EphemeralResourceWithConfigure = &ACLTokenEphemeralResource{}

var aclTokenRoleObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
	},
}

type ACLTokenEphemeralResource struct {
	SDKv2Meta func() any
}

type aclTokenEphemeralModel struct {
	AccessorID     types.String `tfsdk:"accessor_id"`
	SecretID       types.String `tfsdk:"secret_id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	Policies       types.Set    `tfsdk:"policies"`
	Roles          types.Set    `tfsdk:"roles"`
	Global         types.Bool   `tfsdk:"global"`
	CreateTime     types.String `tfsdk:"create_time"`
	ExpirationTTL  types.String `tfsdk:"expiration_ttl"`
	ExpirationTime types.String `tfsdk:"expiration_time"`
}

type aclTokenRoleModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewACLTokenEphemeralResource() ephemeral.EphemeralResource {
	return &ACLTokenEphemeralResource{}
}

func (r *ACLTokenEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl_token"
}

func (r *ACLTokenEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = ephemeralschema.Schema{
		Description: "Reads a Nomad ACL token when the token secret is needed by Terraform but should not be stored in state.",
		Attributes: map[string]ephemeralschema.Attribute{
			"accessor_id": ephemeralschema.StringAttribute{
				Required:    true,
				Description: "Nomad-generated ID for this token.",
			},
			"secret_id": ephemeralschema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The value that grants access to Nomad.",
			},
			"name": ephemeralschema.StringAttribute{
				Computed:    true,
				Description: "Human-readable name for this token.",
			},
			"type": ephemeralschema.StringAttribute{
				Computed:    true,
				Description: "The type of the token.",
			},
			"policies": ephemeralschema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of policy names associated with this token.",
			},
			"roles": ephemeralschema.SetNestedAttribute{
				Computed:    true,
				Description: "The roles that are applied to the token.",
				NestedObject: ephemeralschema.NestedAttributeObject{
					Attributes: map[string]ephemeralschema.Attribute{
						"id": ephemeralschema.StringAttribute{
							Computed:    true,
							Description: "The ID of the ACL role.",
						},
						"name": ephemeralschema.StringAttribute{
							Computed:    true,
							Description: "The name of the ACL role.",
						},
					},
				},
			},
			"global": ephemeralschema.BoolAttribute{
				Computed:    true,
				Description: "Whether the token should be replicated to all regions or not.",
			},
			"create_time": ephemeralschema.StringAttribute{
				Computed:    true,
				Description: "The timestamp the token was created.",
			},
			"expiration_ttl": ephemeralschema.StringAttribute{
				Computed:    true,
				Description: `Provides a TTL for the token in the form of a time duration such as "5m" or "1h".`,
			},
			"expiration_time": ephemeralschema.StringAttribute{
				Computed:    true,
				Description: "The point after which a token is considered expired and eligible for destruction.",
			},
		},
	}
}

func (r *ACLTokenEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (r *ACLTokenEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var config aclTokenEphemeralModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		log.Printf("[DEBUG] nomad_acl_token: config decoding returned diagnostics")
		return
	}

	if r.SDKv2Meta == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Nomad Provider",
			"The provider has not been configured. Configure the nomad provider before using nomad_acl_token.",
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

	client := providerConfig.Client()
	if client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Nomad Client",
			"The provider did not expose a configured Nomad API client.",
		)
		return
	}

	accessorID := config.AccessorID.ValueString()
	log.Printf("[DEBUG] nomad_acl_token: reading ACL token %q", accessorID)
	token, _, err := client.ACLTokens().Info(accessorID, nil)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.Diagnostics.AddError("ACL token not found", fmt.Sprintf("No ACL token found for accessor_id %q.", accessorID))
			return
		}

		resp.Diagnostics.AddError("Error reading ACL token", err.Error())
		return
	}
	log.Printf("[DEBUG] nomad_acl_token: successfully read ACL token %q", accessorID)

	result, diags := buildACLTokenResult(ctx, token)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &result)...)
}

func buildACLTokenResult(ctx context.Context, token *api.ACLToken) (aclTokenEphemeralModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	policies, policiesDiags := types.SetValueFrom(ctx, types.StringType, token.Policies)
	diags.Append(policiesDiags...)

	roleModels := make([]aclTokenRoleModel, len(token.Roles))
	for index, roleLink := range token.Roles {
		roleModels[index] = aclTokenRoleModel{
			ID:   types.StringValue(roleLink.ID),
			Name: types.StringValue(roleLink.Name),
		}
	}

	roles, rolesDiags := types.SetValueFrom(ctx, aclTokenRoleObjectType, roleModels)
	diags.Append(rolesDiags...)

	var expirationTime types.String
	if token.ExpirationTime != nil {
		expirationTime = types.StringValue(token.ExpirationTime.Format(time.RFC3339))
	} else {
		expirationTime = types.StringNull()
	}

	return aclTokenEphemeralModel{
		AccessorID:     types.StringValue(token.AccessorID),
		SecretID:       types.StringValue(token.SecretID),
		Name:           types.StringValue(token.Name),
		Type:           types.StringValue(token.Type),
		Policies:       policies,
		Roles:          roles,
		Global:         types.BoolValue(token.Global),
		CreateTime:     types.StringValue(token.CreateTime.UTC().String()),
		ExpirationTTL:  types.StringValue(token.ExpirationTTL.String()),
		ExpirationTime: expirationTime,
	}, diags
}
