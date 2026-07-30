// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package job_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	r "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-nomad/internal/framework/provider/testutil"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
	"github.com/shoenig/test/must"
	"github.com/stretchr/testify/require"
)

// nomadClient returns a Nomad API client using the configured SDKv2 provider meta.
func nomadClient(t *testing.T) *api.Client {
	t.Helper()
	meta := testutil.SDKV2ProviderMeta(t)()
	return meta.(nomad.ProviderConfig).Client()
}

// stopJobExternally deregisters (stops) a job out-of-band.
func stopJobExternally(t *testing.T, jobID string) func() {
	return func() {
		t.Helper()
		client := nomadClient(t)
		_, _, err := client.Jobs().Deregister(jobID, false, nil)
		require.NoError(t, err, "external job stop failed")
		require.Eventually(t, func() bool {
			job, _, e := client.Jobs().Info(jobID, nil)
			return e == nil && job != nil && job.Status != nil && *job.Status == "dead"
		}, 30*time.Second, time.Second, "job did not reach dead status after external stop")
	}
}

// scaleTaskGroupExternally scales a task group out-of-band.
func scaleTaskGroupExternally(t *testing.T, jobID, groupName string, count int) func() {
	return func() {
		t.Helper()
		client := nomadClient(t)
		resp, _, err := client.Jobs().Scale(jobID, groupName, &count, "terraform acceptance test scale", false, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)

		job, _, err := client.Jobs().Info(jobID, nil)
		require.NoError(t, err)
		namespace := "default"
		if job != nil && job.Namespace != nil && *job.Namespace != "" {
			namespace = *job.Namespace
		}
		if resp.EvalID != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			_, err = waitForEval(ctx, client, namespace, resp.EvalID)
			require.NoError(t, err)
		}
	}
}

// updateTaskResourcesExternally re-registers the job with new task resources out-of-band.
func updateTaskResourcesExternally(t *testing.T, jobID, groupName, taskName string, cpu, memory int) func() {
	return func() {
		t.Helper()
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(jobID, nil)
		must.NoError(t, err)
		must.NotNil(t, job)

		for _, tg := range job.TaskGroups {
			if tg == nil || tg.Name == nil || *tg.Name != groupName {
				continue
			}
			for _, tk := range tg.Tasks {
				if tk.Name != taskName {
					continue
				}
				if tk.Resources == nil {
					tk.Resources = &api.Resources{}
				}
				tk.Resources.CPU = pointerOf(cpu)
				tk.Resources.MemoryMB = pointerOf(memory)
			}
		}

		namespace := "default"
		if job.Namespace != nil && *job.Namespace != "" {
			namespace = *job.Namespace
		}
		resp, _, err := client.Jobs().Register(job, &api.WriteOptions{Namespace: namespace})
		must.NoError(t, err)
		must.NotNil(t, resp)

		if resp.EvalID != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			_, err = waitForEval(ctx, client, namespace, resp.EvalID)
			must.NoError(t, err)
		}
	}
}

// waitForEval waits until an evaluation is complete.
func waitForEval(ctx context.Context, client *api.Client, namespace, evalID string) (*api.Evaluation, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
		eval, _, err := client.Evaluations().Info(evalID, &api.QueryOptions{Namespace: namespace})
		if err != nil {
			return nil, err
		}
		switch eval.Status {
		case "complete":
			return eval, nil
		case "failed", "cancelled":
			return nil, fmt.Errorf("evaluation %s %s: %s", evalID, eval.Status, eval.StatusDescription)
		}
	}
}

// checkNomadJobCount verifies the live task group count directly via the API.
func checkNomadJobCount(t *testing.T, jobID, groupName string, want int) r.TestCheckFunc {
	return func(*terraform.State) error {
		t.Helper()
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(jobID, nil)
		if err != nil {
			return fmt.Errorf("error reading job %q: %s", jobID, err)
		}
		for _, tg := range job.TaskGroups {
			if tg == nil || tg.Name == nil || *tg.Name != groupName {
				continue
			}
			if tg.Count == nil {
				return fmt.Errorf("task group %q has nil count", groupName)
			}
			if got := *tg.Count; got != want {
				return fmt.Errorf("task group %q count = %d, want %d", groupName, got, want)
			}
			return nil
		}
		return fmt.Errorf("task group %q not found in job %q", groupName, jobID)
	}
}

// checkNomadJobResources verifies the live task resources directly via the API.
func checkNomadJobResources(t *testing.T, jobID, groupName, taskName string, wantCPU, wantMemory int) r.TestCheckFunc {
	return func(*terraform.State) error {
		t.Helper()
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(jobID, nil)
		if err != nil {
			return fmt.Errorf("error reading job %q: %s", jobID, err)
		}
		for _, tg := range job.TaskGroups {
			if tg == nil || tg.Name == nil || *tg.Name != groupName {
				continue
			}
			for _, tk := range tg.Tasks {
				if tk.Name != taskName {
					continue
				}
				if tk.Resources == nil {
					return fmt.Errorf("task %q has nil resources", taskName)
				}
				if tk.Resources.CPU == nil || *tk.Resources.CPU != wantCPU {
					return fmt.Errorf("task %q cpu = %v, want %d", taskName, tk.Resources.CPU, wantCPU)
				}
				if tk.Resources.MemoryMB == nil || *tk.Resources.MemoryMB != wantMemory {
					return fmt.Errorf("task %q memory = %v, want %d", taskName, tk.Resources.MemoryMB, wantMemory)
				}
				return nil
			}
		}
		return fmt.Errorf("task %q not found in group %q of job %q", taskName, groupName, jobID)
	}
}

// checkNomadJobDestroyed checks that a job is stopped or absent via the API.
func checkNomadJobDestroyed(t *testing.T, jobID string) r.TestCheckFunc {
	return func(*terraform.State) error {
		t.Helper()
		client := nomadClient(t)
		for i := 0; i < 5; i++ {
			job, _, err := client.Jobs().Info(jobID, nil)
			if err != nil && isNotFound(err) {
				return nil
			}
			if job != nil && job.Status != nil && *job.Status == "dead" {
				return nil
			}
			time.Sleep(time.Second)
		}
		return fmt.Errorf("job %q was not stopped or absent after destroy", jobID)
	}
}

// checkNomadJobDestroyedNS checks that a namespaced job is stopped or absent.
func checkNomadJobDestroyedNS(t *testing.T, jobID, ns string) r.TestCheckFunc {
	return func(*terraform.State) error {
		t.Helper()
		client := nomadClient(t)
		for i := 0; i < 5; i++ {
			job, _, err := client.Jobs().Info(jobID, &api.QueryOptions{Namespace: ns})
			if err != nil && isNotFound(err) {
				return nil
			}
			if job != nil && job.Status != nil && *job.Status == "dead" {
				return nil
			}
			time.Sleep(time.Second)
		}
		return fmt.Errorf("job %q in namespace %q was not stopped or absent after destroy", jobID, ns)
	}
}

// checkNomadJobExistsNS asserts a job exists in a given namespace.
func checkNomadJobExistsNS(t *testing.T, jobID, ns string) r.TestCheckFunc {
	return func(*terraform.State) error {
		t.Helper()
		client := nomadClient(t)
		_, _, err := client.Jobs().Info(jobID, &api.QueryOptions{Namespace: ns})
		if err != nil {
			return fmt.Errorf("error reading back job %q in namespace %q: %s", jobID, ns, err)
		}
		return nil
	}
}

