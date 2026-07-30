// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"context"
	"sort"
	"strconv"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type constraintModel struct {
	LTarget types.String `tfsdk:"ltarget"`
	RTarget types.String `tfsdk:"rtarget"`
	Operand types.String `tfsdk:"operand"`
}

type updateStrategyModel struct {
	Stagger         types.String `tfsdk:"stagger"`
	MaxParallel     types.Int64  `tfsdk:"max_parallel"`
	HealthCheck     types.String `tfsdk:"health_check"`
	MinHealthyTime  types.String `tfsdk:"min_healthy_time"`
	HealthyDeadline types.String `tfsdk:"healthy_deadline"`
	AutoRevert      types.Bool   `tfsdk:"auto_revert"`
	Canary          types.Int64  `tfsdk:"canary"`
}

type periodicConfigModel struct {
	Enabled         types.Bool   `tfsdk:"enabled"`
	Spec            types.String `tfsdk:"spec"`
	SpecType        types.String `tfsdk:"spec_type"`
	ProhibitOverlap types.Bool   `tfsdk:"prohibit_overlap"`
	Timezone        types.String `tfsdk:"timezone"`
}

type volumeMountModel struct {
	Volume      types.String `tfsdk:"volume"`
	Destination types.String `tfsdk:"destination"`
	ReadOnly    types.Bool   `tfsdk:"read_only"`
}

type taskModel struct {
	Name         types.String `tfsdk:"name"`
	Driver       types.String `tfsdk:"driver"`
	Meta         types.Map    `tfsdk:"meta"`
	Resources    types.Object `tfsdk:"resources"`
	VolumeMounts types.List   `tfsdk:"volume_mounts"`
}

type resourcesModel struct {
	CPU         types.Int64  `tfsdk:"cpu"`
	Cores       types.Int64  `tfsdk:"cores"`
	MemoryMB    types.Int64  `tfsdk:"memory_mb"`
	MemoryMaxMB types.Int64  `tfsdk:"memory_max_mb"`
	DiskMB      types.Int64  `tfsdk:"disk_mb"`
	SecretsMB   types.Int64  `tfsdk:"secrets_mb"`
	IOPS        types.Int64  `tfsdk:"iops"`
	NUMA        types.Object `tfsdk:"numa"`
	Networks    types.List   `tfsdk:"networks"`
	Devices     types.List   `tfsdk:"devices"`
}

type numaModel struct {
	Affinity types.String `tfsdk:"affinity"`
	Devices  types.List   `tfsdk:"devices"`
}

type networkModel struct {
	Mode          types.String `tfsdk:"mode"`
	Device        types.String `tfsdk:"device"`
	CIDR          types.String `tfsdk:"cidr"`
	IP            types.String `tfsdk:"ip"`
	Hostname      types.String `tfsdk:"hostname"`
	MBits         types.Int64  `tfsdk:"mbits"`
	DNS           types.Object `tfsdk:"dns"`
	ReservedPorts types.List   `tfsdk:"reserved_ports"`
	DynamicPorts  types.List   `tfsdk:"dynamic_ports"`
	CNI           types.Object `tfsdk:"cni"`
}

type dnsModel struct {
	Servers  types.List `tfsdk:"servers"`
	Searches types.List `tfsdk:"searches"`
	Options  types.List `tfsdk:"options"`
}

type portModel struct {
	Label           types.String `tfsdk:"label"`
	Value           types.Int64  `tfsdk:"value"`
	To              types.Int64  `tfsdk:"to"`
	HostNetwork     types.String `tfsdk:"host_network"`
	IgnoreCollision types.Bool   `tfsdk:"ignore_collision"`
}

type cniModel struct {
	Args types.Map `tfsdk:"args"`
}

type requestedDeviceModel struct {
	Name        types.String `tfsdk:"name"`
	Count       types.Int64  `tfsdk:"count"`
	Constraints types.List   `tfsdk:"constraints"`
	Affinities  types.List   `tfsdk:"affinities"`
}

