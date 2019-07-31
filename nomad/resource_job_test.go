package nomad

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/nomad/api"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceJob_basic(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_initialConfig,
				Check:  testResourceJob_initialCheck(t),
			},
		},

		CheckDestroy: testResourceJob_checkDestroy("foo"),
	})
}

func TestResourceJob_namespace(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_initialConfigNamespace,
				Check:  testResourceJob_initialCheckNS(t, "jobresource-test-namespace"),
			},
		},

		CheckDestroy: testResourceJob_checkDestroyNS("foo", "jobresource-test-namespace"),
	})
}

func TestResourceJob_v086(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_v086config,
				Check:  testResourceJob_v086Check,
			},
		},

		CheckDestroy: testResourceJob_checkDestroy("foov086"),
	})
}

func TestResourceJob_v090(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_v090config,
				Check:  testResourceJob_v090Check,
			},
		},

		CheckDestroy: testResourceJob_checkDestroy("foov086"),
	})
}

func TestResourceJob_json(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_jsonConfig,
				Check:  testResourceJob_initialCheck(t),
			},
		},

		CheckDestroy: testResourceJob_checkDestroy("foo-json"),
	})
}

func TestResourceJob_refresh(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_initialConfig,
				Check:  testResourceJob_initialCheck(t),
			},

			// This should successfully cause the job to be recreated,
			// testing the Exists function.
			{
				PreConfig: testResourceJob_deregister(t, "foo"),
				Config:    testResourceJob_initialConfig,
			},
		},
	})
}

func TestResourceJob_disableDestroyDeregister(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			// create the resource
			{
				Config: testResourceJob_noDestroy,
				Check:  testResourceJob_initialCheck(t),
			},
			// "Destroy" with 'deregister_on_destroy = false', check that it wasn't destroyed
			{
				Destroy: true,
				Config:  testResourceJob_noDestroy,
				Check: func(*terraform.State) error {
					providerConfig := testProvider.Meta().(ProviderConfig)
					client := providerConfig.client
					job, _, err := client.Jobs().Info("foo", nil)
					if err != nil {
						return err
					}
					if *job.Stop == true {
						return fmt.Errorf("job was unexpectedly stopped")
					}
					return nil
				},
			},
		},

		// Somewhat-abuse CheckDestroy to clean up
		CheckDestroy: testResourceJob_forceDestroyWithPurge("foo", "default"),
	})
}

func TestResourceJob_rename(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_initialConfig,
				Check:  testResourceJob_initialCheck(t),
			},
			{
				Config: testResourceJob_renameConfig,
				Check: resource.ComposeTestCheckFunc(
					testResourceJob_checkDestroy("foo"),
					testResourceJob_checkExists("bar"),
				),
			},
		},

		CheckDestroy: testResourceJob_checkDestroy("bar"),
	})
}

func TestResourceJob_change_namespace(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_initialConfigNamespace,
				Check:  testResourceJob_initialCheckNS(t, "jobresource-test-namespace"),
			},
			{
				Config: testResourceJob_changeNamespaceConfig,
				Check: resource.ComposeTestCheckFunc(
					testResourceJob_checkDestroyNS("foo", "jobresource-test-namespace"),
					testResourceJob_checkExistsNS("foo", "jobresource-updated-namespace"),
				),
			},
		},

		CheckDestroy: resource.ComposeTestCheckFunc(
			testResourceJob_checkDestroyNS("bar", "jobresource-test-namespace"),
			testResourceJob_checkDestroyNS("bar", "jobresource-updated-namespace"),
		),
	})
}

func TestResourceJob_policyOverride(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckEnt(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_policyOverrideConfig(),
				Check:  testResourceJob_initialCheck(t),
			},
		},
	})
}

func TestResourceJob_parameterizedJob(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_parameterizedJob,
				Check:  testResourceJob_parameterizedCheck,
			},
		},
	})
}

