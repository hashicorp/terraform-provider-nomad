// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec2"
)

type jobParserConfig struct {
	JSON bool
	HCL2 hcl2ParserConfig
}

type hcl2ParserConfig struct {
	AllowFS bool
	Vars    map[string]string
}

func parserConfigFromModel(ctx context.Context, data jobResourceModel) (jobParserConfig, error) {
	config := jobParserConfig{JSON: data.JSON.ValueBool()}
	if len(data.HCL2) == 0 {
		return config, nil
	}
	if len(data.HCL2) > 1 {
		return config, fmt.Errorf("failed to unpack hcl2 configuration block")
	}

	hcl2Config := data.HCL2[0]
	config.HCL2.AllowFS = hcl2Config.AllowFS.ValueBool()
	if !hcl2Config.Vars.IsNull() && !hcl2Config.Vars.IsUnknown() {
		if diags := hcl2Config.Vars.ElementsAs(ctx, &config.HCL2.Vars, false); diags.HasError() {
			return config, fmt.Errorf("failed to unpack hcl2 variables: %s", diags.Errors()[0].Detail())
		}
	}

	return config, nil
}

func parseJobspec(raw string, config jobParserConfig) (*api.Job, error) {
	var (
		job *api.Job
		err error
	)

	if config.JSON {
		job, err = parseJSONJobspec(raw)
	} else {
		job, err = parseHCL2Jobspec(raw, config.HCL2)
	}
	if err != nil {
		return nil, fmt.Errorf("error parsing jobspec: %s", err)
	}
	if job == nil || reflect.DeepEqual(job, &api.Job{}) {
		return nil, fmt.Errorf("error parsing jobspec: input JSON is not a valid Nomad jobspec")
	}

	job.Canonicalize()

	return job, nil
}

func parseJSONJobspec(raw string) (*api.Job, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		return nil, err
	}

	jobBytes, ok := root["Job"]
	if !ok {
		jobBytes = []byte(raw)
	}

	var job api.Job
	if err := json.Unmarshal(jobBytes, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func parseHCL2Jobspec(raw string, config hcl2ParserConfig) (*api.Job, error) {
	argVars := make([]string, 0, len(config.Vars))
	for key, value := range config.Vars {
		argVars = append(argVars, fmt.Sprintf("%s=%s", key, value))
	}

	return jobspec2.ParseWithConfig(&jobspec2.ParseConfig{
		Path:    "",
		Body:    []byte(raw),
		AllowFS: config.AllowFS,
		ArgVars: argVars,
		Strict:  true,
	})
}

func jobspecsEqual(oldRaw string, oldConfig jobParserConfig, newRaw string, newConfig jobParserConfig) bool {
	oldJob, oldErr := parseJobspec(oldRaw, oldConfig)
	newJob, newErr := parseJobspec(newRaw, newConfig)
	if oldErr != nil || newErr != nil {
		return false
	}

	oldJob.Canonicalize()
	newJob.Canonicalize()
	return reflect.DeepEqual(oldJob, newJob)
}

func defaultJobNamespace(job *api.Job) string {
	if job.Namespace == nil || *job.Namespace == "" {
		job.Namespace = pointerOf("default")
	}
	return *job.Namespace
}

func defaultJobRegion(job *api.Job) string {
	if job.Region == nil || *job.Region == "" {
		job.Region = pointerOf("global")
	}

	return *job.Region
}

func pointerOf[T any](value T) *T {
	return &value
}