type affinityModel struct {
	LTarget types.String `tfsdk:"ltarget"`
	RTarget types.String `tfsdk:"rtarget"`
	Operand types.String `tfsdk:"operand"`
	Weight  types.Int64  `tfsdk:"weight"`
}

type volumeModel struct {
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	ReadOnly types.Bool   `tfsdk:"read_only"`
	Source   types.String `tfsdk:"source"`
}

type taskGroupModel struct {
	Name           types.String `tfsdk:"name"`
	Count          types.Int64  `tfsdk:"count"`
	UpdateStrategy types.List   `tfsdk:"update_strategy"`
	Tasks          types.List   `tfsdk:"task"`
	Volumes        types.List   `tfsdk:"volumes"`
	Meta           types.Map    `tfsdk:"meta"`
}

var (
	constraintObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"ltarget": types.StringType,
		"rtarget": types.StringType,
		"operand": types.StringType,
	}}
	updateStrategyObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"stagger":          types.StringType,
		"max_parallel":     types.Int64Type,
		"health_check":     types.StringType,
		"min_healthy_time": types.StringType,
		"healthy_deadline": types.StringType,
		"auto_revert":      types.BoolType,
		"canary":           types.Int64Type,
	}}
	periodicConfigObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"enabled":          types.BoolType,
		"spec":             types.StringType,
		"spec_type":        types.StringType,
		"prohibit_overlap": types.BoolType,
		"timezone":         types.StringType,
	}}
	volumeMountObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"volume":      types.StringType,
		"destination": types.StringType,
		"read_only":   types.BoolType,
	}}
	numaObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"affinity": types.StringType,
		"devices":  types.ListType{ElemType: types.StringType},
	}}
	dnsObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"servers":  types.ListType{ElemType: types.StringType},
		"searches": types.ListType{ElemType: types.StringType},
		"options":  types.ListType{ElemType: types.StringType},
	}}
	portObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"label":            types.StringType,
		"value":            types.Int64Type,
		"to":               types.Int64Type,
		"host_network":     types.StringType,
		"ignore_collision": types.BoolType,
	}}
	cniObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"args": types.MapType{ElemType: types.StringType},
	}}
	affinityObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"ltarget": types.StringType,
		"rtarget": types.StringType,
		"operand": types.StringType,
		"weight":  types.Int64Type,
	}}
	requestedDeviceObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":        types.StringType,
		"count":       types.Int64Type,
		"constraints": types.ListType{ElemType: constraintObjectType},
		"affinities":  types.ListType{ElemType: affinityObjectType},
	}}
	networkObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"mode":           types.StringType,
		"device":         types.StringType,
		"cidr":           types.StringType,
		"ip":             types.StringType,
		"hostname":       types.StringType,
		"mbits":          types.Int64Type,
		"dns":            dnsObjectType,
		"reserved_ports": types.ListType{ElemType: portObjectType},
		"dynamic_ports":  types.ListType{ElemType: portObjectType},
		"cni":            cniObjectType,
	}}
	resourcesObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"cpu":           types.Int64Type,
		"cores":         types.Int64Type,
		"memory_mb":     types.Int64Type,
		"memory_max_mb": types.Int64Type,
		"disk_mb":       types.Int64Type,
		"secrets_mb":    types.Int64Type,
		"iops":          types.Int64Type,
		"numa":          numaObjectType,
		"networks":      types.ListType{ElemType: networkObjectType},
		"devices":       types.ListType{ElemType: requestedDeviceObjectType},
	}}
	taskObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":          types.StringType,
		"driver":        types.StringType,
		"meta":          types.MapType{ElemType: types.StringType},
		"resources":     resourcesObjectType,
		"volume_mounts": types.ListType{ElemType: volumeMountObjectType},
	}}
	volumeObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":      types.StringType,
		"type":      types.StringType,
		"read_only": types.BoolType,
		"source":    types.StringType,
	}}
	taskGroupObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"name":            types.StringType,
		"count":           types.Int64Type,
		"update_strategy": types.ListType{ElemType: updateStrategyObjectType},
		"task":            types.ListType{ElemType: taskObjectType},
		"volumes":         types.ListType{ElemType: volumeObjectType},
		"meta":            types.MapType{ElemType: types.StringType},
	}}
)