func testResourceJob_parameterizedCheck(s *terraform.State) error {
	resourceState := s.Modules[0].Resources["nomad_job.parameterized"]
	if resourceState == nil {
		return errors.New("resource not found in state")
	}

	instanceState := resourceState.Primary
	if instanceState == nil {
		return errors.New("resource has no primary instance")
	}

	jobID := instanceState.ID

	providerConfig := testProvider.Meta().(ProviderConfig)
	client := providerConfig.client
	job, _, err := client.Jobs().Info(jobID, nil)
	if err != nil {
		return fmt.Errorf("error reading back job: %s", err)
	}

	if got, want := *job.ID, jobID; got != want {
		return fmt.Errorf("jobID is %q; want %q", got, want)
	}

	return nil
}

var testResourceJob_parameterizedJob = `
resource "nomad_job" "parameterized" {
	jobspec = <<EOT
		job "parameterized" {
			datacenters = ["dc1"]
			type = "batch"
			parameterized {
				payload = "required"
			}
			group "foo" {
				task "foo" {
					driver = "raw_exec"
					config {
						command = "/bin/sleep"
						args = ["1"]
					}
					resources {
						cpu = 100
						memory = 10
					}
					
					logs {
						max_files = 3
						max_file_size = 10
					}
				}
			}
		}
	EOT
}
`

var testResourceJob_initialConfig = `
resource "nomad_job" "test" {
	jobspec = <<EOT
		job "foo" {
			datacenters = ["dc1"]
			type = "service"
			group "foo" {
				task "foo" {
					leader = true ## new in Nomad 0.5.6
					
					driver = "raw_exec"
					config {
						command = "/bin/sleep"
						args = ["10"]
					}
					
					resources {
						cpu = 100
						memory = 10
					}
					
					logs {
						max_files = 3
						max_file_size = 10
					}
				}
			}
		}
	EOT
}
`

var testResourceJob_initialConfigNamespace = `
resource "nomad_namespace" "test-namespace" {
  name = "jobresource-test-namespace" 
}

resource "nomad_job" "test" {
	jobspec = <<EOT
		job "foo" {
			datacenters = ["dc1"]
			type = "batch"
			namespace = "${nomad_namespace.test-namespace.name}"
			group "foo" {
				task "foo" {
					driver = "raw_exec"
					config {
						command = "/bin/sleep"
						args = ["10"]
					}
					
					resources {
						cpu = 100
						memory = 10
					}
					
					logs {
						max_files = 3
						max_file_size = 10
					}
				}
			}
		}
	EOT
}
`

var testResourceJob_changeNamespaceConfig = `
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
			type = "batch"
			namespace = "${nomad_namespace.new-namespace.name}"
			group "foo" {
				task "foo" {
					driver = "raw_exec"
					config {
						command = "/bin/sleep"
						args = ["10"]
					}
					
					resources {
						cpu = 100
						memory = 10
					}
					
					logs {
						max_files = 3
						max_file_size = 10
					}
				}
			}
		}
	EOT
}
`

var testResourceJob_jsonConfig = `
resource "nomad_job" "test" {
	json = true
	jobspec = <<EOT
{
  "Datacenters": [ "dc1" ],
  "ID": "foo-json",
  "Name": "foo-json",
  "Type": "service",
  "TaskGroups": [
    {
      "Name": "foo",
      "Tasks": [{
        "Config": {
          "command": "/bin/sleep",
          "args": [ "1" ]
        },
        "Driver": "raw_exec",
        "Leader": true,
        "LogConfig": {
          "MaxFileSizeMB": 10,
          "MaxFiles": 3
        },
        "Name": "foo",
        "Resources": {
          "CPU": 100,
          "MemoryMB": 10
        }
      }
      ]
    }
  ]
}
EOT
}
`

