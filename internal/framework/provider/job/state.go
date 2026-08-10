// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type jobStateV0 struct {
	Jobspec              string                  `json:"jobspec"`
	PolicyOverride       bool                    `json:"policy_override"`
	PreserveCounts       bool                    `json:"preserve_counts"`
	PreserveResources    bool                    `json:"preserve_resources"`
	DeregisterOnDestroy  bool                    `json:"deregister_on_destroy"`
	DeregisterOnIDChange bool                    `json:"deregister_on_id_change"`
	Detach               bool                    `json:"detach"`
	DeploymentID         string                  `json:"deployment_id"`
	DeploymentStatus     string                  `json:"deployment_status"`
	HCL2                 []hcl2StateV0           `json:"hcl2"`
	JSON                 bool                    `json:"json"`
	ModifyIndex          string                  `json:"modify_index"`
	Name                 string                  `json:"name"`
	Namespace            string                  `json:"namespace"`
	Type                 string                  `json:"type"`
	RerunIfDead          bool                    `json:"rerun_if_dead"`
	Status               string                  `json:"status"`
	StatusDescription    string                  `json:"status_description"`
	Version              int64                   `json:"version"`
	SubmitTime           string                  `json:"submit_time"`
	CreateIndex          int64                   `json:"create_index"`
	Stop                 bool                    `json:"stop"`
	Priority             int64                   `json:"priority"`
	ParentID             string                  `json:"parent_id"`
	Stable               bool                    `json:"stable"`
	AllAtOnce            bool                    `json:"all_at_once"`
	Constraints          []constraintStateV0     `json:"constraints"`
	UpdateStrategy       []updateStrategyStateV0 `json:"update_strategy"`
	PeriodicConfig       []periodicConfigStateV0 `json:"periodic_config"`
	Region               string                  `json:"region"`
	Datacenters          []string                `json:"datacenters"`
	ReadAllocationIDs    bool                    `json:"read_allocation_ids"`
	AllocationIDs        []string                `json:"allocation_ids"`
	TaskGroups           []taskGroupStateV0      `json:"task_groups"`
	PurgeOnDestroy       bool                    `json:"purge_on_destroy"`
}

type hcl2StateV0 struct {
	AllowFS bool              `json:"allow_fs"`
	Vars    map[string]string `json:"vars"`
}

type constraintStateV0 struct {
	LTarget string `json:"ltarget"`
	RTarget string `json:"rtarget"`
	Operand string `json:"operand"`
}

type updateStrategyStateV0 struct {
	Stagger         string `json:"stagger"`
	MaxParallel     int64  `json:"max_parallel"`
	HealthCheck     string `json:"health_check"`
	MinHealthyTime  string `json:"min_healthy_time"`
	HealthyDeadline string `json:"healthy_deadline"`
	AutoRevert      bool   `json:"auto_revert"`
	Canary          int64  `json:"canary"`
}

type periodicConfigStateV0 struct {
	Enabled         bool   `json:"enabled"`
	Spec            string `json:"spec"`
	SpecType        string `json:"spec_type"`
	ProhibitOverlap bool   `json:"prohibit_overlap"`
	Timezone        string `json:"timezone"`
}

type taskGroupStateV0 struct {
	Name           string                  `json:"name"`
	Count          int64                   `json:"count"`
	UpdateStrategy []updateStrategyStateV0 `json:"update_strategy"`
	Tasks          []taskStateV0           `json:"task"`
	Volumes        []volumeStateV0         `json:"volumes"`
	Meta           map[string]string       `json:"meta"`
}

type taskStateV0 struct {
	Name         string               `json:"name"`
	Driver       string               `json:"driver"`
	Meta         map[string]string    `json:"meta"`
	VolumeMounts []volumeMountStateV0 `json:"volume_mounts"`
}

type volumeMountStateV0 struct {
	Volume      string `json:"volume"`
	Destination string `json:"destination"`
	ReadOnly    bool   `json:"read_only"`
}

type volumeStateV0 struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	ReadOnly bool   `json:"read_only"`
	Source   string `json:"source"`
}

func (r *JobResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {StateUpgrader: upgradeJobStateV0toV1},
	}
}

