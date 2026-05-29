// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package volumes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

type capabilityModel struct {
	AccessMode     types.String `tfsdk:"access_mode"`
	AttachmentMode types.String `tfsdk:"attachment_mode"`
}

type mountOptionsModel struct {
	FSType     types.String   `tfsdk:"fs_type"`
	MountFlags []types.String `tfsdk:"mount_flags"`
}

type topologyModel struct {
	Segments map[string]types.String `tfsdk:"segments"`
}

type topologyGroupModel struct {
	Topology []topologyModel `tfsdk:"topology"`
}

type topologyRequestModel struct {
	Required  *topologyGroupModel `tfsdk:"required"`
	Preferred *topologyGroupModel `tfsdk:"preferred"`
}

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
	MountOptions     *mountOptionsModel      `tfsdk:"mount_options"`
	Secrets          map[string]types.String `tfsdk:"secrets"`
	SecretsWO        types.String            `tfsdk:"secrets_wo"`
	SecretsWOVersion types.Int64             `tfsdk:"secrets_wo_version"`
	Parameters       map[string]types.String `tfsdk:"parameters"`
	TopologyRequest  *topologyRequestModel   `tfsdk:"topology_request"`

	// Computed
	Capacity              types.Int64             `tfsdk:"capacity"`
	CapacityMinBytes      types.Int64             `tfsdk:"capacity_min_bytes"`
	CapacityMaxBytes      types.Int64             `tfsdk:"capacity_max_bytes"`
	ControllerRequired    types.Bool              `tfsdk:"controller_required"`
	ControllersExpected   types.Int64             `tfsdk:"controllers_expected"`
	ControllersHealthy    types.Int64             `tfsdk:"controllers_healthy"`
	PluginProvider        types.String            `tfsdk:"plugin_provider"`
	PluginProviderVersion types.String            `tfsdk:"plugin_provider_version"`
	NodesHealthy          types.Int64             `tfsdk:"nodes_healthy"`
	NodesExpected         types.Int64             `tfsdk:"nodes_expected"`
	Schedulable           types.Bool              `tfsdk:"schedulable"`
	Topologies            []topologyModel         `tfsdk:"topologies"`
	ExternalID            types.String            `tfsdk:"external_id"`
	Context               map[string]types.String `tfsdk:"context"`
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

