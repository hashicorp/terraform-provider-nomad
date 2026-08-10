// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec2"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// parseHCLJobspec parses a raw Nomad HCL jobspec string and returns a
// canonicalized *api.Job, using the same jobspec2 parser the provider uses at
// runtime.  The path argument is only used for error messages.
func parseHCLJobspec(t *testing.T, raw string) *api.Job {
	t.Helper()
	job, err := jobspec2.ParseWithConfig(&jobspec2.ParseConfig{
		Body:    []byte(raw),
		AllowFS: false,
		Strict:  true,
	})
	if err != nil {
		t.Fatalf("parseHCLJobspec: %s", err)
	}
	job.Canonicalize()
	return job
}

// registerJobViaAPI registers a job directly through the Nomad API and returns
// a cleanup function that deregisters (purges) it.  Call this inside PreCheck
// so the job exists before Terraform's plan/apply runs.
func registerJobViaAPI(t *testing.T, job *api.Job) func() {
	t.Helper()
	client := testProvider.Meta().(ProviderConfig).client
	ns := "default"
	if job.Namespace != nil && *job.Namespace != "" {
		ns = *job.Namespace
	}
	resp, _, err := client.Jobs().Register(job, &api.WriteOptions{Namespace: ns})
	if err != nil {
		t.Fatalf("failed to register job %q via API: %s", *job.ID, err)
	}
	// Wait for the evaluation to complete so the job is visible.
	if resp.EvalID != "" {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			eval, _, err := client.Evaluations().Info(resp.EvalID, &api.QueryOptions{Namespace: ns})
			if err == nil && (eval.Status == "complete" || eval.Status == "failed" || eval.Status == "cancelled") {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
	return func() {
		_, _, _ = client.Jobs().Deregister(*job.ID, true, &api.WriteOptions{Namespace: ns})
	}
}

// createNamespaceViaAPI creates a Nomad namespace directly and returns a cleanup func.
func createNamespaceViaAPI(t *testing.T, ns string) func() {
	t.Helper()
	client := testProvider.Meta().(ProviderConfig).client
	_, err := client.Namespaces().Register(&api.Namespace{Name: ns}, nil)
	if err != nil {
		t.Fatalf("failed to create namespace %q via API: %s", ns, err)
	}
	return func() {
		_, _ = client.Namespaces().Delete(ns, nil)
	}
}

// purgeJobViaAPI returns a TestCheckFunc that purges a job after the test,
// used as CheckDestroy when no Terraform resource manages the job.
func purgeJobViaAPI(t *testing.T, jobID, ns string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		t.Helper()
		client := testProvider.Meta().(ProviderConfig).client
		_, _, _ = client.Jobs().Deregister(jobID, true, &api.WriteOptions{Namespace: ns})
		return nil
	}
}

func TestAccDataSourceNomadJob_Basic(t *testing.T) {
	jobID := "testjobds"
	var cleanup func()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			cleanup = registerJobViaAPI(t, basicServiceJob(t, jobID))
		},
		Providers:    testProviders,
		CheckDestroy: purgeJobViaAPI(t, jobID, "default"),
		Steps: []resource.TestStep{
			{
				Config: testAccJobDataSourceConfig(jobID),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceNomadJobExists("data.nomad_job.test-job", "default"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "name", jobID),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "type", "service"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "priority", "50"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "namespace", "default"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "update_strategy.#", "1"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "update_strategy.0.max_parallel", "2"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "task_groups.0.update_strategy.#", "1"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "task_groups.0.update_strategy.0.max_parallel", "2"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "task_groups.0.update_strategy.0.min_healthy_time", "11s"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "task_groups.0.update_strategy.0.healthy_deadline", "6m0s"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "task_groups.0.update_strategy.0.auto_revert", "true"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "task_groups.0.update_strategy.0.canary", "1"),
				),
			},
		},
	})

	if cleanup != nil {
		cleanup()
	}
}

func TestAccDataSourceNomadJob_Periodic(t *testing.T) {
	jobID := "testjobds_periodic"
	var cleanup func()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			cleanup = registerJobViaAPI(t, periodicBatchJob(t, jobID))
		},
		Providers:    testProviders,
		CheckDestroy: purgeJobViaAPI(t, jobID, "default"),
		Steps: []resource.TestStep{
			{
				Config: testAccJobDataSourceConfigPeriodic(jobID),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceNomadJobExists("data.nomad_job.test-job-periodic", "default"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job-periodic", "name", jobID),
					resource.TestCheckResourceAttr("data.nomad_job.test-job-periodic", "type", "batch"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job-periodic", "priority", "50"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job-periodic", "namespace", "default"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job-periodic", "periodic_config.#", "1"),
				),
			},
		},
	})

	if cleanup != nil {
		cleanup()
	}
}

