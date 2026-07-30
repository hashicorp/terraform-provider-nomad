// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"context"
	"strconv"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (r *JobResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan jobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || plan.Jobspec.IsUnknown() {
		return
	}

	var config jobResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	parserConfig, err := parserConfigFromModel(ctx, config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid jobspec parser configuration", err.Error())
		return
	}
	job, err := parseJobspec(config.Jobspec.ValueString(), parserConfig)
	if err != nil {
		resp.Diagnostics.AddError("Invalid jobspec", err.Error())
		return
	}
	namespace := defaultJobNamespace(job)
	region := defaultJobRegion(job)

	var state *jobResourceModel
	if !req.State.Raw.IsNull() {
		var stateData jobResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		state = &stateData
	}

	configChanged := state == nil
	var stateParserConfig jobParserConfig
	if state != nil {
		var configErr error
		stateParserConfig, configErr = parserConfigFromModel(ctx, *state)
		if configErr != nil || !jobspecsEqual(state.Jobspec.ValueString(), stateParserConfig, config.Jobspec.ValueString(), parserConfig) {
			configChanged = true
		} else {
			plan.Jobspec = state.Jobspec
		}

		if state.Namespace.ValueString() != namespace {
			resp.RequiresReplace = append(resp.RequiresReplace, path.Root("namespace"))
		} else if state.Name.ValueString() != pointerValue(job.ID) && config.DeregisterOnIDChange.ValueBool() {
			resp.RequiresReplace = append(resp.RequiresReplace,
				path.Root("name"),
				path.Root("id"),
			)
		}

		if state.Status.ValueString() == "dead" && config.RerunIfDead.ValueBool() {
			configChanged = true
			plan.Status = types.StringUnknown()
		}

		if state.Stop.ValueBool() {
			configChanged = true
			plan.Stop = types.BoolValue(false)
		}
	}

	if state != nil {
		resp.Diagnostics.Append(applyPreservation(ctx, job, state.TaskGroups, config.PreserveCounts.ValueBool(), config.PreserveResources.ValueBool())...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	plan.Name = types.StringValue(pointerValue(job.ID))
	plan.Namespace = types.StringValue(namespace)
	plan.Type = stringPointerValue(job.Type)
	plan.Region = types.StringValue(region)

	datacenters, valueDiags := types.SetValueFrom(ctx, types.StringType, job.Datacenters)
	resp.Diagnostics.Append(valueDiags...)
	plan.Datacenters = datacenters

	periodicConfig, pcDiags := flattenPeriodicConfig(ctx, job.Periodic)
	resp.Diagnostics.Append(pcDiags...)
	plan.PeriodicConfig = periodicConfig

	configTaskGroups, valueDiags := flattenTaskGroups(ctx, job.TaskGroups)
	resp.Diagnostics.Append(valueDiags...)

	configConstraints, cDiags := flattenConstraints(ctx, job.Constraints)
	resp.Diagnostics.Append(cDiags...)
	plan.Constraints = configConstraints

	configUpdateStrategy, uDiags := flattenJobLevelUpdateStrategy(ctx, job.Update)
	resp.Diagnostics.Append(uDiags...)
	plan.UpdateStrategy = configUpdateStrategy

	if resp.Diagnostics.HasError() {
		return
	}

	planResponse, _, planErr := r.providerConfig.Client().Jobs().PlanOpts(job, &api.PlanOptions{
		Diff:           true,
		PolicyOverride: config.PolicyOverride.ValueBool(),
	}, &api.WriteOptions{Namespace: namespace, Region: region})
	if planErr != nil {
		resp.Diagnostics.AddError("Error validating job plan with Nomad", planErr.Error())
		return
	}

	if state != nil && state.Name.ValueString() == pointerValue(job.ID) && state.Namespace.ValueString() == namespace {
		wantModifyIndex, parseErr := strconv.ParseUint(state.ModifyIndex.ValueString(), 10, 64)
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid job modify index", parseErr.Error())
			return
		}
		if planResponse != nil && planResponse.JobModifyIndex != wantModifyIndex {
			resp.Diagnostics.AddError("Job changed during planning", "The job modify index changed since the latest refresh.")
			return
		}
	}

	var jobDiff *api.JobDiff
	if planResponse != nil && planResponse.Diff != nil {
		jobDiff = planResponse.Diff
	}

	serverDrift := false
	if state != nil && jobDiff != nil && jobDiff.Type != "None" {
		tflog.Debug(ctx, "Nomad server detected job drift", map[string]any{
			"diff_type": jobDiff.Type,
			"job_id":    pointerValue(job.ID),
		})
		serverDrift = true
	}

	stateTaskGroups := types.ListNull(taskGroupObjectType)
	if state != nil {
		stateTaskGroups = state.TaskGroups
	}

	var taskGroupDiags diag.Diagnostics
	plan.TaskGroups, taskGroupDiags = mergeTaskGroupsWithDiff(ctx, stateTaskGroups, configTaskGroups, jobDiff)
	resp.Diagnostics.Append(taskGroupDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if configChanged || serverDrift {
		// The job will be re-registered: all server-assigned fields will get new
		// values that are not known until after apply.
		plan.ModifyIndex = types.StringUnknown()
		plan.AllocationIDs = types.ListUnknown(types.StringType)
		plan.DeploymentID = types.StringUnknown()
		plan.DeploymentStatus = types.StringUnknown()
		plan.Version = types.Int64Unknown()
		plan.SubmitTime = types.StringUnknown()
		plan.Status = types.StringUnknown()
		plan.StatusDescription = types.StringUnknown()
		plan.Stable = types.BoolUnknown()
		plan.Priority = types.Int64Unknown()
		plan.ParentID = types.StringUnknown()
		plan.AllAtOnce = types.BoolUnknown()
		plan.CreateIndex = types.Int64Unknown()
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func mergeTaskGroupsWithDiff(
	ctx context.Context,
	stateGroups, configGroups types.List,
	diff *api.JobDiff,
) (types.List, diag.Diagnostics) {
	tgDiffByName := make(map[string]*api.TaskGroupDiff)
	if diff != nil {
		for _, tgd := range diff.TaskGroups {
			if tgd != nil {
				tgDiffByName[tgd.Name] = tgd
			}
		}
	}

	var stateTGs []taskGroupModel
	if !stateGroups.IsNull() && !stateGroups.IsUnknown() {
		if diags := stateGroups.ElementsAs(ctx, &stateTGs, false); diags.HasError() {
			return types.ListNull(taskGroupObjectType), diags
		}
	}
	stateTGByName := make(map[string]taskGroupModel, len(stateTGs))
	for _, tg := range stateTGs {
		stateTGByName[tg.Name.ValueString()] = tg
	}

	var configTGs []taskGroupModel
	if diags := configGroups.ElementsAs(ctx, &configTGs, false); diags.HasError() {
		return types.ListNull(taskGroupObjectType), diags
	}

	for i, configTG := range configTGs {
		tgd := tgDiffByName[configTG.Name.ValueString()]

		var configTasks []taskModel
		if diags := configTG.Tasks.ElementsAs(ctx, &configTasks, false); diags.HasError() {
			return types.ListNull(taskGroupObjectType), diags
		}

		planned := append([]taskModel(nil), configTasks...)

		configNames := make(map[string]struct{}, len(configTasks))
		for _, t := range configTasks {
			configNames[t.Name.ValueString()] = struct{}{}
		}

		var stateByName map[string]taskModel
		if tgd == nil {
			continue
		}
		for _, td := range tgd.Tasks {
			if td == nil {
				continue
			}
			if _, inConfig := configNames[td.Name]; inConfig {
				continue
			}
			switch td.Type {
			case "Added", "Edited":
				// Server-injected task: name is known, all other fields unknown.
				planned = append(planned, taskModel{
					Name:         types.StringValue(td.Name),
					Driver:       types.StringUnknown(),
					Meta:         types.MapUnknown(types.StringType),
					Resources:    types.ObjectUnknown(resourcesObjectType.AttrTypes),
					VolumeMounts: types.ListUnknown(volumeMountObjectType),
				})
			case "None":
				// Unchanged server-injected task: carry from state.
				if stateByName == nil {
					stateByName = make(map[string]taskModel)
					if stateTG, ok := stateTGByName[configTG.Name.ValueString()]; ok &&
						!stateTG.Tasks.IsNull() && !stateTG.Tasks.IsUnknown() {
						var stateTasks []taskModel
						if diags := stateTG.Tasks.ElementsAs(ctx, &stateTasks, false); diags.HasError() {
							return types.ListNull(taskGroupObjectType), diags
						}
						for _, t := range stateTasks {
							stateByName[t.Name.ValueString()] = t
						}
					}
				}
				if st, ok := stateByName[td.Name]; ok {
					planned = append(planned, st)
				}
				// "Deleted": server-injected task being removed — skip.
			}
		}

		taskList, diags := types.ListValueFrom(ctx, taskObjectType, planned)
		if diags.HasError() {
			return types.ListNull(taskGroupObjectType), diags
		}
		configTGs[i].Tasks = taskList
	}

	return types.ListValueFrom(ctx, taskGroupObjectType, configTGs)
}

func pointerValue[T any](value *T) T {
	if value == nil {
		var zero T
		return zero
	}
	return *value
}
