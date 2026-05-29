// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package volumes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

type topologyRequestRequiredOnlyModel struct {
	Required *topologyGroupModel `tfsdk:"required"`
}

type csiVolumeRegistrationModel struct {
	ID                  types.String                      `tfsdk:"id"`
	Namespace           types.String                      `tfsdk:"namespace"`
	VolumeID            types.String                      `tfsdk:"volume_id"`
	Name                types.String                      `tfsdk:"name"`
	PluginID            types.String                      `tfsdk:"plugin_id"`
	ExternalID          types.String                      `tfsdk:"external_id"`
	CapacityMin         types.String                      `tfsdk:"capacity_min"`
	CapacityMax         types.String                      `tfsdk:"capacity_max"`
	Capability          []capabilityModel                 `tfsdk:"capability"`
	MountOptions        *mountOptionsModel                `tfsdk:"mount_options"`
	Secrets             map[string]types.String           `tfsdk:"secrets"`
	SecretsWO           types.String                      `tfsdk:"secrets_wo"`
	SecretsWOVersion    types.Int64                       `tfsdk:"secrets_wo_version"`
	Parameters          map[string]types.String           `tfsdk:"parameters"`
	TopologyRequest     *topologyRequestRequiredOnlyModel `tfsdk:"topology_request"`
	Context             map[string]types.String           `tfsdk:"context"`
	DeregisterOnDestroy types.Bool                        `tfsdk:"deregister_on_destroy"`

	// Computed
	Capacity              types.Int64  `tfsdk:"capacity"`
	CapacityMinBytes      types.Int64  `tfsdk:"capacity_min_bytes"`
	CapacityMaxBytes      types.Int64  `tfsdk:"capacity_max_bytes"`
	ControllerRequired    types.Bool   `tfsdk:"controller_required"`
	ControllersExpected   types.Int64  `tfsdk:"controllers_expected"`
	ControllersHealthy    types.Int64  `tfsdk:"controllers_healthy"`
	PluginProvider        types.String `tfsdk:"plugin_provider"`
	PluginProviderVersion types.String `tfsdk:"plugin_provider_version"`
	NodesHealthy          types.Int64  `tfsdk:"nodes_healthy"`
	NodesExpected         types.Int64  `tfsdk:"nodes_expected"`
	Schedulable           types.Bool   `tfsdk:"schedulable"`
	Topologies            types.List   `tfsdk:"topologies"`
}

var (
	_ resource.Resource                = &CSIVolumeRegistrationResource{}
	_ resource.ResourceWithConfigure   = &CSIVolumeRegistrationResource{}
	_ resource.ResourceWithImportState = &CSIVolumeRegistrationResource{}
	_ resource.ResourceWithModifyPlan  = &CSIVolumeRegistrationResource{}
)

type CSIVolumeRegistrationResource struct {
	providerConfig nomad.ProviderConfig
}

func NewCSIVolumeRegistrationResource() resource.Resource {
	return &CSIVolumeRegistrationResource{}
}

func (r *CSIVolumeRegistrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_csi_volume_registration"
}

