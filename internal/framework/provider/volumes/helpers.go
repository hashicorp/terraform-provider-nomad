// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package volumes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type capabilityModel struct {
	AccessMode     types.String `tfsdk:"access_mode"`
	AttachmentMode types.String `tfsdk:"attachment_mode"`
}

type mountOptionsModel struct {
	FSType              types.String   `tfsdk:"fs_type"`
	MountFlags          []types.String `tfsdk:"mount_flags"`
	MountFlagsWO        types.List     `tfsdk:"mount_flags_wo"`
	MountFlagsWOVersion types.Int64    `tfsdk:"mount_flags_wo_version"`
}

type topologyModel struct {
	Segments map[string]types.String `tfsdk:"segments"`
}

type topologyGroupModel struct {
	Topology []topologyModel `tfsdk:"topology"`
}

type topologyRequestModel struct {
	Required  []topologyGroupModel `tfsdk:"required"`
	Preferred []topologyGroupModel `tfsdk:"preferred"`
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

func mapValueToStringMap(m types.Map) map[string]string {
	if m.IsNull() || m.IsUnknown() || len(m.Elements()) == 0 {
		return nil
	}
	result := make(map[string]string, len(m.Elements()))
	for k, v := range m.Elements() {
		result[k] = v.(types.String).ValueString()
	}
	return result
}

func stringMapToMapValue(m map[string]string) types.Map {
	if m == nil {
		return types.MapNull(types.StringType)
	}
	elems := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elems[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, elems)
}

// capacityValidator validates that a string is parseable as a human-readable byte size.
type capacityValidator struct{}

func (v capacityValidator) Description(_ context.Context) string {
	return "value must be a valid human-readable byte size (e.g., \"10 GiB\", \"500MB\")"
}

func (v capacityValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v capacityValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	if _, err := humanize.ParseBytes(s); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid capacity value",
			fmt.Sprintf("unable to parse %q as human-readable capacity: %s", s, err),
		)
	}
}

// capacityPlanModifier normalizes capacity strings and performs semantic equality
// comparison to suppress false diffs (e.g., "5mb" vs "5.0 MiB" are the same value).
type capacityPlanModifier struct{}

func (m capacityPlanModifier) Description(_ context.Context) string {
	return "Normalizes capacity values and suppresses diffs when values are semantically equal."
}

func (m capacityPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m capacityPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	configBytes, err := humanize.ParseBytes(req.ConfigValue.ValueString())
	if err != nil {
		return
	}

	// If state exists and resolves to the same byte count, preserve state value
	// to avoid a spurious diff.
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		stateBytes, err := humanize.ParseBytes(req.StateValue.ValueString())
		if err == nil && configBytes == stateBytes {
			resp.PlanValue = req.StateValue
			return
		}
	}

	// When the value is changing, pass config through as-is. State will store
	// the user's exact string; semantic equality handles future no-op diffs.
	resp.PlanValue = req.ConfigValue
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

func resolveSecrets(secretsWO types.Map, secretsLegacy map[string]types.String) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !secretsWO.IsNull() && !secretsWO.IsUnknown() {
		return mapValueToStringMap(secretsWO), diags
	}

	return toMapStringString(secretsLegacy), diags
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

// hashMapValue produces a deterministic SHA256 hash of a types.Map by sorting
// keys and encoding as JSON.
func hashMapValue(m types.Map) string {
	canonical, _ := json.Marshal(mapValueToStringMap(m))
	h := sha256.New()
	h.Write(canonical)
	return hex.EncodeToString(h.Sum(nil))
}

// hashListValue produces a deterministic SHA256 hash of a types.List by
// encoding its elements as JSON.
func hashListValue(l types.List) string {
	var elems []string
	l.ElementsAs(context.Background(), &elems, false)
	canonical, _ := json.Marshal(elems)
	h := sha256.New()
	h.Write(canonical)
	return hex.EncodeToString(h.Sum(nil))
}

func handleSecretsWOHash(ctx context.Context, secretsWO types.Map, private privateStateSetter, diags *diag.Diagnostics) {
	if !secretsWO.IsNull() && !secretsWO.IsUnknown() {
		hash := hashMapValue(secretsWO)
		data, err := json.Marshal(privateHashWrapper{Hash: hash})
		if err != nil {
			diags.AddError("Error storing secrets hash", err.Error())
			return
		}
		d := private.SetKey(ctx, "secrets_wo_hash", data)
		diags.Append(d...)
	}
}

