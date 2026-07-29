// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccDataSourceDeployments tests the nomad_deployments data source.
//
// Flow:
//  1. PreCheck registers the job via API so a deployment is created.
//  2. Step 1 reads deployments and asserts ≥1 exist, then deregisters the job
//     via the API so the deployment transitions to "cancelled".
//  3. Step 2 re-reads deployments and asserts the deployment for "foo_deploy"
//     is in "cancelled" status.
func TestAccDataSourceDeployments(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck: func() {
			testAccPreCheck(t)
			registerJobViaAPI(t, deploymentTestJob(t))
		},
		CheckDestroy: testJobForceDestroyWithPurge("foo_deploy", "default"),
		Steps: []resource.TestStep{
			{
				// Step 1: job is running → at least one deployment exists.
				// After checking, deregister the job so its deployment becomes cancelled.
				Config: testAccCheckDataSourceNomadDeploymentsCfg,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs, _ := s.RootModule().Resources["data.nomad_deployments.foobar"]
						is := rs.Primary
						v, ok := is.Attributes["deployments.#"]
						if !ok {
							return fmt.Errorf("attribute 'deployments.#' not found")
						}
						numDeployments, err := strconv.Atoi(v)
						if err != nil {
							return fmt.Errorf("error parsing 'deployments.#': %v", err)
						}
						if numDeployments < 1 {
							return fmt.Errorf("'deployments.#' should be >= 1, got %v", v)
						}
						return nil
					},
					// Deregister the job out-of-band so Nomad cancels the deployment.
					func(*terraform.State) error {
						client := testProvider.Meta().(ProviderConfig).client
						_, _, err := client.Jobs().Deregister("foo_deploy", false, nil)
						if err != nil {
							return fmt.Errorf("failed to deregister job foo_deploy: %s", err)
						}
						return nil
					},
				),
			},
			{
				// Step 2: job is deregistered → deployment should be "cancelled".
				Config: testAccCheckDataSourceNomadDeploymentsCfg,
				Check: func(s *terraform.State) error {
					re := regexp.MustCompile(`^deployments\.(\d+)\.JobID$`)
					rs, _ := s.RootModule().Resources["data.nomad_deployments.foobar"]
					is := rs.Primary
					index := -1
					for k, v := range is.Attributes {
						if submatch := re.FindStringSubmatch(k); submatch != nil && v == "foo_deploy" {
							index, _ = strconv.Atoi(submatch[1])
							break
						}
					}
					if index < 0 {
						return fmt.Errorf("did not find expected deployment for job 'foo_deploy'")
					}
					statusAttr := fmt.Sprintf("deployments.%d.Status", index)
					if s, ok := is.Attributes[statusAttr]; !ok || s != "cancelled" {
						if !ok {
							return fmt.Errorf("did not find expected attribute '%v'", statusAttr)
						}
						return fmt.Errorf("'%v': expected 'cancelled', got '%v'", statusAttr, s)
					}
					return nil
				},
			},
		},
	})
}

// deploymentTestJob parses the HCL for the service job that creates a deployment.
func deploymentTestJob(t *testing.T) *api.Job {
	t.Helper()
	const hcl = `
job "foo_deploy" {
  datacenters = ["dc1"]
  type        = "service"

  update {
    max_parallel = 1
  }

  group "foo" {
    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["1"]
      }

      resources {
        cpu    = 100
        memory = 10
      }

      logs {
        max_files     = 3
        max_file_size = 10
      }
    }
  }
}
`
	return parseHCLJobspec(t, hcl)
}

var testAccCheckDataSourceNomadDeploymentsCfg = `
data "nomad_deployments" "foobar" {}
`
