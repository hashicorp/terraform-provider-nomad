// Copyright IBM Corp. 2024, 2026
// SPDX-License-Identifier: MPL-2.0

package services

import (
	"context"
	"fmt"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

var _ datasource.DataSource = &ServicesDataSource{}
var _ datasource.DataSourceWithConfigure = &ServicesDataSource{}

type ServicesDataSource struct {
	providerConfig nomad.ProviderConfig
}

func NewServicesDataSource() datasource.DataSource {
	return &ServicesDataSource{}
}

type servicesModel struct {
	Namespace types.String `tfsdk:"namespace"`
	Services  types.List   `tfsdk:"services"`
}

var serviceStubObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"namespace": types.StringType,
		"name":      types.StringType,
		"tags":      types.ListType{ElemType: types.StringType},
	},
}

func (d *ServicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_services"
}

func (d *ServicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieve the list of all registered Nomad services.",
		Attributes: map[string]schema.Attribute{
			"namespace": schema.StringAttribute{
				Optional:    true,
				Description: "The namespace to filter services. If not provided, defaults to \"default\". Use \"*\" for all namespaces.",
			},
			"services": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The list of services found.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"namespace": schema.StringAttribute{
							Computed:    true,
							Description: "The namespace in which the service is registered.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the service.",
						},
						"tags": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "The tags associated with the service.",
						},
					},
				},
			},
		},
	}
}

func (d *ServicesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	metaFunc, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
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

	d.providerConfig = providerConfig
}

func (d *ServicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data servicesModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.providerConfig.Client()

	qOpts := &api.QueryOptions{}
	if !data.Namespace.IsNull() && !data.Namespace.IsUnknown() {
		qOpts.Namespace = data.Namespace.ValueString()
	}

	tflog.Debug(ctx, "listing Nomad services", map[string]interface{}{
		"namespace": qOpts.Namespace,
	})

	serviceStubs, _, err := client.Services().List(qOpts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing Nomad services",
			fmt.Sprintf("Unable to list services: %s", err),
		)
		return
	}

	svcList := make([]attr.Value, 0)
	for _, nsStub := range serviceStubs {
		for _, svc := range nsStub.Services {
			tagValues := make([]attr.Value, 0, len(svc.Tags))
			for _, tag := range svc.Tags {
				tagValues = append(tagValues, types.StringValue(tag))
			}
			tagsList, diags := types.ListValue(types.StringType, tagValues)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			svcObj, diags := types.ObjectValue(
				serviceStubObjectType.AttrTypes,
				map[string]attr.Value{
					"namespace": types.StringValue(nsStub.Namespace),
					"name":      types.StringValue(svc.ServiceName),
					"tags":      tagsList,
				},
			)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			svcList = append(svcList, svcObj)
		}
	}

	servicesList, diags := types.ListValue(serviceStubObjectType, svcList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Services = servicesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