func (r *CSIVolumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
	}

	for k, v := range secretsAttributes() {
		attrs[k] = v
	}

	resp.Schema = schema.Schema{
		Description: "Manages the lifecycle of creating and deleting CSI volumes.",
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
					"preferred": schema.SingleNestedBlock{
						Description: "Preferred topologies indicate that the volume should be created in a location accessible from some of the listed topologies.",
						Blocks: map[string]schema.Block{
							"topology": topologyBlock(),
						},
					},
				},
			},
			"topologies": schema.ListNestedBlock{
				Description: "The topologies of the volume.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"segments": schema.MapAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
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
		SnapshotID:            data.SnapshotID.ValueString(),
		CloneID:               data.CloneID.ValueString(),
		RequestedCapacityMin:  int64(capMin),
		RequestedCapacityMax:  int64(capMax),
		RequestedCapabilities: parseCapabilities(data.Capability),
		RequestedTopologies:   parseTopologyRequest(data.TopologyRequest),
		Secrets:               secrets,
		Parameters:            toMapStringString(data.Parameters),
	}

	if data.MountOptions != nil {
		volume.MountOptions = parseMountOptions(data.MountOptions)
	}

	ns := data.Namespace.ValueString()
	if ns == "" {
		ns = "default"
	}
	opts := &api.WriteOptions{Namespace: ns}

	tflog.Debug(ctx, "Creating CSI volume", map[string]any{"volume_id": volume.ID, "namespace": ns})

	deadline := time.Now().Add(10 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		_, _, err := client.CSIVolumes().Create(volume, opts)
		if err != nil {
			lastErr = fmt.Errorf("error creating CSI volume: %s", err)
			if csiErrIsRetryable(lastErr) {
				time.Sleep(2 * time.Second)
				continue
			}
			resp.Diagnostics.AddError("Error creating CSI volume", lastErr.Error())
			return
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		resp.Diagnostics.AddError("Error creating CSI volume", lastErr.Error())
		return
	}

	tflog.Debug(ctx, "Created CSI volume", map[string]any{"volume_id": volume.ID, "namespace": ns})
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
		SnapshotID:            data.SnapshotID.ValueString(),
		CloneID:               data.CloneID.ValueString(),
		RequestedCapacityMin:  int64(capMin),
		RequestedCapacityMax:  int64(capMax),
		RequestedCapabilities: parseCapabilities(data.Capability),
		RequestedTopologies:   parseTopologyRequest(data.TopologyRequest),
		Secrets:               secrets,
		Parameters:            toMapStringString(data.Parameters),
	}

	if data.MountOptions != nil {
		volume.MountOptions = parseMountOptions(data.MountOptions)
	}

	ns := data.Namespace.ValueString()
	if ns == "" {
		ns = "default"
	}
	opts := &api.WriteOptions{Namespace: ns}

	tflog.Debug(ctx, "Updating CSI volume", map[string]any{"volume_id": volume.ID, "namespace": ns})

	_, _, err := client.CSIVolumes().Create(volume, opts)
	if err != nil {
		resp.Diagnostics.AddError("Error updating CSI volume", err.Error())
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

	deadline := time.Now().Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		err := client.CSIVolumes().Delete(id, opts)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		tflog.Debug(ctx, "Deleted CSI volume", map[string]any{"volume_id": id})
		return
	}
	resp.Diagnostics.AddError("Error deleting CSI volume", fmt.Sprintf("timed out deleting CSI volume %q", id))
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

	stateVersion := types.Int64Null()
	if !req.State.Raw.IsNull() {
		var stateData csiVolumeModel
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

	if !data.CapacityMin.IsNull() && data.CapacityMin.ValueString() != "" {
		data.CapacityMin = types.StringValue(humanize.IBytes(uint64(volume.RequestedCapacityMin)))
	}
	if !data.CapacityMax.IsNull() && data.CapacityMax.ValueString() != "" {
		data.CapacityMax = types.StringValue(humanize.IBytes(uint64(volume.RequestedCapacityMax)))
	}

	data.ControllerRequired = types.BoolValue(volume.ControllerRequired)
	data.ControllersExpected = types.Int64Value(int64(volume.ControllersExpected))
	data.ControllersHealthy = types.Int64Value(int64(volume.ControllersHealthy))
	data.NodesHealthy = types.Int64Value(int64(volume.NodesHealthy))
	data.NodesExpected = types.Int64Value(int64(volume.NodesExpected))
	data.Schedulable = types.BoolValue(volume.Schedulable)

	data.Capability = flattenCapabilities(volume.RequestedCapabilities)
	data.Topologies = flattenTopologies(volume.Topologies)

	if volume.RequestedTopologies != nil {
		data.TopologyRequest = flattenTopologyRequests(volume.RequestedTopologies)
	}

	data.Context = fromMapStringString(volume.Context)

	// The Nomad API redacts mount_options and secrets, so we don't update them
	// from the response payload; they will remain as is.
}

func capabilityBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Capabilities intended to be used in a job. At least one capability must be provided.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"access_mode": schema.StringAttribute{
					Required:    true,
					Description: "Defines whether a volume should be available concurrently.",
					Validators: []validator.String{
						stringvalidator.OneOf(
							"single-node-reader-only",
							"single-node-writer",
							"multi-node-reader-only",
							"multi-node-single-writer",
							"multi-node-multi-writer",
						),
					},
				},
				"attachment_mode": schema.StringAttribute{
					Required:    true,
					Description: "The storage API that will be used by the volume.",
					Validators: []validator.String{
						stringvalidator.OneOf(
							"block-device",
							"file-system",
						),
					},
				},
			},
		},
	}
}

func mountOptionsBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Options for mounting 'block-device' volumes without a pre-formatted file system.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"fs_type": schema.StringAttribute{
					Optional:    true,
					Description: "The file system type.",
				},
				"mount_flags": schema.ListAttribute{
					ElementType: types.StringType,
					Optional:    true,
					Description: "The flags passed to mount.",
				},
			},
		},
	}
}

func topologyBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Defines the location for the volume.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"segments": schema.MapAttribute{
					ElementType: types.StringType,
					Required:    true,
					Description: "Define the attributes for the topology request.",
					PlanModifiers: []planmodifier.Map{
						mapplanmodifier.RequiresReplace(),
					},
				},
			},
		},
	}
}

func computedAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"capacity": schema.Int64Attribute{
			Computed:    true,
			Description: "The capacity of the volume in bytes.",
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"capacity_min_bytes": schema.Int64Attribute{
			Computed: true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"capacity_max_bytes": schema.Int64Attribute{
			Computed: true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"controller_required": schema.BoolAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"controllers_expected": schema.Int64Attribute{
			Computed: true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"controllers_healthy": schema.Int64Attribute{
			Computed: true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"plugin_provider": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"plugin_provider_version": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"nodes_healthy": schema.Int64Attribute{
			Computed: true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"nodes_expected": schema.Int64Attribute{
			Computed: true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"schedulable": schema.BoolAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func secretsAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"secrets": schema.MapAttribute{
			ElementType:        types.StringType,
			Optional:           true,
			Sensitive:          true,
			Description:        "An optional key-value map of strings used as credentials for publishing and unpublishing volumes. Deprecated: use secrets_wo instead.",
			DeprecationMessage: "Use secrets_wo to avoid storing secrets in Terraform state.",
		},
		"secrets_wo": schema.StringAttribute{
			Optional:    true,
			WriteOnly:   true,
			Description: "JSON-encoded map of secrets used as credentials for publishing and unpublishing volumes. Use jsonencode() to set this value. This value is write-only and will not be stored in Terraform state.",
		},
		"secrets_wo_version": schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: "Version counter for secrets_wo. Increments automatically when the write-only secret changes, or set manually to trigger an update.",
		},
	}
}

func parseCapabilities(caps []capabilityModel) []*api.CSIVolumeCapability {
	result := make([]*api.CSIVolumeCapability, len(caps))
	for i, c := range caps {
		result[i] = &api.CSIVolumeCapability{
			AccessMode:     api.CSIVolumeAccessMode(c.AccessMode.ValueString()),
			AttachmentMode: api.CSIVolumeAttachmentMode(c.AttachmentMode.ValueString()),
		}
	}
	return result
}

func flattenCapabilities(caps []*api.CSIVolumeCapability) []capabilityModel {
	result := make([]capabilityModel, 0, len(caps))
	for _, c := range caps {
		if c == nil {
			continue
		}
		result = append(result, capabilityModel{
			AccessMode:     types.StringValue(string(c.AccessMode)),
			AttachmentMode: types.StringValue(string(c.AttachmentMode)),
		})
	}
	return result
}

func parseMountOptions(m *mountOptionsModel) *api.CSIMountOptions {
	if m == nil {
		return nil
	}
	opts := &api.CSIMountOptions{}
	if !m.FSType.IsNull() && !m.FSType.IsUnknown() {
		opts.FSType = m.FSType.ValueString()
	}
	if len(m.MountFlags) > 0 {
		opts.MountFlags = make([]string, len(m.MountFlags))
		for i, f := range m.MountFlags {
			opts.MountFlags[i] = f.ValueString()
		}
	}
	return opts
}

func parseTopologyRequest(req *topologyRequestModel) *api.CSITopologyRequest {
	if req == nil {
		return nil
	}
	result := &api.CSITopologyRequest{}
	if req.Required != nil {
		result.Required = parseTopologies(req.Required.Topology)
	}
	if req.Preferred != nil {
		result.Preferred = parseTopologies(req.Preferred.Topology)
	}
	return result
}

func parseTopologies(topos []topologyModel) []*api.CSITopology {
	result := make([]*api.CSITopology, len(topos))
	for i, t := range topos {
		segments := make(map[string]string, len(t.Segments))
		for k, v := range t.Segments {
			segments[k] = v.ValueString()
		}
		result[i] = &api.CSITopology{Segments: segments}
	}
	return result
}

func flattenTopologies(topos []*api.CSITopology) []topologyModel {
	result := make([]topologyModel, 0, len(topos))
	for _, t := range topos {
		if t == nil {
			continue
		}
		segments := make(map[string]types.String, len(t.Segments))
		for k, v := range t.Segments {
			segments[k] = types.StringValue(v)
		}
		result = append(result, topologyModel{Segments: segments})
	}
	return result
}

func flattenTopologyRequests(req *api.CSITopologyRequest) *topologyRequestModel {
	if req == nil {
		return nil
	}
	result := &topologyRequestModel{}
	if req.Required != nil {
		result.Required = &topologyGroupModel{
			Topology: flattenTopologies(req.Required),
		}
	}
	if req.Preferred != nil {
		result.Preferred = &topologyGroupModel{
			Topology: flattenTopologies(req.Preferred),
		}
	}
	return result
}

func toMapStringString(m map[string]types.String) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v.ValueString()
	}
	return result
}

func fromMapStringString(m map[string]string) map[string]types.String {
	if m == nil {
		return nil
	}
	result := make(map[string]types.String, len(m))
	for k, v := range m {
		result[k] = types.StringValue(v)
	}
	return result
}

func parseCapacity(capMinAttr, capMaxAttr types.String) (uint64, uint64, diag.Diagnostics) {
	var diags diag.Diagnostics
	var capMin, capMax uint64
	var err error

	if !capMinAttr.IsNull() && capMinAttr.ValueString() != "" {
		capMin, err = humanize.ParseBytes(capMinAttr.ValueString())
		if err != nil {
			diags.AddError("Invalid capacity_min", err.Error())
		}
	}
	if !capMaxAttr.IsNull() && capMaxAttr.ValueString() != "" {
		capMax, err = humanize.ParseBytes(capMaxAttr.ValueString())
		if err != nil {
			diags.AddError("Invalid capacity_max", err.Error())
		}
	}

	if capMax > 0 && capMax < capMin {
		diags.AddError("Invalid capacity", "capacity_max less than capacity_min")
	}

	return capMin, capMax, diags
}

func resolveSecrets(secretsWO types.String, secretsLegacy map[string]types.String) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !secretsWO.IsNull() && !secretsWO.IsUnknown() {
		var result map[string]string
		if err := json.Unmarshal([]byte(secretsWO.ValueString()), &result); err != nil {
			diags.AddError("Invalid secrets_wo", fmt.Sprintf("secrets_wo must be a JSON-encoded map of strings: %s", err))
			return nil, diags
		}
		return result, diags
	}

	return toMapStringString(secretsLegacy), diags
}

func checkCapacity(capacity, reqMin uint64) diag.Diagnostics {
	var diags diag.Diagnostics
	if capacity == 0 || reqMin == 0 {
		return diags
	}
	if reqMin > capacity {
		diags.AddWarning(
			"capacity out of requested range",
			fmt.Sprintf(
				"A volume expand operation may not have occurred on Nomad prior to v1.6.3.\n"+
					"Real capacity after apply: %d -- Requested min: %d\n\n"+
					"Upgrading Nomad will enable volume expansion (for CSI plugins that support it),\n"+
					"or you may change capacity_min to fit the actual capacity.",
				capacity, reqMin,
			),
		)
	}
	return diags
}

func csiErrIsRetryable(err error) bool {
	ignore := []string{
		"already exist",
		"requested capacity",
		"LimitBytes cannot be less than",
	}
	for _, e := range ignore {
		if strings.Contains(err.Error(), e) {
			return false
		}
	}
	return true
}

type privateStateSetter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

type privateStateGetter interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

type privateHashWrapper struct {
	Hash string `json:"hash"`
}

func handleSecretsWOHash(ctx context.Context, secretsWO types.String, private privateStateSetter, diags *diag.Diagnostics) {
	if !secretsWO.IsNull() && !secretsWO.IsUnknown() {
		h := sha256.New()
		h.Write([]byte(secretsWO.ValueString()))
		hash := hex.EncodeToString(h.Sum(nil))

		data, err := json.Marshal(privateHashWrapper{Hash: hash})
		if err != nil {
			diags.AddError("Error storing secrets hash", err.Error())
			return
		}
		d := private.SetKey(ctx, "secrets_wo_hash", data)
		diags.Append(d...)
	}
}

func modifySecretsWOPlan(ctx context.Context, private privateStateGetter, configSecretsWO types.String, configSecretsWOVersion types.Int64, stateVersion types.Int64, planVersion *types.Int64, diags *diag.Diagnostics) bool {
	versionExplicit := !configSecretsWOVersion.IsNull()

	if configSecretsWO.IsNull() && !versionExplicit {
		if !planVersion.Equal(stateVersion) {
			*planVersion = stateVersion
			return true
		}
		return false
	}

	if !configSecretsWO.IsNull() && !versionExplicit {
		h := sha256.New()
		h.Write([]byte(configSecretsWO.ValueString()))
		newHash := hex.EncodeToString(h.Sum(nil))

		oldHash := ""
		data, d := private.GetKey(ctx, "secrets_wo_hash")
		diags.Append(d...)
		if diags.HasError() {
			return false
		}
		if len(data) > 0 {
			var wrapper privateHashWrapper
			if err := json.Unmarshal(data, &wrapper); err == nil {
				oldHash = wrapper.Hash
			}
		}

		if newHash != oldHash {
			oldVer := int64(0)
			if !stateVersion.IsNull() {
				oldVer = stateVersion.ValueInt64()
			}
			*planVersion = types.Int64Value(oldVer + 1)
		} else {
			*planVersion = stateVersion
		}
		return true
	}

	return false
}

func parseImportID(importID string) (string, string, diag.Diagnostics) {
	var diags diag.Diagnostics
	sepIdx := strings.LastIndex(importID, "@")
	if sepIdx == -1 {
		diags.AddError("Invalid import ID", "Import ID should follow the pattern <id>@<namespace>")
		return "", "", diags
	}
	id := importID[:sepIdx]
	ns := importID[sepIdx+1:]
	if id == "" {
		diags.AddError("Invalid import ID", "Missing resource ID in import")
		return "", "", diags
	}
	if ns == "" {
		diags.AddError("Invalid import ID", "Missing namespace in import")
		return "", "", diags
	}
	return id, ns, diags
}