// forceDestroyWithPurge purges a job; used in CheckDestroy after deregister_on_destroy=false tests.
func forceDestroyWithPurge(t *testing.T, jobID, namespace string) r.TestCheckFunc {
	return func(*terraform.State) error {
		t.Helper()
		client := nomadClient(t)
		_, _, err := client.Jobs().Deregister(jobID, true, &api.WriteOptions{Namespace: namespace})
		if err != nil && !isNotFound(err) {
			return fmt.Errorf("failed to clean up job %q after test: %s", jobID, err)
		}
		return nil
	}
}

func isNotFound(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found"))
}

func pointerOf[T any](v T) *T { return &v }

// Tests: drift detection & preserve-*

func TestJobResource_ExternalStop(t *testing.T) {
	jobID := "framework-job-external-stop"
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testJobConfig(jobID, 50),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_job.test", "status", "running"),
					resource.TestCheckResourceAttr("nomad_job.test", "stop", "false"),
				),
			},
			{
				PreConfig:          stopJobExternally(t, jobID),
				Config:             testJobConfig(jobID, 50),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testJobConfig(jobID, 50),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_job.test", "stop", "false"),
					resource.TestCheckResourceAttr("nomad_job.test", "status", "running"),
				),
			},
		},
		CheckDestroy: checkNomadJobDestroyed(t, jobID),
	})
}

func TestJobResource_ExternalScaleDetected(t *testing.T) {
	const jobID = "framework-job-external-scale-detect"
	const groupName = "foo"

	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testPreserveCountsConfig(jobID, false, 50),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_job.test", "task_groups.0.count", "1"),
					checkNomadJobCount(t, jobID, groupName, 1),
				),
			},
			{
				PreConfig:          scaleTaskGroupExternally(t, jobID, groupName, 3),
				Config:             testPreserveCountsConfig(jobID, false, 50),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testPreserveCountsConfig(jobID, false, 50),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_job.test", "task_groups.0.count", "1"),
					checkNomadJobCount(t, jobID, groupName, 1),
				),
			},
		},
		CheckDestroy: checkNomadJobDestroyed(t, jobID),
	})
}

func TestJobResource_PreserveCounts(t *testing.T) {
	const jobID = "framework-job-preserve-counts"
	const groupName = "foo"

	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testPreserveCountsConfig(jobID, false, 50),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_job.test", "task_groups.0.count", "1"),
					checkNomadJobCount(t, jobID, groupName, 1),
				),
			},
			{
				PreConfig: scaleTaskGroupExternally(t, jobID, groupName, 3),
				Config:    testPreserveCountsConfig(jobID, true, 75),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_job.test", "task_groups.0.count", "3"),
					checkNomadJobCount(t, jobID, groupName, 3),
				),
			},
		},
		CheckDestroy: checkNomadJobDestroyed(t, jobID),
	})
}

func TestJobResource_ExternalResourceMutationDetected(t *testing.T) {
	const jobID = "framework-job-resource-detect"
	const groupName = "foo"
	const taskName = "server"

	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testPreserveResourcesConfig(jobID, false, 50),
				Check:  checkNomadJobResources(t, jobID, groupName, taskName, 100, 32),
			},
			{
				PreConfig:          updateTaskResourcesExternally(t, jobID, groupName, taskName, 250, 64),
				Config:             testPreserveResourcesConfig(jobID, false, 50),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testPreserveResourcesConfig(jobID, false, 50),
				Check:  checkNomadJobResources(t, jobID, groupName, taskName, 100, 32),
			},
		},
		CheckDestroy: checkNomadJobDestroyed(t, jobID),
	})
}

func TestJobResource_PreserveResources(t *testing.T) {
	const jobID = "framework-job-preserve-resources"
	const groupName = "foo"
	const taskName = "server"

	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testPreserveResourcesConfig(jobID, false, 50),
				Check:  checkNomadJobResources(t, jobID, groupName, taskName, 100, 32),
			},
			{
				PreConfig: updateTaskResourcesExternally(t, jobID, groupName, taskName, 250, 64),
				Config:    testPreserveResourcesConfig(jobID, true, 75),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_job.test", "priority", "75"),
					checkNomadJobResources(t, jobID, groupName, taskName, 250, 64),
				),
			},
		},
		CheckDestroy: checkNomadJobDestroyed(t, jobID),
	})
}

// Migrated from nomad/resource_job_test.go

func TestJobResource_Basic(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps:                    []r.TestStep{{Config: testJobInitialConfig, Check: checkJobInitial(t)}},
		CheckDestroy:             checkNomadJobDestroyed(t, "foo"),
	})
}

func TestJobResource_Service(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps:                    []r.TestStep{{Config: testJobInitialConfigService, Check: checkJobInitial(t)}},
		CheckDestroy:             checkNomadJobDestroyed(t, "foo-service"),
	})
}

// TestJobResource_Namespace uses the muxed provider so nomad_namespace is available.
func TestJobResource_Namespace(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccMuxedProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps:                    []r.TestStep{{Config: testJobInitialConfigNamespace, Check: checkJobInitialNS(t, "jobresource-test-namespace")}},
		CheckDestroy:             checkNomadJobDestroyedNS(t, "foo", "jobresource-test-namespace"),
	})
}

func TestJobResource_PreserveCountsMigrated(t *testing.T) {
	const jobID = "foo-preserve-counts"
	const groupName = "foo"

	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testPreserveCountsConfig(jobID, false, 50),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_job.test", "task_groups.0.count", "1"),
					resource.TestCheckResourceAttr("nomad_job.test", "priority", "50"),
					checkNomadJobCount(t, jobID, groupName, 1),
				),
			},
			{
				PreConfig: scaleTaskGroupExternally(t, jobID, groupName, 3),
				Config:    testPreserveCountsConfig(jobID, true, 75),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_job.test", "task_groups.0.count", "3"),
					resource.TestCheckResourceAttr("nomad_job.test", "priority", "75"),
					checkNomadJobCount(t, jobID, groupName, 3),
				),
			},
		},
		CheckDestroy: checkNomadJobDestroyed(t, jobID),
	})
}

func TestJobResource_PreserveResourcesMigrated(t *testing.T) {
	const jobID = "foo-preserve-resources"
	const groupName = "foo"
	const taskName = "server"

	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testPreserveResourcesConfig(jobID, false, 50),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_job.test", "priority", "50"),
					checkNomadJobResources(t, jobID, groupName, taskName, 100, 32),
				),
			},
			{
				PreConfig: updateTaskResourcesExternally(t, jobID, groupName, taskName, 250, 64),
				Config:    testPreserveResourcesConfig(jobID, true, 75),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_job.test", "priority", "75"),
					checkNomadJobResources(t, jobID, groupName, taskName, 250, 64),
				),
			},
		},
		CheckDestroy: checkNomadJobDestroyed(t, jobID),
	})
}

func TestJobResource_V086(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps:                    []r.TestStep{{Config: testJobV086Config, Check: checkJobV086(t)}},
		CheckDestroy:             checkNomadJobDestroyed(t, "foov086"),
	})
}

func TestJobResource_V090(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps:                    []r.TestStep{{Config: testJobV090Config, Check: checkJobV090(t)}},
		CheckDestroy:             checkNomadJobDestroyed(t, "foov090"),
	})
}

func TestJobResource_Volumes(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testutil.CheckMinVersion(t, "0.10.0-beta1")
		},
		Steps:        []r.TestStep{{Config: testJobVolumesConfig, Check: checkJobVolumes(t)}},
		CheckDestroy: checkNomadJobDestroyed(t, "foo-volumes"),
	})
}

