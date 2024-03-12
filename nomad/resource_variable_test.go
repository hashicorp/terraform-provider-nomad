// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceVariable_basic(t *testing.T) {
	path := acctest.RandomWithPrefix("tf-nomad-test")

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.4.0") },
		Steps: []resource.TestStep{
			{
				Config: testResourceVariable_initialConfig(api.DefaultNamespace, path),
				Check:  testResourceVariable_initialCheck(api.DefaultNamespace, path),
			},
		},

		CheckDestroy: testResourceVariable_checkDestroy(api.DefaultNamespace, path),
	})
}

func TestResourceVariable_pathChange(t *testing.T) {
	path := acctest.RandomWithPrefix("tf-nomad-test")
	newPath := acctest.RandomWithPrefix("tf-nomad-test")

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceVariable_initialConfig(api.DefaultNamespace, path),
				Check:  testResourceVariable_initialCheck(api.DefaultNamespace, path),
			},
			{
				Config: testResourceVariable_initialConfig(api.DefaultNamespace, newPath),
				Check:  testResourceVariable_initialCheck(api.DefaultNamespace, newPath),
			},
		},

		CheckDestroy: testResourceVariable_checkDestroy(api.DefaultNamespace, path),
	})
}

func TestResourceVariable_namespaceChange(t *testing.T) {
	path := acctest.RandomWithPrefix("tf-nomad-test")
	newPath := acctest.RandomWithPrefix("tf-nomad-test")

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceVariable_initialConfig(api.DefaultNamespace, path),
				Check:  testResourceVariable_initialCheck(api.DefaultNamespace, path),
			},
			{
				Config: testResourceVariable_initialConfigWithNamespace("var-test-namespace", newPath),
				Check:  testResourceVariable_initialCheck("var-test-namespace", newPath),
			},
		},

		CheckDestroy: resource.ComposeTestCheckFunc(
			testResourceVariable_checkDestroy(api.DefaultNamespace, path),
			testResourceVariable_checkDestroy("var-test-namespace", path),
		),
	})
}

func testResourceVariable_initialConfig(namespace, path string) string {
	return fmt.Sprintf(`
resource "nomad_variable" "test" {
  namespace = "%s"
  path      = "%s"

  items = {
    test_key = "test_value"
  }
}
`, namespace, path)
}

func testResourceVariable_initialConfigWithNamespace(namespace, path string) string {
	return fmt.Sprintf(`
resource nomad_namespace "nomad_var_test" {
  name = "%s"
}
%s
`, namespace, testResourceVariable_initialConfig("${nomad_namespace.nomad_var_test.name}", path))
}

func testResourceVariable_initialCheck(namespace, path string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
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

func testResourceVariable_checkDestroy(namespace, path string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client
		resourceID := path + "@" + namespace

		variable, _, err := client.Variables().Read(path, &api.QueryOptions{Namespace: namespace})
		if err != nil && strings.Contains(err.Error(), "404") || variable == nil {
			return nil
		}

		return fmt.Errorf("variable %q has not been deleted.", resourceID)
	}
}