func modifySecretsWOPlan(ctx context.Context, private privateStateGetter, configSecretsWO types.Map, configSecretsWOVersion types.Int64, stateVersion types.Int64, planVersion *types.Int64, diags *diag.Diagnostics) bool {
	versionExplicit := !configSecretsWOVersion.IsNull()

	if configSecretsWO.IsNull() && !versionExplicit {
		if !planVersion.Equal(stateVersion) {
			*planVersion = stateVersion
			return true
		}
		return false
	}

	if !configSecretsWO.IsNull() && !versionExplicit {
		newHash := hashMapValue(configSecretsWO)

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

func parseMountOptions(m *mountOptionsModel) *api.CSIMountOptions {
	if m == nil {
		return nil
	}
	opts := &api.CSIMountOptions{}
	if !m.FSType.IsNull() && !m.FSType.IsUnknown() {
		opts.FSType = m.FSType.ValueString()
	}
	// Use mount_flags_wo if set, otherwise fall back to mount_flags.
	if !m.MountFlagsWO.IsNull() && !m.MountFlagsWO.IsUnknown() {
		var flags []types.String
		m.MountFlagsWO.ElementsAs(context.Background(), &flags, false)
		if len(flags) > 0 {
			opts.MountFlags = make([]string, len(flags))
			for i, f := range flags {
				opts.MountFlags[i] = f.ValueString()
			}
		}
	} else if len(m.MountFlags) > 0 {
		opts.MountFlags = make([]string, len(m.MountFlags))
		for i, f := range m.MountFlags {
			opts.MountFlags[i] = f.ValueString()
		}
	}
	return opts
}

func handleMountFlagsWOHash(ctx context.Context, mountOpts []mountOptionsModel, private privateStateSetter, diags *diag.Diagnostics) {
	if len(mountOpts) == 0 {
		return
	}
	flagsWO := mountOpts[0].MountFlagsWO
	if !flagsWO.IsNull() && !flagsWO.IsUnknown() {
		hash := hashListValue(flagsWO)
		data, err := json.Marshal(privateHashWrapper{Hash: hash})
		if err != nil {
			diags.AddError("Error storing mount_flags hash", err.Error())
			return
		}
		d := private.SetKey(ctx, "mount_flags_wo_hash", data)
		diags.Append(d...)
	}
}

func modifyMountFlagsWOPlan(ctx context.Context, private privateStateGetter, configMountOpts []mountOptionsModel, stateMountOpts []mountOptionsModel, planMountOpts []mountOptionsModel) ([]mountOptionsModel, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(configMountOpts) == 0 || len(planMountOpts) == 0 {
		return planMountOpts, false, diags
	}

	configFlagsWO := configMountOpts[0].MountFlagsWO
	configVersion := configMountOpts[0].MountFlagsWOVersion
	versionExplicit := !configVersion.IsNull()

	stateVersion := types.Int64Null()
	if len(stateMountOpts) > 0 {
		stateVersion = stateMountOpts[0].MountFlagsWOVersion
	}

	planVersion := &planMountOpts[0].MountFlagsWOVersion

	if configFlagsWO.IsNull() && !versionExplicit {
		if !planVersion.Equal(stateVersion) {
			*planVersion = stateVersion
			return planMountOpts, true, diags
		}
		return planMountOpts, false, diags
	}

	if !configFlagsWO.IsNull() && !versionExplicit {
		newHash := hashListValue(configFlagsWO)

		oldHash := ""
		data, d := private.GetKey(ctx, "mount_flags_wo_hash")
		diags.Append(d...)
		if diags.HasError() {
			return planMountOpts, false, diags
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
		return planMountOpts, true, diags
	}

	return planMountOpts, false, diags
}

func secretsAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"secrets": schema.MapAttribute{
			ElementType:        types.StringType,
			Optional:           true,
			Sensitive:          true,
			Description:        "An optional key-value map of strings used as credentials for publishing and unpublishing volumes. Deprecated: use secrets_wo instead.",
			DeprecationMessage: "Use secrets_wo to avoid storing secrets in Terraform state.",
			Validators: []validator.Map{
				mapvalidator.PreferWriteOnlyAttribute(path.MatchRoot("secrets_wo")),
				mapvalidator.ConflictsWith(path.MatchRoot("secrets_wo")),
			},
		},
		"secrets_wo": schema.MapAttribute{
			ElementType: types.StringType,
			Optional:    true,
			WriteOnly:   true,
			Description: "Write-only map of secrets used as credentials for publishing and unpublishing volumes. This value will not be stored in Terraform state.",
			Validators: []validator.Map{
				mapvalidator.ConflictsWith(path.MatchRoot("secrets")),
			},
		},
		"secrets_wo_version": schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: "Version counter for secrets_wo. Increments automatically when the write-only secret changes, or set manually to trigger an update.",
			Validators: []validator.Int64{
				int64validator.ConflictsWith(path.MatchRoot("secrets")),
				int64validator.AlsoRequires(path.MatchRoot("secrets_wo")),
			},
		},
	}
}

func mountOptionsBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Options for mounting 'block-device' volumes without a pre-formatted file system.",
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Validators: []validator.Object{
				mountOptionsValidator{},
			},
			Attributes: map[string]schema.Attribute{
				"fs_type": schema.StringAttribute{
					Optional:    true,
					Description: "The file system type.",
				},
				"mount_flags": schema.ListAttribute{
					ElementType:        types.StringType,
					Optional:           true,
					Sensitive:          true,
					Description:        "The flags passed to mount. Deprecated: use mount_flags_wo instead.",
					DeprecationMessage: "Use mount_flags_wo to avoid storing mount flags in Terraform state.",
				},
				"mount_flags_wo": schema.ListAttribute{
					ElementType: types.StringType,
					Optional:    true,
					WriteOnly:   true,
					Description: "Write-only list of flags passed to mount. This value will not be stored in Terraform state.",
				},
				"mount_flags_wo_version": schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Description: "Version counter for mount_flags_wo. Increments automatically when the write-only mount flags change, or set manually to trigger an update.",
				},
			},
		},
	}
}

// mountOptionsValidator validates cross-attribute constraints within the
// mount_options block.
type mountOptionsValidator struct{}

func (v mountOptionsValidator) Description(_ context.Context) string {
	return "validates mount_flags/mount_flags_wo mutual exclusivity and mount_flags_wo_version dependency"
}

func (v mountOptionsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v mountOptionsValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	var opts mountOptionsModel
	resp.Diagnostics.Append(req.ConfigValue.As(ctx, &opts, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasMountFlags := len(opts.MountFlags) > 0
	hasMountFlagsWO := !opts.MountFlagsWO.IsNull()

	if hasMountFlags && hasMountFlagsWO {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("mount_flags"),
			"Conflicting Attributes",
			"mount_flags and mount_flags_wo cannot both be set. Use mount_flags_wo to avoid storing mount flags in Terraform state.",
		)
	}
	if hasMountFlags && !hasMountFlagsWO {
		resp.Diagnostics.AddAttributeWarning(
			req.Path.AtName("mount_flags"),
			"Prefer Write-Only Attribute",
			"Prefer mount_flags_wo over mount_flags to avoid storing sensitive mount flags in Terraform state.",
		)
	}
	if !opts.MountFlagsWOVersion.IsNull() && !hasMountFlagsWO {
		resp.Diagnostics.AddAttributeError(
			req.Path.AtName("mount_flags_wo_version"),
			"Missing Required Attribute",
			"mount_flags_wo_version requires mount_flags_wo to be set.",
		)
	}
}

func capabilityBlock(requiresReplace bool) schema.SetNestedBlock {
	accessModeAttr := schema.StringAttribute{
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
	}
	attachmentModeAttr := schema.StringAttribute{
		Required:    true,
		Description: "The storage API that will be used by the volume.",
		Validators: []validator.String{
			stringvalidator.OneOf(
				"block-device",
				"file-system",
			),
		},
	}

	if requiresReplace {
		accessModeAttr.PlanModifiers = []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		}
		attachmentModeAttr.PlanModifiers = []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		}
	}

	return schema.SetNestedBlock{
		Description: "Capabilities intended to be used in a job. At least one capability must be provided.",
		Validators: []validator.Set{
			setvalidator.SizeAtLeast(1),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"access_mode":     accessModeAttr,
				"attachment_mode": attachmentModeAttr,
			},
		},
	}
}