func TestJobResource_ScalingPolicy(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testutil.CheckMinVersion(t, "0.11.0-beta1")
		},
		Steps:        []r.TestStep{{Config: testJobScalingPolicyConfig, Check: checkJobScalingPolicy(t)}},
		CheckDestroy: checkNomadJobDestroyed(t, "foo-scaling"),
	})

	// Dynamic Application Sizing (Enterprise only).
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testutil.CheckEnt(t)
			testutil.CheckMinVersion(t, "1.0.0-beta2+ent")
		},
		Steps:        []r.TestStep{{Config: testJobScalingPolicyDASConfig, Check: checkJobScalingPolicyDAS(t)}},
		CheckDestroy: checkNomadJobDestroyed(t, "foo-scaling-das"),
	})
}

// TestJobResource_Lifecycle also covers the sidecar task lifecycle stanza.
func TestJobResource_Lifecycle(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testutil.CheckMinVersion(t, "0.11.0-beta1")
		},
		Steps:        []r.TestStep{{Config: testJobLifecycleConfig, Check: checkJobLifecycle(t)}},
		CheckDestroy: checkNomadJobDestroyed(t, "foo-lifecycle"),
	})
}

func TestJobResource_Actions(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testutil.CheckMinVersion(t, "1.7.0")
		},
		Steps:        []r.TestStep{{Config: testJobActionsConfig, Check: checkJobActions(t)}},
		CheckDestroy: checkNomadJobDestroyed(t, "actions"),
	})
}

func TestJobResource_ServiceDeploymentInfo(t *testing.T) {
	t.Skip("This test started failing when running the full suite on Nomad v1.5.1+")
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps:                    []r.TestStep{{Config: testJobServiceDeploymentInfo, Check: checkJobServiceDeploymentInfo(t)}},
		CheckDestroy:             checkNomadJobDestroyed(t, "foo-service-with-deployment"),
	})
}

func TestJobResource_BatchNoDetach(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{{
			Config: testJobBatchNoDetach,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("nomad_job.batch_no_detach", "deployment_id", ""),
				resource.TestCheckResourceAttr("nomad_job.batch_no_detach", "deployment_status", ""),
			),
		}},
		CheckDestroy: checkNomadJobDestroyed(t, "foo-batch"),
	})
}

func TestJobResource_ServiceWithoutDeployment(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{{
			Config: testJobServiceNoDeployment,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("nomad_job.service", "deployment_id", ""),
				resource.TestCheckResourceAttr("nomad_job.service", "deployment_status", ""),
				resource.TestCheckResourceAttr("nomad_job.service", "update_strategy.0.max_parallel", "0"),
			),
		}},
		CheckDestroy: checkNomadJobDestroyed(t, "foo-service-without-deployment"),
	})
}

func TestJobResource_PeriodicConfig(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{{
			Config: testJobPeriodicConfig,
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("nomad_job.periodic", "periodic_config.0.enabled", "true"),
				resource.TestCheckResourceAttr("nomad_job.periodic", "periodic_config.0.spec", "*/1 * * * * *"),
				resource.TestCheckResourceAttr("nomad_job.periodic", "periodic_config.0.prohibit_overlap", "true"),
				resource.TestCheckResourceAttr("nomad_job.periodic", "periodic_config.0.timezone", "UTC"),
			),
		}},
		CheckDestroy: checkNomadJobDestroyed(t, "foo-periodic"),
	})
}

// TestJobResource_Multiregion uses the muxed provider (Enterprise feature check
// calls nomad_sentinel_policy indirectly via the SDKv2 provider).
func TestJobResource_Multiregion(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testutil.CheckMinVersion(t, "0.12.0-beta1")
			testutil.CheckEntFeatures(t, "Multiregion Deployments")
		},
		Steps:        []r.TestStep{{Config: testJobMultiregion, Check: checkJobMultiregion(t)}},
		CheckDestroy: checkNomadJobDestroyed(t, "foo-multiregion"),
	})
}

func TestJobResource_Schedule(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testutil.CheckMinVersion(t, "1.8.0-rc.1")
			testutil.CheckEnt(t)
		},
		Steps:        []r.TestStep{{Config: testJobScheduleBlock, Check: checkJobSchedule(t)}},
		CheckDestroy: checkNomadJobDestroyed(t, "foo-schedule"),
	})
}

func TestJobResource_UI(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testutil.CheckMinVersion(t, "1.8.0-rc.1")
		},
		Steps:        []r.TestStep{{Config: testJobUIBlock, Check: checkJobUI(t)}},
		CheckDestroy: checkNomadJobDestroyed(t, "foo-ui"),
	})
}

func TestJobResource_CSIController(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testutil.CheckMinVersion(t, "0.11.0-beta1")
		},
		Steps:        []r.TestStep{{Config: testJobCSIController, Check: checkJobCSIController(t)}},
		CheckDestroy: checkNomadJobDestroyed(t, "foo-csi-controller"),
	})
}

func TestJobResource_CPUCores(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testutil.CheckMinVersion(t, "1.1.0-beta1")
		},
		Steps: []r.TestStep{{Config: testJobCPUCoresConfig, Check: checkJobCPUCores(t)}},
	})
}

func TestJobResource_JSON(t *testing.T) {
	re := regexp.MustCompile("error parsing jobspec")

	// Invalid JSON inputs must be rejected at plan time.
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{Config: testJobInvalidJSONConfig, ExpectError: re},
			{Config: testJobInvalidJSONConfig_notJobspec, ExpectError: re},
		},
	})

	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps:                    []r.TestStep{{Config: testJobJSONConfigWithRoot, Check: checkJobInitialJSON(t)}},
		CheckDestroy:             checkNomadJobDestroyed(t, "foo-json"),
	})

	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps:                    []r.TestStep{{Config: testJobJSONConfig, Check: checkJobInitialJSON(t)}},
		CheckDestroy:             checkNomadJobDestroyed(t, "foo-json"),
	})
}

func TestJobResource_Refresh(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{Config: testJobInitialConfig, Check: checkJobInitial(t)},
			{PreConfig: deregisterJobExternally(t, "foo"), Config: testJobInitialConfig},
		},
		CheckDestroy: checkNomadJobDestroyed(t, "foo"),
	})
}

func TestJobResource_DisableDestroyDeregister(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{Config: testJobNoDestroy, Check: checkJobInitial(t)},
			{
				Destroy: true,
				Config:  testJobNoDestroy,
				Check: func(*terraform.State) error {
					client := nomadClient(t)
					job, _, err := client.Jobs().Info("foo-nodestroy", nil)
					if err != nil {
						return err
					}
					if job.Stop != nil && *job.Stop {
						return fmt.Errorf("job was unexpectedly stopped")
					}
					return nil
				},
			},
		},
		CheckDestroy: forceDestroyWithPurge(t, "foo-nodestroy", "default"),
	})
}

func TestJobResource_Rename(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{Config: testJobInitialConfig, Check: checkJobInitial(t)},
			{
				Config: testJobRenameConfig,
				Check: resource.ComposeTestCheckFunc(
					checkNomadJobDestroyed(t, "foo"),
					checkNomadJobExistsNS(t, "bar", "default"),
				),
			},
		},
		CheckDestroy: checkNomadJobDestroyed(t, "bar"),
	})
}

