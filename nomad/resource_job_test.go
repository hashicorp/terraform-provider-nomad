package nomad

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/nomad/api"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
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
				Check:  testResourceJob_initialCheck,
			},
		},

		CheckDestroy: testResourceJob_checkDestroy("foo"),
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

func TestResourceJob_json(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_jsonConfig,
				Check:  testResourceJob_initialCheck,
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
				Check:  testResourceJob_initialCheck,
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
			{
				Config: testResourceJob_noDestroy,
				Check:  testResourceJob_initialCheck,
			},

			// Destroy with our setting set
			{
				Destroy: true,
				Config:  testResourceJob_noDestroy,
				Check:   testResourceJob_checkExists,
			},

			// Re-apply without the setting set
			{
				Config: testResourceJob_initialConfig,
				Check:  testResourceJob_checkExists,
			},
		},
	})
}

func TestResourceJob_idChange(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_initialConfig,
				Check:  testResourceJob_initialCheck,
			},

			// Change our ID
			{
				Config: testResourceJob_updateConfig,
				Check:  testResourceJob_updateCheck,
			},
		},
	})
}

func TestResourceJob_policyOverride(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceJob_policyOverrideConfig(),
				Check:  testResourceJob_initialCheck,
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

var testResourceJob_jsonConfig = `
resource "nomad_job" "json_test" {
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

func testResourceJob_initialCheck(s *terraform.State) error {
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

	return nil
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

func testResourceJob_checkExists(s *terraform.State) error {
	jobID := "foo"

	providerConfig := testProvider.Meta().(ProviderConfig)
	client := providerConfig.client
	_, _, err := client.Jobs().Info(jobID, nil)
	if err != nil {
		return fmt.Errorf("error reading back job: %s", err)
	}

	return nil
}

func testResourceJob_checkDestroy(jobID string) r.TestCheckFunc {
	return func(*terraform.State) error {
		providerConfig := testProvider.Meta().(ProviderConfig)
		client := providerConfig.client

		tries := 0
		for {
			job, _, err := client.Jobs().Info(jobID, nil)
			// This should likely never happen, due to how nomad caches jobs
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
				break
			}
		}

		return fmt.Errorf("Job %q has not been stopped.", jobID)
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

var testResourceJob_updateConfig = `
resource "nomad_job" "test" {
	jobspec = <<EOT
		job "bar" {
			datacenters = ["dc1"]
			type = "service"
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
				}
			}
		}
	EOT
}
`

func testResourceJob_updateCheck(s *terraform.State) error {
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

	{
		// Verify foo doesn't exist
		job, _, err := client.Jobs().Info("foo", nil)
		if err != nil {
			// Job could have already been purged from nomad server
			if !strings.Contains(err.Error(), "(job not found)") {
				return fmt.Errorf("error reading %q job: %s", "foo", err)
			}
			return nil
		}

		if *job.Status != "dead" {
			return fmt.Errorf("%q job is not dead. Status: %q", "foo", *job.Status)
		}
	}

	return nil
}

func TestResourceJob_vault(t *testing.T) {
	re, err := regexp.Compile("bad token")
	if err != nil {
		t.Errorf("Error compiling regex: %s", err)
	}
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config:      testResourceJob_invalidVaultConfig,
				Check:       testResourceJob_initialCheck,
				ExpectError: re,
			},
			{
				Config: testResourceJob_validVaultConfig,
				Check:  testResourceJob_initialCheck,
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
