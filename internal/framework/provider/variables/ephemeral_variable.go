// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package variables

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	ephemeralschema "github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

var _ ephemeral.EphemeralResource = &VariableEphemeralResource{}
var _ ephemeral.EphemeralResourceWithConfigure = &VariableEphemeralResource{}

type VariableEphemeralResource struct {
	SDKv2Meta func() any
}

type variableEphemeralModel struct {
	Path      types.String `tfsdk:"path"`
	Namespace types.String `tfsdk:"namespace"`
	Items     types.Map    `tfsdk:"items"`
}

func NewVariableEphemeralResource() ephemeral.EphemeralResource {
	return &VariableEphemeralResource{}
}

func (r *VariableEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_variable"
}

func (r *VariableEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = ephemeralschema.Schema{
		Description: "Reads a Nomad variable when its items are needed by Terraform but should not be stored in state.",
		Attributes: map[string]ephemeralschema.Attribute{
			"path": ephemeralschema.StringAttribute{
				Required:    true,
				Description: "The path of the variable.",
			},
			"namespace": ephemeralschema.StringAttribute{
				Optional:    true,
				Description: "Variable namespace. Defaults to default.",
			},
			"items": ephemeralschema.MapAttribute{
				Computed:    true,
				Sensitive:   true,
				ElementType: types.StringType,
				Description: "A map of values from the stored variable.",
			},
		},
	}
}

func (r *VariableEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (r *VariableEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var config variableEphemeralModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		log.Printf("[DEBUG] nomad_variable: config decoding returned diagnostics")
		return
	}

	if r.SDKv2Meta == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Nomad Provider",
			"The provider has not been configured. Configure the nomad provider before using nomad_variable.",
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

	path := config.Path.ValueString()
	namespace := api.DefaultNamespace
	if !config.Namespace.IsNull() && !config.Namespace.IsUnknown() && config.Namespace.ValueString() != "" {
		namespace = config.Namespace.ValueString()
	}

	log.Printf("[DEBUG] nomad_variable: reading variable %q in namespace %q", path, namespace)
	variable, _, err := client.Variables().Read(path, &api.QueryOptions{Namespace: namespace})
	if err != nil {
		if strings.Contains(err.Error(), "404") || errors.Is(err, api.ErrVariablePathNotFound) {
			resp.Diagnostics.AddError("Variable not found", fmt.Sprintf("No variable found for path %q in namespace %q.", path, namespace))
			return
		}

		resp.Diagnostics.AddError("Error reading variable", err.Error())
		return
	}
	log.Printf("[DEBUG] nomad_variable: successfully read variable %q in namespace %q", path, namespace)

	result, diags := buildVariableResult(ctx, variable)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &result)...)
}

func buildVariableResult(ctx context.Context, variable *api.Variable) (variableEphemeralModel, diag.Diagnostics) {
	items, diags := types.MapValueFrom(ctx, types.StringType, variable.Items)

	return variableEphemeralModel{
		Path:      types.StringValue(variable.Path),
		Namespace: types.StringValue(variable.Namespace),
		Items:     items,
	}, diags
}