func topologyRequestBlock(includePreferred bool) schema.ListNestedBlock {
	blocks := map[string]schema.Block{
		"required": schema.ListNestedBlock{
			Description: "Required topologies indicate that the volume must be created in a location accessible from all the listed topologies.",
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
			},
			NestedObject: schema.NestedBlockObject{
				Blocks: map[string]schema.Block{
					"topology": topologyBlock(),
				},
			},
		},
	}

	if includePreferred {
		blocks["preferred"] = schema.ListNestedBlock{
			Description: "Preferred topologies indicate that the volume should be created in a location accessible from some of the listed topologies.",
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
			},
			NestedObject: schema.NestedBlockObject{
				Blocks: map[string]schema.Block{
					"topology": topologyBlock(),
				},
			},
		}
	}

	return schema.ListNestedBlock{
		Description: "Specify locations (region, zone, rack, etc.) where the provisioned volume is accessible from.",
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		NestedObject: schema.NestedBlockObject{
			Blocks: blocks,
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
		},
		"capacity_min_bytes": schema.Int64Attribute{
			Computed: true,
		},
		"capacity_max_bytes": schema.Int64Attribute{
			Computed: true,
		},
		"controller_required": schema.BoolAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"controllers_expected": schema.Int64Attribute{
			Computed: true,
		},
		"controllers_healthy": schema.Int64Attribute{
			Computed: true,
		},
		"plugin_provider": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"plugin_provider_version": schema.StringAttribute{
			Computed: true,
		},
		"nodes_healthy": schema.Int64Attribute{
			Computed: true,
		},
		"nodes_expected": schema.Int64Attribute{
			Computed: true,
		},
		"schedulable": schema.BoolAttribute{
			Computed: true,
		},
		"topologies": schema.ListNestedAttribute{
			Computed:    true,
			Description: "The topologies of the volume.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"segments": schema.MapAttribute{
						ElementType: types.StringType,
						Computed:    true,
					},
				},
			},
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
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

func parseTopologyRequest(req []topologyRequestModel) *api.CSITopologyRequest {
	if len(req) == 0 {
		return nil
	}
	tr := req[0]
	result := &api.CSITopologyRequest{}
	if len(tr.Required) > 0 {
		result.Required = parseTopologies(tr.Required[0].Topology)
	}
	if len(tr.Preferred) > 0 {
		result.Preferred = parseTopologies(tr.Preferred[0].Topology)
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

var topologyAttrTypes = map[string]attr.Type{
	"segments": types.MapType{ElemType: types.StringType},
}

func flattenTopologiesToList(topos []*api.CSITopology) types.List {
	if len(topos) == 0 {
		return types.ListValueMust(types.ObjectType{AttrTypes: topologyAttrTypes}, []attr.Value{})
	}
	elems := make([]attr.Value, 0, len(topos))
	for _, t := range topos {
		if t == nil {
			continue
		}
		segElems := make(map[string]attr.Value, len(t.Segments))
		for k, v := range t.Segments {
			segElems[k] = types.StringValue(v)
		}
		segMap := types.MapValueMust(types.StringType, segElems)
		obj := types.ObjectValueMust(topologyAttrTypes, map[string]attr.Value{
			"segments": segMap,
		})
		elems = append(elems, obj)
	}
	return types.ListValueMust(types.ObjectType{AttrTypes: topologyAttrTypes}, elems)
}

func flattenTopologyRequests(req *api.CSITopologyRequest) []topologyRequestModel {
	if req == nil {
		return nil
	}
	result := topologyRequestModel{}
	if req.Required != nil {
		result.Required = []topologyGroupModel{{
			Topology: flattenTopologies(req.Required),
		}}
	}
	if req.Preferred != nil {
		result.Preferred = []topologyGroupModel{{
			Topology: flattenTopologies(req.Preferred),
		}}
	}
	return []topologyRequestModel{result}
}

// topologyRequestStructureChanged detects structural changes in topology_request
// that should trigger resource replacement (equivalent to SDKv2 ForceNew on blocks).
// This covers adding/removing blocks or changing the number of topology entries,
// which RequiresReplace on the segments attribute alone cannot detect.
func topologyRequestStructureChanged(state, plan []topologyGroupModel) bool {
	if len(state) != len(plan) {
		return true
	}
	for i := range state {
		if len(state[i].Topology) != len(plan[i].Topology) {
			return true
		}
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

func csiErrIsRetryable(err error) bool {
	ignore := []string{
		"already exist",
		"already exists",
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
