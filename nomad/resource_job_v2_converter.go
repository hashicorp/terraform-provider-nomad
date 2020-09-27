package nomad

import (
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/helper"
	"github.com/hashicorp/nomad/nomad/structs"
)

func ApiJobToStructJob(job *api.Job) *structs.Job {
	job.Canonicalize()

	j := &structs.Job{
		Stop:           *job.Stop,
		Region:         *job.Region,
		Namespace:      *job.Namespace,
		ID:             *job.ID,
		ParentID:       *job.ParentID,
		Name:           *job.Name,
		Type:           *job.Type,
		Priority:       *job.Priority,
		AllAtOnce:      *job.AllAtOnce,
		Datacenters:    job.Datacenters,
		Payload:        job.Payload,
		Meta:           job.Meta,
		ConsulToken:    *job.ConsulToken,
		VaultToken:     *job.VaultToken,
		VaultNamespace: *job.VaultNamespace,
		Constraints:    ApiConstraintsToStructs(job.Constraints),
		Affinities:     ApiAffinitiesToStructs(job.Affinities),
	}

	// Update has been pushed into the task groups. stagger and max_parallel are
	// preserved at the job level, but all other values are discarded. The job.Update
	// api value is merged into TaskGroups already in api.Canonicalize
	if job.Update != nil && job.Update.MaxParallel != nil && *job.Update.MaxParallel > 0 {
		j.Update = structs.UpdateStrategy{}

		if job.Update.Stagger != nil {
			j.Update.Stagger = *job.Update.Stagger
		}
		if job.Update.MaxParallel != nil {
			j.Update.MaxParallel = *job.Update.MaxParallel
		}
	}

	if l := len(job.Spreads); l != 0 {
		j.Spreads = make([]*structs.Spread, l)
		for i, apiSpread := range job.Spreads {
			j.Spreads[i] = ApiSpreadToStructs(apiSpread)
		}
	}

	if job.Periodic != nil {
		j.Periodic = &structs.PeriodicConfig{
			Enabled:         *job.Periodic.Enabled,
			SpecType:        *job.Periodic.SpecType,
			ProhibitOverlap: *job.Periodic.ProhibitOverlap,
			TimeZone:        *job.Periodic.TimeZone,
		}

		if job.Periodic.Spec != nil {
			j.Periodic.Spec = *job.Periodic.Spec
		}
	}

	if job.ParameterizedJob != nil {
		j.ParameterizedJob = &structs.ParameterizedJobConfig{
			Payload:      job.ParameterizedJob.Payload,
			MetaRequired: job.ParameterizedJob.MetaRequired,
			MetaOptional: job.ParameterizedJob.MetaOptional,
		}
	}

	if job.Multiregion != nil {
		j.Multiregion = &structs.Multiregion{}
		j.Multiregion.Strategy = &structs.MultiregionStrategy{
			MaxParallel: *job.Multiregion.Strategy.MaxParallel,
			OnFailure:   *job.Multiregion.Strategy.OnFailure,
		}
		j.Multiregion.Regions = []*structs.MultiregionRegion{}
		for _, region := range job.Multiregion.Regions {
			r := &structs.MultiregionRegion{}
			r.Name = region.Name
			r.Count = *region.Count
			r.Datacenters = region.Datacenters
			r.Meta = region.Meta
			j.Multiregion.Regions = append(j.Multiregion.Regions, r)
		}
	}

	if l := len(job.TaskGroups); l != 0 {
		j.TaskGroups = make([]*structs.TaskGroup, l)
		for i, taskGroup := range job.TaskGroups {
			tg := &structs.TaskGroup{}
			ApiTgToStructsTG(j, taskGroup, tg)
			j.TaskGroups[i] = tg
		}
	}

	return j
}

