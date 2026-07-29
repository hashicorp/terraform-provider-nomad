// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/shoenig/test/must"
	"github.com/shoenig/test/wait"
)

func TestDataSourceAllocations_basic(t *testing.T) {
	jobID := acctest.RandomWithPrefix("tf-nomad-test")

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck: func() {
			testAccPreCheck(t)
			registerJobViaAPI(t, allocationsTestJob(t, jobID))
		},
		CheckDestroy: testJobForceDestroyWithPurge(jobID, "default"),
		Steps: []resource.TestStep{
			{
				Config: testDataSourceAllocations_basicConfig(jobID),
				Check: resource.ComposeTestCheckFunc(
					testDataSourceAllocations_waitForAllocs(t, jobID, "default", 3),
					resource.TestCheckResourceAttrSet("data.nomad_allocations.all", "allocations.#"),
					resource.TestCheckResourceAttr("data.nomad_allocations.by_job", "allocations.#", "3"),
					func(s *terraform.State) error {
						resourceName := "data.nomad_allocations.by_job"
						for i := 0; i < 2; i++ {
							keyPrefix := fmt.Sprintf("allocations.%d", i)
							err := resource.ComposeTestCheckFunc(
								resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("%s.eval_id", keyPrefix)),
								resource.TestMatchResourceAttr(resourceName, fmt.Sprintf("%s.name", keyPrefix),
									regexp.MustCompile(fmt.Sprintf("%s\\.sleep\\[\\d+\\]", jobID))),
								resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.namespace", keyPrefix), api.DefaultNamespace),
								resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("%s.node_id", keyPrefix)),
								resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("%s.node_name", keyPrefix)),
								resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.job_id", keyPrefix), jobID),
								resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.job_type", keyPrefix), "service"),
								resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.job_version", keyPrefix), "0"),
								resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.task_group", keyPrefix), "sleep"),
								resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.desired_status", keyPrefix), api.AllocDesiredStatusRun),
								resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("%s.client_status", keyPrefix)),
								resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.followup_eval_id", keyPrefix), ""),
								resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.next_allocation", keyPrefix), ""),
								resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.preempted_by_allocation", keyPrefix), ""),
								resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("%s.create_index", keyPrefix)),
								resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("%s.modify_index", keyPrefix)),
								resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("%s.create_time", keyPrefix)),
								resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("%s.modify_time", keyPrefix)),
							)(s)
							if err != nil {
								return err
							}
						}
						return nil
					},
				),
			},
		},
	})
}

// allocationsTestJob parses the HCL for the service job the allocations test needs.
func allocationsTestJob(t *testing.T, jobID string) *api.Job {
	t.Helper()
	hcl := fmt.Sprintf(`
job %q {
  datacenters = ["dc1"]
  type        = "service"

  group "sleep" {
    count = 3

    task "sleep" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["10"]
      }

      resources {
        cpu    = 10
        memory = 10
      }

      logs {
        max_files     = 1
        max_file_size = 1
      }
    }
  }
}
`, jobID)
	return parseHCLJobspec(t, hcl)
}

// testDataSourceAllocations_basicConfig returns the Terraform config with only
// the two data sources — no nomad_job resource; the job is pre-registered via API.
func testDataSourceAllocations_basicConfig(jobID string) string {
	return fmt.Sprintf(`
data "nomad_allocations" "all" {}

data "nomad_allocations" "by_job" {
  filter = "JobID == %q"
}
`, jobID)
}

func testDataSourceAllocations_waitForAllocs(t *testing.T, jobID, ns string, expected int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client

		must.Wait(t, wait.InitialSuccess(
			wait.ErrorFunc(func() error {
				allocs, _, err := client.Jobs().Allocations(
					jobID, true, &api.QueryOptions{Namespace: ns})
				must.NoError(t, err)
				if len(allocs) != expected {
					return fmt.Errorf("expected %d allocs, got %d", expected, len(allocs))
				}
				t.Logf("got %d allocs", expected)
				return nil
			}),
			wait.Timeout(30*time.Second),
			wait.Gap(200*time.Millisecond),
		))

		return nil
	}
}

// testJobForceDestroyWithPurge purges a job unconditionally via the Nomad API.
func testJobForceDestroyWithPurge(jobID, namespace string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client
		_, _, err := client.Jobs().Deregister(jobID, true, &api.WriteOptions{Namespace: namespace})
		if err != nil && !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("failed to purge job %q: %s", jobID, err)
		}
		return nil
	}
}