var testResourceJob_renameConfig = `
resource "nomad_job" "test" {
    jobspec = <<EOT
		job "bar" {
		    datacenters = ["dc1"]
		    type = "service"
		    group "foo" {
		        task "foo" {
		            leader = true ## new in Nomad 0.5.6
							
		            driver = "raw_exec"
		            config {
		                command = "/bin/sleep"
		                args = ["1"]
		            }
							
		            resources {
		                cpu = 100
		                memory = 10
		            }
							
		            logs {
		                max_files = 3
		                max_file_size = 10
		            }
		        }
		    }
		}
	EOT
}
`

var testResourceJob_noDestroy = `
resource "nomad_job" "test" {
    deregister_on_destroy = false
    jobspec = <<EOT
		job "foo" {
			datacenters = ["dc1"]
			type = "service"
			group "foo" {
				task "foo" {
					driver = "raw_exec"
					config {
						command = "/bin/sleep"
						args = ["30"]
					}
						
					resources {
						cpu = 100
						memory = 10
					}
					
					logs {
						max_files = 3
						max_file_size = 10
					}
				}
			}
		}
	EOT
}
`

func testResourceJob_initialCheck(t *testing.T) r.TestCheckFunc {
	return testResourceJob_initialCheckNS(t, "default")
}

func testResourceJob_initialCheckNS(t *testing.T, expectedNamespace string) r.TestCheckFunc {
	return func(s *terraform.State) error {

		resourceState := s.Modules[0].Resources["nomad_job.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}

		jobID := instanceState.ID

		if setNamespace, ok := instanceState.Attributes["namespace"]; !ok || setNamespace != expectedNamespace {
			return errors.New("resource does not have expected namespace")
		}

		providerConfig := testProvider.Meta().(ProviderConfig)
		client := providerConfig.client
		job, _, err := client.Jobs().Info(jobID, &api.QueryOptions{
			Namespace: expectedNamespace,
		})
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}

		if got, want := *job.ID, jobID; got != want {
			return fmt.Errorf("jobID is %q; want %q", got, want)
		}

		if got, want := *job.Namespace, expectedNamespace; got != want {
			return fmt.Errorf("job namespace is %q; want %q", got, want)
		}

		wantAllocs, _, err := client.Jobs().Allocations(jobID, false, nil)
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}
		wantAllocIds := make([]string, 0, len(wantAllocs))
		for _, a := range wantAllocs {
			wantAllocIds = append(wantAllocIds, a.ID)
		}
		numGotAllocs, _ := strconv.Atoi(instanceState.Attributes["allocation_ids.#"])
		gotAllocs := make([]string, 0, numGotAllocs)
		for i := 0; i < numGotAllocs; i++ {
			id := instanceState.Attributes[fmt.Sprintf("allocation_ids.%d", i)]
			gotAllocs = append(gotAllocs, id)
		}
		if !assert.ElementsMatch(t, gotAllocs, wantAllocIds) {
			return fmt.Errorf("job 'allocation_ids' is '%v'; want '%v'", gotAllocs, wantAllocIds)
		}

		return nil
	}
}