func ApiTgToStructsTG(job *structs.Job, taskGroup *api.TaskGroup, tg *structs.TaskGroup) {
	tg.Name = *taskGroup.Name
	tg.Count = *taskGroup.Count
	tg.Meta = taskGroup.Meta
	tg.Constraints = ApiConstraintsToStructs(taskGroup.Constraints)
	tg.Affinities = ApiAffinitiesToStructs(taskGroup.Affinities)
	tg.Networks = ApiNetworkResourceToStructs(taskGroup.Networks)
	tg.Services = ApiServicesToStructs(taskGroup.Services)

	tg.RestartPolicy = &structs.RestartPolicy{
		Attempts: *taskGroup.RestartPolicy.Attempts,
		Interval: *taskGroup.RestartPolicy.Interval,
		Delay:    *taskGroup.RestartPolicy.Delay,
		Mode:     *taskGroup.RestartPolicy.Mode,
	}

	if taskGroup.ShutdownDelay != nil {
		tg.ShutdownDelay = taskGroup.ShutdownDelay
	}

	if taskGroup.StopAfterClientDisconnect != nil {
		tg.StopAfterClientDisconnect = taskGroup.StopAfterClientDisconnect
	}

	if taskGroup.ReschedulePolicy != nil {
		tg.ReschedulePolicy = &structs.ReschedulePolicy{
			Attempts:      *taskGroup.ReschedulePolicy.Attempts,
			Interval:      *taskGroup.ReschedulePolicy.Interval,
			Delay:         *taskGroup.ReschedulePolicy.Delay,
			DelayFunction: *taskGroup.ReschedulePolicy.DelayFunction,
			MaxDelay:      *taskGroup.ReschedulePolicy.MaxDelay,
			Unlimited:     *taskGroup.ReschedulePolicy.Unlimited,
		}
	}

	if taskGroup.Migrate != nil {
		tg.Migrate = &structs.MigrateStrategy{
			MaxParallel:     *taskGroup.Migrate.MaxParallel,
			HealthCheck:     *taskGroup.Migrate.HealthCheck,
			MinHealthyTime:  *taskGroup.Migrate.MinHealthyTime,
			HealthyDeadline: *taskGroup.Migrate.HealthyDeadline,
		}
	}

	if taskGroup.Scaling != nil {
		tg.Scaling = ApiScalingPolicyToStructs(tg.Count, taskGroup.Scaling).TargetTaskGroup(job, tg)
	}

	tg.EphemeralDisk = &structs.EphemeralDisk{
		Sticky:  *taskGroup.EphemeralDisk.Sticky,
		SizeMB:  *taskGroup.EphemeralDisk.SizeMB,
		Migrate: *taskGroup.EphemeralDisk.Migrate,
	}

	if l := len(taskGroup.Spreads); l != 0 {
		tg.Spreads = make([]*structs.Spread, l)
		for k, spread := range taskGroup.Spreads {
			tg.Spreads[k] = ApiSpreadToStructs(spread)
		}
	}

	if l := len(taskGroup.Volumes); l != 0 {
		tg.Volumes = make(map[string]*structs.VolumeRequest, l)
		for k, v := range taskGroup.Volumes {
			if v.Type != structs.VolumeTypeHost && v.Type != structs.VolumeTypeCSI {
				// Ignore volumes we don't understand in this iteration currently.
				// - This is because we don't currently have a way to return errors here.
				continue
			}

			vol := &structs.VolumeRequest{
				Name:     v.Name,
				Type:     v.Type,
				ReadOnly: v.ReadOnly,
				Source:   v.Source,
			}

			if v.MountOptions != nil {
				vol.MountOptions = &structs.CSIMountOptions{
					FSType:     v.MountOptions.FSType,
					MountFlags: v.MountOptions.MountFlags,
				}
			}

			tg.Volumes[k] = vol
		}
	}

	if taskGroup.Update != nil {
		tg.Update = &structs.UpdateStrategy{
			Stagger:          *taskGroup.Update.Stagger,
			MaxParallel:      *taskGroup.Update.MaxParallel,
			HealthCheck:      *taskGroup.Update.HealthCheck,
			MinHealthyTime:   *taskGroup.Update.MinHealthyTime,
			HealthyDeadline:  *taskGroup.Update.HealthyDeadline,
			ProgressDeadline: *taskGroup.Update.ProgressDeadline,
			Canary:           *taskGroup.Update.Canary,
		}

		// boolPtr fields may be nil, others will have pointers to default values via Canonicalize
		if taskGroup.Update.AutoRevert != nil {
			tg.Update.AutoRevert = *taskGroup.Update.AutoRevert
		}

		if taskGroup.Update.AutoPromote != nil {
			tg.Update.AutoPromote = *taskGroup.Update.AutoPromote
		}
	}

	if l := len(taskGroup.Tasks); l != 0 {
		tg.Tasks = make([]*structs.Task, l)
		for l, task := range taskGroup.Tasks {
			t := &structs.Task{}
			ApiTaskToStructsTask(task, t)

			// Set the tasks vault namespace from Job if it was not
			// specified by the task or group
			if t.Vault != nil && t.Vault.Namespace == "" && job.VaultNamespace != "" {
				t.Vault.Namespace = job.VaultNamespace
			}
			tg.Tasks[l] = t
		}
	}
}