func upgradeJobStateV0toV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var old jobStateV0
	if err := json.Unmarshal(req.RawState.JSON, &old); err != nil {
		resp.Diagnostics.AddError("Failed to parse prior job state for upgrade", err.Error())
		return
	}

	data := jobResourceModel{
		Jobspec:              types.StringValue(old.Jobspec),
		PolicyOverride:       types.BoolValue(old.PolicyOverride),
		PreserveCounts:       types.BoolValue(old.PreserveCounts),
		PreserveResources:    types.BoolValue(old.PreserveResources),
		DeregisterOnDestroy:  types.BoolValue(old.DeregisterOnDestroy),
		DeregisterOnIDChange: types.BoolValue(old.DeregisterOnIDChange),
		Detach:               types.BoolValue(old.Detach),
		JSON:                 types.BoolValue(old.JSON),
		RerunIfDead:          types.BoolValue(old.RerunIfDead),
		ReadAllocationIDs:    types.BoolValue(old.ReadAllocationIDs),
		PurgeOnDestroy:       types.BoolValue(old.PurgeOnDestroy),
		DeploymentID:         types.StringValue(old.DeploymentID),
		DeploymentStatus:     types.StringValue(old.DeploymentStatus),
		ModifyIndex:          types.StringValue(old.ModifyIndex),
		Name:                 types.StringValue(old.Name),
		ID:                   types.StringValue(old.Name),
		Namespace:            types.StringValue(old.Namespace),
		Type:                 types.StringValue(old.Type),
		Status:               types.StringValue(old.Status),
		StatusDescription:    types.StringValue(old.StatusDescription),
		Version:              types.Int64Value(old.Version),
		SubmitTime:           types.StringValue(old.SubmitTime),
		CreateIndex:          types.Int64Value(old.CreateIndex),
		Stop:                 types.BoolValue(old.Stop),
		Priority:             types.Int64Value(old.Priority),
		ParentID:             types.StringValue(old.ParentID),
		Stable:               types.BoolValue(old.Stable),
		AllAtOnce:            types.BoolValue(old.AllAtOnce),
		Region:               types.StringValue(old.Region),
	}

	for _, oldHCL2 := range old.HCL2 {
		vars, diags := types.MapValueFrom(ctx, types.StringType, oldHCL2.Vars)
		resp.Diagnostics.Append(diags...)
		data.HCL2 = append(data.HCL2, hcl2Model{
			AllowFS: types.BoolValue(oldHCL2.AllowFS),
			Vars:    vars,
		})
	}

	constraints := make([]constraintModel, 0, len(old.Constraints))
	for _, constraint := range old.Constraints {
		constraints = append(constraints, constraintModel{
			LTarget: types.StringValue(constraint.LTarget),
			RTarget: types.StringValue(constraint.RTarget),
			Operand: types.StringValue(constraint.Operand),
		})
	}
	data.Constraints = listValueFromV0(ctx, constraintObjectType, constraints, old.Constraints == nil, &resp.Diagnostics)
	data.UpdateStrategy = upgradeUpdateStrategies(ctx, old.UpdateStrategy, &resp.Diagnostics)

	periodicConfigs := make([]periodicConfigModel, 0, len(old.PeriodicConfig))
	for _, periodic := range old.PeriodicConfig {
		periodicConfigs = append(periodicConfigs, periodicConfigModel{
			Enabled:         types.BoolValue(periodic.Enabled),
			Spec:            types.StringValue(periodic.Spec),
			SpecType:        types.StringValue(periodic.SpecType),
			ProhibitOverlap: types.BoolValue(periodic.ProhibitOverlap),
			Timezone:        types.StringValue(periodic.Timezone),
		})
	}
	data.PeriodicConfig = listValueFromV0(ctx, periodicConfigObjectType, periodicConfigs, old.PeriodicConfig == nil, &resp.Diagnostics)

	datacenters, diags := types.SetValueFrom(ctx, types.StringType, old.Datacenters)
	resp.Diagnostics.Append(diags...)
	data.Datacenters = datacenters
	allocationIDs, diags := types.ListValueFrom(ctx, types.StringType, old.AllocationIDs)
	resp.Diagnostics.Append(diags...)
	data.AllocationIDs = allocationIDs
	data.TaskGroups = upgradeTaskGroups(ctx, old.TaskGroups, &resp.Diagnostics)

	// Construct a valid Timeouts value with null create/update durations.
	// V0 (SDKv2) state has no timeouts block; the framework requires the
	// object to have exactly the attribute keys declared in the schema.
	timeoutsAttrTypes := map[string]attr.Type{
		"create": types.StringType,
		"update": types.StringType,
	}
	timeoutsObj := types.ObjectNull(timeoutsAttrTypes)
	data.Timeouts = timeouts.Value{Object: timeoutsObj}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func upgradeUpdateStrategies(ctx context.Context, old []updateStrategyStateV0, diags *diag.Diagnostics) types.List {
	models := make([]updateStrategyModel, 0, len(old))
	for _, update := range old {
		models = append(models, updateStrategyModel{
			Stagger:         types.StringValue(update.Stagger),
			MaxParallel:     types.Int64Value(update.MaxParallel),
			HealthCheck:     types.StringValue(update.HealthCheck),
			MinHealthyTime:  types.StringValue(update.MinHealthyTime),
			HealthyDeadline: types.StringValue(update.HealthyDeadline),
			AutoRevert:      types.BoolValue(update.AutoRevert),
			Canary:          types.Int64Value(update.Canary),
		})
	}
	return listValueFromV0(ctx, updateStrategyObjectType, models, old == nil, diags)
}