func flattenJob(ctx context.Context, job *api.Job, data *jobResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.Name = stringPointerValue(job.ID)
	data.ID = data.Name
	data.Type = stringPointerValue(job.Type)
	data.Region = stringPointerValue(job.Region)
	data.Namespace = stringPointerValue(job.Namespace)
	data.Status = stringPointerValue(job.Status)
	data.StatusDescription = stringPointerValue(job.StatusDescription)
	data.Version = uint64PointerValue(job.Version)
	data.SubmitTime = int64PointerStringValue(job.SubmitTime)
	data.CreateIndex = uint64PointerValue(job.CreateIndex)
	data.Stop = boolPointerValue(job.Stop)
	data.Priority = intPointerValue(job.Priority)
	data.ParentID = stringPointerValue(job.ParentID)
	data.Stable = boolPointerValue(job.Stable)
	data.AllAtOnce = boolPointerValue(job.AllAtOnce)
	if job.JobModifyIndex == nil {
		data.ModifyIndex = types.StringValue("0")
	} else {
		data.ModifyIndex = types.StringValue(strconv.FormatUint(*job.JobModifyIndex, 10))
	}

	datacenters, valueDiags := types.SetValueFrom(ctx, types.StringType, job.Datacenters)
	diags.Append(valueDiags...)
	data.Datacenters = datacenters

	data.Constraints, valueDiags = flattenConstraints(ctx, job.Constraints)
	diags.Append(valueDiags...)

	data.UpdateStrategy, valueDiags = flattenUpdateStrategy(ctx, job.Update)
	diags.Append(valueDiags...)
	data.PeriodicConfig, valueDiags = flattenPeriodicConfig(ctx, job.Periodic)
	diags.Append(valueDiags...)
	data.TaskGroups, valueDiags = flattenTaskGroups(ctx, job.TaskGroups)
	diags.Append(valueDiags...)

	return diags
}

func flattenConstraints(ctx context.Context, constraints []*api.Constraint) (types.List, diag.Diagnostics) {
	models := make([]constraintModel, 0, len(constraints))
	for _, constraint := range constraints {
		if constraint == nil {
			continue
		}
		models = append(models, constraintModel{
			LTarget: types.StringValue(constraint.LTarget),
			RTarget: types.StringValue(constraint.RTarget),
			Operand: types.StringValue(constraint.Operand),
		})
	}
	return types.ListValueFrom(ctx, constraintObjectType, models)
}

func flattenUpdateStrategy(ctx context.Context, update *api.UpdateStrategy) (types.List, diag.Diagnostics) {
	if update == nil {
		return types.ListNull(updateStrategyObjectType), nil
	}

	model := updateStrategyModel{
		MaxParallel: intPointerValue(update.MaxParallel),
		HealthCheck: stringPointerValue(update.HealthCheck),
		AutoRevert:  boolPointerValue(update.AutoRevert),
		Canary:      intPointerValue(update.Canary),
	}
	if update.Stagger == nil {
		model.Stagger = types.StringNull()
	} else {
		model.Stagger = types.StringValue(update.Stagger.String())
	}
	if update.MinHealthyTime == nil {
		model.MinHealthyTime = types.StringNull()
	} else {
		model.MinHealthyTime = types.StringValue(update.MinHealthyTime.String())
	}
	if update.HealthyDeadline == nil {
		model.HealthyDeadline = types.StringNull()
	} else {
		model.HealthyDeadline = types.StringValue(update.HealthyDeadline.String())
	}

	return types.ListValueFrom(ctx, updateStrategyObjectType, []updateStrategyModel{model})
}

