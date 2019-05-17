package nomad

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-nomad/nomad/jobspec"
)

func resourceJob() *schema.Resource {
	return &schema.Resource{
		Create: resourceJobRegister,
		Update: resourceJobRegister,
		Delete: resourceJobDeregister,
		Read:   resourceJobRead,

		CustomizeDiff: resourceJobCustomizeDiff,

		Schema: map[string]*schema.Schema{
			"jobspec": {
				Description:      "Job specification. If you want to point to a file use the file() function.",
				Required:         true,
				Type:             schema.TypeString,
				DiffSuppressFunc: jobspecDiffSuppress,
			},

			"policy_override": {
				Description: "Override any soft-mandatory Sentinel policies that fail.",
				Optional:    true,
				Type:        schema.TypeBool,
			},

			"deregister_on_destroy": {
				Description: "If true, the job will be deregistered on destroy.",
				Optional:    true,
				Default:     true,
				Type:        schema.TypeBool,
			},

			"deregister_on_id_change": {
				Description: "If true, the job will be deregistered when the job ID changes.",
				Optional:    true,
				Default:     true,
				Type:        schema.TypeBool,
			},

			"detach": {
				Description: "If true, the provider will return immediately after creating or updating, instead of monitoring.",
				Optional:    true,
				Default:     true,
				Type:        schema.TypeBool,
			},

			"deployment_id": {
				Description: "If detach = false, the ID for the deployment associated with the last job create/update, if one exists.",
				Computed:    true,
				Type:        schema.TypeString,
			},

			"deployment_status": {
				Description: "If detach = false, the status for the deployment associated with the last job create/update, if one exists.",
				Computed:    true,
				Type:        schema.TypeString,
			},

			"json": {
				Description: "If true, the `jobspec` will be parsed as json instead of HCL.",
				Optional:    true,
				Type:        schema.TypeBool,
			},

			"modify_index": {
				Description: "Integer that increments for each change. Used to detect any changes between plan and apply.",
				Computed:    true,
				Type:        schema.TypeString, // it's an int64, so won't fit in our TypeInt
			},

			"name": {
				Description: "The name of the job, as derived from the jobspec.",
				Computed:    true,
				Type:        schema.TypeString,
			},

			"type": {
				Description: "The type of the job, as derived from the jobspec.",
				Computed:    true,
				Type:        schema.TypeString,
			},

			"region": {
				Description: "The target region for the job, as derived from the jobspec.",
				Computed:    true,
				Type:        schema.TypeString,
			},

			"datacenters": {
				Description: "The target datacenters for the job, as derived from the jobspec.",
				Computed:    true,
				Type:        schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"allocation_ids": {
				Description: "The IDs for allocations associated with this job.",
				Computed:    true,
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"task_groups": {
				Computed: true,
				Type:     schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"count": {
							Computed: true,
							Type:     schema.TypeInt,
						},
						"task": {
							Computed: true,
							Type:     schema.TypeList,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Computed: true,
										Type:     schema.TypeString,
									},
									"driver": {
										Computed: true,
										Type:     schema.TypeString,
									},
									"meta": {
										Computed: true,
										Type:     schema.TypeMap,
									},
								},
							},
						},
						"meta": {
							Computed: true,
							Type:     schema.TypeMap,
						},
					},
				},
			},
		},
	}
}

func resourceJobRegister(d *schema.ResourceData, meta interface{}) error {
	timeout := d.Timeout(schema.TimeoutCreate)
	if !d.IsNewResource() {
		timeout = d.Timeout(schema.TimeoutUpdate)
	}

	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	// Get the jobspec itself
	jobspecRaw := d.Get("jobspec").(string)
	is_json := d.Get("json").(bool)
	job, err := parseJobspec(jobspecRaw, is_json, providerConfig.vaultToken)
	if err != nil {
		return err
	}

	// Register the job
	wantModifyIndexStrI, _ := d.GetChange("modify_index")
	wantModifyIndex, err := strconv.ParseUint(wantModifyIndexStrI.(string), 10, 64)
	if err != nil {
		wantModifyIndex = 0
	}

	resp, _, err := client.Jobs().RegisterOpts(job, &api.RegisterOptions{
		PolicyOverride: d.Get("policy_override").(bool),
		ModifyIndex:    wantModifyIndex,
	}, nil)
	if err != nil {
		return fmt.Errorf("error applying jobspec: %s", err)
	}

	log.Printf("[DEBUG] job '%s' registered", *job.ID)
	d.SetId(*job.ID)
	d.Set("name", job.ID)
	d.Set("modify_index", strconv.FormatUint(resp.JobModifyIndex, 10))

	evalId := resp.EvalID
	if !d.Get("detach").(bool) {
		log.Printf("[DEBUG] will monitor deployment of job '%s'", *job.ID)
		stateConf := &resource.StateChangeConf{
			Pending:    []string{"monitoring_deployment", "monitoring_evaluation"},
			Target:     []string{"job_scheduled_without_deployment", "deployment_successful"},
			Refresh:    deploymentStateRefreshFunc(client, evalId),
			Timeout:    timeout,
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		state, err := stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf(
				"error waiting for job '%s' to deploy successfully: %s",
				*job.ID, err)
		}

		deployment := state.(*api.Deployment)

		if deployment != nil {
			d.Set("deployment_id", deployment.ID)
			d.Set("deployment_status", deployment.Status)
		} else {
			d.Set("deployment_id", "")
			d.Set("deployment_status", "")
		}
	}

	return resourceJobRead(d, meta) // populate other computed attributes
}