// TestJobResource_ChangeNamespace uses the muxed provider so nomad_namespace is available.
func TestJobResource_ChangeNamespace(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccMuxedProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{Config: testJobInitialConfigNamespace, Check: checkJobInitialNS(t, "jobresource-test-namespace")},
			{
				Config: testJobChangeNamespaceConfig,
				Check: resource.ComposeTestCheckFunc(
					checkNomadJobDestroyedNS(t, "foo", "jobresource-test-namespace"),
					checkNomadJobExistsNS(t, "foo", "jobresource-updated-namespace"),
				),
			},
		},
		CheckDestroy: resource.ComposeTestCheckFunc(
			checkNomadJobDestroyedNS(t, "foo", "jobresource-test-namespace"),
			checkNomadJobDestroyedNS(t, "foo", "jobresource-updated-namespace"),
		),
	})
}

// TestJobResource_PolicyOverride uses the muxed provider so nomad_sentinel_policy is available.
func TestJobResource_PolicyOverride(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccMuxedProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testutil.CheckEnt(t)
		},
		Steps:        []r.TestStep{{Config: testJobPolicyOverrideConfig(), Check: checkJobInitial(t)}},
		CheckDestroy: checkNomadJobDestroyed(t, "foo"),
	})
}

func TestJobResource_ParameterizedJob(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps:                    []r.TestStep{{Config: testJobParameterizedJob, Check: checkJobParameterized(t)}},
	})
}

func TestJobResource_PurgeOnDestroy(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{Config: testJobPurgeOnDestroy, Check: checkJobInitial(t)},
			{
				Destroy: true,
				Config:  testJobPurgeOnDestroy,
				Check: func(*terraform.State) error {
					client := nomadClient(t)
					job, _, err := client.Jobs().Info("purge-test", nil)
					if err == nil {
						return fmt.Errorf("expected job to be purged; found: %#v", job)
					}
					if !isNotFound(err) {
						return fmt.Errorf("unexpected error checking purged job: %s", err)
					}
					return nil
				},
			},
		},
	})
}

func TestJobResource_HCL2(t *testing.T) {
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testutil.CheckMinVersion(t, "1.0.0")
		},
		Steps: []r.TestStep{
			{Config: testJobHCL2NoFS, ExpectError: regexp.MustCompile("filesystem function disabled")},
			{Config: testJobHCL2, Check: checkJobHCL2(t)},
		},
		CheckDestroy: checkNomadJobDestroyed(t, "foo-hcl2"),
	})
}

// TestJobResource_RerunIfDead covers the rerun_if_dead attribute behaviour.
func TestJobResource_RerunIfDead(t *testing.T) {
	jobID := "rerun-if-dead"
	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{Config: testJobRerunIfDead(jobID, false), Check: checkJobInitial(t)},
			{PreConfig: deregisterJobExternally(t, jobID), RefreshState: true, ExpectNonEmptyPlan: false},
			{Config: testJobRerunIfDead(jobID, false), Check: checkJobStatus(t, "dead")},
			{Config: testJobRerunIfDead(jobID, true), Check: checkJobStatus(t, "running")},
			{PreConfig: deregisterJobExternally(t, jobID), RefreshState: true, ExpectNonEmptyPlan: true},
			{Config: testJobRerunIfDead(jobID, true), Check: checkJobStatus(t, "running")},
		},
		CheckDestroy: checkNomadJobDestroyed(t, jobID),
	})
}

// TestJobResource_OutOfBandConstraintChange verifies that an out-of-band mutation
// to a task constraint (with no job submission record available) is detected as
// drift and corrected on the next apply.
func TestJobResource_OutOfBandConstraintChange(t *testing.T) {
	const jobID = "framework-job-constraint-drift"

	r.Test(t, r.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testJobWithConstraint(jobID),
				Check: func(s *terraform.State) error {
					client := nomadClient(t)
					job, _, err := client.Jobs().Info(jobID, nil)
					if err != nil {
						return fmt.Errorf("error reading job: %s", err)
					}
					if len(job.Constraints) == 0 {
						return fmt.Errorf("expected at least one constraint, got none")
					}
					return nil
				},
			},
			{
				PreConfig:          mutateJobConstraintExternally(t, jobID),
				Config:             testJobWithConstraint(jobID),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testJobWithConstraint(jobID),
				Check: func(s *terraform.State) error {
					client := nomadClient(t)
					job, _, err := client.Jobs().Info(jobID, nil)
					if err != nil {
						return fmt.Errorf("error reading job: %s", err)
					}
					for _, c := range job.Constraints {
						if c.LTarget == "${attr.kernel.name}" && c.RTarget == "linux" {
							return nil
						}
					}
					return fmt.Errorf("constraint ${attr.kernel.name}=linux not restored after apply")
				},
			},
		},
		CheckDestroy: checkNomadJobDestroyed(t, jobID),
	})
}

// Check helpers

func checkJobInitial(t *testing.T) r.TestCheckFunc {
	return checkJobInitialNS(t, "default")
}

func checkJobInitialNS(t *testing.T, expectedNamespace string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.test"]
		if rs == nil {
			return errors.New("resource nomad_job.test not found in state")
		}
		is := rs.Primary
		if is == nil {
			return errors.New("resource has no primary instance")
		}
		if ns := is.Attributes["namespace"]; ns != expectedNamespace {
			return fmt.Errorf("namespace is %q; want %q", ns, expectedNamespace)
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(is.ID, &api.QueryOptions{Namespace: expectedNamespace})
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		if *job.ID != is.ID {
			return fmt.Errorf("jobID is %q; want %q", *job.ID, is.ID)
		}
		if *job.Namespace != expectedNamespace {
			return fmt.Errorf("job namespace is %q; want %q", *job.Namespace, expectedNamespace)
		}
		return nil
	}
}

// checkJobInitialJSON is a lighter check for JSON-jobspec tests (no namespace attr present).
func checkJobInitialJSON(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.test"]
		if rs == nil {
			return errors.New("resource nomad_job.test not found in state")
		}
		if rs.Primary == nil {
			return errors.New("resource has no primary instance")
		}
		client := nomadClient(t)
		_, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		return nil
	}
}

func checkJobV086(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.test"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		if len(job.TaskGroups) != 1 {
			return fmt.Errorf("expected a single TaskGroup")
		}
		tg := job.TaskGroups[0]

		expUpdate := api.UpdateStrategy{}
		json.Unmarshal([]byte(`{"Stagger":30000000000,"MaxParallel":2,"HealthCheck":"checks","MinHealthyTime":12000000000,"HealthyDeadline":360000000000,"ProgressDeadline":720000000000,"AutoRevert":true,"AutoPromote":false,"Canary":1}`), &expUpdate)
		if !reflect.DeepEqual(tg.Update, &expUpdate) {
			return fmt.Errorf("job update strategy not as expected: %+v", tg.Update)
		}
		expMigrate := api.MigrateStrategy{}
		json.Unmarshal([]byte(`{"MaxParallel":2,"HealthCheck":"checks","MinHealthyTime":12000000000,"HealthyDeadline":360000000000}`), &expMigrate)
		if !reflect.DeepEqual(tg.Migrate, &expMigrate) {
			return fmt.Errorf("job migrate strategy not as expected: %+v", tg.Migrate)
		}
		expReschedule := api.ReschedulePolicy{}
		json.Unmarshal([]byte(`{"Attempts":0,"Interval":7200000000000,"Delay":12000000000,"DelayFunction":"exponential","MaxDelay":100000000000,"Unlimited":true}`), &expReschedule)
		if !reflect.DeepEqual(tg.ReschedulePolicy, &expReschedule) {
			return fmt.Errorf("job reschedule policy not as expected: %+v", tg.ReschedulePolicy)
		}
		if len(tg.Tasks) != 1 {
			return fmt.Errorf("expected a single task")
		}
		if len(tg.Tasks[0].Services) != 1 || !reflect.DeepEqual(tg.Tasks[0].Services[0].CanaryTags, []string{"canary-tag-a"}) {
			return fmt.Errorf("expected canary tags [canary-tag-a], got %v", tg.Tasks[0].Services[0].CanaryTags)
		}
		return nil
	}
}

