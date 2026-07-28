// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
)

var (
	_ resource.Resource                 = (*JobResource)(nil)
	_ resource.ResourceWithConfigure    = (*JobResource)(nil)
	_ resource.ResourceWithImportState  = (*JobResource)(nil)
	_ resource.ResourceWithModifyPlan   = (*JobResource)(nil)
	_ resource.ResourceWithUpgradeState = (*JobResource)(nil)
)

type JobResource struct {
	providerConfig nomad.ProviderConfig
}

func NewJobResource() resource.Resource {
	return &JobResource{}
}

func (r *JobResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job"
}

func (r *JobResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = jobSchema(ctx)
}

func (r *JobResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	metaFunc, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected func() any, got %T.", req.ProviderData))
		return
	}
	providerConfig, ok := metaFunc().(nomad.ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Provider Meta Type", fmt.Sprintf("Expected nomad.ProviderConfig, got %T.", metaFunc()))
		return
	}
	r.providerConfig = providerConfig
}

func (r *JobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data jobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	registerResponse := r.register(ctx, &data, 0, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.monitor(ctx, &data, registerResponse, true, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readIntoModel(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *JobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data jobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state jobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	wantModifyIndex, err := strconv.ParseUint(state.ModifyIndex.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid job modify index", err.Error())
		return
	}

	registerResponse := r.register(ctx, &data, wantModifyIndex, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.monitor(ctx, &data, registerResponse, false, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readIntoModel(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *JobResource) register(ctx context.Context, data *jobResourceModel, modifyIndex uint64, diags *diag.Diagnostics) *api.JobRegisterResponse {
	parserConfig, err := parserConfigFromModel(ctx, *data)
	if err != nil {
		diags.AddError("Invalid jobspec parser configuration", err.Error())
		return nil
	}

	job, err := parseJobspec(data.Jobspec.ValueString(), parserConfig)
	if err != nil {
		diags.AddError("Invalid jobspec", err.Error())
		return nil
	}
	namespace := defaultJobNamespace(job)
	region := defaultJobRegion(job)

	submission := &api.JobSubmission{
		Source:        data.Jobspec.ValueString(),
		Format:        "hcl2",
		VariableFlags: parserConfig.HCL2.Vars,
	}
	if parserConfig.JSON {
		submission.Format = "json"
	}

	tflog.Debug(ctx, "Registering Nomad job", map[string]any{"job_id": *job.ID, "namespace": namespace, "region": region})
	registerResponse, _, err := r.providerConfig.Client().Jobs().RegisterOpts(job, &api.RegisterOptions{
		PolicyOverride:    data.PolicyOverride.ValueBool(),
		PreserveCounts:    data.PreserveCounts.ValueBool(),
		PreserveResources: data.PreserveResources.ValueBool(),
		ModifyIndex:       modifyIndex,
		Submission:        submission,
	}, &api.WriteOptions{Namespace: namespace, Region: region})
	if err != nil {
		diags.AddError("Error applying jobspec", err.Error())
		return nil
	}

	data.Name = types.StringValue(*job.ID)
	data.ID = data.Name
	data.Namespace = types.StringValue(namespace)
	data.Region = types.StringValue(region)
	data.ModifyIndex = types.StringValue(strconv.FormatUint(registerResponse.JobModifyIndex, 10))
	return registerResponse
}

func (r *JobResource) monitor(ctx context.Context, data *jobResourceModel, registerResponse *api.JobRegisterResponse, create bool, diags *diag.Diagnostics) {
	if data.Detach.ValueBool() || registerResponse == nil || registerResponse.EvalID == "" {
		return
	}

	timeout := 5 * time.Minute
	var timeoutDiags diag.Diagnostics
	if create {
		timeout, timeoutDiags = data.Timeouts.Create(ctx, timeout)
	} else {
		timeout, timeoutDiags = data.Timeouts.Update(ctx, timeout)
	}
	diags.Append(timeoutDiags...)
	if diags.HasError() {
		return
	}

	deployment, err := monitorDeployment(ctx, r.providerConfig.Client(), timeout, data.Namespace.ValueString(), registerResponse.EvalID)
	if err != nil {
		diags.AddError("Error waiting for job to schedule or deploy", err.Error())
		return
	}
	if deployment == nil {
		data.DeploymentID = types.StringNull()
		data.DeploymentStatus = types.StringNull()
		return
	}
	data.DeploymentID = types.StringValue(deployment.ID)
	data.DeploymentStatus = types.StringValue(deployment.Status)
}

func (r *JobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data jobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found := r.readIntoModel(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *JobResource) readIntoModel(ctx context.Context, data *jobResourceModel, diags *diag.Diagnostics) bool {
	namespace := data.Namespace.ValueString()
	if namespace == "" {
		namespace = "default"
	}
	region := data.Region.ValueString()
	if region == "" {
		region = "global"
	}
	id := data.Name.ValueString()
	queryOptions := &api.QueryOptions{Namespace: namespace, Region: region}
	client := r.providerConfig.Client()

	job, _, err := client.Jobs().Info(id, queryOptions)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false
		}
		diags.AddError("Error checking for job", err.Error())
		return false
	}

	diags.Append(flattenJob(ctx, job, data)...)
	if diags.HasError() {
		return false
	}

	if data.ReadAllocationIDs.ValueBool() {
		allocationStubs, _, allocationErr := client.Jobs().Allocations(id, false, queryOptions)
		if allocationErr != nil {
			tflog.Warn(ctx, "Failed to list allocations for job", map[string]any{"job_id": id, "error": allocationErr.Error()})
		}
		allocationIDs := make([]string, 0, len(allocationStubs))
		for _, allocation := range allocationStubs {
			allocationIDs = append(allocationIDs, allocation.ID)
		}
		var valueDiags diag.Diagnostics
		data.AllocationIDs, valueDiags = types.ListValueFrom(ctx, types.StringType, allocationIDs)
		diags.Append(valueDiags...)
	} else {
		data.AllocationIDs = types.ListNull(types.StringType)
	}

	if job.ID != nil && job.Version != nil {
		submission, _, submissionErr := client.Jobs().Submission(*job.ID, int(*job.Version), queryOptions)
		if submissionErr != nil {
			tflog.Warn(ctx, "Failed to read job submission", map[string]any{"job_id": id, "error": submissionErr.Error()})
		} else {
			diags.Append(updateSubmissionState(ctx, submission, data)...)
		}
	}

	return !diags.HasError()
}

func updateSubmissionState(ctx context.Context, submission *api.JobSubmission, data *jobResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if submission == nil {
		return diags
	}
	if submission.Source != "" {
		data.Jobspec = types.StringValue(submission.Source)
	}
	if submission.Format != "hcl2" {
		return diags
	}

	currentVars := map[string]string(nil)
	if len(data.HCL2) > 0 && !data.HCL2[0].Vars.IsNull() && !data.HCL2[0].Vars.IsUnknown() {
		diags.Append(data.HCL2[0].Vars.ElementsAs(ctx, &currentVars, false)...)
		if diags.HasError() {
			return diags
		}
	}
	if stringMapsEqual(currentVars, submission.VariableFlags) {
		return diags
	}

	vars, valueDiags := types.MapValueFrom(ctx, types.StringType, submission.VariableFlags)
	diags.Append(valueDiags...)
	if len(data.HCL2) == 0 {
		data.HCL2 = []hcl2Model{{AllowFS: types.BoolValue(false), Vars: vars}}
	} else {
		data.HCL2[0].Vars = vars
	}
	return diags
}

func stringMapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func (r *JobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data jobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || !data.DeregisterOnDestroy.ValueBool() {
		return
	}

	namespace := data.Namespace.ValueString()
	if namespace == "" {
		namespace = "default"
	}
	region := data.Region.ValueString()
	if region == "" {
		region = "global"
	}

	// purge_on_destroy: prefer the value carried through private state (written
	// by ModifyPlan during the destroy plan), which reflects the user's current
	// config. Fall back to the state value for resources that haven't gone
	// through a destroy plan with this version of the provider.
	purge := data.PurgeOnDestroy.ValueBool()
	if purgeBytes, diags := req.Private.GetKey(ctx, "purge_on_destroy"); !diags.HasError() && len(purgeBytes) > 0 {
		purge = purgeBytes[0] == 1
	}

	_, _, err := r.providerConfig.Client().Jobs().Deregister(
		data.Name.ValueString(),
		purge,
		&api.WriteOptions{Namespace: namespace, Region: region},
	)
	if err != nil {
		resp.Diagnostics.AddError("Error deregistering job", err.Error())
	}
}

func (r *JobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, namespace, importDiags := parseImportID(req.ID)
	resp.Diagnostics.Append(importDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data := jobResourceModel{
		Name:      types.StringValue(id),
		ID:        types.StringValue(id),
		Namespace: types.StringValue(namespace),
	}
	found := r.readIntoModel(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Job not found", fmt.Sprintf("No Nomad job %q exists in namespace %q.", id, namespace))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func parseImportID(importID string) (string, string, diag.Diagnostics) {
	var diags diag.Diagnostics
	separator := strings.LastIndex(importID, "@")
	if separator == -1 {
		diags.AddError("Invalid import ID", "Import ID should follow the pattern <id>@<namespace>")
		return "", "", diags
	}
	id := importID[:separator]
	namespace := importID[separator+1:]
	if id == "" {
		diags.AddError("Invalid import ID", "Missing resource ID in import")
	}
	if namespace == "" {
		diags.AddError("Invalid import ID", "Missing namespace in import")
	}
	return id, namespace, diags
}