func upgradeTaskGroups(ctx context.Context, old []taskGroupStateV0, diags *diag.Diagnostics) types.List {
	groups := make([]taskGroupModel, 0, len(old))
	for _, group := range old {
		meta, valueDiags := types.MapValueFrom(ctx, types.StringType, group.Meta)
		diags.Append(valueDiags...)

		tasks := make([]taskModel, 0, len(group.Tasks))
		for _, oldTask := range group.Tasks {
			taskMeta, taskDiags := types.MapValueFrom(ctx, types.StringType, oldTask.Meta)
			diags.Append(taskDiags...)
			mounts := make([]volumeMountModel, 0, len(oldTask.VolumeMounts))
			for _, mount := range oldTask.VolumeMounts {
				mounts = append(mounts, volumeMountModel{
					Volume:      types.StringValue(mount.Volume),
					Destination: types.StringValue(mount.Destination),
					ReadOnly:    types.BoolValue(mount.ReadOnly),
				})
			}
			volumeMounts := listValueFromV0(ctx, volumeMountObjectType, mounts, oldTask.VolumeMounts == nil, diags)
			tasks = append(tasks, taskModel{
				Name:         types.StringValue(oldTask.Name),
				Driver:       types.StringValue(oldTask.Driver),
				Meta:         taskMeta,
				Resources:    types.ObjectNull(resourcesObjectType.AttrTypes),
				VolumeMounts: volumeMounts,
			})
		}

		volumes := make([]volumeModel, 0, len(group.Volumes))
		for _, volume := range group.Volumes {
			volumes = append(volumes, volumeModel{
				Name:     types.StringValue(volume.Name),
				Type:     types.StringValue(volume.Type),
				ReadOnly: types.BoolValue(volume.ReadOnly),
				Source:   types.StringValue(volume.Source),
			})
		}

		groups = append(groups, taskGroupModel{
			Name:           types.StringValue(group.Name),
			Count:          types.Int64Value(group.Count),
			UpdateStrategy: upgradeUpdateStrategies(ctx, group.UpdateStrategy, diags),
			Tasks:          listValueFromV0(ctx, taskObjectType, tasks, group.Tasks == nil, diags),
			Volumes:        listValueFromV0(ctx, volumeObjectType, volumes, group.Volumes == nil, diags),
			Meta:           meta,
		})
	}
	return listValueFromV0(ctx, taskGroupObjectType, groups, old == nil, diags)
}

func listValueFromV0(ctx context.Context, elementType attr.Type, value any, null bool, diags *diag.Diagnostics) types.List {
	if null {
		return types.ListNull(elementType)
	}
	result, valueDiags := types.ListValueFrom(ctx, elementType, value)
	diags.Append(valueDiags...)
	return result
}