type monitorState struct {
	// evalId is the evaluation that we are currently monitoring. This will change
	// along with follow-up evals.
	evalId string

	// deploymentId is the deployment that we are monitoring. This is captured from the
	// final evaluation.
	deploymentId string
}

// deploymentStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// the deployment from a job create/update
func deploymentStateRefreshFunc(client *api.Client, initialEvalId string) resource.StateRefreshFunc {
	watching := &monitorState{
		evalId:       initialEvalId,
		deploymentId: "",
	}
	return func() (interface{}, string, error) {

		var deployment *api.Deployment
		var state string
		if watching.deploymentId == "" {
			// monitor the eval
			log.Printf("[DEBUG] monitoring evaluation '%s'", watching.evalId)
			eval, _, err := client.Evaluations().Info(watching.evalId, nil)
			if err != nil {
				log.Printf("[ERROR] error on Evaluation.Info during deploymentStateRefresh: %s", err)
				return nil, "", err
			}

			switch eval.Status {
			case "complete":
				// Monitor the next eval in the chain, if present
				if eval.NextEval != "" {
					log.Printf("[DEBUG] will monitor follow-up eval '%v'", watching.evalId)
					watching.evalId = eval.NextEval
					state = "monitoring_evaluation"
				} else if eval.DeploymentID != "" {
					log.Printf("[DEBUG] job has been scheduled, will monitor deployment '%s'", eval.DeploymentID)
					watching.deploymentId = eval.DeploymentID
					state = "monitoring_deployment"
				} else {
					log.Printf("[WARN] job has been scheduled, but there is no deployment to monitor")
					state = "job_scheduled_without_deployment"
				}
			case "failed", "cancelled":
				return nil, "", fmt.Errorf("evaluation failed: %v", eval.StatusDescription)
			default:
				state = "monitoring_evaluation"
			}
		} else {
			// monitor the deployment
			deployment, _, err := client.Deployments().Info(watching.deploymentId, nil)
			if err != nil {
				log.Printf("[ERROR] error on Deployment.Info during deploymentStateRefresh: %s", err)
				return nil, "", err
			}
			switch deployment.Status {
			case "successful":
				log.Printf("[DEBUG] deployment '%s' successful", deployment.ID)
				state = "deployment_successful"
			case "failed", "cancelled":
				log.Printf("[DEBUG] deployment unsuccessful: %s", deployment.StatusDescription)
				return deployment, "",
					fmt.Errorf("deployment '%s' terminated with status '%s': '%s'",
						deployment.ID, deployment.Status, deployment.StatusDescription)
			default:
				// don't overwhelm the API server
				state = "monitoring_deployment"
			}
		}
		return deployment, state, nil
	}
}

func resourceJobDeregister(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	// If deregistration is disabled, then do nothing
	if !d.Get("deregister_on_destroy").(bool) {
		log.Printf(
			"[WARN] job %q will not deregister since 'deregister_on_destroy'"+
				" is false", d.Id())
		return nil
	}

	id := d.Id()
	log.Printf("[DEBUG] deregistering job: %q", id)
	_, _, err := client.Jobs().Deregister(id, false, nil)
	if err != nil {
		return fmt.Errorf("error deregistering job: %s", err)
	}

	return nil
}

