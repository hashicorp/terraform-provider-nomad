// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package acl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

var _ resource.Resource = &ACLBindingRuleResource{}
var _ resource.ResourceWithConfigure = &ACLBindingRuleResource{}
var _ resource.ResourceWithImportState = &ACLBindingRuleResource{}
var _ resource.ResourceWithUpgradeState = &ACLBindingRuleResource{}

type ACLBindingRuleResource struct {
	providerConfig nomad.ProviderConfig
}

func NewACLBindingRuleResource() resource.Resource {
	return &ACLBindingRuleResource{}
}

type aclBindingRuleModel struct {
	ID          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	AuthMethod  types.String `tfsdk:"auth_method"`
	Selector    types.String `tfsdk:"selector"`
	BindType    types.String `tfsdk:"bind_type"`
	BindName    types.String `tfsdk:"bind_name"`
}

func (r *ACLBindingRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl_binding_rule"
}

func (r *ACLBindingRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an ACL binding rule in Nomad.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description for this ACL binding rule.",
			},
			"auth_method": schema.StringAttribute{
				Required:    true,
				Description: "Name of the auth method for which this rule applies to.",
			},
			"selector": schema.StringAttribute{
				Optional:    true,
				Description: "A boolean expression that matches against verified identity attributes returned from the auth method during login.",
			},
			"bind_type": schema.StringAttribute{
				Required:    true,
				Description: `Adjusts how this binding rule is applied at login time. Valid values are "role", "policy", and "management".`,
				Validators: []validator.String{
					stringvalidator.OneOf(
						api.ACLBindingRuleBindTypeManagement,
						api.ACLBindingRuleBindTypePolicy,
						api.ACLBindingRuleBindTypeRole,
					),
				},
			},
			"bind_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Target of the binding.",
			},
		},
	}
}

func (r *ACLBindingRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	metaFunc, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected func() any, got %T.", req.ProviderData),
		)
		return
	}

	providerConfig, ok := metaFunc().(nomad.ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Meta Type",
			fmt.Sprintf("Expected nomad.ProviderConfig, got %T.", metaFunc()),
		)
		return
	}

	r.providerConfig = providerConfig
}

func (r *ACLBindingRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data aclBindingRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateBindingRule(data); err != nil {
		resp.Diagnostics.AddError("Invalid binding rule configuration", err.Error())
		return
	}

	rule := modelToAPIBindingRule(data)

	tflog.Debug(ctx, "Creating ACL Binding Rule")
	created, _, err := r.providerConfig.Client().ACLBindingRules().Create(rule, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ACL Binding Rule", err.Error())
		return
	}
	tflog.Debug(ctx, "Created ACL Binding Rule", map[string]any{"id": created.ID})

	data.ID = types.StringValue(created.ID)
	data.BindName = types.StringValue(created.BindName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLBindingRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data aclBindingRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	tflog.Debug(ctx, "Reading ACL Binding Rule", map[string]any{"id": id})

	rule, _, err := r.providerConfig.Client().ACLBindingRules().Get(id, nil)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ACL Binding Rule", fmt.Sprintf("error reading %q: %s", id, err))
		return
	}

	data.Description = types.StringValue(rule.Description)
	data.AuthMethod = types.StringValue(rule.AuthMethod)
	data.Selector = types.StringValue(rule.Selector)
	data.BindType = types.StringValue(rule.BindType)
	data.BindName = types.StringValue(rule.BindName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLBindingRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data aclBindingRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state aclBindingRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = state.ID

	if err := validateBindingRule(data); err != nil {
		resp.Diagnostics.AddError("Invalid binding rule configuration", err.Error())
		return
	}

	rule := modelToAPIBindingRule(data)

	tflog.Debug(ctx, "Updating ACL Binding Rule", map[string]any{"id": rule.ID})
	_, _, err := r.providerConfig.Client().ACLBindingRules().Update(rule, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ACL Binding Rule", fmt.Sprintf("error updating %q: %s", rule.ID, err))
		return
	}
	tflog.Debug(ctx, "Updated ACL Binding Rule", map[string]any{"id": rule.ID})

	if data.BindName.IsUnknown() {
		data.BindName = types.StringValue(rule.BindName)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLBindingRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data aclBindingRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	tflog.Debug(ctx, "Deleting ACL Binding Rule", map[string]any{"id": id})

	_, err := r.providerConfig.Client().ACLBindingRules().Delete(id, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting ACL Binding Rule", fmt.Sprintf("error deleting %q: %s", id, err))
		return
	}
	tflog.Debug(ctx, "Deleted ACL Binding Rule", map[string]any{"id": id})
}

func (r *ACLBindingRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data aclBindingRuleModel
	data.ID = types.StringValue(req.ID)

	rule, _, err := r.providerConfig.Client().ACLBindingRules().Get(req.ID, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error importing ACL Binding Rule", fmt.Sprintf("error reading %q: %s", req.ID, err))
		return
	}

	data.Description = types.StringValue(rule.Description)
	data.AuthMethod = types.StringValue(rule.AuthMethod)
	data.Selector = types.StringValue(rule.Selector)
	data.BindType = types.StringValue(rule.BindType)
	data.BindName = types.StringValue(rule.BindName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLBindingRuleResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {StateUpgrader: upgradeBindingRuleStateV0toV1},
	}
}

func upgradeBindingRuleStateV0toV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	type v0State struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		AuthMethod  string `json:"auth_method"`
		Selector    string `json:"selector"`
		BindType    string `json:"bind_type"`
		BindName    string `json:"bind_name"`
	}

	var old v0State
	if err := json.Unmarshal(req.RawState.JSON, &old); err != nil {
		resp.Diagnostics.AddError("Failed to parse prior state for upgrade", err.Error())
		return
	}

	upgraded := aclBindingRuleModel{
		ID:          types.StringValue(old.ID),
		Description: types.StringValue(old.Description),
		AuthMethod:  types.StringValue(old.AuthMethod),
		Selector:    types.StringValue(old.Selector),
		BindType:    types.StringValue(old.BindType),
		BindName:    types.StringValue(old.BindName),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
}

func modelToAPIBindingRule(m aclBindingRuleModel) *api.ACLBindingRule {
	return &api.ACLBindingRule{
		ID:          m.ID.ValueString(),
		Description: m.Description.ValueString(),
		AuthMethod:  m.AuthMethod.ValueString(),
		Selector:    m.Selector.ValueString(),
		BindType:    m.BindType.ValueString(),
		BindName:    m.BindName.ValueString(),
	}
}

func validateBindingRule(m aclBindingRuleModel) error {
	if m.BindType.ValueString() == api.ACLBindingRuleBindTypeManagement {
		if m.BindName.ValueString() != "" {
			return fmt.Errorf("bind_name must not be set when bind_type is %q", api.ACLBindingRuleBindTypeManagement)
		}
	} else {
		if m.BindName.ValueString() == "" {
			return fmt.Errorf("bind_name must be set when bind_type is %q", m.BindType.ValueString())
		}
	}
	return nil
}
