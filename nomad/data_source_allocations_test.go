package nomad

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestDataSourceAllocations_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testDataSourceAllocations_basicConfig(name),
				Check: resource.ComposeTestCheckFunc(
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
		CheckDestroy: testResourceJob_checkDestroy("foo"),
	})
}

func testDataSourceAllocations_basicConfig(prefix string) string {
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

data "nomad_allocations" "all" {
  depends_on = [nomad_job.test]
}

data "nomad_allocations" "by_job" {
  filter = "JobID == \"${nomad_job.test.id}\""
}
`, prefix)
}