func resourceJobRead(d *schema.ResourceData, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	id := d.Id()
	log.Printf("[DEBUG] reading information for job %q", id)
	job, _, err := client.Jobs().Info(id, nil)
	if err != nil {
		// As of Nomad 0.4.1, the API client returns an error for 404
		// rather than a nil result, so we must check this way.
		if strings.Contains(err.Error(), "404") {
			log.Printf("[DEBUG] job %q does not exist, so removing", id)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("error checking for job: %s", err)
	}

	allocStubs, _, err := client.Jobs().Allocations(id, false, nil)
	if err != nil {
		log.Printf("[WARN] error listing allocations for Job %q, will return empty list", id)
	}
	allocIds := make([]string, 0, len(allocStubs))
	for _, a := range allocStubs {
		allocIds = append(allocIds, a.ID)
	}

	d.Set("name", job.ID)
	d.Set("type", job.Type)
	d.Set("region", job.Region)
	d.Set("datacenters", job.Datacenters)
	d.Set("task_groups", jobTaskGroupsRaw(job.TaskGroups))
	d.Set("allocation_ids", allocIds)
	if job.JobModifyIndex != nil {
		d.Set("modify_index", strconv.FormatUint(*job.JobModifyIndex, 10))
	} else {
		d.Set("modify_index", "0")
	}

	return nil
}

func resourceJobCustomizeDiff(d *schema.ResourceDiff, meta interface{}) error {
	providerConfig := meta.(ProviderConfig)
	client := providerConfig.client

	oldSpecRaw, newSpecRaw := d.GetChange("jobspec")
	if oldSpecRaw.(string) == newSpecRaw.(string) {
		// nothing to do!
		return nil
	}

	is_json := d.Get("json").(bool)
	job, err := parseJobspec(newSpecRaw.(string), is_json, providerConfig.vaultToken) // catch syntax errors client-side during plan
	if err != nil {
		return err
	}

	resp, _, err := client.Jobs().PlanOpts(job, &api.PlanOptions{
		Diff:           false,
		PolicyOverride: d.Get("policy_override").(bool),
	}, nil)
	if err != nil {
		return fmt.Errorf("error from 'nomad plan': %s", err)
	}

	// If we were able to successfully plan then we can safely populate our
	// diff with new values based on the job object we got from parsing,
	// causing the Terraform diff to correctly reflect the planned changes
	// to the subset of job attributes we include in our schema.

	d.SetNew("name", job.ID)
	d.SetNew("type", job.Type)
	d.SetNew("region", job.Region)
	d.SetNew("datacenters", job.Datacenters)

	// If the id has changed and the config asks us to deregister on id
	// change then the id field "forces new resource".
	if d.Id() != *job.ID {
		if d.Get("deregister_on_id_change").(bool) {
			log.Printf("[DEBUG] name change forces new resource because deregister_on_id_change is set")
			d.ForceNew("id")
			d.ForceNew("name")
		} else {
			log.Printf("[DEBUG] allowing name change as update because deregister_on_id_change is not set")
		}
	} else {
		// If the id _isn't_ changing, then we require consistency of the
		// job modify index to ensure that the "old" part of our diff
		// will show what Nomad currently knows.
		wantModifyIndexStr := d.Get("modify_index").(string)
		wantModifyIndex, err := strconv.ParseUint(wantModifyIndexStr, 10, 64)
		if err != nil {
			// should never happen, because we always write with FormatUint
			// in Read above.
			return fmt.Errorf("invalid modify_index in state: %s", err)
		}

		if resp.JobModifyIndex != wantModifyIndex {
			// Should rarely happen, but might happen if there was a concurrent
			// other process writing to Nomad since our Read call.
			return fmt.Errorf("job modify index has changed since last refresh")
		}
	}

	// We know that applying changes here will change the modify index
	// _somehow_, but we won't know how much it will increment until
	// after we complete registration.
	d.SetNewComputed("modify_index")
	d.SetNewComputed("allocation_ids")

	d.SetNew("task_groups", jobTaskGroupsRaw(job.TaskGroups))

	return nil
}

func parseJobspec(raw string, is_json bool, vaultToken *string) (*api.Job, error) {
	var job *api.Job
	if is_json {
		err := json.Unmarshal([]byte(raw), &job)
		if err != nil {
			return nil, fmt.Errorf("error parsing jobspec: %s", err)
		}
	} else {
		var err error
		job, err = jobspec.Parse(strings.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("error parsing jobspec: %s", err)
		}
	}

	// Inject the Vault token
	job.VaultToken = vaultToken

	return job, nil
}

func jobTaskGroupsRaw(tgs []*api.TaskGroup) []interface{} {
	ret := make([]interface{}, 0, len(tgs))

	for _, tg := range tgs {
		tgM := make(map[string]interface{})

		if tg.Name != nil {
			tgM["name"] = *tg.Name
		} else {
			tgM["name"] = ""
		}
		if tg.Count != nil {
			tgM["count"] = *tg.Count
		} else {
			tgM["count"] = 1
		}
		if tg.Meta != nil {
			tgM["meta"] = tg.Meta
		} else {
			tgM["meta"] = make(map[string]interface{})
		}

		tasksI := make([]interface{}, 0, len(tg.Tasks))
		for _, task := range tg.Tasks {
			taskM := make(map[string]interface{})

			taskM["name"] = task.Name
			taskM["driver"] = task.Driver
			if task.Meta != nil {
				taskM["meta"] = task.Meta
			} else {
				taskM["meta"] = make(map[string]interface{})
			}

			tasksI = append(tasksI, taskM)
		}
		tgM["task"] = tasksI

		ret = append(ret, tgM)
	}

	return ret
}

// jobspecDiffSuppress is the DiffSuppressFunc used by the schema to
// check if two jobspecs are equal.
func jobspecDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
	// TODO: does this need to consider is_json ???
	// Parse the old job
	oldJob, err := jobspec.Parse(strings.NewReader(old))
	if err != nil {
		return false
	}

	// Parse the new job
	newJob, err := jobspec.Parse(strings.NewReader(new))
	if err != nil {
		return false
	}

	// Init
	oldJob.Canonicalize()
	newJob.Canonicalize()

	// Check for jobspec equality
	return reflect.DeepEqual(oldJob, newJob)
}