func flattenJobLevelUpdateStrategy(ctx context.Context, update *api.UpdateStrategy) (types.List, diag.Diagnostics) {
	if update == nil || update.MaxParallel == nil {
		model := updateStrategyModel{
			MaxParallel:     types.Int64Value(0),
			Stagger:         types.StringValue("0s"),
			HealthCheck:     types.StringValue(""),
			MinHealthyTime:  types.StringValue("0s"),
			HealthyDeadline: types.StringValue("0s"),
			AutoRevert:      types.BoolValue(false),
			Canary:          types.Int64Value(0),
		}
		return types.ListValueFrom(ctx, updateStrategyObjectType, []updateStrategyModel{model})
	}
	model := updateStrategyModel{
		MaxParallel:     intPointerValue(update.MaxParallel),
		HealthCheck:     types.StringValue(""),
		MinHealthyTime:  types.StringValue("0s"),
		HealthyDeadline: types.StringValue("0s"),
		AutoRevert:      types.BoolValue(false),
		Canary:          types.Int64Value(0),
	}
	if update.Stagger == nil {
		model.Stagger = types.StringNull()
	} else {
		if *update.MaxParallel == 0 {
			model.Stagger = types.StringValue("0s")
		} else {
			model.Stagger = types.StringValue(update.Stagger.String())
		}
	}
	return types.ListValueFrom(ctx, updateStrategyObjectType, []updateStrategyModel{model})
}

func flattenPeriodicConfig(ctx context.Context, periodic *api.PeriodicConfig) (types.List, diag.Diagnostics) {
	if periodic == nil {
		return types.ListNull(periodicConfigObjectType), nil
	}

	return types.ListValueFrom(ctx, periodicConfigObjectType, []periodicConfigModel{{
		Enabled:         boolPointerValue(periodic.Enabled),
		Spec:            stringPointerValue(periodic.Spec),
		SpecType:        stringPointerValue(periodic.SpecType),
		ProhibitOverlap: boolPointerValue(periodic.ProhibitOverlap),
		Timezone:        stringPointerValue(periodic.TimeZone),
	}})
}

func flattenTaskGroups(ctx context.Context, taskGroups []*api.TaskGroup) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	models := make([]taskGroupModel, 0, len(taskGroups))
	for _, taskGroup := range taskGroups {
		if taskGroup == nil {
			continue
		}

		updateStrategy, valueDiags := flattenUpdateStrategy(ctx, taskGroup.Update)
		diags.Append(valueDiags...)
		meta, valueDiags := stringMapValue(ctx, taskGroup.Meta)
		diags.Append(valueDiags...)

		tasks := make([]taskModel, 0, len(taskGroup.Tasks))
		for _, task := range taskGroup.Tasks {
			if task == nil {
				continue
			}
			taskMeta, taskDiags := stringMapValue(ctx, task.Meta)
			diags.Append(taskDiags...)
			resources, resourceDiags := flattenResources(ctx, task.Resources)
			diags.Append(resourceDiags...)

			mounts := make([]volumeMountModel, 0, len(task.VolumeMounts))
			for _, mount := range task.VolumeMounts {
				if mount == nil {
					continue
				}
				mounts = append(mounts, volumeMountModel{
					Volume:      stringPointerValue(mount.Volume),
					Destination: stringPointerValue(mount.Destination),
					ReadOnly:    boolPointerValue(mount.ReadOnly),
				})
			}
			volumeMounts, taskDiags := types.ListValueFrom(ctx, volumeMountObjectType, mounts)
			diags.Append(taskDiags...)
			tasks = append(tasks, taskModel{
				Name:         types.StringValue(task.Name),
				Driver:       types.StringValue(task.Driver),
				Meta:         taskMeta,
				Resources:    resources,
				VolumeMounts: volumeMounts,
			})
		}
		taskList, valueDiags := types.ListValueFrom(ctx, taskObjectType, tasks)
		diags.Append(valueDiags...)

		volumes := make([]volumeModel, 0, len(taskGroup.Volumes))
		for _, volume := range taskGroup.Volumes {
			if volume == nil {
				continue
			}
			volumes = append(volumes, volumeModel{
				Name:     types.StringValue(volume.Name),
				Type:     types.StringValue(volume.Type),
				ReadOnly: types.BoolValue(volume.ReadOnly),
				Source:   types.StringValue(volume.Source),
			})
		}
		sort.Slice(volumes, func(i, j int) bool {
			return volumes[i].Name.ValueString() < volumes[j].Name.ValueString()
		})
		volumeList, valueDiags := types.ListValueFrom(ctx, volumeObjectType, volumes)
		diags.Append(valueDiags...)

		count := int64(1)
		if taskGroup.Count != nil {
			count = int64(*taskGroup.Count)
		}
		models = append(models, taskGroupModel{
			Name:           stringPointerValue(taskGroup.Name),
			Count:          types.Int64Value(count),
			UpdateStrategy: updateStrategy,
			Tasks:          taskList,
			Volumes:        volumeList,
			Meta:           meta,
		})
	}

	value, valueDiags := types.ListValueFrom(ctx, taskGroupObjectType, models)
	diags.Append(valueDiags...)
	return value, diags
}