// ApiTaskToStructsTask is a copy and type conversion between the API
// representation of a task from a struct representation of a task.
func ApiTaskToStructsTask(apiTask *api.Task, structsTask *structs.Task) {
	structsTask.Name = apiTask.Name
	structsTask.Driver = apiTask.Driver
	structsTask.User = apiTask.User
	structsTask.Leader = apiTask.Leader
	structsTask.Config = apiTask.Config
	structsTask.Env = apiTask.Env
	structsTask.Meta = apiTask.Meta
	structsTask.KillTimeout = *apiTask.KillTimeout
	structsTask.ShutdownDelay = apiTask.ShutdownDelay
	structsTask.KillSignal = apiTask.KillSignal
	structsTask.Kind = structs.TaskKind(apiTask.Kind)
	structsTask.Constraints = ApiConstraintsToStructs(apiTask.Constraints)
	structsTask.Affinities = ApiAffinitiesToStructs(apiTask.Affinities)
	structsTask.CSIPluginConfig = ApiCSIPluginConfigToStructsCSIPluginConfig(apiTask.CSIPluginConfig)

	if apiTask.RestartPolicy != nil {
		structsTask.RestartPolicy = &structs.RestartPolicy{
			Attempts: *apiTask.RestartPolicy.Attempts,
			Interval: *apiTask.RestartPolicy.Interval,
			Delay:    *apiTask.RestartPolicy.Delay,
			Mode:     *apiTask.RestartPolicy.Mode,
		}
	}

	if l := len(apiTask.VolumeMounts); l != 0 {
		structsTask.VolumeMounts = make([]*structs.VolumeMount, l)
		for i, mount := range apiTask.VolumeMounts {
			structsTask.VolumeMounts[i] = &structs.VolumeMount{
				Volume:          *mount.Volume,
				Destination:     *mount.Destination,
				ReadOnly:        *mount.ReadOnly,
				PropagationMode: *mount.PropagationMode,
			}
		}
	}

	if l := len(apiTask.Services); l != 0 {
		structsTask.Services = make([]*structs.Service, l)
		for i, service := range apiTask.Services {
			structsTask.Services[i] = &structs.Service{
				Name:              service.Name,
				PortLabel:         service.PortLabel,
				Tags:              service.Tags,
				CanaryTags:        service.CanaryTags,
				EnableTagOverride: service.EnableTagOverride,
				AddressMode:       service.AddressMode,
				Meta:              helper.CopyMapStringString(service.Meta),
				CanaryMeta:        helper.CopyMapStringString(service.CanaryMeta),
			}

			if l := len(service.Checks); l != 0 {
				structsTask.Services[i].Checks = make([]*structs.ServiceCheck, l)
				for j, check := range service.Checks {
					structsTask.Services[i].Checks[j] = &structs.ServiceCheck{
						Name:                   check.Name,
						Type:                   check.Type,
						Command:                check.Command,
						Args:                   check.Args,
						Path:                   check.Path,
						Protocol:               check.Protocol,
						PortLabel:              check.PortLabel,
						AddressMode:            check.AddressMode,
						Interval:               check.Interval,
						Timeout:                check.Timeout,
						InitialStatus:          check.InitialStatus,
						TLSSkipVerify:          check.TLSSkipVerify,
						Header:                 check.Header,
						Method:                 check.Method,
						GRPCService:            check.GRPCService,
						GRPCUseTLS:             check.GRPCUseTLS,
						SuccessBeforePassing:   check.SuccessBeforePassing,
						FailuresBeforeCritical: check.FailuresBeforeCritical,
					}
					if check.CheckRestart != nil {
						structsTask.Services[i].Checks[j].CheckRestart = &structs.CheckRestart{
							Limit:          check.CheckRestart.Limit,
							Grace:          *check.CheckRestart.Grace,
							IgnoreWarnings: check.CheckRestart.IgnoreWarnings,
						}
					}
				}
			}
		}
	}

	structsTask.Resources = ApiResourcesToStructs(apiTask.Resources)

	structsTask.LogConfig = &structs.LogConfig{
		MaxFiles:      *apiTask.LogConfig.MaxFiles,
		MaxFileSizeMB: *apiTask.LogConfig.MaxFileSizeMB,
	}

	if l := len(apiTask.Artifacts); l != 0 {
		structsTask.Artifacts = make([]*structs.TaskArtifact, l)
		for k, ta := range apiTask.Artifacts {
			structsTask.Artifacts[k] = &structs.TaskArtifact{
				GetterSource:  *ta.GetterSource,
				GetterOptions: ta.GetterOptions,
				GetterMode:    *ta.GetterMode,
				RelativeDest:  *ta.RelativeDest,
			}
		}
	}

	if apiTask.Vault != nil {
		structsTask.Vault = &structs.Vault{
			Policies:     apiTask.Vault.Policies,
			Namespace:    *apiTask.Vault.Namespace,
			Env:          *apiTask.Vault.Env,
			ChangeMode:   *apiTask.Vault.ChangeMode,
			ChangeSignal: *apiTask.Vault.ChangeSignal,
		}
	}

	if l := len(apiTask.Templates); l != 0 {
		structsTask.Templates = make([]*structs.Template, l)
		for i, template := range apiTask.Templates {
			structsTask.Templates[i] = &structs.Template{
				SourcePath:   *template.SourcePath,
				DestPath:     *template.DestPath,
				EmbeddedTmpl: *template.EmbeddedTmpl,
				ChangeMode:   *template.ChangeMode,
				ChangeSignal: *template.ChangeSignal,
				Splay:        *template.Splay,
				Perms:        *template.Perms,
				LeftDelim:    *template.LeftDelim,
				RightDelim:   *template.RightDelim,
				Envvars:      *template.Envvars,
				VaultGrace:   *template.VaultGrace,
			}
		}
	}

	if apiTask.DispatchPayload != nil {
		structsTask.DispatchPayload = &structs.DispatchPayloadConfig{
			File: apiTask.DispatchPayload.File,
		}
	}

	if apiTask.Lifecycle != nil {
		structsTask.Lifecycle = &structs.TaskLifecycleConfig{
			Hook:    apiTask.Lifecycle.Hook,
			Sidecar: apiTask.Lifecycle.Sidecar,
		}
	}
}

