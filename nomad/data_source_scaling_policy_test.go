// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDataSourceScalingPolicy_Basic(t *testing.T) {
	dataSourceName := "data.nomad_scaling_policy.policy"
	const jobID = "foo-scaling-policy"

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck: func() {
			testAccPreCheck(t)
			testCheckMinVersion(t, "0.11.0")
			registerJobViaAPI(t, scalingPolicyTestJob(t, jobID))
		},
		CheckDestroy: testJobForceDestroyWithPurge(jobID, "default"),
		Steps: []resource.TestStep{
			{
				Config: testDataSourceScalingPolicyConfig(jobID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "type", "horizontal"),
					resource.TestCheckResourceAttr(dataSourceName, "min", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "max", "20"),
					resource.TestCheckResourceAttr(dataSourceName, "policy", `{"cooldown":"20s"}`),
					resource.TestCheckResourceAttr(dataSourceName, "target.Namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "target.Job", jobID),
					resource.TestCheckResourceAttr(dataSourceName, "target.Group", "foo"),
				),
			},
		},
	})
}

// scalingPolicyTestJob parses the HCL for the job the scaling policy test needs.
func scalingPolicyTestJob(t *testing.T, jobID string) *api.Job {
	t.Helper()
	hcl := fmt.Sprintf(`
job %q {
  datacenters = ["dc1"]

  group "foo" {
    scaling {
      enabled = false
      min     = 1
      max     = 20
      type    = "horizontal"

      policy {
        cooldown = "20s"
      }
    }

    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["10"]
      }
    }
  }
}
`, jobID)
	return parseHCLJobspec(t, hcl)
}

// testDataSourceScalingPolicyConfig returns the Terraform config that reads the
// scaling policy for the given job — no nomad_job resource; the job is pre-registered.
func testDataSourceScalingPolicyConfig(jobID string) string {
	return fmt.Sprintf(`
data "nomad_scaling_policies" "policies" {
  job_id = %q
}

data "nomad_scaling_policy" "policy" {
  id = data.nomad_scaling_policies.policies.policies[0].id
}
`, jobID)
}