func flattenResources(ctx context.Context, resources *api.Resources) (types.Object, diag.Diagnostics) {
	if resources == nil {
		return types.ObjectNull(resourcesObjectType.AttrTypes), nil
	}

	var diags diag.Diagnostics
	model := resourcesModel{
		CPU:         intPointerValue(resources.CPU),
		Cores:       intPointerValue(resources.Cores),
		MemoryMB:    intPointerValue(resources.MemoryMB),
		MemoryMaxMB: intPointerValue(resources.MemoryMaxMB),
		DiskMB:      intPointerValue(resources.DiskMB),
		SecretsMB:   intPointerValue(resources.SecretsMB),
		IOPS:        intPointerValue(resources.IOPS),
		NUMA:        types.ObjectNull(numaObjectType.AttrTypes),
	}

	if resources.NUMA != nil {
		devices, valueDiags := types.ListValueFrom(ctx, types.StringType, resources.NUMA.Devices)
		diags.Append(valueDiags...)
		model.NUMA, valueDiags = types.ObjectValueFrom(ctx, numaObjectType.AttrTypes, numaModel{
			Affinity: types.StringValue(resources.NUMA.Affinity),
			Devices:  devices,
		})
		diags.Append(valueDiags...)
	}

	networks := make([]networkModel, 0, len(resources.Networks))
	for _, network := range resources.Networks {
		if network == nil {
			continue
		}
		reservedPorts, valueDiags := flattenPorts(ctx, network.ReservedPorts)
		diags.Append(valueDiags...)
		dynamicPorts, valueDiags := flattenPorts(ctx, network.DynamicPorts)
		diags.Append(valueDiags...)

		dns := types.ObjectNull(dnsObjectType.AttrTypes)
		if network.DNS != nil {
			servers, dnsDiags := types.ListValueFrom(ctx, types.StringType, network.DNS.Servers)
			diags.Append(dnsDiags...)
			searches, dnsDiags := types.ListValueFrom(ctx, types.StringType, network.DNS.Searches)
			diags.Append(dnsDiags...)
			options, dnsDiags := types.ListValueFrom(ctx, types.StringType, network.DNS.Options)
			diags.Append(dnsDiags...)
			dns, dnsDiags = types.ObjectValueFrom(ctx, dnsObjectType.AttrTypes, dnsModel{
				Servers: servers, Searches: searches, Options: options,
			})
			diags.Append(dnsDiags...)
		}

		cni := types.ObjectNull(cniObjectType.AttrTypes)
		if network.CNI != nil {
			args, cniDiags := stringMapValue(ctx, network.CNI.Args)
			diags.Append(cniDiags...)
			cni, cniDiags = types.ObjectValueFrom(ctx, cniObjectType.AttrTypes, cniModel{Args: args})
			diags.Append(cniDiags...)
		}

		networks = append(networks, networkModel{
			Mode:          types.StringValue(network.Mode),
			Device:        types.StringValue(network.Device),
			CIDR:          types.StringValue(network.CIDR),
			IP:            types.StringValue(network.IP),
			Hostname:      types.StringValue(network.Hostname),
			MBits:         intPointerValue(network.MBits),
			DNS:           dns,
			ReservedPorts: reservedPorts,
			DynamicPorts:  dynamicPorts,
			CNI:           cni,
		})
	}
	networkList, valueDiags := types.ListValueFrom(ctx, networkObjectType, networks)
	diags.Append(valueDiags...)
	model.Networks = networkList

	devices := make([]requestedDeviceModel, 0, len(resources.Devices))
	for _, device := range resources.Devices {
		if device == nil {
			continue
		}
		constraints := make([]constraintModel, 0, len(device.Constraints))
		for _, constraint := range device.Constraints {
			if constraint == nil {
				continue
			}
			constraints = append(constraints, constraintModel{
				LTarget: types.StringValue(constraint.LTarget),
				RTarget: types.StringValue(constraint.RTarget),
				Operand: types.StringValue(constraint.Operand),
			})
		}
		constraintList, valueDiags := types.ListValueFrom(ctx, constraintObjectType, constraints)
		diags.Append(valueDiags...)

		affinities := make([]affinityModel, 0, len(device.Affinities))
		for _, affinity := range device.Affinities {
			if affinity == nil {
				continue
			}
			affinities = append(affinities, affinityModel{
				LTarget: types.StringValue(affinity.LTarget),
				RTarget: types.StringValue(affinity.RTarget),
				Operand: types.StringValue(affinity.Operand),
				Weight:  int8PointerValue(affinity.Weight),
			})
		}
		affinityList, valueDiags := types.ListValueFrom(ctx, affinityObjectType, affinities)
		diags.Append(valueDiags...)

		devices = append(devices, requestedDeviceModel{
			Name:        types.StringValue(device.Name),
			Count:       uint64PointerValue(device.Count),
			Constraints: constraintList,
			Affinities:  affinityList,
		})
	}
	model.Devices, valueDiags = types.ListValueFrom(ctx, requestedDeviceObjectType, devices)
	diags.Append(valueDiags...)

	value, valueDiags := types.ObjectValueFrom(ctx, resourcesObjectType.AttrTypes, model)
	diags.Append(valueDiags...)
	return value, diags
}