func ApiCSIPluginConfigToStructsCSIPluginConfig(apiConfig *api.TaskCSIPluginConfig) *structs.TaskCSIPluginConfig {
	if apiConfig == nil {
		return nil
	}

	sc := &structs.TaskCSIPluginConfig{}
	sc.ID = apiConfig.ID
	sc.Type = structs.CSIPluginType(apiConfig.Type)
	sc.MountDir = apiConfig.MountDir
	return sc
}

func ApiResourcesToStructs(in *api.Resources) *structs.Resources {
	if in == nil {
		return nil
	}

	out := &structs.Resources{
		CPU:      *in.CPU,
		MemoryMB: *in.MemoryMB,
	}

	// COMPAT(0.10): Only being used to issue warnings
	if in.IOPS != nil {
		out.IOPS = *in.IOPS
	}

	if len(in.Networks) != 0 {
		out.Networks = ApiNetworkResourceToStructs(in.Networks)
	}

	if l := len(in.Devices); l != 0 {
		out.Devices = make([]*structs.RequestedDevice, l)
		for i, d := range in.Devices {
			out.Devices[i] = &structs.RequestedDevice{
				Name:        d.Name,
				Count:       *d.Count,
				Constraints: ApiConstraintsToStructs(d.Constraints),
				Affinities:  ApiAffinitiesToStructs(d.Affinities),
			}
		}
	}

	return out
}