func (r *CSIVolumeRegistrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := computedAttributes()

	attrs["id"] = schema.StringAttribute{
		Computed:    true,
		Description: "The ID of this resource.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attrs["namespace"] = schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Default:     stringdefault.StaticString("default"),
		Description: "The namespace in which to register the volume.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
	attrs["volume_id"] = schema.StringAttribute{
		Required:    true,
		Description: "The unique ID of the volume, how jobs will refer to the volume.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
	attrs["name"] = schema.StringAttribute{
		Required:    true,
		Description: "The display name of the volume.",
	}
	attrs["plugin_id"] = schema.StringAttribute{
		Required:    true,
		Description: "The ID of the CSI plugin that manages this volume.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
	attrs["external_id"] = schema.StringAttribute{
		Required:    true,
		Description: "The ID of the physical volume from the storage provider.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
	attrs["capacity_min"] = schema.StringAttribute{
		Optional:    true,
		Description: "Defines how small the volume can be. The storage provider may return a volume that is larger than this value.",
	}
	attrs["capacity_max"] = schema.StringAttribute{
		Optional:    true,
		Description: "Defines how large the volume can be. The storage provider may return a volume that is smaller than this value.",
	}
	attrs["parameters"] = schema.MapAttribute{
		ElementType: types.StringType,
		Optional:    true,
		Description: "An optional key-value map of strings passed directly to the CSI plugin to configure the volume.",
	}
	attrs["context"] = schema.MapAttribute{
		ElementType: types.StringType,
		Optional:    true,
		Description: "An optional key-value map of strings passed directly to the CSI plugin to validate the volume.",
	}
	attrs["deregister_on_destroy"] = schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Default:     booldefault.StaticBool(true),
		Description: "If true, the volume will be deregistered on destroy.",
	}

	for k, v := range secretsAttributes() {
		attrs[k] = v
	}

	resp.Schema = schema.Schema{
		Description: "Manages the registration of a CSI volume in Nomad.",
		Attributes:  attrs,
		Blocks: map[string]schema.Block{
			"capability":    capabilityBlock(),
			"mount_options": mountOptionsBlock(),
			"topology_request": schema.SingleNestedBlock{
				Description: "Specify locations (region, zone, rack, etc.) where the provisioned volume is accessible from.",
				Blocks: map[string]schema.Block{
					"required": schema.SingleNestedBlock{
						Description: "Required topologies indicate that the volume must be created in a location accessible from all the listed topologies.",
						Blocks: map[string]schema.Block{
							"topology": topologyBlock(),
						},
					},
				},
			},
		},
	}
}

func (r *CSIVolumeRegistrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CSIVolumeRegistrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data csiVolumeRegistrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configData csiVolumeRegistrationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.SecretsWO = configData.SecretsWO

	client := r.providerConfig.Client()

	capMin, capMax, capDiags := parseCapacity(data.CapacityMin, data.CapacityMax)
	resp.Diagnostics.Append(capDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secrets, secretDiags := resolveSecrets(data.SecretsWO, data.Secrets)
	resp.Diagnostics.Append(secretDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	volume := &api.CSIVolume{
		ID:                    data.VolumeID.ValueString(),
		PluginID:              data.PluginID.ValueString(),
		Name:                  data.Name.ValueString(),
		ExternalID:            data.ExternalID.ValueString(),
		RequestedCapacityMin:  int64(capMin),
		RequestedCapacityMax:  int64(capMax),
		RequestedCapabilities: parseCapabilities(data.Capability),
		RequestedTopologies:   parseTopologyRequestRequiredOnly(data.TopologyRequest),
		Secrets:               secrets,
		Parameters:            toMapStringString(data.Parameters),
		Context:               toMapStringString(data.Context),
	}

	if data.MountOptions != nil {
		volume.MountOptions = parseMountOptions(data.MountOptions)
	}

	ns := data.Namespace.ValueString()
	if ns == "" {
		ns = "default"
	}
	opts := &api.WriteOptions{Namespace: ns}

	tflog.Debug(ctx, "Registering CSI volume", map[string]any{"volume_id": volume.ID, "namespace": ns})

	deadline := time.Now().Add(10 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		_, lastErr = client.CSIVolumes().Register(volume, opts)
		if lastErr != nil {
			lastErr = fmt.Errorf("error registering CSI volume: %s", lastErr)
			if csiErrIsRetryable(lastErr) {
				time.Sleep(2 * time.Second)
				continue
			}
			resp.Diagnostics.AddError("Error registering CSI volume", lastErr.Error())
			return
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		resp.Diagnostics.AddError("Error registering CSI volume", lastErr.Error())
		return
	}

	tflog.Debug(ctx, "Registered CSI volume", map[string]any{"volume_id": volume.ID, "namespace": ns})
	data.ID = types.StringValue(volume.ID)

	r.readVolumeIntoModel(ctx, client, ns, volume.ID, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(checkCapacity(uint64(data.Capacity.ValueInt64()), capMin)...)

	handleSecretsWOHash(ctx, data.SecretsWO, resp.Private, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CSIVolumeRegistrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data csiVolumeRegistrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.providerConfig.Client()
	ns := data.Namespace.ValueString()
	if ns == "" {
		ns = "default"
	}

	r.readVolumeIntoModel(ctx, client, ns, data.ID.ValueString(), &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CSIVolumeRegistrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data csiVolumeRegistrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state csiVolumeRegistrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = state.ID

	var configData csiVolumeRegistrationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.SecretsWO = configData.SecretsWO

	client := r.providerConfig.Client()

	capMin, capMax, capDiags := parseCapacity(data.CapacityMin, data.CapacityMax)
	resp.Diagnostics.Append(capDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secrets, secretDiags := resolveSecrets(data.SecretsWO, data.Secrets)
	resp.Diagnostics.Append(secretDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	volume := &api.CSIVolume{
		ID:                    data.VolumeID.ValueString(),
		PluginID:              data.PluginID.ValueString(),
		Name:                  data.Name.ValueString(),
		ExternalID:            data.ExternalID.ValueString(),
		RequestedCapacityMin:  int64(capMin),
		RequestedCapacityMax:  int64(capMax),
		RequestedCapabilities: parseCapabilities(data.Capability),
		RequestedTopologies:   parseTopologyRequestRequiredOnly(data.TopologyRequest),
		Secrets:               secrets,
		Parameters:            toMapStringString(data.Parameters),
		Context:               toMapStringString(data.Context),
	}

	if data.MountOptions != nil {
		volume.MountOptions = parseMountOptions(data.MountOptions)
	}

	ns := data.Namespace.ValueString()
	if ns == "" {
		ns = "default"
	}
	opts := &api.WriteOptions{Namespace: ns}

	tflog.Debug(ctx, "Updating CSI volume registration", map[string]any{"volume_id": volume.ID, "namespace": ns})

	_, err := client.CSIVolumes().Register(volume, opts)
	if err != nil {
		resp.Diagnostics.AddError("Error updating CSI volume registration", err.Error())
		return
	}

	r.readVolumeIntoModel(ctx, client, ns, volume.ID, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	handleSecretsWOHash(ctx, data.SecretsWO, resp.Private, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CSIVolumeRegistrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data csiVolumeRegistrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.DeregisterOnDestroy.IsNull() && data.DeregisterOnDestroy.ValueBool() {
		client := r.providerConfig.Client()
		id := data.ID.ValueString()
		ns := data.Namespace.ValueString()
		if ns == "" {
			ns = "default"
		}
		opts := &api.WriteOptions{Namespace: ns}

		tflog.Debug(ctx, "Deregistering CSI volume", map[string]any{"volume_id": id, "namespace": ns})

		deadline := time.Now().Add(10 * time.Minute)
		for time.Now().Before(deadline) {
			err := client.CSIVolumes().Deregister(id, true, opts)
			if err != nil {
				time.Sleep(2 * time.Second)
				continue
			}
			tflog.Debug(ctx, "Deregistered CSI volume", map[string]any{"volume_id": id})
			return
		}
		resp.Diagnostics.AddError("Error deregistering CSI volume", fmt.Sprintf("timed out deregistering CSI volume %q", id))
	}
}

func (r *CSIVolumeRegistrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, ns, diags := parseImportID(req.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.providerConfig.Client()

	var data csiVolumeRegistrationModel
	data.ID = types.StringValue(id)
	data.Namespace = types.StringValue(ns)

	r.readVolumeIntoModel(ctx, client, ns, id, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CSIVolumeRegistrationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan csiVolumeRegistrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configData csiVolumeRegistrationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateVersion := types.Int64Null()
	if !req.State.Raw.IsNull() {
		var stateData csiVolumeRegistrationModel
		resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		stateVersion = stateData.SecretsWOVersion
	}

	planModified := modifySecretsWOPlan(ctx, req.Private, configData.SecretsWO, configData.SecretsWOVersion, stateVersion, &plan.SecretsWOVersion, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if planModified {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func (r *CSIVolumeRegistrationResource) readVolumeIntoModel(ctx context.Context, client *api.Client, ns, id string, data *csiVolumeRegistrationModel, diags *diag.Diagnostics) {
	queryOpts := &api.QueryOptions{Namespace: ns}

	tflog.Debug(ctx, "Reading CSI volume registration", map[string]any{"id": id, "namespace": ns})
	volume, _, err := client.CSIVolumes().Info(id, queryOpts)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			data.ID = types.StringValue("")
			return
		}
		diags.AddError("Error reading CSI volume", fmt.Sprintf("error reading CSI volume %q: %s", id, err))
		return
	}

	data.Name = types.StringValue(volume.Name)
	data.Namespace = types.StringValue(volume.Namespace)
	data.VolumeID = types.StringValue(volume.ID)
	data.ExternalID = types.StringValue(volume.ExternalID)
	data.PluginID = types.StringValue(volume.PluginID)
	data.PluginProvider = types.StringValue(volume.Provider)
	data.PluginProviderVersion = types.StringValue(volume.ProviderVersion)

	data.Capacity = types.Int64Value(int64(volume.Capacity))
	data.CapacityMinBytes = types.Int64Value(volume.RequestedCapacityMin)
	data.CapacityMaxBytes = types.Int64Value(volume.RequestedCapacityMax)

	data.ControllerRequired = types.BoolValue(volume.ControllerRequired)
	data.ControllersExpected = types.Int64Value(int64(volume.ControllersExpected))
	data.ControllersHealthy = types.Int64Value(int64(volume.ControllersHealthy))
	data.NodesHealthy = types.Int64Value(int64(volume.NodesHealthy))
	data.NodesExpected = types.Int64Value(int64(volume.NodesExpected))
	data.Schedulable = types.BoolValue(volume.Schedulable)

	data.Capability = flattenCapabilities(volume.RequestedCapabilities)
	data.Topologies = flattenTopologiesToList(volume.Topologies)

	if volume.RequestedTopologies != nil {
		data.TopologyRequest = flattenTopologyRequestRequiredOnly(volume.RequestedTopologies)
	}
}

func parseTopologyRequestRequiredOnly(req *topologyRequestRequiredOnlyModel) *api.CSITopologyRequest {
	if req == nil {
		return nil
	}
	result := &api.CSITopologyRequest{}
	if req.Required != nil {
		result.Required = parseTopologies(req.Required.Topology)
	}
	return result
}

func flattenTopologyRequestRequiredOnly(req *api.CSITopologyRequest) *topologyRequestRequiredOnlyModel {
	if req == nil {
		return nil
	}
	result := &topologyRequestRequiredOnlyModel{}
	if req.Required != nil {
		result.Required = &topologyGroupModel{
			Topology: flattenTopologies(req.Required),
		}
	}
	return result
}