func flattenPorts(ctx context.Context, ports []api.Port) (types.List, diag.Diagnostics) {
	models := make([]portModel, 0, len(ports))
	for _, port := range ports {
		models = append(models, portModel{
			Label:           types.StringValue(port.Label),
			Value:           types.Int64Value(int64(port.Value)),
			To:              types.Int64Value(int64(port.To)),
			HostNetwork:     types.StringValue(port.HostNetwork),
			IgnoreCollision: types.BoolValue(port.IgnoreCollision),
		})
	}
	return types.ListValueFrom(ctx, portObjectType, models)
}

func stringMapValue(ctx context.Context, values map[string]string) (types.Map, diag.Diagnostics) {
	if values == nil {
		values = map[string]string{}
	}
	return types.MapValueFrom(ctx, types.StringType, values)
}

func stringPointerValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func boolPointerValue(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

func intPointerValue(value *int) types.Int64 {
	if value == nil {
		return types.Int64Value(0)
	}
	return types.Int64Value(int64(*value))
}

func uint64PointerValue(value *uint64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*value))
}

func int8PointerValue(value *int8) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*value))
}

func int64PointerStringValue(value *int64) types.String {
	if value == nil {
		return types.StringValue("")
	}
	return types.StringValue(strconv.FormatInt(*value, 10))
}
