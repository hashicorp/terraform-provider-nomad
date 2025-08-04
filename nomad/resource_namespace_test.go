// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceNamespace_import(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceNamespace_initialConfig(name),
				Check:  testResourceNamespace_initialCheck(name),
			},
			{
				ResourceName:      "nomad_namespace.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},

		CheckDestroy: testResourceNamespace_checkDestroy(name),
	})
}

func TestResourceNamespace_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceNamespace_initialConfig(name),
				Check:  testResourceNamespace_initialCheck(name),
			},
		},

		CheckDestroy: testResourceNamespace_checkDestroy(name),
	})
}

func TestResourceNamespace_refresh(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceNamespace_initialConfig(name),
				Check:  testResourceNamespace_initialCheck(name),
			},

			// This should successfully cause the policy to be recreated,
			// testing the Exists function.
			{
				PreConfig: testResourceNamespace_delete(t, name),
				Config:    testResourceNamespace_initialConfig(name),
			},
		},
	})
}

func TestResourceNamespace_nameChange(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	newName := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceNamespace_initialConfig(name),
				Check:  testResourceNamespace_initialCheck(name),
			},

			// Change our name
			{
				Config: testResourceNamespace_updateConfig(newName),
				Check:  testResourceNamespace_updateCheck(newName),
			},
		},
	})
}

func TestResourceNamespace_update(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceNamespace_initialConfig(name),
				Check:  testResourceNamespace_initialCheck(name),
			},
			{
				Config: testResourceNamespace_updateConfig(name),
				Check:  testResourceNamespace_updateCheck(name),
			},
		},
	})
}

func TestResourceNamespace_deleteDefault(t *testing.T) {
	name := api.DefaultNamespace
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceNamespace_initialConfig(name),
				Check:  testResourceNamespace_initialCheck(name),
			},
		},

		CheckDestroy: testResourceNamespace_checkResetDefault(),
	})
}

func TestResourceNamespace_deleteNSWithQuota(t *testing.T) {
	nsName := "nsWithQuota1"
	quotaName := "quota1"

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckEnt(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceNamespace_configWithQuota(nsName, quotaName),
				Check: resource.ComposeTestCheckFunc(
					testResourceNamespace_initialCheck(nsName),
					testResourceNamespaceWithQuota_check(nsName, quotaName),
				),
			},
		},

		CheckDestroy: resource.ComposeTestCheckFunc(
			testResourceNamespace_checkDestroy(nsName),
			testResourceQuotaSpecification_checkDestroy(quotaName),
		),
	})
}

