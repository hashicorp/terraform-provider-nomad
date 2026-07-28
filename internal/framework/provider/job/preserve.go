// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"context"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func applyPreservation(ctx context.Context, configJob *api.Job, stateTaskGroups types.List, preserveCounts, preserveResources bool) diag.Diagnostics {
	if configJob == nil || (!preserveCounts && !preserveResources) {
		return nil
	}
	if stateTaskGroups.IsNull() || stateTaskGroups.IsUnknown() {
		return nil
	}

	var liveGroups []taskGroupModel
	diags := stateTaskGroups.ElementsAs(ctx, &liveGroups, false)
	if diags.HasError() {
		return diags
	}

	preserveCounts = preserveCounts && jobTypeSupportsCounts(pointerValue(configJob.Type))

	liveGroupMap := make(map[string]taskGroupModel, len(liveGroups))
	for _, tg := range liveGroups {
		liveGroupMap[tg.Name.ValueString()] = tg
	}

	for _, configTaskGroup := range configJob.TaskGroups {
		if configTaskGroup == nil || configTaskGroup.Name == nil {
			continue
		}
		liveGroup, ok := liveGroupMap[*configTaskGroup.Name]
		if !ok {
			continue
		}

		if preserveCounts {
			count := int(liveGroup.Count.ValueInt64())
			configTaskGroup.Count = &count
		}

		if !preserveResources {
			continue
		}

		var liveTasks []taskModel
		if diags = liveGroup.Tasks.ElementsAs(ctx, &liveTasks, false); diags.HasError() {
			return diags
		}
		liveTaskMap := make(map[string]taskModel, len(liveTasks))
		for _, t := range liveTasks {
			liveTaskMap[t.Name.ValueString()] = t
		}

		for _, configTask := range configTaskGroup.Tasks {
			if configTask == nil {
				continue
			}
			liveTask, ok := liveTaskMap[configTask.Name]
			if !ok {
				continue
			}
			configTask.Resources = expandResources(ctx, liveTask.Resources)
		}
	}

	return nil
}

func expandResources(ctx context.Context, obj types.Object) *api.Resources {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}
	var m resourcesModel
	if diags := obj.As(ctx, &m, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil
	}
	r := &api.Resources{}
	if !m.CPU.IsNull() && !m.CPU.IsUnknown() {
		v := int(m.CPU.ValueInt64())
		r.CPU = &v
	}
	if !m.Cores.IsNull() && !m.Cores.IsUnknown() {
		v := int(m.Cores.ValueInt64())
		r.Cores = &v
	}
	if !m.MemoryMB.IsNull() && !m.MemoryMB.IsUnknown() {
		v := int(m.MemoryMB.ValueInt64())
		r.MemoryMB = &v
	}
	if !m.MemoryMaxMB.IsNull() && !m.MemoryMaxMB.IsUnknown() {
		v := int(m.MemoryMaxMB.ValueInt64())
		r.MemoryMaxMB = &v
	}
	if !m.DiskMB.IsNull() && !m.DiskMB.IsUnknown() {
		v := int(m.DiskMB.ValueInt64())
		r.DiskMB = &v
	}
	if !m.SecretsMB.IsNull() && !m.SecretsMB.IsUnknown() {
		v := int(m.SecretsMB.ValueInt64())
		r.SecretsMB = &v
	}
	if !m.IOPS.IsNull() && !m.IOPS.IsUnknown() {
		v := int(m.IOPS.ValueInt64())
		r.IOPS = &v
	}
	return r
}

func jobTypeSupportsCounts(jobType string) bool {
	return jobType == api.JobTypeService || jobType == api.JobTypeBatch
}
