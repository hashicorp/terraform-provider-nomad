package nomad

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/nomad/api"
	"github.com/terraform-providers/terraform-provider-nomad/nomad/core/helper"
)

func TestFlattenTaksGroups(t *testing.T) {
	cases := []struct {
		input []*api.TaskGroup
		want  []interface{}
	}{
		// simple task group
		{
			input: []*api.TaskGroup{
				&api.TaskGroup{
					Name:  helper.StringToPtr("cache"),
					Count: helper.IntToPtr(1),
					Tasks: []*api.Task{&simpleTask},
				},
			},
			want: []interface{}{
				map[string]interface{}{
					"name":  "cache",
					"count": 1,
					"task":  []interface{}{simpleTaskFlattened},
				},
			},
		},
	}

	for _, c := range cases {
		got := flattenTaskGroups(c.input)
		if diff := cmp.Diff(c.want, got); diff != "" {
			t.Errorf("flattenTaskGroups mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenTasks(t *testing.T) {
	cases := []struct {
		input []*api.Task
		want  []interface{}
	}{
		// simple task
		{
			input: []*api.Task{&simpleTask},
			want:  []interface{}{simpleTaskFlattened},
		},
		// multiple tasks
		{
			input: []*api.Task{&simpleTask, &simpleTask},
			want:  []interface{}{simpleTaskFlattened, simpleTaskFlattened},
		},
	}

	for _, c := range cases {
		got := flattenTasks(c.input)
		if diff := cmp.Diff(c.want, got); diff != "" {
			t.Errorf("flattenTasks mismatch (-want +got):\n%s", diff)
		}
	}
}

var simpleTask = api.Task{
	Name:   "example",
	Driver: "docker",
	Config: map[string]interface{}{
		"image": "redis:3.2",
		"port_map": map[string]int{
			"db": 6379,
		},
	},
}

var simpleTaskFlattened = map[string]interface{}{
	"name":   "example",
	"driver": "docker",
	"config": `{"image":"redis:3.2","port_map":{"db":6379}}`,
}
