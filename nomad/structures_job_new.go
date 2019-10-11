package nomad

import (
	"encoding/json"
	"log"

	"github.com/hashicorp/nomad/api"
)

func flattenTaskGroups(in []*api.TaskGroup) []interface{} {
	out := make([]interface{}, 0, len(in))

	for _, tg := range in {
		log.Printf("group: %s", *tg.Name)
		m := make(map[string]interface{})

		m["name"] = tg.Name
		m["count"] = tg.Count
		m["task"] = flattenTasks(tg.Tasks)

		out = append(out, m)
	}

	return out
}

func flattenTasks(in []*api.Task) []interface{} {
	out := make([]interface{}, 0, len(in))

	for _, t := range in {
		m := make(map[string]interface{})
		m["name"] = t.Name
		m["driver"] = t.Driver

		c, err := json.Marshal(t.Config)
		if err != nil {
			// shouldn't really happen but let's log it
			log.Printf("[WARN] failed to parse task config: %s", err)
		}
		m["config"] = string(c)

		out = append(out, m)
	}
	return out

}