func TestAccDataSourceNomadJob_Namespaced(t *testing.T) {
	ns := "jobds-test-namespace"
	jobID := "testjobds-namespace"
	var cleanupNS, cleanupJob func()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			cleanupNS = createNamespaceViaAPI(t, ns)
			cleanupJob = registerJobViaAPI(t, namespacedBatchJob(t, jobID, ns))
		},
		Providers:    testProviders,
		CheckDestroy: purgeJobViaAPI(t, jobID, ns),
		Steps: []resource.TestStep{
			{
				Config: testAccJobDataSourceConfigNamespaced(jobID, ns),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceNomadJobExists("data.nomad_job.test-job", ns),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "name", jobID),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "type", "batch"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "priority", "50"),
					resource.TestCheckResourceAttr("data.nomad_job.test-job", "namespace", ns),
				),
			},
		},
	})

	if cleanupJob != nil {
		cleanupJob()
	}
	if cleanupNS != nil {
		cleanupNS()
	}
}

func testAccDataSourceNomadJobExists(n, namespace string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no Job ID is set")
		}

		client := testProvider.Meta().(ProviderConfig).client
		job, _, err := client.Jobs().Info(rs.Primary.ID, &api.QueryOptions{Namespace: namespace})
		if err != nil {
			return err
		}
		if *job.ID != rs.Primary.ID {
			return fmt.Errorf("job not found")
		}
		return nil
	}
}

// HCL jobspec fixtures
// Each fixture is defined as a raw HCL string (matching the same HCL syntax
// used in the resource tests) and parsed via jobspec2 — the same parser the
// provider uses at runtime — so the *api.Job the API receives is always
// semantically identical to what Terraform would submit.

// basicServiceJob parses the HCL for a service job with an update strategy,
// then stamps in the caller-supplied jobID so the same spec can be reused with
// different names across test runs.
func basicServiceJob(t *testing.T, jobID string) *api.Job {
	t.Helper()
	// The update strategy values (max_parallel=2, min_healthy_time=11s,
	// healthy_deadline=6m, auto_revert=true, canary=1) are what the datasource
	// test assertions expect — they match testJobV086Config's group-level update.
	const hcl = `
job "PLACEHOLDER" {
  datacenters = ["dc1"]
  type        = "service"

  update {
    max_parallel      = 2
    min_healthy_time  = "11s"
    healthy_deadline  = "6m"
    progress_deadline = "11m"
    auto_revert       = true
    auto_promote      = true
    canary            = 1
  }

  group "foo" {
    count = 1

    update {
      max_parallel      = 2
      min_healthy_time  = "11s"
      healthy_deadline  = "6m"
      progress_deadline = "11m"
      auto_revert       = true
      auto_promote      = true
      canary            = 1
    }

    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/echo"
        args    = ["test"]
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
	job := parseHCLJobspec(t, strings.ReplaceAll(hcl, "PLACEHOLDER", jobID))
	job.ID = &jobID
	job.Name = &jobID
	return job
}

// periodicBatchJob parses the HCL for a periodic batch job, stamping in jobID.
func periodicBatchJob(t *testing.T, jobID string) *api.Job {
	t.Helper()
	const hcl = `
job "PLACEHOLDER" {
  datacenters = ["dc1"]
  type        = "batch"

  periodic {
    crons            = ["*/15 * * * * *"]
    prohibit_overlap = true
  }

  group "foo" {
    count = 1

    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/echo"
        args    = ["test"]
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
	job := parseHCLJobspec(t, strings.ReplaceAll(hcl, "PLACEHOLDER", jobID))
	job.ID = &jobID
	job.Name = &jobID
	return job
}

// namespacedBatchJob parses the HCL for a simple batch job in a given namespace.
func namespacedBatchJob(t *testing.T, jobID, ns string) *api.Job {
	t.Helper()
	hcl := fmt.Sprintf(`
job %q {
  datacenters = ["dc1"]
  type        = "batch"
  namespace   = %q

  group "foo" {
    count = 1

    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/echo"
        args    = ["test"]
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
`, jobID, ns)
	return parseHCLJobspec(t, hcl)
}

// Terraform config (data source only)

func testAccJobDataSourceConfig(jobID string) string {
	return fmt.Sprintf(`
data "nomad_job" "test-job" {
  job_id = %q
}
`, jobID)
}

func testAccJobDataSourceConfigPeriodic(jobID string) string {
	return fmt.Sprintf(`
data "nomad_job" "test-job-periodic" {
  job_id = %q
}
`, jobID)
}

func testAccJobDataSourceConfigNamespaced(jobID, ns string) string {
	return fmt.Sprintf(`
data "nomad_job" "test-job" {
  job_id    = %q
  namespace = %q
}
`, jobID, ns)
}
