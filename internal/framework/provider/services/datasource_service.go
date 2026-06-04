// Copyright IBM Corp. 2016, 2026
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

var _ datasource.DataSource = &ServiceDataSource{}
var _ datasource.DataSourceWithConfigure = &ServiceDataSource{}

type ServiceDataSource struct {
	providerConfig nomad.ProviderConfig
}

func NewServiceDataSource() datasource.DataSource {
	return &ServiceDataSource{}
}

type serviceModel struct {
	ServiceName   types.String `tfsdk:"service_name"`
	Namespace     types.String `tfsdk:"namespace"`
	Filter        types.String `tfsdk:"filter"`
	Choose        types.String `tfsdk:"choose"`
	Registrations types.List   `tfsdk:"registrations"`
}

var registrationObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":           types.StringType,
		"address":      types.StringType,
		"port":         types.Int64Type,
		"node_id":      types.StringType,
		"datacenter":   types.StringType,
		"alloc_id":     types.StringType,
		"job_id":       types.StringType,
		"namespace":    types.StringType,
		"tags":         types.ListType{ElemType: types.StringType},
		"create_index": types.Int64Type,
		"modify_index": types.Int64Type,
	},
}

func (d *ServiceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *ServiceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieve information about a specific Nomad service and its registrations.",
		Attributes: map[string]schema.Attribute{
			"service_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the service to look up.",
			},
			"namespace": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The namespace of the service. Defaults to \"default\".",
			},
			"filter": schema.StringAttribute{
				Optional:    true,
				Description: "A Nomad filter expression to apply to the service registrations. See https://developer.hashicorp.com/nomad/api-docs#filtering for syntax.",
			},
			"choose": schema.StringAttribute{
				Optional:    true,
				Description: "Specifies the number of services to return and a hash key for rendezvous hashing. Must be in the form \"<number>|<key>\".",
			},
			"registrations": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The list of service registrations matching the query.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The unique identifier of the service registration.",
						},
						"address": schema.StringAttribute{
							Computed:    true,
							Description: "The IP address of the service registration.",
						},
						"port": schema.Int64Attribute{
							Computed:    true,
							Description: "The port number of the service registration.",
						},
						"node_id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the node where the service is running.",
						},
						"datacenter": schema.StringAttribute{
							Computed:    true,
							Description: "The datacenter of the node where the service is running.",
						},
						"alloc_id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the allocation providing this service.",
						},
						"job_id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the job that registered this service.",
						},
						"namespace": schema.StringAttribute{
							Computed:    true,
							Description: "The namespace of the service registration.",
						},
						"tags": schema.ListAttribute{
							Computed:    true,
							ElementType: types.StringType,
							Description: "The tags associated with this service registration.",
						},
						"create_index": schema.Int64Attribute{
							Computed:    true,
							Description: "The Raft index when this registration was created.",
						},
						"modify_index": schema.Int64Attribute{
							Computed:    true,
							Description: "The Raft index when this registration was last modified.",
						},
					},
				},
			},
		},
	}
}

func (d *ServiceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serviceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.providerConfig.Client()
	serviceName := data.ServiceName.ValueString()

	qOpts := &api.QueryOptions{}
	if !data.Namespace.IsNull() && !data.Namespace.IsUnknown() {
		qOpts.Namespace = data.Namespace.ValueString()
	}

	if !data.Choose.IsNull() && !data.Choose.IsUnknown() {
		qOpts.Params = map[string]string{
			"choose": data.Choose.ValueString(),
		}
	}

	if !data.Filter.IsNull() && !data.Filter.IsUnknown() {
		qOpts.Filter = data.Filter.ValueString()
	}

	tflog.Debug(ctx, "reading Nomad service", map[string]interface{}{
		"service_name": serviceName,
		"namespace":    qOpts.Namespace,
	})

	registrations, _, err := client.Services().Get(serviceName, qOpts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Nomad service",
			fmt.Sprintf("Unable to read service %q: %s", serviceName, err),
		)
		return
	}

	if data.Namespace.IsNull() || data.Namespace.IsUnknown() {
		data.Namespace = types.StringValue("default")
	}

	regList := make([]attr.Value, 0, len(registrations))
	for _, reg := range registrations {
		tagValues := make([]attr.Value, 0, len(reg.Tags))
		for _, tag := range reg.Tags {
			tagValues = append(tagValues, types.StringValue(tag))
		}
		tagsList, diags := types.ListValue(types.StringType, tagValues)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		regObj, diags := types.ObjectValue(
			registrationObjectType.AttrTypes,
			map[string]attr.Value{
				"id":           types.StringValue(reg.ID),
				"address":      types.StringValue(reg.Address),
				"port":         types.Int64Value(int64(reg.Port)),
				"node_id":      types.StringValue(reg.NodeID),
				"datacenter":   types.StringValue(reg.Datacenter),
				"alloc_id":     types.StringValue(reg.AllocID),
				"job_id":       types.StringValue(reg.JobID),
				"namespace":    types.StringValue(reg.Namespace),
				"tags":         tagsList,
				"create_index": types.Int64Value(int64(reg.CreateIndex)),
				"modify_index": types.Int64Value(int64(reg.ModifyIndex)),
			},
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		regList = append(regList, regObj)
	}

	registrationsList, diags := types.ListValue(registrationObjectType, regList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Registrations = registrationsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