func checkJobV090(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.test"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		expAffinities := []*api.Affinity{}
		json.Unmarshal([]byte(`[{"LTarget":"${node.datacenter}","Operand":"=","RTarget":"dc1","Weight":50},{"LTarget":"${meta.tag}","Operand":"=","RTarget":"foo","Weight":50}]`), &expAffinities)
		if !reflect.DeepEqual(job.Affinities, expAffinities) {
			return fmt.Errorf("job affinities not as expected: %+v", job.Affinities)
		}
		expSpreads := []*api.Spread{}
		json.Unmarshal([]byte(`[{"Attribute":"${node.datacenter}","SpreadTarget":[{"Percent":35,"Value":"dc1"},{"Percent":65,"Value":"dc2"}],"Weight":80}]`), &expSpreads)
		if !reflect.DeepEqual(job.Spreads, expSpreads) {
			return fmt.Errorf("job spreads not as expected: %+v", job.Spreads)
		}
		if exp := job.TaskGroups[0].Update.AutoPromote; exp == nil || !*exp {
			return fmt.Errorf("group auto_promote not as expected: %v", exp)
		}
		return nil
	}
}

func checkJobVolumes(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.test"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		var tg *api.TaskGroup
		for _, g := range job.TaskGroups {
			if *g.Name == "foo" {
				tg = g
				break
			}
		}
		if tg == nil {
			return fmt.Errorf("task group foo not found")
		}
		expVolumes := map[string]*api.VolumeRequest{}
		json.Unmarshal([]byte(`{"data":{"Name":"data","Type":"host","ReadOnly":true,"Source":"data"}}`), &expVolumes)
		if diff := cmp.Diff(expVolumes, tg.Volumes); diff != "" {
			return fmt.Errorf("task group volume mismatch (-want +got):\n%s", diff)
		}
		var task *api.Task
		for _, tk := range tg.Tasks {
			if tk.Name == "foo" {
				task = tk
				break
			}
		}
		if task == nil {
			return fmt.Errorf("task foo not found")
		}
		expMounts := []*api.VolumeMount{}
		json.Unmarshal([]byte(`[{"Volume":"data","Destination":"/var/lib/data","ReadOnly":true,"PropagationMode":"private","SELinuxLabel":""}]`), &expMounts)
		if diff := cmp.Diff(expMounts, task.VolumeMounts); diff != "" {
			return fmt.Errorf("task volume mount mismatch (-want +got):\n%s", diff)
		}
		return nil
	}
}

func checkJobScalingPolicy(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.test"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		var tg *api.TaskGroup
		for _, g := range job.TaskGroups {
			if *g.Name == "foo" {
				tg = g
				break
			}
		}
		if tg == nil || tg.Scaling == nil {
			return fmt.Errorf("task group foo with scaling not found")
		}
		if tg.Scaling.Type != "horizontal" {
			return fmt.Errorf("scaling type = %q, want horizontal", tg.Scaling.Type)
		}
		wantPolicy := map[string]interface{}{"opaque": true}
		if diff := cmp.Diff(wantPolicy, tg.Scaling.Policy); diff != "" {
			return fmt.Errorf("scaling policy content mismatch (-want +got):\n%s", diff)
		}
		return nil
	}
}

func checkJobScalingPolicyDAS(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.test_das"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		var tg *api.TaskGroup
		for _, g := range job.TaskGroups {
			if *g.Name == "foo" {
				tg = g
				break
			}
		}
		if tg == nil {
			return fmt.Errorf("task group foo not found")
		}
		var task *api.Task
		for _, tk := range tg.Tasks {
			if tk.Name == "foo" {
				task = tk
				break
			}
		}
		if task == nil {
			return fmt.Errorf("task foo not found")
		}
		for _, p := range task.ScalingPolicies {
			if p.Type == "vertical_cpu" {
				return nil
			}
		}
		return fmt.Errorf("vertical_cpu scaling policy not found")
	}
}

// checkJobLifecycle verifies the sidecar task lifecycle stanza is registered correctly.
func checkJobLifecycle(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.test"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		var tg *api.TaskGroup
		for _, g := range job.TaskGroups {
			if *g.Name == "foo" {
				tg = g
				break
			}
		}
		if tg == nil {
			return fmt.Errorf("task group foo not found")
		}
		expLC := api.TaskLifecycle{}
		json.Unmarshal([]byte(`{"Hook":"prestart","Sidecar":true}`), &expLC)
		expRestart := api.RestartPolicy{}
		json.Unmarshal([]byte(`{"Interval":600000000000,"Delay":15000000000,"Mode":"delay","Attempts":10,"RenderTemplates":false}`), &expRestart)
		if diff := cmp.Diff(expLC, *tg.Tasks[0].Lifecycle); diff != "" {
			return fmt.Errorf("task lifecycle mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(expRestart, *tg.Tasks[0].RestartPolicy); diff != "" {
			return fmt.Errorf("task restart policy mismatch (-want +got):\n%s", diff)
		}
		return nil
	}
}

func checkJobActions(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.test"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		if len(job.TaskGroups) != 1 {
			return fmt.Errorf("expected 1 group, got %d", len(job.TaskGroups))
		}
		task := job.TaskGroups[0].Tasks[0]
		expected := []*api.Action{{Name: "echo", Command: "/bin/echo", Args: []string{"hi"}}}
		if diff := cmp.Diff(expected, task.Actions); diff != "" {
			return fmt.Errorf("task actions mismatch (-want +got):\n%s", diff)
		}
		return nil
	}
}

func checkJobServiceDeploymentInfo(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.service"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		is := rs.Primary
		client := nomadClient(t)
		deployment, _, err := client.Jobs().LatestDeployment(is.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading deployment: %s", err)
		}
		if deployment == nil {
			return fmt.Errorf("missing latest deployment")
		}
		if is.Attributes["deployment_id"] != deployment.ID {
			return fmt.Errorf("deployment_id is %q; want %q", is.Attributes["deployment_id"], deployment.ID)
		}
		if is.Attributes["deployment_status"] != deployment.Status {
			return fmt.Errorf("deployment_status is %q; want %q", is.Attributes["deployment_status"], deployment.Status)
		}
		return nil
	}
}

func checkJobCSIController(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.test"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		var tg *api.TaskGroup
		for _, g := range job.TaskGroups {
			if *g.Name == "foo-controller" {
				tg = g
				break
			}
		}
		if tg == nil || tg.Tasks[0].CSIPluginConfig == nil {
			return fmt.Errorf("task group foo-controller or CSIPluginConfig not found")
		}
		exp := api.TaskCSIPluginConfig{
			ID:                  "aws-ebs0",
			Type:                "controller",
			MountDir:            "/csi",
			StagePublishBaseDir: "/local/csi",
			HealthTimeout:       30 * time.Second,
		}
		if diff := cmp.Diff(exp, *tg.Tasks[0].CSIPluginConfig); diff != "" {
			return fmt.Errorf("CSI plugin config mismatch (-want +got):\n%s", diff)
		}
		return nil
	}
}

