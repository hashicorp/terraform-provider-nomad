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
	var testDeployment api.Deployment
	deploymentId := acctest.RandString(8)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testProviders,
		CheckDestroy: testAccCheckNomadDeploymentFail,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckNomadDeploymentConfig_basic(deploymentId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNomadDeploymentExists("nomad_deployment.foobar", &testDeployment),
				),
			},
			{
				Config: testAccCheckNomadDeploymentConfig_basic(deploymentId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.nomad_deployment.foobar", "id", fmt.Sprintf("%s", deploymentId)),
				),
			},
			{
				Config:      testAccCheckNomadDeploymentConfig_nonexisting(deploymentId),
				Destroy:     false,
				ExpectError: regexp.MustCompile(`.*no deployment found with that id`),
			},
		},
	})
}

func testAccCheckNomadDeploymentExists(n string, deployment *api.Deployment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Deployment ID is set")
		}

		providerConfig := testProvider.Meta().(ProviderConfig)
		client := providerConfig.client

		id := rs.Primary.ID

		// Try to find the deployment
		test_deployment, _, err := client.Deployments().Info(id, nil)

		if err != nil {
			return err
		}

		if test_deployment.ID != rs.Primary.ID {
			return fmt.Errorf("Deployment not found")
		}

		*deployment = *test_deployment

		return nil
	}
}

func testAccCheckNomadDeploymentFail(s *terraform.State) error {
	providerConfig := testProvider.Meta().(ProviderConfig)
	client := providerConfig.client

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "nomad_deployment" {
			continue
		}

		id := rs.Primary.ID

		// Try to find the Droplet
		_, _, err := client.Deployments().Info(id, nil)

		// Wait

		if err != nil && !strings.Contains(err.Error(), "404") {
			return fmt.Errorf(
				"Error waiting for deployment (%s) to fail: %s",
				rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckNomadDeploymentConfig_basic(str string) string {
	return fmt.Sprintf(`
data "nomad_deployment" "foobar" {
  id               = "%s"
}
`, str)
}

func testAccCheckNomadDeploymentConfig_nonexisting(str string) string {
	return fmt.Sprintf(`
data "nomad_deployment" "foobar" {
  id               = "%s-nonexisting"
}
`, str)
}
