package nomad

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccNomadJob_Basic(t *testing.T) {
	var testJob api.Job
	jobId := acctest.RandString(8)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testProviders,
		CheckDestroy: testAccCheckNomadJobDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckNomadJobConfig_basic(jobId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNomadJobExists("nomad_job.foobar", &testJob),
					resource.TestCheckResourceAttr(
						"data.nomad_job.foobar", "job_id", fmt.Sprintf("%s", jobId),
					),
				),
			},
			{
				Config:      testAccCheckNomadJobConfig_nonexisting(jobId),
				Destroy:     false,
				ExpectError: regexp.MustCompile(`.*no job found with that id`),
			},
		},
	})
}

func testAccCheckNomadJobExists(n string, job *api.Job) resource.TestCheckFunc {
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
		test_job, _, err := client.Jobs().Info(id, nil)

		if err != nil {
			return err
		}

		if *test_job.ID != rs.Primary.ID {
			return fmt.Errorf("Job not found")
		}

		*job = *test_job

		return nil
	}
}

func testAccCheckNomadJobDestroy(s *terraform.State) error {
	providerConfig := testProvider.Meta().(ProviderConfig)
	client := providerConfig.client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "nomad_job" {
			continue
		}

		id := rs.Primary.ID

		// Try to find the Droplet
		_, _, err := client.Jobs().Info(id, nil)

		// Wait

		if err != nil && !strings.Contains(err.Error(), "404") {
			return fmt.Errorf(
				"Error waiting for job (%s) to be deregistered: %s",
				rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckNomadJobConfig_basic(str string) string {
	return fmt.Sprintf(`
data "nomad_job" "foobar" {
  job_id               = "%s"
}
`, str)
}

func testAccCheckNomadJobConfig_nonexisting(str string) string {
	return fmt.Sprintf(`
data "nomad_job" "foobar" {
  job_id               = "%s-nonexisting"
}
`, str)
}