func TestResourceNamespace_nodePoolConfig(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.6.0"); testCheckEnt(t) },
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "nomad_namespace" "test" {
  name = "%s"

  node_pool_config {
    default = "dev"
    allowed = ["prod"]
    denied  = ["qa"]
  }
}
`, name),
				ExpectError: regexp.MustCompile(".+allowed.+conflicts with.+denied"),
			},
			{
				Config: fmt.Sprintf(`
resource "nomad_namespace" "test" {
  name = "%s"

  node_pool_config {
    default = "dev"
    allowed = ["prod", "qa"]
  }
}
`, name),
				Check: testResourceNamespace_nodePoolConfigCheck(name, &api.NamespaceNodePoolConfiguration{
					Default: "dev",
					Allowed: []string{"prod", "qa"},
					Denied:  nil,
				}),
			},
			{
				Config: fmt.Sprintf(`
resource "nomad_namespace" "test" {
  name = "%s"

  node_pool_config {
    default = "dev"
    denied  = ["prod", "qa"]
  }
}
`, name),
				Check: testResourceNamespace_nodePoolConfigCheck(name, &api.NamespaceNodePoolConfiguration{
					Default: "dev",
					Denied:  []string{"prod", "qa"},
					Allowed: nil,
				}),
			},
		},
	})
}

func testResourceNamespace_initialConfig(name string) string {
	return fmt.Sprintf(`
resource "nomad_namespace" "test" {
  name = "%s"
  description = "A Terraform acctest namespace"

  meta = {
    key = "value",
  }

  capabilities {
    enabled_task_drivers  = ["docker", "exec"]
    disabled_task_drivers = ["raw_exec"]
    enabled_network_modes = ["bridge", "none"]
    disabled_network_modes = ["host"]
  }
}
`, name)
}

func testResourceNamespace_configWithQuota(name, quota string) string {
	return fmt.Sprintf(`
resource "nomad_quota_specification" "test_quota" {
  name        = "%[2]s"
  description = "A Terraform acctest quota spec"

  limits {
    region = "global"

    region_limit {
      cpu       = 2400
      memory_mb = 1200
    }
  }
}

resource "nomad_namespace" "test" {
  name = "%[1]s"
  description = "A Terraform acctest namespace"
  quota = "%[2]s"
  depends_on = [
    nomad_quota_specification.test_quota
  ]

  meta = {
    key = "value",
  }

  capabilities {
    enabled_task_drivers  = ["docker", "exec"]
    disabled_task_drivers = ["raw_exec"]
    enabled_network_modes = ["bridge", "none"]
    disabled_network_modes = ["host"]
  }
}
`, name, quota)
}

func testResourceNamespace_initialCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const (
			description = "A Terraform acctest namespace"
		)
		resourceState := s.Modules[0].Resources["nomad_namespace.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}

		if instanceState.ID != name {
			return fmt.Errorf("expected ID to be %q, got %q", name, instanceState.ID)
		}

		if instanceState.Attributes["name"] != name {
			return fmt.Errorf("expected name to be %q, is %q in state", name, instanceState.Attributes["name"])
		}

		if instanceState.Attributes["description"] != description {
			return fmt.Errorf("expected description to be %q, is %q in state", description, instanceState.Attributes["description"])
		}

		client := testProvider.Meta().(ProviderConfig).client
		namespace, _, err := client.Namespaces().Info(name, nil)
		if err != nil {
			return fmt.Errorf("error reading back namespace %q: %s", name, err)
		}

		if namespace.Name != name {
			return fmt.Errorf("expected name to be %q, is %q in API", name, namespace.Name)
		}
		if namespace.Description != description {
			return fmt.Errorf("expected description to be %q, is %q in API", description, namespace.Description)
		}

		expectedMeta := map[string]string{
			"key": "value",
		}
		if diff := cmp.Diff(namespace.Meta, expectedMeta); diff != "" {
			return fmt.Errorf("namespace meta mismatch (-want +got):\n%s", diff)
		}

		expectedCapabilities := &api.NamespaceCapabilities{
			EnabledTaskDrivers:   []string{"docker", "exec"},
			DisabledTaskDrivers:  []string{"raw_exec"},
			EnabledNetworkModes:  []string{"bridge", "none"},
			DisabledNetworkModes: []string{"host"},
		}
		if diff := cmp.Diff(namespace.Capabilities, expectedCapabilities); diff != "" {
			return fmt.Errorf("namespace capabilities mismatch (-want +got):\n%s", diff)
		}

		return nil
	}
}

func testResourceNamespaceWithQuota_check(name, quota string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client
		namespace, _, err := client.Namespaces().Info(name, nil)
		if err != nil {
			return fmt.Errorf("error reading back namespace %q: %w", name, err)
		}

		if namespace.Quota != quota {
			return fmt.Errorf("expected quota spec to be %q, is %q in API", quota, namespace.Quota)
		}
		return nil
	}
}

func testResourceNamespace_checkExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client
		namespace, _, err := client.Namespaces().Info(name, nil)
		if err != nil {
			return fmt.Errorf("error reading back namespace %q: %s", name, err)
		}
		if namespace == nil {
			return fmt.Errorf("no namespace returned for %q", name)
		}

		return nil
	}
}

func testResourceNamespace_checkDestroy(name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client
		namespace, _, err := client.Namespaces().Info(name, nil)
		if err != nil && strings.Contains(err.Error(), "404") || namespace == nil {
			return nil
		}
		return fmt.Errorf("namespace %q has not been deleted.", name)
	}
}

func testResourceNamespace_checkResetDefault() resource.TestCheckFunc {
	return func(*terraform.State) error {
		defaultNamespace := api.Namespace{
			Name:        api.DefaultNamespace,
			Description: "Default shared namespace",
			Quota:       "",
		}
		client := testProvider.Meta().(ProviderConfig).client
		namespace, _, err := client.Namespaces().Info(defaultNamespace.Name, nil)
		if err != nil {
			return fmt.Errorf("failed to find default namespace %q.", defaultNamespace.Name)
		}
		if namespace.Description != defaultNamespace.Description || namespace.Quota != defaultNamespace.Quota {
			return fmt.Errorf("default namespace %q not reset.", defaultNamespace.Name)
		}

		return nil
	}
}

func testResourceNamespace_delete(t *testing.T, name string) func() {
	return func() {
		client := testProvider.Meta().(ProviderConfig).client
		_, err := client.Namespaces().Delete(name, nil)
		if err != nil {
			t.Fatalf("error deleting namespace %q: %s", name, err)
		}
	}
}

func testResourceNamespace_updateConfig(name string) string {
	return fmt.Sprintf(`
resource "nomad_namespace" "test" {
  name = "%s"
  description = "An updated Terraform acctest namespace"
}
`, name)
}

func testResourceNamespace_updateCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const (
			description = "An updated Terraform acctest namespace"
		)
		resourceState := s.Modules[0].Resources["nomad_namespace.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}

		if instanceState.ID != name {
			return fmt.Errorf("expected ID to be %q, got %q", name, instanceState.ID)
		}

		if instanceState.Attributes["name"] != name {
			return fmt.Errorf("expected name to be %q, is %q in state", name, instanceState.Attributes["name"])
		}

		if instanceState.Attributes["description"] != description {
			return fmt.Errorf("expected description to be %q, is %q in state", description, instanceState.Attributes["description"])
		}

		client := testProvider.Meta().(ProviderConfig).client
		namespace, _, err := client.Namespaces().Info(name, nil)
		if err != nil {
			return fmt.Errorf("error reading back namespace %q: %s", name, err)
		}

		if namespace.Name != name {
			return fmt.Errorf("expected name to be %q, is %q in API", name, namespace.Name)
		}
		if namespace.Description != description {
			return fmt.Errorf("expected description to be %q, is %q in API", description, namespace.Description)
		}

		return nil
	}
}

func testResourceNamespace_nodePoolConfigCheck(name string, expected *api.NamespaceNodePoolConfiguration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState := s.Modules[0].Resources["nomad_namespace.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}

		if instanceState.ID != name {
			return fmt.Errorf("expected ID to be %q, got %q", name, instanceState.ID)
		}

		client := testProvider.Meta().(ProviderConfig).client
		namespace, _, err := client.Namespaces().Info(name, nil)
		if err != nil {
			return fmt.Errorf("error reading back namespace %q: %s", name, err)
		}

		if namespace.Name != name {
			return fmt.Errorf("expected name to be %q, is %q in API", name, namespace.Name)
		}

		npConfig := namespace.NodePoolConfiguration
		if npConfig == nil {
			return errors.New("expected node pool configuration to exist")
		}

		sortNpConfigSets := cmp.Transformer(
			"Sort",
			func(npConfig *api.NamespaceNodePoolConfiguration) *api.NamespaceNodePoolConfiguration {
				allowed := append([]string(nil), npConfig.Allowed...)
				denied := append([]string(nil), npConfig.Denied...)
				sort.Strings(allowed)
				sort.Strings(denied)
				return &api.NamespaceNodePoolConfiguration{
					Default: npConfig.Default,
					Allowed: allowed,
					Denied:  denied,
				}
			},
		)
		if diff := cmp.Diff(npConfig, expected, sortNpConfigSets); diff != "" {
			return fmt.Errorf("node pool configuration mismatch (-want +got):\n%s", diff)
		}

		return nil
	}
}