func checkJobCPUCores(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.test_cpu_cores"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		if len(job.TaskGroups) != 1 || len(job.TaskGroups[0].Tasks) != 1 {
			return fmt.Errorf("unexpected job shape")
		}
		task := job.TaskGroups[0].Tasks[0]
		if task.Resources.Cores == nil || *task.Resources.Cores != 1 {
			return fmt.Errorf("expected 1 core, got %v", task.Resources.Cores)
		}
		return nil
	}
}

func checkJobMultiregion(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.multiregion"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		if job.Multiregion == nil {
			return fmt.Errorf("multiregion config not found")
		}
		return nil
	}
}

func checkJobSchedule(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.schedule"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		if len(job.TaskGroups) != 1 || len(job.TaskGroups[0].Tasks) != 1 {
			return fmt.Errorf("unexpected job shape")
		}
		if job.TaskGroups[0].Tasks[0].Schedule == nil {
			return fmt.Errorf("schedule config not found")
		}
		return nil
	}
}

func checkJobUI(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.ui"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		if job.UI == nil {
			return fmt.Errorf("UI config not found")
		}
		return nil
	}
}

func checkJobParameterized(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.parameterized"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		if *job.ID != rs.Primary.ID {
			return fmt.Errorf("jobID is %q; want %q", *job.ID, rs.Primary.ID)
		}
		return nil
	}
}

func checkJobHCL2(t *testing.T) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.hcl2"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		is := rs.Primary
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(is.ID, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		if diff := cmp.Diff([]string{"dc1", "dc2"}, job.Datacenters); diff != "" {
			return fmt.Errorf("datacenters mismatch (-want +got):\n%s", diff)
		}
		if len(job.TaskGroups) != 1 || len(job.TaskGroups[0].Tasks) != 1 {
			return fmt.Errorf("unexpected job shape")
		}
		if got, want := *job.TaskGroups[0].RestartPolicy.Attempts, 5; got != want {
			return fmt.Errorf("restart attempts = %d; want %d", got, want)
		}
		task := job.TaskGroups[0].Tasks[0]
		if len(task.Templates) != 1 || task.Templates[0].EmbeddedTmpl == nil {
			return fmt.Errorf("expected 1 template with content")
		}
		// Content matches the literal inlined in testJobHCL2.
		const wantTmpl = "Hello :)\n"
		if diff := cmp.Diff(wantTmpl, *task.Templates[0].EmbeddedTmpl); diff != "" {
			return fmt.Errorf("template content mismatch (-want +got):\n%s", diff)
		}
		return nil
	}
}

func checkJobStatus(t *testing.T, status string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		t.Helper()
		rs := s.Modules[0].Resources["nomad_job.test"]
		if rs == nil {
			return errors.New("resource not found in state")
		}
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(rs.Primary.ID, &api.QueryOptions{
			Namespace: rs.Primary.Attributes["namespace"],
		})
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		if *job.Status != status {
			return fmt.Errorf("job status is %q, want %q", *job.Status, status)
		}
		return nil
	}
}

// PreConfig helpers

func deregisterJobExternally(t *testing.T, jobID string) func() {
	return func() {
		t.Helper()
		client := nomadClient(t)
		_, _, err := client.Jobs().Deregister(jobID, false, nil)
		if err != nil {
			t.Fatalf("error deregistering job %q: %s", jobID, err)
		}
	}
}

// mutateJobConstraintExternally re-registers the job with a mutated constraint
// value, simulating an out-of-band change with no submission record.
func mutateJobConstraintExternally(t *testing.T, jobID string) func() {
	return func() {
		t.Helper()
		client := nomadClient(t)
		job, _, err := client.Jobs().Info(jobID, nil)
		must.NoError(t, err)
		must.NotNil(t, job)

		for _, c := range job.Constraints {
			if c.LTarget == "${attr.kernel.name}" {
				c.RTarget = "windows"
			}
		}
		namespace := "default"
		if job.Namespace != nil && *job.Namespace != "" {
			namespace = *job.Namespace
		}
		resp, _, err := client.Jobs().Register(job, &api.WriteOptions{Namespace: namespace})
		must.NoError(t, err)
		must.NotNil(t, resp)
		if resp.EvalID != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			_, err = waitForEval(ctx, client, namespace, resp.EvalID)
			must.NoError(t, err)
		}
	}
}

// Terraform config generators

// testJobConfig renders a minimal service job; used by ExternalStop tests.
func testJobConfig(jobID string, priority int) string {
	return fmt.Sprintf(`
resource "nomad_job" "test" {
  jobspec = <<EOT
job %q {
  datacenters = ["dc1"]
  type        = "service"
  priority    = %d

  group "foo" {
    count = 1

    task "server" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["300"]
      }

      resources {
        cpu    = 100
        memory = 32
      }
    }
  }
}
EOT
  detach = false
}
`, jobID, priority)
}

func testPreserveCountsConfig(jobID string, preserveCounts bool, priority int) string {
	return fmt.Sprintf(`
resource "nomad_job" "test" {
  jobspec = <<EOT
job %q {
  datacenters = ["dc1"]
  type        = "service"
  priority    = %d

  group "foo" {
    count = 1

    task "server" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["60"]
      }

      resources {
        cpu    = 100
        memory = 32
      }
    }
  }
}
EOT
  detach          = false
  preserve_counts = %t
}
`, jobID, priority, preserveCounts)
}

func testPreserveResourcesConfig(jobID string, preserveResources bool, priority int) string {
	return fmt.Sprintf(`
resource "nomad_job" "test" {
  jobspec = <<EOT
job %q {
  datacenters = ["dc1"]
  type        = "service"
  priority    = %d

  group "foo" {
    count = 1

    task "server" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["60"]
      }

      resources {
        cpu    = 100
        memory = 32
      }
    }
  }
}
EOT
  detach             = false
  preserve_resources = %t
}
`, jobID, priority, preserveResources)
}

func testJobConfigCompact(jobID string) string {
	return fmt.Sprintf(`
resource "nomad_job" "test" {
  jobspec = <<EOT
job %q {
  datacenters = ["dc1"]
  type        = "batch"

  group "foo" {
    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/true"
      }

      resources {
        cpu    = 100
        memory = 32
      }
    }
  }
}
EOT
}
`, jobID)
}

func testJobConfigSpacious(jobID string) string {
	return fmt.Sprintf(`
resource "nomad_job" "test" {
  # extra whitespace and a comment — must not cause a diff
  jobspec = <<EOT
job %q {
  datacenters = ["dc1"]
  type        = "batch"


  group "foo" {

    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/true"
      }

      resources {
        cpu    = 100
        memory = 32
      }
    }
  }
}
EOT
}
`, jobID)
}

func testJobRerunIfDead(name string, rerunIfDead bool) string {
	return fmt.Sprintf(`
resource "nomad_job" "test" {
  jobspec = <<EOT
job %q {
  group "foo" {
    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["300"]
      }
    }
  }
}
EOT
  detach        = false
  rerun_if_dead = %t
}
`, name, rerunIfDead)
}

func testJobWithConstraint(jobID string) string {
	return fmt.Sprintf(`
resource "nomad_job" "test" {
  jobspec = <<EOT
job %q {
  datacenters = ["dc1"]
  type        = "batch"

  constraint {
    attribute = "$${attr.kernel.name}"
    value     = "linux"
  }

  group "foo" {
    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/true"
      }

      resources {
        cpu    = 100
        memory = 32
      }
    }
  }
}
EOT
  detach = false
}
`, jobID)
}