func testResourceJob_v086Check(s *terraform.State) error {

	resourceState := s.Modules[0].Resources["nomad_job.test"]
	if resourceState == nil {
		return errors.New("resource not found in state")
	}

	instanceState := resourceState.Primary
	if instanceState == nil {
		return errors.New("resource has no primary instance")
	}

	jobID := instanceState.ID

	providerConfig := testProvider.Meta().(ProviderConfig)
	client := providerConfig.client
	job, _, err := client.Jobs().Info(jobID, nil)
	if err != nil {
		return fmt.Errorf("error reading back job: %s", err)
	}

	if got, want := *job.ID, jobID; got != want {
		return fmt.Errorf("jobID is %q; want %q", got, want)
	}

	if len(job.TaskGroups) != 1 {
		return fmt.Errorf("expected a single TaskGroup")
	}
	tg := job.TaskGroups[0]

	// 0.8.x jobs support migrate and update stanzas
	expUpdate := api.UpdateStrategy{}
	json.Unmarshal([]byte(`{
      "Stagger":  		   30000000000,
      "MaxParallel": 2,
      "HealthCheck": "checks",
      "MinHealthyTime":    12000000000,
      "HealthyDeadline":  360000000000,
      "ProgressDeadline": 720000000000,
      "AutoRevert": true,
      "Canary": 1
    }`), &expUpdate)
	if !reflect.DeepEqual(tg.Update, &expUpdate) {
		return fmt.Errorf("job update strategy not as expected")
	}

	expMigrate := api.MigrateStrategy{}
	json.Unmarshal([]byte(`{
      "MaxParallel": 2,
      "HealthCheck": "checks",
      "MinHealthyTime":   12000000000,
      "HealthyDeadline": 360000000000
	}`), &expMigrate)
	if !reflect.DeepEqual(tg.Migrate, &expMigrate) {
		return fmt.Errorf("job migrate strategy not as expected")
	}

	// 0.8.x TaskGroups support reschedule stanza
	expReschedule := api.ReschedulePolicy{}
	json.Unmarshal([]byte(`{
	  "Attempts": 0,
	  "Interval": 7200000000000,
	  "Delay": 	    12000000000,
	  "DelayFunction": "exponential",
	  "MaxDelay":  100000000000,
	  "Unlimited": true
	}`), &expReschedule)
	if !reflect.DeepEqual(tg.ReschedulePolicy, &expReschedule) {
		return fmt.Errorf("job reschedule strategy not as expected")
	}

	if len(tg.Tasks) != 1 {
		return fmt.Errorf("expected a single task in the task group")
	}
	t := tg.Tasks[0]

	// 0.8.x Task service stanza supports canary tags
	if len(t.Services) != 1 {
		return fmt.Errorf("expected task Services stanza with a single element")
	}
	if sv := t.Services[0]; reflect.DeepEqual(sv.CanaryTags, []string{"canary-tag-a"}) != true {
		return fmt.Errorf("expected task canary tags")
	}

	return nil
}

func testResourceJob_v090Check(s *terraform.State) error {

	resourceState := s.Modules[0].Resources["nomad_job.test"]
	if resourceState == nil {
		return errors.New("resource not found in state")
	}

	instanceState := resourceState.Primary
	if instanceState == nil {
		return errors.New("resource has no primary instance")
	}

	jobID := instanceState.ID

	providerConfig := testProvider.Meta().(ProviderConfig)
	client := providerConfig.client
	job, _, err := client.Jobs().Info(jobID, nil)
	if err != nil {
		return fmt.Errorf("error reading back job: %s", err)
	}

	if got, want := *job.ID, jobID; got != want {
		return fmt.Errorf("jobID is %q; want %q", got, want)
	}

	// 0.9.x jobs support affinity stanzas
	expAffinities := []*api.Affinity{}
	json.Unmarshal([]byte(`[
        {
            "LTarget": "${node.datacenter}",
            "Operand": "=",
            "RTarget": "dc1",
            "Weight": 50
        },
        {
            "LTarget": "${meta.tag}",
            "Operand": "=",
            "RTarget": "foo",
            "Weight": 50
        }
    ]`), &expAffinities)
	if !reflect.DeepEqual(job.Affinities, expAffinities) {
		return fmt.Errorf("job affinities not as expected")
	}

	// 0.9.x jobs support spread stanzas
	expSpreads := []*api.Spread{}
	json.Unmarshal([]byte(`[
        {
            "Attribute": "${node.datacenter}",
            "SpreadTarget": [
                {
                    "Percent": 35,
                    "Value": "dc1"
                },
                {
                    "Percent": 65,
                    "Value": "dc2"
                }
            ],
            "Weight": 80
        }
    ]`), &expSpreads)
	if !reflect.DeepEqual(job.Spreads, expSpreads) {
		return fmt.Errorf("job spreads not as expected")
	}

	return nil
}

