// Copyright IBM Corp. 2024, 2026
// SPDX-License-Identifier: MPL-2.0

package volumes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

type csiVolumeModel struct {
	ID               types.String            `tfsdk:"id"`
	Namespace        types.String            `tfsdk:"namespace"`
	VolumeID         types.String            `tfsdk:"volume_id"`
	Name             types.String            `tfsdk:"name"`
	PluginID         types.String            `tfsdk:"plugin_id"`
	SnapshotID       types.String            `tfsdk:"snapshot_id"`
	CloneID          types.String            `tfsdk:"clone_id"`
	CapacityMin      types.String            `tfsdk:"capacity_min"`
	CapacityMax      types.String            `tfsdk:"capacity_max"`
	Capability       []capabilityModel       `tfsdk:"capability"`
	MountOptions     []mountOptionsModel     `tfsdk:"mount_options"`
	Secrets          map[string]types.String `tfsdk:"secrets"`
	SecretsWO        types.Map               `tfsdk:"secrets_wo"`
	SecretsWOVersion types.Int64             `tfsdk:"secrets_wo_version"`
	Parameters       map[string]types.String `tfsdk:"parameters"`
	TopologyRequest  []topologyRequestModel  `tfsdk:"topology_request"`
	Timeouts         timeouts.Value          `tfsdk:"timeouts"`

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
	ExternalID            types.String `tfsdk:"external_id"`
	Context               types.Map    `tfsdk:"context"`
}

var (
	_ resource.Resource                = &CSIVolumeResource{}
	_ resource.ResourceWithConfigure   = &CSIVolumeResource{}
	_ resource.ResourceWithImportState = &CSIVolumeResource{}
	_ resource.ResourceWithModifyPlan  = &CSIVolumeResource{}
)

type CSIVolumeResource struct {
	providerConfig nomad.ProviderConfig
}

func NewCSIVolumeResource() resource.Resource {
	return &CSIVolumeResource{}
}

func (r *CSIVolumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_csi_volume"
}

func (r *CSIVolumeResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
		Description: "The namespace in which to create the volume.",
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
	attrs["snapshot_id"] = schema.StringAttribute{
		Optional:    true,
		Description: "The snapshot ID to restore when creating this volume. Storage provider must support snapshots. Conflicts with 'clone_id'.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
		Validators: []validator.String{
			stringvalidator.ConflictsWith(path.MatchRoot("clone_id")),
		},
	}
	attrs["clone_id"] = schema.StringAttribute{
		Optional:    true,
		Description: "The volume ID to clone when creating this volume. Storage provider must support cloning. Conflicts with 'snapshot_id'.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
		Validators: []validator.String{
			stringvalidator.ConflictsWith(path.MatchRoot("snapshot_id")),
		},
	}
	attrs["capacity_min"] = schema.StringAttribute{
		Optional:      true,
		Description:   "Defines how small the volume can be. The storage provider may return a volume that is larger than this value.",
		Validators:    []validator.String{capacityValidator{}},
		PlanModifiers: []planmodifier.String{capacityPlanModifier{}},
	}
	attrs["capacity_max"] = schema.StringAttribute{
		Optional:      true,
		Description:   "Defines how large the volume can be. The storage provider may return a volume that is smaller than this value.",
		Validators:    []validator.String{capacityValidator{}},
		PlanModifiers: []planmodifier.String{capacityPlanModifier{}},
	}
	attrs["parameters"] = schema.MapAttribute{
		ElementType: types.StringType,
		Optional:    true,
		Description: "An optional key-value map of strings passed directly to the CSI plugin to configure the volume.",
	}
	attrs["external_id"] = schema.StringAttribute{
		Computed:    true,
		Description: "The ID of the physical volume from the storage provider.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attrs["context"] = schema.MapAttribute{
		ElementType: types.StringType,
		Computed:    true,
		Description: "The volume context provided by the storage provider.",
		PlanModifiers: []planmodifier.Map{
			mapplanmodifier.UseStateForUnknown(),
		},
	}

	for k, v := range secretsAttributes() {
		attrs[k] = v
	}

	resp.Schema = schema.Schema{
		Description: "Manages the lifecycle of creating and deleting CSI volumes.",
		Attributes:  attrs,
		Blocks: map[string]schema.Block{
			"capability":       capabilityBlock(),
			"mount_options":    mountOptionsBlock(),
			"topology_request": topologyRequestBlock(true),
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Delete: true,
			}),
		},
	}
}

