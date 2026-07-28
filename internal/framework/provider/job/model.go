// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type jobResourceModel struct {
	ID                   types.String   `tfsdk:"id"`
	Jobspec              types.String   `tfsdk:"jobspec"`
	PolicyOverride       types.Bool     `tfsdk:"policy_override"`
	PreserveCounts       types.Bool     `tfsdk:"preserve_counts"`
	PreserveResources    types.Bool     `tfsdk:"preserve_resources"`
	DeregisterOnDestroy  types.Bool     `tfsdk:"deregister_on_destroy"`
	DeregisterOnIDChange types.Bool     `tfsdk:"deregister_on_id_change"`
	Detach               types.Bool     `tfsdk:"detach"`
	HCL2                 []hcl2Model    `tfsdk:"hcl2"`
	JSON                 types.Bool     `tfsdk:"json"`
	RerunIfDead          types.Bool     `tfsdk:"rerun_if_dead"`
	ReadAllocationIDs    types.Bool     `tfsdk:"read_allocation_ids"`
	PurgeOnDestroy       types.Bool     `tfsdk:"purge_on_destroy"`
	Timeouts             timeouts.Value `tfsdk:"timeouts"`

	DeploymentID      types.String `tfsdk:"deployment_id"`
	DeploymentStatus  types.String `tfsdk:"deployment_status"`
	ModifyIndex       types.String `tfsdk:"modify_index"`
	Name              types.String `tfsdk:"name"`
	Namespace         types.String `tfsdk:"namespace"`
	Type              types.String `tfsdk:"type"`
	Status            types.String `tfsdk:"status"`
	StatusDescription types.String `tfsdk:"status_description"`
	Version           types.Int64  `tfsdk:"version"`
	SubmitTime        types.String `tfsdk:"submit_time"`
	CreateIndex       types.Int64  `tfsdk:"create_index"`
	Stop              types.Bool   `tfsdk:"stop"`
	Priority          types.Int64  `tfsdk:"priority"`
	ParentID          types.String `tfsdk:"parent_id"`
	Stable            types.Bool   `tfsdk:"stable"`
	AllAtOnce         types.Bool   `tfsdk:"all_at_once"`
	Constraints       types.List   `tfsdk:"constraints"`
	UpdateStrategy    types.List   `tfsdk:"update_strategy"`
	PeriodicConfig    types.List   `tfsdk:"periodic_config"`
	Region            types.String `tfsdk:"region"`
	Datacenters       types.Set    `tfsdk:"datacenters"`
	AllocationIDs     types.List   `tfsdk:"allocation_ids"`
	TaskGroups        types.List   `tfsdk:"task_groups"`
}

type hcl2Model struct {
	AllowFS types.Bool `tfsdk:"allow_fs"`
	Vars    types.Map  `tfsdk:"vars"`
}