func ApiNetworkResourceToStructs(in []*api.NetworkResource) []*structs.NetworkResource {
	var out []*structs.NetworkResource
	if len(in) == 0 {
		return out
	}
	out = make([]*structs.NetworkResource, len(in))
	for i, nw := range in {
		out[i] = &structs.NetworkResource{
			Mode:  nw.Mode,
			CIDR:  nw.CIDR,
			IP:    nw.IP,
			MBits: *nw.MBits,
		}

		if nw.DNS != nil {
			out[i].DNS = &structs.DNSConfig{
				Servers:  nw.DNS.Servers,
				Searches: nw.DNS.Searches,
				Options:  nw.DNS.Options,
			}
		}

		if l := len(nw.DynamicPorts); l != 0 {
			out[i].DynamicPorts = make([]structs.Port, l)
			for j, dp := range nw.DynamicPorts {
				out[i].DynamicPorts[j] = ApiPortToStructs(dp)
			}
		}

		if l := len(nw.ReservedPorts); l != 0 {
			out[i].ReservedPorts = make([]structs.Port, l)
			for j, rp := range nw.ReservedPorts {
				out[i].ReservedPorts[j] = ApiPortToStructs(rp)
			}
		}
	}

	return out
}

func ApiPortToStructs(in api.Port) structs.Port {
	return structs.Port{
		Label:       in.Label,
		Value:       in.Value,
		To:          in.To,
		HostNetwork: in.HostNetwork,
	}
}

//TODO(schmichael) refactor and reuse in service parsing above
func ApiServicesToStructs(in []*api.Service) []*structs.Service {
	if len(in) == 0 {
		return nil
	}

	out := make([]*structs.Service, len(in))
	for i, s := range in {
		out[i] = &structs.Service{
			Name:              s.Name,
			PortLabel:         s.PortLabel,
			TaskName:          s.TaskName,
			Tags:              s.Tags,
			CanaryTags:        s.CanaryTags,
			EnableTagOverride: s.EnableTagOverride,
			AddressMode:       s.AddressMode,
			Meta:              helper.CopyMapStringString(s.Meta),
			CanaryMeta:        helper.CopyMapStringString(s.CanaryMeta),
		}

		if l := len(s.Checks); l != 0 {
			out[i].Checks = make([]*structs.ServiceCheck, l)
			for j, check := range s.Checks {
				out[i].Checks[j] = &structs.ServiceCheck{
					Name:          check.Name,
					Type:          check.Type,
					Command:       check.Command,
					Args:          check.Args,
					Path:          check.Path,
					Protocol:      check.Protocol,
					PortLabel:     check.PortLabel,
					Expose:        check.Expose,
					AddressMode:   check.AddressMode,
					Interval:      check.Interval,
					Timeout:       check.Timeout,
					InitialStatus: check.InitialStatus,
					TLSSkipVerify: check.TLSSkipVerify,
					Header:        check.Header,
					Method:        check.Method,
					GRPCService:   check.GRPCService,
					GRPCUseTLS:    check.GRPCUseTLS,
					TaskName:      check.TaskName,
				}
				if check.CheckRestart != nil {
					out[i].Checks[j].CheckRestart = &structs.CheckRestart{
						Limit:          check.CheckRestart.Limit,
						Grace:          *check.CheckRestart.Grace,
						IgnoreWarnings: check.CheckRestart.IgnoreWarnings,
					}
				}
			}
		}

		if s.Connect != nil {
			out[i].Connect = ApiConsulConnectToStructs(s.Connect)
		}

	}

	return out
}

func ApiConsulConnectToStructs(in *api.ConsulConnect) *structs.ConsulConnect {
	if in == nil {
		return nil
	}
	return &structs.ConsulConnect{
		Native:         in.Native,
		SidecarService: apiConnectSidecarServiceToStructs(in.SidecarService),
		SidecarTask:    apiConnectSidecarTaskToStructs(in.SidecarTask),
		Gateway:        apiConnectGatewayToStructs(in.Gateway),
	}
}

func apiConnectGatewayToStructs(in *api.ConsulGateway) *structs.ConsulGateway {
	if in == nil {
		return nil
	}

	return &structs.ConsulGateway{
		Proxy:   apiConnectGatewayProxyToStructs(in.Proxy),
		Ingress: apiConnectIngressGatewayToStructs(in.Ingress),
	}
}