func testJobPolicyOverrideConfig() string {
	return fmt.Sprintf(`
resource "nomad_sentinel_policy" "policy" {
  name              = %q
  policy            = "main = rule { false }"
  scope             = "submit-job"
  enforcement_level = "soft-mandatory"
  description       = "Fail all jobs for testing policy overrides in terraform acctests"
}

resource "nomad_job" "test" {
  depends_on      = [nomad_sentinel_policy.policy]
  policy_override = true

  jobspec = <<EOT
job "foo" {
  datacenters = ["dc1"]
  type        = "service"

  group "foo" {
    task "foo" {
      leader = true

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
EOT
}
`, acctest.RandomWithPrefix("tf-nomad-test"))
}

// ── Static config vars ────────────────────────────────────────────────────────

var testJobInitialConfig = `
resource "nomad_job" "test" {
  jobspec = <<EOT
job "foo" {
  datacenters = ["dc1"]
  type        = "service"

  group "foo" {
    task "foo" {
      leader = true

      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["10"]
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
EOT
}
`

var testJobInitialConfigNamespace = `
resource "nomad_namespace" "test-namespace" {
  name = "jobresource-test-namespace"
}

resource "nomad_job" "test" {
  jobspec = <<EOT
job "foo" {
  datacenters = ["dc1"]
  type        = "batch"
  namespace   = "${nomad_namespace.test-namespace.name}"

  group "foo" {
    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["10"]
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
EOT
}
`

var testJobInitialConfigService = `
resource "nomad_job" "test" {
  jobspec = <<EOT
job "foo-service" {
  datacenters = ["dc1"]
  type        = "service"

  group "foo" {
    service {
      name         = "foo-service"
      port         = "8080"
      address_mode = "host"
      tags         = ["foor", "test", "tf"]
      canary_tags  = ["canary"]
      enable_tag_override = false

      meta {
        key = "value"
      }

      canary_meta {
        canary = "true"
      }

      check {
        type     = "http"
        interval = "10s"
        timeout  = "2s"
        method   = "GET"
        path     = "/health"
        protocol = "https"
        tls_skip_verify = true

        header {
          Authorization = ["Basic ZWxhc3RpYzpjaGFuZ2VtZQ=="]
        }
      }
    }

    task "foo" {
      leader = true

      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["10"]
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
EOT
}
`

var testJobChangeNamespaceConfig = `
resource "nomad_namespace" "test-namespace" {
  name = "jobresource-test-namespace"
}

resource "nomad_namespace" "new-namespace" {
  name = "jobresource-updated-namespace"
}

resource "nomad_job" "test" {
  jobspec = <<EOT
job "foo" {
  datacenters = ["dc1"]
  type        = "batch"
  namespace   = "${nomad_namespace.new-namespace.name}"

  group "foo" {
    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["10"]
      }

      resources {
        cpu    = 100
        memory = 10
      }
    }
  }
}
EOT
}
`

var testJobRenameConfig = `
resource "nomad_job" "test" {
  jobspec = <<EOT
job "bar" {
  datacenters = ["dc1"]
  type        = "service"

  group "foo" {
    task "foo" {
      leader = true

      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["1"]
      }

      resources {
        cpu    = 100
        memory = 10
      }
    }
  }
}
EOT
}
`

var testJobNoDestroy = `
resource "nomad_job" "test" {
  deregister_on_destroy = false

  jobspec = <<EOT
job "foo-nodestroy" {
  datacenters = ["dc1"]
  type        = "service"

  group "foo" {
    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["30"]
      }

      resources {
        cpu    = 100
        memory = 10
      }
    }
  }
}
EOT
}
`

var testJobPurgeOnDestroy = `
resource "nomad_job" "test" {
  purge_on_destroy = true

  jobspec = <<EOT
job "purge-test" {
  datacenters = ["dc1"]
  type        = "service"

  group "foo" {
    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["30"]
      }

      resources {
        cpu    = 100
        memory = 10
      }
    }
  }
}
EOT
}
`

var testJobInvalidJSONConfig = `
resource "nomad_job" "test" {
  json    = true
  jobspec = "not json"
}
`

var testJobInvalidJSONConfig_notJobspec = `
resource "nomad_job" "test" {
  json    = true
  jobspec = <<EOT
{"not":"job"}
EOT
}
`

// JSON configs use detach=false so deployment_id/deployment_status are fully resolved.
var testJobJSONConfigWithRoot = `
resource "nomad_job" "test" {
  json   = true
  detach = false

  jobspec = <<EOT
{
  "Job": {
    "Datacenters": ["dc1"],
    "ID":   "foo-json",
    "Name": "foo-json",
    "Type": "service",
    "TaskGroups": [{
      "Name": "foo",
      "Tasks": [{
        "Name":      "foo",
        "Driver":    "raw_exec",
        "Leader":    true,
        "Config":    {"command": "/bin/sleep", "args": ["1"]},
        "LogConfig": {"MaxFileSizeMB": 10, "MaxFiles": 3},
        "Resources": {"CPU": 100, "MemoryMB": 10}
      }]
    }]
  }
}
EOT
}
`

var testJobJSONConfig = `
resource "nomad_job" "test" {
  json   = true
  detach = false

  jobspec = <<EOT
{
  "Datacenters": ["dc1"],
  "ID":   "foo-json",
  "Name": "foo-json",
  "Type": "service",
  "TaskGroups": [{
    "Name": "foo",
    "Tasks": [{
      "Name":      "foo",
      "Driver":    "raw_exec",
      "Leader":    true,
      "Config":    {"command": "/bin/sleep", "args": ["1"]},
      "LogConfig": {"MaxFileSizeMB": 10, "MaxFiles": 3},
      "Resources": {"CPU": 100, "MemoryMB": 10}
    }]
  }]
}
EOT
}
`

var testJobV086Config = `
resource "nomad_job" "test" {
  jobspec = <<EOT
job "foov086" {
  datacenters = ["dc1"]
  type        = "service"

  migrate {
    max_parallel     = 2
    health_check     = "checks"
    min_healthy_time = "11s"
    healthy_deadline = "6m"
  }

  update {
    max_parallel      = 2
    min_healthy_time  = "11s"
    healthy_deadline  = "6m"
    progress_deadline = "11m"
    auto_revert       = true
    canary            = 1
  }

  reschedule {
    attempts       = 11
    interval       = "2h"
    delay          = "11s"
    delay_function = "exponential"
    max_delay      = "100s"
    unlimited      = false
  }

  group "foo" {
    migrate {
      min_healthy_time = "12s"
    }

    update {
      min_healthy_time  = "12s"
      progress_deadline = "12m"
    }

    reschedule {
      attempts  = 0
      delay     = "12s"
      unlimited = true
    }

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

      service {
        canary_tags = ["canary-tag-a"]
      }

      logs {
        max_files     = 3
        max_file_size = 10
      }
    }
  }
}
EOT
}
`

var testJobV090Config = `
resource "nomad_job" "test" {
  jobspec = <<EOT
job "foov090" {
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

  affinity {
    attribute = "$${node.datacenter}"
    value     = "dc1"
    weight    = 50
  }

  affinity {
    attribute = "$${meta.tag}"
    value     = "foo"
    weight    = 50
  }

  spread {
    attribute = "$${node.datacenter}"
    weight    = 80

    target "dc1" {
      percent = 35
    }

    target "dc2" {
      percent = 65
    }
  }

  group "foo" {
    update {
      min_healthy_time  = "12s"
      progress_deadline = "12m"
    }

    reschedule {
      attempts  = 0
      delay     = "12s"
      unlimited = true
    }

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

      service {
        canary_tags = ["canary-tag-a"]
      }

      logs {
        max_files     = 3
        max_file_size = 10
      }
    }
  }
}
EOT
}
`