func (r *CSIVolumeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CSIVolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data csiVolumeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configData csiVolumeModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.SecretsWO = configData.SecretsWO
	if len(configData.MountOptions) > 0 && len(data.MountOptions) > 0 {
		data.MountOptions[0].MountFlagsWO = configData.MountOptions[0].MountFlagsWO
	}

	client := r.providerConfig.Client()

	r.createVolume(ctx, client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Persist the ID to state immediately so Terraform tracks the resource
	// even if the subsequent read fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ns := data.Namespace.ValueString()
	if ns == "" {
		ns = "default"
	}

	r.readVolumeIntoModel(ctx, client, ns, data.ID.ValueString(), &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(checkCapacity(uint64(data.Capacity.ValueInt64()), uint64(data.CapacityMinBytes.ValueInt64()))...)

	handleSecretsWOHash(ctx, configData.SecretsWO, resp.Private, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	handleMountFlagsWOHash(ctx, configData.MountOptions, resp.Private, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CSIVolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data csiVolumeModel
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

	// If the volume was not found (404), remove it from state.
	if data.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CSIVolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data csiVolumeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state csiVolumeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = state.ID

	var configData csiVolumeModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.SecretsWO = configData.SecretsWO
	if len(configData.MountOptions) > 0 && len(data.MountOptions) > 0 {
		data.MountOptions[0].MountFlagsWO = configData.MountOptions[0].MountFlagsWO
	}

	client := r.providerConfig.Client()

	r.createVolume(ctx, client, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	ns := data.Namespace.ValueString()
	if ns == "" {
		ns = "default"
	}

	r.readVolumeIntoModel(ctx, client, ns, data.ID.ValueString(), &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(checkCapacity(uint64(data.Capacity.ValueInt64()), uint64(data.CapacityMinBytes.ValueInt64()))...)

	handleSecretsWOHash(ctx, configData.SecretsWO, resp.Private, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	handleMountFlagsWOHash(ctx, configData.MountOptions, resp.Private, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// createVolume contains the shared create + retry logic used by both
// Create and Update, mirroring the SDKv2 pattern where Update reuses Create.
// After success, data.ID is set for use by the caller.
func (r *CSIVolumeResource) createVolume(ctx context.Context, client *api.Client, data *csiVolumeModel, diags *diag.Diagnostics) {
	capMin, capMax, capDiags := parseCapacity(data.CapacityMin, data.CapacityMax)
	diags.Append(capDiags...)
	if diags.HasError() {
		return
	}

	secrets, secretDiags := resolveSecrets(data.SecretsWO, data.Secrets)
	diags.Append(secretDiags...)
	if diags.HasError() {
		return
	}

	capabilities := parseCapabilities(data.Capability)

	volume := &api.CSIVolume{
		ID:                    data.VolumeID.ValueString(),
		PluginID:              data.PluginID.ValueString(),
		Name:                  data.Name.ValueString(),
		SnapshotID:            data.SnapshotID.ValueString(),
		CloneID:               data.CloneID.ValueString(),
		RequestedCapacityMin:  int64(capMin),
		RequestedCapacityMax:  int64(capMax),
		RequestedCapabilities: capabilities,
		RequestedTopologies:   parseTopologyRequest(data.TopologyRequest),
		Secrets:               secrets,
		Parameters:            toMapStringString(data.Parameters),

		// COMPAT(1.5.0)
		// Maintain backwards compatibility.
		AccessMode:     capabilities[0].AccessMode,
		AttachmentMode: capabilities[0].AttachmentMode,
	}

	if len(data.MountOptions) > 0 {
		volume.MountOptions = parseMountOptions(&data.MountOptions[0])
	} else {
		// Send an empty struct rather than nil to match Nomad's internal
		// representation and avoid a spurious "can not update mount options
		// while volume is in use" error from CSIVolume.Merge.
		volume.MountOptions = &api.CSIMountOptions{}
	}

	ns := data.Namespace.ValueString()
	if ns == "" {
		ns = "default"
	}
	opts := &api.WriteOptions{Namespace: ns}

	tflog.Debug(ctx, "Creating CSI volume", map[string]any{"volume_id": volume.ID, "namespace": ns})

	createTimeout, tDiags := data.Timeouts.Create(ctx, 10*time.Minute)
	diags.Append(tDiags...)
	if diags.HasError() {
		return
	}

	err := retry.RetryContext(ctx, createTimeout-time.Minute, func() *retry.RetryError {
		_, _, err := client.CSIVolumes().Create(volume, opts)
		if err != nil {
			err = fmt.Errorf("error creating CSI volume: %s", err)
			if csiErrIsRetryable(err) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		diags.AddError("Error creating CSI volume", err.Error())
		return
	}

	tflog.Debug(ctx, "Created CSI volume", map[string]any{"volume_id": volume.ID, "namespace": ns})
	data.ID = types.StringValue(volume.ID)
}

func (r *CSIVolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data csiVolumeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.providerConfig.Client()
	id := data.ID.ValueString()
	ns := data.Namespace.ValueString()
	if ns == "" {
		ns = "default"
	}
	opts := &api.WriteOptions{Namespace: ns}

	tflog.Debug(ctx, "Deleting CSI volume", map[string]any{"volume_id": id, "namespace": ns})

	deleteTimeout, diags := data.Timeouts.Delete(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := retry.RetryContext(ctx, deleteTimeout-time.Minute, func() *retry.RetryError {
		err := client.CSIVolumes().Delete(id, opts)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("error deleting CSI volume: %s", err))
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting CSI volume", err.Error())
		return
	}

	tflog.Debug(ctx, "Deleted CSI volume", map[string]any{"volume_id": id})
}

func (r *CSIVolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, ns, diags := parseImportID(req.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.providerConfig.Client()

	var data csiVolumeModel
	data.ID = types.StringValue(id)
	data.Namespace = types.StringValue(ns)

	r.readVolumeIntoModel(ctx, client, ns, id, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CSIVolumeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan csiVolumeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configData csiVolumeModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateSecretsVersion := types.Int64Null()
	var stateMountOpts []mountOptionsModel
	if !req.State.Raw.IsNull() {
		var stateData csiVolumeModel
		resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		stateSecretsVersion = stateData.SecretsWOVersion
		stateMountOpts = stateData.MountOptions

		// Detect structural changes in topology_request that require replacement.
		if len(stateData.TopologyRequest) != len(plan.TopologyRequest) {
			resp.RequiresReplace = append(resp.RequiresReplace, path.Root("topology_request"))
		} else if len(stateData.TopologyRequest) > 0 && len(plan.TopologyRequest) > 0 {
			state := stateData.TopologyRequest[0]
			p := plan.TopologyRequest[0]
			if topologyRequestStructureChanged(state.Required, p.Required) {
				resp.RequiresReplace = append(resp.RequiresReplace, path.Root("topology_request").AtListIndex(0).AtName("required"))
			}
			if topologyRequestStructureChanged(state.Preferred, p.Preferred) {
				resp.RequiresReplace = append(resp.RequiresReplace, path.Root("topology_request").AtListIndex(0).AtName("preferred"))
			}
		}
	}

	secretsModified := modifySecretsWOPlan(ctx, req.Private, configData.SecretsWO, configData.SecretsWOVersion, stateSecretsVersion, &plan.SecretsWOVersion, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedMountOpts, mountFlagsModified, mountDiags := modifyMountFlagsWOPlan(ctx, req.Private, configData.MountOptions, stateMountOpts, plan.MountOptions)
	resp.Diagnostics.Append(mountDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if mountFlagsModified {
		plan.MountOptions = updatedMountOpts
	}

	if secretsModified || mountFlagsModified {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func (r *CSIVolumeResource) readVolumeIntoModel(ctx context.Context, client *api.Client, ns, id string, data *csiVolumeModel, diags *diag.Diagnostics) {
	queryOpts := &api.QueryOptions{Namespace: ns}

	tflog.Debug(ctx, "Reading CSI volume", map[string]any{"id": id, "namespace": ns})
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

	// Don't overwrite capacity_min/capacity_max from API response.
	// The user-configured string (e.g. "10MiB") must remain unchanged to
	// avoid plan/apply mismatches with humanize formatting ("10 MiB").

	data.ControllerRequired = types.BoolValue(volume.ControllerRequired)
	data.ControllersExpected = types.Int64Value(int64(volume.ControllersExpected))
	data.ControllersHealthy = types.Int64Value(int64(volume.ControllersHealthy))
	data.NodesHealthy = types.Int64Value(int64(volume.NodesHealthy))
	data.NodesExpected = types.Int64Value(int64(volume.NodesExpected))
	data.Schedulable = types.BoolValue(volume.Schedulable)

	data.Capability = flattenCapabilities(volume.RequestedCapabilities)
	data.Topologies = flattenTopologiesToList(volume.Topologies)

	if volume.RequestedTopologies != nil {
		data.TopologyRequest = flattenTopologyRequests(volume.RequestedTopologies)
	}

	if volume.Context != nil {
		ctxElems := make(map[string]attr.Value, len(volume.Context))
		for k, v := range volume.Context {
			ctxElems[k] = types.StringValue(v)
		}
		data.Context = types.MapValueMust(types.StringType, ctxElems)
	} else {
		data.Context = types.MapNull(types.StringType)
	}

	// The Nomad API redacts mount_options and secrets, so we don't update them
	// from the response payload; they will remain as is.
}
