// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceNodePool_import(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.6.0") },
		Steps: []resource.TestStep{
			{
				Config: testResourceNodePoolConfig_basic(name),
				Check:  testResourceNodePoolCheck_basic(name),
			},
			{
				ResourceName:      "nomad_node_pool.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: testResourceNodePool_checkDestroy(name),
	})
}

func TestResourceNodePool_schedulerConfig(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.6.0"); testCheckEnt(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceNodePoolConfig_schedConfig(name),
				Check:  testResourceNodePoolCheck_schedConfig(name),
			},
		},
		CheckDestroy: testResourceNodePool_checkDestroy(name),
	})
}

func TestResourceNodePool_refresh(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.6.0") },
		Steps: []resource.TestStep{
			{
				Config: testResourceNodePoolConfig_basic(name),
				Check:  testResourceNodePoolCheck_basic(name),
			},

			// This should successfully cause the policy to be recreated,
			// testing the Exists function.
			{
				PreConfig: testResourceNodePool_delete(t, name),
				Config:    testResourceNodePoolConfig_basic(name),
			},
		},
		CheckDestroy: testResourceNodePool_checkDestroy(name),
	})
}

func TestResourceNodePool_update(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.6.0"); testCheckEnt(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceNodePoolConfig_schedConfig(name),
				Check:  testResourceNodePoolCheck_schedConfig(name),
			},
			{
				Config: testResourceNodePoolConfig_updated(name),
				Check:  testResourceNodePoolCheck_updated(name),
			},
		},
		CheckDestroy: testResourceNodePool_checkDestroy(name),
	})
}

func TestResourceNodePool_error(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, "1.6.0") },
		Steps: []resource.TestStep{
			{
				Config: `
resource "nomad_node_pool" "empty_name" {
  name = ""
}
`,
				ExpectError: regexp.MustCompile("expected length of name to be in the range"),
			},
			{
				Config: fmt.Sprintf(`
resource "nomad_node_pool" "name_too_long" {
  name = "%s"
}
`, strings.Repeat("A", 200)),
				ExpectError: regexp.MustCompile("expected length of name to be in the range"),
			},
			{
				Config: fmt.Sprintf(`
resource "nomad_node_pool" "desc_too_long" {
  name        = "tf-test-pool"
  description = "%s"
}
`, strings.Repeat("A", 300)),
				ExpectError: regexp.MustCompile("expected length of description to be in the range"),
			},
		},
	})
}

func testResourceNodePoolConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "nomad_node_pool" "test" {
  name              = "%s"
  description       = "Terraform test node pool"
  node_identity_ttl = "168h0m0s"

  meta = {
    test = "true"
  }
}
`, name)
}

func testResourceNodePoolCheck_basic(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState := s.Modules[0].Resources["nomad_node_pool.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}

		if instanceState.ID != name {
			return fmt.Errorf("expected resource ID to be %q, got %q", name, instanceState.ID)
		}

		client := testProvider.Meta().(ProviderConfig).client
		pool, _, err := client.NodePools().Info(name, nil)
		if err != nil {
			return fmt.Errorf("error fetching node pool %q: %v", name, err)
		}

		if pool.Name != name {
			return fmt.Errorf("expected name to be %q, got %q", name, pool.Name)
		}

		expectedDescription := "Terraform test node pool"
		if pool.Description != expectedDescription {
			return fmt.Errorf("expected description to be %q, got %q", expectedDescription, pool.Description)
		}

		expectedMeta := map[string]string{
			"test": "true",
		}
		if diff := cmp.Diff(pool.Meta, expectedMeta); diff != "" {
			return fmt.Errorf("meta mismatch (-want +got):\n%s", diff)
		}

		expectedTTL := "168h0m0s"
		if pool.NodeIdentityTTL.String() != expectedTTL {
			return fmt.Errorf("expected node identity TTL to be %q, got %q", expectedTTL, pool.NodeIdentityTTL.String())
		}

		return nil
	}
}

func testResourceNodePoolConfig_schedConfig(name string) string {
	return fmt.Sprintf(`