func testResourceJob_checkExistsNS(jobID, ns string) r.TestCheckFunc {
	return func(*terraform.State) error {
		providerConfig := testProvider.Meta().(ProviderConfig)
		client := providerConfig.client
		_, _, err := client.Jobs().Info(jobID, &api.QueryOptions{
			Namespace: ns,
		})
		if err != nil {
			return fmt.Errorf("error reading back job: %s", err)
		}

		return nil
	}
}

func testResourceJob_checkExists(jobID string) r.TestCheckFunc {
	return testResourceJob_checkExistsNS(jobID, "default")
}

func testResourceJob_checkDestroy(jobID string) r.TestCheckFunc {
	return testResourceJob_checkDestroyNS(jobID, "default")
}

func testResourceJob_checkDestroyNS(jobID, ns string) r.TestCheckFunc {
	return func(*terraform.State) error {
		providerConfig := testProvider.Meta().(ProviderConfig)
		client := providerConfig.client

		tries := 0
	TRY:
		for {
			job, _, err := client.Jobs().Info(jobID, &api.QueryOptions{
				Namespace: ns,
			})
			// This should likely never happen because we aren't purging jobs on delete
			if err != nil && strings.Contains(err.Error(), "404") || job == nil {
				return nil
			}

			switch {
			case *job.Status == "dead":
				return nil
			case tries < 5:
				tries++
				time.Sleep(time.Second)
			default:
				break TRY
			}
		}

		return fmt.Errorf("Job %q in namespace %q has not been stopped.", jobID, ns)
	}
}

func testResourceJob_forceDestroyWithPurge(jobID, namespace string) r.TestCheckFunc {
	return func(*terraform.State) error {
		providerConfig := testProvider.Meta().(ProviderConfig)
		client := providerConfig.client
		_, _, err := client.Jobs().Deregister(jobID, true, &api.WriteOptions{
			Namespace: namespace,
		})
		if err != nil {
			return fmt.Errorf("failed to clean up job %q after test: %s", jobID, err)
		}
		return nil
	}
}

func testResourceJob_deregister(t *testing.T, jobID string) func() {
	return func() {
		providerConfig := testProvider.Meta().(ProviderConfig)
		client := providerConfig.client
		_, _, err := client.Jobs().Deregister(jobID, false, nil)
		if err != nil {
			t.Fatalf("error deregistering job: %s", err)
		}
	}
}

func TestResourceJob_vault(t *testing.T) {
	re, err := regexp.Compile("bad token")
	if err != nil {
		t.Errorf("Error compiling regex: %s", err)
	}
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckVaultEnabled(t) },
		Steps: []r.TestStep{
			{
				Config:      testResourceJob_invalidVaultConfig,
				Check:       testResourceJob_initialCheck(t),
				ExpectError: re,
			},
			{
				Config: testResourceJob_validVaultConfig,
				Check:  testResourceJob_initialCheck(t),
			},
		},
		CheckDestroy: testResourceJob_checkDestroy("test"),
	})
}

var testResourceJob_validVaultConfig = `
provider "nomad" {
}

resource "nomad_job" "test" {
	jobspec = <<EOT
		job "test" {
			datacenters = ["dc1"]
			type = "batch"
			group "foo" {
				task "foo" {
					driver = "raw_exec"
					config {
						command = "/usr/bin/true"
					}

					resources {
						cpu = 100
						memory = 10
					}

					logs {
						max_files = 3
						max_file_size = 10
					}

					vault {
						policies = ["default"]
					}
				}
			}
		}
	EOT
}
`

var testResourceJob_invalidVaultConfig = `
provider "nomad" {
	vault_token = "bad-token"
}

resource "nomad_job" "test" {
	jobspec = <<EOT
		job "test" {
			datacenters = ["dc1"]
			type = "batch"
			group "foo" {
				task "foo" {
					leader = true ## new in Nomad 0.5.6

					driver = "raw_exec"
					config {
						command = "/usr/bin/true"
					}

					resources {
						cpu = 100
						memory = 10
					}

					logs {
						max_files = 3
						max_file_size = 10
					}

					vault {
						policies = ["default"]
					}
				}
			}
		}
	EOT
}
`

