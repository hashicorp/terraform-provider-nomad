// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"errors"
	"fmt"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestResourceVariable_basic(t *testing.T) {
	path := acctest.RandomWithPrefix("tf-nomad-test")

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceVariable_initialConfig(path),
				Check:  testResourceVariable_initialCheck(path),
			},
		},

		CheckDestroy: testResourceVariable_checkDestroy(path),
	})
}

func testResourceVariable_initialConfig(path string) string {
	return fmt.Sprintf(`
resource "nomad_variable" "test" {
  path = "%s"
  
  items = {
    test_key = "test_value"
  }
}
`, path)
}

func testResourceVariable_initialCheck(path string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		namespace := api.DefaultNamespace
		resourceID := path + "@" + namespace

		resourceState := s.Modules[0].Resources["nomad_variable.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}

		if instanceState.ID != resourceID {
			return fmt.Errorf("expected ID to be %q, got %q", resourceID, instanceState.ID)
		}

		if instanceState.Attributes["path"] != path {
			return fmt.Errorf("expected path to be %q, is %q in state", path, instanceState.Attributes["path"])
		}

		if instanceState.Attributes["namespace"] != namespace {
			return fmt.Errorf("expected namespace to be %q, is %q in state", namespace, instanceState.Attributes["description"])
		}

		client := testProvider.Meta().(ProviderConfig).client
		variable, _, err := client.Variables().Read(path, &api.QueryOptions{Namespace: namespace})
		if err != nil {
			return fmt.Errorf("error reading back variable %q: %s", resourceID, err)
		}

		if variable.Path != path {
			return fmt.Errorf("expected path to be %q, is %q in API", path, variable.Path)
		}

		if variable.Namespace != namespace {
			return fmt.Errorf("expected namespace to be %q, is %q in API", namespace, variable.Namespace)
		}

		return nil
	}
}

func testResourceVariable_checkDestroy(path string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client
		namespace := api.DefaultNamespace
		resourceID := path + "@" + namespace

		variable, _, err := client.Variables().Read(path, &api.QueryOptions{Namespace: namespace})
		if err != nil && strings.Contains(err.Error(), "404") || variable == nil {
			return nil
		}

		return fmt.Errorf("variable %q has not been deleted.", resourceID)
	}
}