func apiConnectGatewayProxyToStructs(in *api.ConsulGatewayProxy) *structs.ConsulGatewayProxy {
	if in == nil {
		return nil
	}

	var bindAddresses map[string]*structs.ConsulGatewayBindAddress
	if in.EnvoyGatewayBindAddresses != nil {
		bindAddresses = make(map[string]*structs.ConsulGatewayBindAddress)
		for k, v := range in.EnvoyGatewayBindAddresses {
			bindAddresses[k] = &structs.ConsulGatewayBindAddress{
				Address: v.Address,
				Port:    v.Port,
			}
		}
	}

	return &structs.ConsulGatewayProxy{
		ConnectTimeout:                  in.ConnectTimeout,
		EnvoyGatewayBindTaggedAddresses: in.EnvoyGatewayBindTaggedAddresses,
		EnvoyGatewayBindAddresses:       bindAddresses,
		EnvoyGatewayNoDefaultBind:       in.EnvoyGatewayNoDefaultBind,
		Config:                          helper.CopyMapStringInterface(in.Config),
	}
}

func apiConnectIngressGatewayToStructs(in *api.ConsulIngressConfigEntry) *structs.ConsulIngressConfigEntry {
	if in == nil {
		return nil
	}

	return &structs.ConsulIngressConfigEntry{
		TLS:       apiConnectGatewayTLSConfig(in.TLS),
		Listeners: apiConnectIngressListenersToStructs(in.Listeners),
	}
}

func apiConnectGatewayTLSConfig(in *api.ConsulGatewayTLSConfig) *structs.ConsulGatewayTLSConfig {
	if in == nil {
		return nil
	}

	return &structs.ConsulGatewayTLSConfig{
		Enabled: in.Enabled,
	}
}

func apiConnectIngressListenersToStructs(in []*api.ConsulIngressListener) []*structs.ConsulIngressListener {
	if len(in) == 0 {
		return nil
	}

	listeners := make([]*structs.ConsulIngressListener, len(in))
	for i, listener := range in {
		listeners[i] = apiConnectIngressListenerToStructs(listener)
	}
	return listeners
}

func apiConnectIngressListenerToStructs(in *api.ConsulIngressListener) *structs.ConsulIngressListener {
	if in == nil {
		return nil
	}

	return &structs.ConsulIngressListener{
		Port:     in.Port,
		Protocol: in.Protocol,
		Services: apiConnectIngressServicesToStructs(in.Services),
	}
}

func apiConnectIngressServicesToStructs(in []*api.ConsulIngressService) []*structs.ConsulIngressService {
	if len(in) == 0 {
		return nil
	}

	services := make([]*structs.ConsulIngressService, len(in))
	for i, service := range in {
		services[i] = apiConnectIngressServiceToStructs(service)
	}
	return services
}

func apiConnectIngressServiceToStructs(in *api.ConsulIngressService) *structs.ConsulIngressService {
	if in == nil {
		return nil
	}

	return &structs.ConsulIngressService{
		Name:  in.Name,
		Hosts: helper.CopySliceString(in.Hosts),
	}
}

func apiConnectSidecarServiceToStructs(in *api.ConsulSidecarService) *structs.ConsulSidecarService {
	if in == nil {
		return nil
	}
	return &structs.ConsulSidecarService{
		Port:  in.Port,
		Tags:  helper.CopySliceString(in.Tags),
		Proxy: apiConnectSidecarServiceProxyToStructs(in.Proxy),
	}
}

func apiConnectSidecarServiceProxyToStructs(in *api.ConsulProxy) *structs.ConsulProxy {
	if in == nil {
		return nil
	}
	return &structs.ConsulProxy{
		LocalServiceAddress: in.LocalServiceAddress,
		LocalServicePort:    in.LocalServicePort,
		Upstreams:           apiUpstreamsToStructs(in.Upstreams),
		Expose:              apiConsulExposeConfigToStructs(in.ExposeConfig),
		Config:              helper.CopyMapStringInterface(in.Config),
	}
}

func apiUpstreamsToStructs(in []*api.ConsulUpstream) []structs.ConsulUpstream {
	if len(in) == 0 {
		return nil
	}
	upstreams := make([]structs.ConsulUpstream, len(in))
	for i, upstream := range in {
		upstreams[i] = structs.ConsulUpstream{
			DestinationName: upstream.DestinationName,
			LocalBindPort:   upstream.LocalBindPort,
		}
	}
	return upstreams
}

