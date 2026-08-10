// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDataSourceScalingPolicies_Basic(t *testing.T) {
	dataSourceName := "data.nomad_scaling_policies.policies"

	const ossJobID = "foo-scaling-policies"
	const entJobID = "foo-scaling-policies-ent"

	// OSS test: register the job via API in PreCheck.
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck: func() {
			testAccPreCheck(t)
			testCheckMinVersion(t, "0.11.0")
			registerJobViaAPI(t, scalingPoliciesOSSJob(t, ossJobID))
		},
		CheckDestroy: testJobForceDestroyWithPurge(ossJobID, "default"),
		Steps: []resource.TestStep{
			{
				Config: testDataSourceScalingPoliciesConfig(ossJobID, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "policies.#", "1"),
					resource.TestCheckResourceAttrSet(dataSourceName, "policies.0.id"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.enabled", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.type", "horizontal"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Job", ossJobID),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Group", "foo"),
				),
			},
			{
				Config: testDataSourceScalingPoliciesConfig("", "horizontal"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "policies.#", "1"),
					resource.TestCheckResourceAttrSet(dataSourceName, "policies.0.id"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.enabled", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.type", "horizontal"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Job", ossJobID),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Group", "foo"),
				),
			},
		},
	})

	// Ent test: register the Ent job via API in PreCheck.
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck: func() {
			testAccPreCheck(t)
			testCheckEnt(t)
			testCheckMinVersion(t, "1.0.0-beta2")
			registerJobViaAPI(t, scalingPoliciesEntJob(t, entJobID))
		},
		CheckDestroy: testJobForceDestroyWithPurge(entJobID, "default"),
		Steps: []resource.TestStep{
			{
				Config: testDataSourceScalingPoliciesConfig("", ""),
				Check: resource.ComposeTestCheckFunc(
					// We can't guarantee order, so test length only for now.
					resource.TestCheckResourceAttr(dataSourceName, "policies.#", "3"),
				),
			},
			{
				Config: testDataSourceScalingPoliciesConfig(entJobID, ""),
				Check: resource.ComposeTestCheckFunc(
					// We can't guarantee order, so test length only for now.
					resource.TestCheckResourceAttr(dataSourceName, "policies.#", "3"),
				),
			},
			{
				Config: testDataSourceScalingPoliciesConfig("", "vertical"),
				Check: resource.ComposeTestCheckFunc(
					// We can't guarantee order, so test length only for now.
					resource.TestCheckResourceAttr(dataSourceName, "policies.#", "2"),
				),
			},
			{
				Config: testDataSourceScalingPoliciesConfig("", "horizontal"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "policies.#", "1"),
					resource.TestCheckResourceAttrSet(dataSourceName, "policies.0.id"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.enabled", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.type", "horizontal"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Job", entJobID),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Group", "foo"),
				),
			},
			{
				Config: testDataSourceScalingPoliciesConfig("", "vertical_cpu"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "policies.#", "1"),
					resource.TestCheckResourceAttrSet(dataSourceName, "policies.0.id"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.enabled", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.type", "vertical_cpu"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Job", entJobID),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Group", "foo"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Task", "foo"),
				),
			},
			{
				Config: testDataSourceScalingPoliciesConfig("", "vertical_mem"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "policies.#", "1"),
					resource.TestCheckResourceAttrSet(dataSourceName, "policies.0.id"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.enabled", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.type", "vertical_mem"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Job", entJobID),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Group", "foo"),
					resource.TestCheckResourceAttr(dataSourceName, "policies.0.target.Task", "foo"),
				),
			},
		},
	})
}

// API job builders

// scalingPoliciesOSSJob parses the HCL for the OSS horizontal-scaling job.
func scalingPoliciesOSSJob(t *testing.T, jobID string) *api.Job {
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

        check "avg_instance_sessions" {
          source = "prometheus"
          query  = "query"

          strategy "target-value" {
            target = 5
          }
        }
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

// scalingPoliciesEntJob parses the HCL for the Ent job with horizontal +
// vertical_cpu + vertical_mem scaling policies.
func scalingPoliciesEntJob(t *testing.T, jobID string) *api.Job {
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

        check "avg_instance_sessions" {
          source = "prometheus"
          query  = "query"

          strategy "target-value" {
            target = 5
          }
        }
      }
    }

    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["10"]
      }

      scaling "cpu" {
        policy {
          check "check" {
            query = "query"
          }
        }
      }

      scaling "mem" {
        policy {
          check "check" {
            query = "query"
          }
        }
      }
    }
  }
}
`, jobID)
	return parseHCLJobspec(t, hcl)
}

// Terraform config (data sources only)

func testDataSourceScalingPoliciesConfig(jobID string, typeQuery string) string {
	var filters string

	if jobID != "" {
		filters += fmt.Sprintf("job_id = %q\n", jobID)
	}
	if typeQuery != "" {
		filters += fmt.Sprintf("type = %q\n", typeQuery)
	}

	return fmt.Sprintf(`
data "nomad_scaling_policies" "policies" {
%s}
`, filters)
}