resource "nomad_node_pool" "test" {
  name        = "%s"
  description = "Terraform test node pool"

  meta = {
    test = "true"
  }

  scheduler_config {
    scheduler_algorithm     = "spread"
	memory_oversubscription = "enabled"
  }
}
`, name)
}

func testResourceNodePoolCheck_schedConfig(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState := s.Modules[0].Resources["nomad_node_pool.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}

		if instanceState.ID != name {
			return fmt.Errorf("expected resource ID to be %q, got %q", name, instanceState.ID)
		}

		client := testProvider.Meta().(ProviderConfig).client
		pool, _, err := client.NodePools().Info(name, nil)
		if err != nil {
			return fmt.Errorf("error fetching node pool %q: %v", name, err)
		}

		if pool.Name != name {
			return fmt.Errorf("expected name to be %q, got %q", name, pool.Name)
		}

		expectedDescription := "Terraform test node pool"
		if pool.Description != expectedDescription {
			return fmt.Errorf("expected description to be %q, got %q", expectedDescription, pool.Description)
		}

		expectedMeta := map[string]string{
			"test": "true",
		}
		if diff := cmp.Diff(pool.Meta, expectedMeta); diff != "" {
			return fmt.Errorf("meta mismatch (-want +got):\n%s", diff)
		}

		if pool.SchedulerConfiguration == nil {
			return fmt.Errorf("expected node pool to have scheduler configuration")
		}
		schedConfig := pool.SchedulerConfiguration

		expectedSchedAlgo := api.SchedulerAlgorithmSpread
		if schedConfig.SchedulerAlgorithm != expectedSchedAlgo {
			return fmt.Errorf(
				"expected scheduler algorithm to be %q, got %q",
				expectedSchedAlgo,
				schedConfig.SchedulerAlgorithm,
			)
		}

		if schedConfig.MemoryOversubscriptionEnabled == nil || !*schedConfig.MemoryOversubscriptionEnabled {
			return fmt.Errorf("expected memory oversubscription to be enabled")
		}

		return nil
	}
}

func testResourceNodePoolConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "nomad_node_pool" "test" {
  name        = "%s"
  description = "Updated Terraform test node pool"

  scheduler_config {
    scheduler_algorithm = "spread"
  }
}
`, name)
}

func testResourceNodePoolCheck_updated(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState := s.Modules[0].Resources["nomad_node_pool.test"]
		if resourceState == nil {
			return errors.New("resource not found in state")
		}

		instanceState := resourceState.Primary
		if instanceState == nil {
			return errors.New("resource has no primary instance")
		}

		if instanceState.ID != name {
			return fmt.Errorf("expected resource ID to be %q, got %q", name, instanceState.ID)
		}

		client := testProvider.Meta().(ProviderConfig).client
		pool, _, err := client.NodePools().Info(name, nil)
		if err != nil {
			return fmt.Errorf("error fetching node pool %q: %v", name, err)
		}

		if pool.Name != name {
			return fmt.Errorf("expected name to be %q, got %q", name, pool.Name)
		}

		expectedDescription := "Updated Terraform test node pool"
		if pool.Description != expectedDescription {
			return fmt.Errorf("expected description to be %q, got %q", expectedDescription, pool.Description)
		}

		if len(pool.Meta) != 0 {
			return fmt.Errorf("expected meta to be empty")
		}

		if pool.SchedulerConfiguration == nil {
			return fmt.Errorf("expected node pool to have scheduler configuration")
		}
		schedConfig := pool.SchedulerConfiguration

		expectedSchedAlgo := api.SchedulerAlgorithmSpread
		if schedConfig.SchedulerAlgorithm != expectedSchedAlgo {
			return fmt.Errorf(
				"expected scheduler algorithm to be %q, got %q",
				expectedSchedAlgo,
				schedConfig.SchedulerAlgorithm,
			)
		}

		if schedConfig.MemoryOversubscriptionEnabled != nil {
			return fmt.Errorf("expected memory oversubscription to not be set, got %v",
				*schedConfig.MemoryOversubscriptionEnabled)
		}

		return nil
	}
}

func testResourceNodePool_checkDestroy(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client
		pool, _, err := client.NodePools().Info(name, nil)
		if err != nil && strings.Contains(err.Error(), "404") || pool == nil {
			return nil
		}
		return fmt.Errorf("node pool %q not deleted", name)
	}
}

func testResourceNodePool_delete(t *testing.T, name string) func() {
	return func() {
		client := testProvider.Meta().(ProviderConfig).client
		_, err := client.NodePools().Delete(name, nil)
		if err != nil {
			t.Fatalf("error deleting node pool %q: %v", name, err)
		}
	}
}