func apiConsulExposeConfigToStructs(in *api.ConsulExposeConfig) *structs.ConsulExposeConfig {
	if in == nil {
		return nil
	}
	return &structs.ConsulExposeConfig{
		Paths: apiConsulExposePathsToStructs(in.Path),
	}
}

func apiConsulExposePathsToStructs(in []*api.ConsulExposePath) []structs.ConsulExposePath {
	if len(in) == 0 {
		return nil
	}
	paths := make([]structs.ConsulExposePath, len(in))
	for i, path := range in {
		paths[i] = structs.ConsulExposePath{
			Path:          path.Path,
			Protocol:      path.Protocol,
			LocalPathPort: path.LocalPathPort,
			ListenerPort:  path.ListenerPort,
		}
	}
	return paths
}

func apiConnectSidecarTaskToStructs(in *api.SidecarTask) *structs.SidecarTask {
	if in == nil {
		return nil
	}
	return &structs.SidecarTask{
		Name:          in.Name,
		Driver:        in.Driver,
		User:          in.User,
		Config:        in.Config,
		Env:           in.Env,
		Resources:     ApiResourcesToStructs(in.Resources),
		Meta:          in.Meta,
		ShutdownDelay: in.ShutdownDelay,
		KillSignal:    in.KillSignal,
		KillTimeout:   in.KillTimeout,
		LogConfig:     apiLogConfigToStructs(in.LogConfig),
	}
}

func apiLogConfigToStructs(in *api.LogConfig) *structs.LogConfig {
	if in == nil {
		return nil
	}
	return &structs.LogConfig{
		MaxFiles:      dereferenceInt(in.MaxFiles),
		MaxFileSizeMB: dereferenceInt(in.MaxFileSizeMB),
	}
}

func dereferenceInt(in *int) int {
	if in == nil {
		return 0
	}
	return *in
}

func ApiConstraintsToStructs(in []*api.Constraint) []*structs.Constraint {
	if in == nil {
		return nil
	}

	out := make([]*structs.Constraint, len(in))
	for i, ac := range in {
		out[i] = ApiConstraintToStructs(ac)
	}

	return out
}

func ApiConstraintToStructs(in *api.Constraint) *structs.Constraint {
	if in == nil {
		return nil
	}

	return &structs.Constraint{
		LTarget: in.LTarget,
		RTarget: in.RTarget,
		Operand: in.Operand,
	}
}

func ApiAffinitiesToStructs(in []*api.Affinity) []*structs.Affinity {
	if in == nil {
		return nil
	}

	out := make([]*structs.Affinity, len(in))
	for i, ac := range in {
		out[i] = ApiAffinityToStructs(ac)
	}

	return out
}

func ApiAffinityToStructs(a1 *api.Affinity) *structs.Affinity {
	return &structs.Affinity{
		LTarget: a1.LTarget,
		Operand: a1.Operand,
		RTarget: a1.RTarget,
		Weight:  *a1.Weight,
	}
}

func ApiSpreadToStructs(a1 *api.Spread) *structs.Spread {
	ret := &structs.Spread{}
	ret.Attribute = a1.Attribute
	ret.Weight = *a1.Weight
	if a1.SpreadTarget != nil {
		ret.SpreadTarget = make([]*structs.SpreadTarget, len(a1.SpreadTarget))
		for i, st := range a1.SpreadTarget {
			ret.SpreadTarget[i] = &structs.SpreadTarget{
				Value:   st.Value,
				Percent: st.Percent,
			}
		}
	}
	return ret
}

func ApiScalingPolicyToStructs(count int, ap *api.ScalingPolicy) *structs.ScalingPolicy {
	p := structs.ScalingPolicy{
		Enabled: *ap.Enabled,
		Policy:  ap.Policy,
		Target:  map[string]string{},
	}
	if ap.Max != nil {
		p.Max = *ap.Max
	} else {
		// catch this in Validate
		p.Max = -1
	}
	if ap.Min != nil {
		p.Min = *ap.Min
	} else {
		p.Min = int64(count)
	}
	return &p
}
