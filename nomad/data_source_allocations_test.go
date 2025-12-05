// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/shoenig/test/must"
	"github.com/shoenig/test/wait"
)

func TestDataSourceAllocations_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testDataSourceAllocations_basicConfig_jobOnly(name),
			},
			{
				Config: testDataSourceAllocations_basicConfig(name),
				Check: resource.ComposeTestCheckFunc(

					testDataSourceAllocations_waitForAllocs(t, 3),

					resource.TestCheckResourceAttrSet("data.nomad_allocations.all", "allocations.#"),
					resource.TestCheckResourceAttr("data.nomad_allocations.by_job", "allocations.#", "3"),
					func(s *terraform.State) error {
						resourceName := "data.nomad_allocations.by_job"
						for i := 0; i < 2; i++ {
							keyPrefix := fmt.Sprintf("allocations.%d", i)
							err := resource.ComposeTestCheckFunc(
								resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("%s.eval_id", keyPrefix)),
								resource.TestMatchResourceAttr(resourceName, fmt.Sprintf("%s.name", keyPrefix),
									regexp.MustCompile(fmt.Sprintf("%s\\.sleep\\[\\d+\\]", name))),
								resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.namespace", keyPrefix), api.DefaultNamespace),
								resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("%s.node_id", keyPrefix)),
								resource.TestCheckResourceAttrSet(resourceName, fmt.Sprintf("%s.node_name", keyPrefix)),
								resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.job_id", keyPrefix), name),
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
		CheckDestroy: testResourceJob_checkDestroy(name),
	})
}

func testDataSourceAllocations_basicConfig_jobOnly(prefix string) string {
	return fmt.Sprintf(`
resource "nomad_job" "test" {
  jobspec = <<EOT
    job "%[1]s" {
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
EOT
}
`, prefix)
}

func testDataSourceAllocations_basicConfig(prefix string) string {
	return fmt.Sprintf(`
%s

data "nomad_allocations" "all" {
  depends_on = [nomad_job.test]
}

data "nomad_allocations" "by_job" {
  filter = "JobID == \"${nomad_job.test.id}\""
}
`, testDataSourceAllocations_basicConfig_jobOnly(prefix))
}

func testDataSourceAllocations_waitForAllocs(t *testing.T, expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		resourceState := s.Modules[0].Resources["nomad_job.test"]
		if resourceState == nil {
			return errors.New("job resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("job resource has no primary instance")
		}

		jobID := instanceState.ID

		ns, ok := instanceState.Attributes["namespace"]
		if !ok {
			return errors.New("resource does not have expected namespace")
		}

		providerConfig := testProvider.Meta().(ProviderConfig)
		client := providerConfig.client

		must.Wait(t, wait.InitialSuccess(
			wait.ErrorFunc(func() error {
				allocs, _, err := client.Jobs().Allocations(
					jobID, true, &api.QueryOptions{Namespace: ns})
				must.NoError(t, err)
				if len(allocs) != expected {
					return fmt.Errorf("expected %d allocs, got %d", expected, len(allocs))
				}
				t.Logf("got 3 allocs")
				return nil
			}),
			wait.Timeout(10*time.Second),
			wait.Gap(100*time.Millisecond),
		))

		return nil
	}
}