func testResourceJob_policyOverrideConfig() string {
	return fmt.Sprintf(`
resource "nomad_sentinel_policy" "policy" {
  name = "%s"
  policy = "main = rule { false }"
  scope = "submit-job"
  enforcement_level = "soft-mandatory"
  description = "Fail all jobs for testing policy overrides in terraform acctests"
}

resource "nomad_job" "test" {
    depends_on = ["nomad_sentinel_policy.policy"]
    policy_override = true
    jobspec = <<EOT
job "foo" {
    datacenters = ["dc1"]
    type = "service"
    group "foo" {
        task "foo" {
            leader = true ## new in Nomad 0.5.6
            
            driver = "raw_exec"
            config {
                command = "/bin/sleep"
                args = ["1"]
            }

            resources {
                cpu = 100
                memory = 10
            }

            logs {
                max_files = 3
                max_file_size = 10
            }
        }
    }
}
EOT
}
`, acctest.RandomWithPrefix("tf-nomad-test"))
}

var testResourceJob_v086config = `
resource "nomad_job" "test" {
	jobspec = <<EOT
		job "foov086" {
			datacenters = ["dc1"]
			type = "service"

			migrate {
				max_parallel = 2
				health_check = "checks"
				min_healthy_time = "11s"
				healthy_deadline = "6m"
			}

			update {
			    max_parallel = 2	
				min_healthy_time = "11s"
				healthy_deadline = "6m"
				progress_deadline = "11m"
				auto_revert = true
				canary = 1
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
					min_healthy_time = "12s"
					progress_deadline = "12m"
				}

				reschedule {
					attempts       = 0
					delay          = "12s"
					unlimited 	   = true	
				}

				task "foo" {

					
					driver = "raw_exec"
					config {
						command = "/bin/sleep"
						args = ["1"]
					}

					resources {
						cpu = 100
						memory = 10
					}

					service {
					  canary_tags = ["canary-tag-a"]
					}

					logs {
						max_files = 3
						max_file_size = 10
					}
				}
			}
		}
	EOT
}
`

var testResourceJob_v090config = `
resource "nomad_job" "test" {
	jobspec = <<EOT
		job "foov090" {
			datacenters = ["dc1"]
			type = "service"

			migrate {
				max_parallel = 2
				health_check = "checks"
				min_healthy_time = "11s"
				healthy_deadline = "6m"
			}

			update {
			    max_parallel = 2	
				min_healthy_time = "11s"
				healthy_deadline = "6m"
				progress_deadline = "11m"
				auto_revert = true
				canary = 1
			}

			reschedule {
				attempts       = 11
				interval       = "2h"
				delay          = "11s"
				delay_function = "exponential"
				max_delay      = "100s"
				unlimited      = false
			}
 		
			affinity {
			    attribute = "$${node.datacenter}"
				value = "dc1"
				weight = 50
			}

			affinity {
			    attribute = "$${meta.tag}"
				value = "foo"
				weight = 50
			}
	
			spread {
				attribute = "$${node.datacenter}"
				target "dc1" { percent = 35 }
				target "dc2" { percent = 65 }
				weight = 80
			}

			group "foo" {

				migrate {
					min_healthy_time = "12s"
				}

				update {
					min_healthy_time = "12s"
					progress_deadline = "12m"
				}

				reschedule {
					attempts       = 0
					delay          = "12s"
					unlimited 	   = true	
				}

				task "foo" {

					
					driver = "raw_exec"
					config {
						command = "/bin/sleep"
						args = ["1"]
					}

					resources {
						cpu = 100
						memory = 10
					}

					service {
					  canary_tags = ["canary-tag-a"]
					}

					logs {
						max_files = 3
						max_file_size = 10
					}
				}
			}
		}
	EOT
}
`
