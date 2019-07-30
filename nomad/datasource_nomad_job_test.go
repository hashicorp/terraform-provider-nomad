package nomad

import (
	"fmt"
	"github.com/hashicorp/nomad/api"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

//func cleanupTestJobAndNamespace(jobID, namespace string) resource.TestCheckFunc {
//	return func(*terraform.State) error {
//		providerConfig := testProvider.Meta().(ProviderConfig)
//		client := providerConfig.client
//		_, _, err := client.Jobs().Deregister(jobID, true, &api.WriteOptions{
//			Namespace: namespace,
//		})
//		if err != nil {
//			return fmt.Errorf("failed to clean up job %q after test: %s", jobID, err)
//		}
//		if namespace != "default" && namespace != "" {
//			_, err := client.Namespaces().Delete(namespace, nil)
//			if err != nil {
//				return fmt.Errorf("failed to clean up namespace %q after test: %s", namespace, err)
//			}
//		}
//		return nil
//	}
//}
//
//func setupTestJobAndNamespace(job, namespace string) error {
//	client := testProvider.Meta().(ProviderConfig).client
//	fmt.Println("setupTestJobAndNamespace")
//	if namespace != "" && namespace != "default" {
//		_, err := client.Namespaces().Register(&api.Namespace{
//			Name: namespace,
//		}, nil)
//		if err != nil {
//			return fmt.Errorf("error creating test namespace: %s", err)
//		}
//	}
//	j, _ := client.Jobs().ParseHCL(testJobHCL(job, namespace), false)
//	_, _, err := client.Jobs().Register(j, nil)
//	if err != nil {
//		return fmt.Errorf("error creating test job: %s", err)
//	}
//	return nil
//}

func TestAccDataSourceNomadJob_Basic(t *testing.T) {
	job := "testjobds"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testProviders,
		CheckDestroy: testResourceJob_forceDestroyWithPurge(job, "default"),
		Steps: []resource.TestStep{
			{
				Config: testAccJobDataSourceConfig(job),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceNomadJobExists("data.nomad_job.test-job", "default"),
					resource.TestCheckResourceAttr(
						"data.nomad_job.test-job", "name", job),
					resource.TestCheckResourceAttr(
						"data.nomad_job.test-job", "type", "batch"),
					resource.TestCheckResourceAttr(
						"data.nomad_job.test-job", "priority", "50"),
					resource.TestCheckResourceAttr(
						"data.nomad_job.test-job", "namespace", "default"),
				),
			},
			//{
			//	Config: testAccCheckDataSourceNomadJobConfigErr,
			//	Destroy:     false,
			//	ExpectError: regexp.MustCompile(`.*job not found`),
			//},
		},
	})
}

func TestAccDataSourceNomadJob_Namespaced(t *testing.T) {
	ns := "jobds-test-namespace"
	job := "testjobds-namespace"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); testCheckPro(t) },
		Providers:    testProviders,
		CheckDestroy: testResourceJob_forceDestroyWithPurge(job, ns),
		Steps: []resource.TestStep{
			{
				Config: testAccNSJobDataSourceConfig(job, ns),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceNomadJobExists("data.nomad_job.test-job", ns),
					resource.TestCheckResourceAttr(
						"data.nomad_job.test-job", "name", job),
					resource.TestCheckResourceAttr(
						"data.nomad_job.test-job", "type", "batch"),
					resource.TestCheckResourceAttr(
						"data.nomad_job.test-job", "priority", "50"),
					resource.TestCheckResourceAttr(
						"data.nomad_job.test-job", "namespace", ns),
				),
			},
			//{
			//	Config:      testAccCheckDataSourceNomadJobWrongNamespace,
			//	ExpectError: regexp.MustCompile(`.*job not found`),
			//},
		},
	})
}

func testAccDataSourceNomadJobExists(n, namespace string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Job ID is set")
		}

		providerConfig := testProvider.Meta().(ProviderConfig)
		client := providerConfig.client

		id := rs.Primary.ID

		// Try to find the job
		test_job, _, err := client.Jobs().Info(id, &api.QueryOptions{
			Namespace: namespace,
		})

		if err != nil {
			return err
		}

		if *test_job.ID != rs.Primary.ID {
			return fmt.Errorf("Job not found")
		}

		return nil
	}
}

func testAccNSJobDataSourceConfig(job, ns string) string {
	return `
resource "nomad_namespace" "ns-instance" {
  name = "` + ns + `" 
}

resource "nomad_job" "job-instance" {
	jobspec = <<EOT
		job "` + job + `" {
			datacenters = ["dc1"]
			type = "batch"
			namespace = "${nomad_namespace.ns-instance.name}"
			group "foo" {
				task "foo" {
					driver = "raw_exec"
					config {
						command = "/bin/echo"
						args = ["test"]
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

data "nomad_job" "test-job" {
  job_id    = "${nomad_job.job-instance.id}"
  namespace = "${nomad_job.job-instance.namespace}"
}
`
}

func testAccJobDataSourceConfig(job string) string {
	return `
resource "nomad_job" "job-instance" {
	jobspec = <<EOT
		job "` + job + `" {
			datacenters = ["dc1"]
			type = "batch"
			group "foo" {
				task "foo" {
					driver = "raw_exec"
					config {
						command = "/bin/echo"
						args = ["test"]
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

data "nomad_job" "test-job" {
  job_id    = "${nomad_job.job-instance.id}"
}
`
}