var testJobVolumesConfig = `
resource "nomad_job" "test" {
  jobspec = <<EOT
job "foo-volumes" {
  datacenters = ["dc1"]

  group "foo" {
    volume "data" {
      type      = "host"
      read_only = true
      source    = "data"
    }

    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["10"]
      }

      volume_mount {
        volume           = "data"
        destination      = "/var/lib/data"
        read_only        = true
        propagation_mode = "private"
      }
    }
  }
}
EOT
}
`

var testJobScalingPolicyConfig = `
resource "nomad_job" "test" {
  jobspec = <<EOT
job "foo-scaling" {
  datacenters = ["dc1"]

  group "foo" {
    scaling {
      min     = 10
      max     = 20
      enabled = false

      policy {
        opaque = true
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
EOT
}
`

var testJobScalingPolicyDASConfig = `
resource "nomad_job" "test_das" {
  jobspec = <<EOT
job "foo-scaling-das" {
  datacenters = ["dc1"]

  group "foo" {
    task "foo" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["10"]
      }

      scaling "cpu" {
        min     = 10
        max     = 20
        enabled = false

        policy {
          opaque = true
        }
      }
    }
  }
}
EOT
}
`

var testJobLifecycleConfig = `
resource "nomad_job" "test" {
  jobspec = <<EOT
job "foo-lifecycle" {
  datacenters = ["dc1"]

  group "foo" {
    restart {
      attempts = 5
      interval = "10m"
      delay    = "15s"
      mode     = "delay"
    }

    task "sidecar" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["10"]
      }

      restart {
        attempts = 10
      }

      lifecycle {
        hook    = "prestart"
        sidecar = true
      }
    }
  }
}
EOT
}
`

var testJobActionsConfig = `
resource "nomad_job" "test" {
  jobspec = <<EOT
job "actions" {
  group "foo" {
    task "sidecar" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["10"]
      }

      action "echo" {
        command = "/bin/echo"
        args    = ["hi"]
      }
    }
  }
}
EOT
}
`

var testJobServiceDeploymentInfo = `
resource "nomad_job" "service" {
  detach = false

  jobspec = <<EOT
job "foo-service-with-deployment" {
  type        = "service"
  datacenters = ["dc1"]

  group "service" {
    update {
      min_healthy_time  = "1s"
      healthy_deadline  = "2s"
      progress_deadline = "3s"
    }

    task "sleep" {
      driver = "raw_exec"

      config {
        command = "sleep"
        args    = ["3600"]
      }
    }
  }
}
EOT
}
`

var testJobBatchNoDetach = `
resource "nomad_job" "batch_no_detach" {
  detach = false

  jobspec = <<EOT
job "foo-batch" {
  type        = "batch"
  datacenters = ["dc1"]

  group "service" {
    task "env" {
      driver = "raw_exec"

      config {
        command = "env"
      }
    }
  }
}
EOT
}
`

var testJobServiceNoDeployment = `
resource "nomad_job" "service" {
  detach = false

  jobspec = <<EOT
job "foo-service-without-deployment" {
  type        = "service"
  datacenters = ["dc1"]

  update {
    max_parallel = 0
  }

  group "service" {
    task "sleep" {
      driver = "raw_exec"

      config {
        command = "sleep"
        args    = ["3600"]
      }
    }
  }
}
EOT
}
`

var testJobPeriodicConfig = `
resource "nomad_job" "periodic" {
  jobspec = <<EOT
job "foo-periodic" {
  type        = "batch"
  datacenters = ["dc1"]

  periodic {
    enabled          = true
    cron             = "*/1 * * * * *"
    prohibit_overlap = true
    time_zone        = "UTC"
  }

  group "periodic" {
    task "sleep" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["1"]
      }
    }
  }
}
EOT
}
`

var testJobMultiregion = `
resource "nomad_job" "multiregion" {
  detach = false

  jobspec = <<EOT
job "foo-multiregion" {
  multiregion {
    region "global" {
      datacenters = ["dc1"]
      count       = 2
    }
  }

  group "foo" {
    task "foo" {
      driver = "docker"

      config {
        image = "nginx:alpine"
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
EOT
}
`

var testJobScheduleBlock = `
resource "nomad_job" "schedule" {
  detach = false

  jobspec = <<EOT
job "foo-schedule" {
  group "foo" {
    task "foo" {
      schedule {
        cron {
          start    = "0 12 * * * *"
          end      = "0 16"
          timezone = "EST"
        }
      }

      driver = "docker"

      config {
        image = "nginx:alpine"
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
EOT
}
`

var testJobUIBlock = `
resource "nomad_job" "ui" {
  detach = false

  jobspec = <<EOT
job "foo-ui" {
  ui {
    description = "A job that includes a UI block"
  }

  group "foo" {
    task "foo" {
      driver = "docker"

      config {
        image = "nginx:alpine"
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
EOT
}
`

var testJobCSIController = `
resource "nomad_job" "test" {
  detach = false

  jobspec = <<EOT
job "foo-csi-controller" {
  datacenters = ["dc1"]

  group "foo-controller" {
    stop_after_client_disconnect = "90s"

    task "plugin" {
      driver = "docker"

      config {
        image = "amazon/aws-ebs-csi-driver:latest"
        args  = [
          "controller",
          "--endpoint=unix://csi/csi.sock",
          "--logtostderr",
          "--v=5",
        ]
      }

      csi_plugin {
        id        = "aws-ebs0"
        type      = "controller"
        mount_dir = "/csi"
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
EOT
}
`

var testJobCPUCoresConfig = `
resource "nomad_job" "test_cpu_cores" {
  hcl2 {}

  jobspec = <<EOT
job "test-cpu-cores" {
  datacenters = ["dc1"]

  group "test" {
    task "test" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = ["10"]
      }

      resources {
        cores = 1
      }
    }
  }
}
EOT
}
`

var testJobParameterizedJob = `
resource "nomad_job" "parameterized" {
  jobspec = <<EOT
job "parameterized" {
  datacenters = ["dc1"]
  type        = "batch"

  parameterized {
    payload = "required"
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
EOT
}
`

var testJobHCL2 = `
resource "nomad_job" "hcl2" {
  hcl2 {
    allow_fs = false
    vars = {
      "restart_attempts" = "5",
      "datacenters"      = "[\"dc1\", \"dc2\"]",
    }
  }

  jobspec = <<EOT
variables {
  args = ["10"]
}

variable "datacenters" {
  type = list(string)
}

variable "restart_attempts" {
  type = number
}

job "foo-hcl2" {
  datacenters = var.datacenters

  group "hcl2" {
    restart {
      attempts = var.restart_attempts
      interval = "10m"
      delay    = "15s"
      mode     = "delay"
    }

    task "sleep" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = var.args
      }

      restart {
        attempts = 10
      }

      template {
        data        = "Hello :)\n"
        destination = "local/hello.txt"
      }
    }
  }
}
EOT
}
`

var testJobHCL2NoFS = `
resource "nomad_job" "hcl2" {
  hcl2 {}

  jobspec = <<EOT
variables {
  args = ["10"]
}

job "foo-hcl2" {
  datacenters = ["dc1"]

  group "hcl2" {
    restart {
      attempts = 5
      interval = "10m"
      delay    = "15s"
      mode     = "delay"
    }

    task "sleep" {
      driver = "raw_exec"

      config {
        command = "/bin/sleep"
        args    = var.args
      }

      restart {
        attempts = 10
      }

      template {
        data        = file("../../../nomad/test-fixtures/hello.txt")
        destination = "local/hello.txt"
      }
    }
  }
}
EOT
}
`
