// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/hashicorp/nomad/api"
)

func TestResourceQuotaSpecification_import(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckEnt(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceQuotaSpecification_initialConfig(name),
				Check:  testResourceQuotaSpecification_initialCheck(name),
			},
			{
				ResourceName:      "nomad_quota_specification.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},

		CheckDestroy: testResourceQuotaSpecification_checkDestroy(name),
	})
}

func TestResourceQuotaSpecification_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckEnt(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceQuotaSpecification_initialConfig(name),
				Check:  testResourceQuotaSpecification_initialCheck(name),
			},
		},

		CheckDestroy: testResourceQuotaSpecification_checkDestroy(name),
	})
}

func TestResourceQuotaSpecification_refresh(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckEnt(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceQuotaSpecification_initialConfig(name),
				Check:  testResourceQuotaSpecification_initialCheck(name),
			},

			// This should successfully cause the quota spec to be recreated,
			// testing the Exists function.
			{
				PreConfig: testResourceQuotaSpecification_delete(t, name),
				Config:    testResourceQuotaSpecification_initialConfig(name),
			},
		},
	})
}

func TestResourceQuotaSpecification_nameChange(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	newName := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckEnt(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceQuotaSpecification_initialConfig(name),
				Check:  testResourceQuotaSpecification_initialCheck(name),
			},

			// Change our name
			{
				Config: testResourceQuotaSpecification_updateConfig(newName),
				Check:  testResourceQuotaSpecification_updateCheck(newName),
			},
		},
	})
}

func TestResourceQuotaSpecification_update(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckEnt(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceQuotaSpecification_initialConfig(name),
				Check:  testResourceQuotaSpecification_initialCheck(name),
			},
			{
				Config: testResourceQuotaSpecification_updateConfig(name),
				Check:  testResourceQuotaSpecification_updateCheck(name),
			},
		},
	})
}

func TestResourceQuotaSpecification_allFields(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckEnt(t) },
		Steps: []resource.TestStep{
			{
				Config: testResourceQuotaSpecification_allFieldsConfig(name),
				Check:  testResourceQuotaSpecification_allFieldsCheck(name),
			},
		},
		CheckDestroy: testResourceQuotaSpecification_checkDestroy(name),
	})
}

func testResourceQuotaSpecification_initialConfig(name string) string {
	return fmt.Sprintf(`
resource "nomad_quota_specification" "test" {
  name = "%s"
  description = "A Terraform acctest quota specification"
  limits {
	  region = "global"
	  region_limit {
		  cpu = 2500
	  }
  }
}
`, name)
}

func testResourceQuotaSpecification_initialCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const description = "A Terraform acctest quota specification"
		cpu := 2500
		limits := []*api.QuotaLimit{
			{
				Region: "global",
				RegionLimit: &api.QuotaResources{
					CPU: &cpu,
				},
			},
		}
		resourceState := s.Modules[0].Resources["nomad_quota_specification.test"]
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

		if instanceState.Attributes["limits.#"] != "1" {
			return fmt.Errorf("expected limits.# to be %q, is %q in state", "1", instanceState.Attributes["limits.#"])
		}

		var key, regKey string
		for k := range instanceState.Attributes {
			if !strings.HasPrefix(k, "limits.") {
				continue
			}
			parts := strings.Split(k, ".")
			if len(parts) < 4 {
				continue
			}
			if parts[1] == "#" || parts[3] == "#" {
				continue
			}
			key = parts[1]
			regKey = parts[3]
		}

		if instanceState.Attributes["limits."+key+".region"] != limits[0].Region {
			return fmt.Errorf("expected limits.%s.region to be %q, is %q in state", key, limits[0].Region, instanceState.Attributes["limits."+key+".region"])
		}

		if instanceState.Attributes["limits."+key+".region_limit."+regKey+".cpu"] != strconv.Itoa(*limits[0].RegionLimit.CPU) {
			return fmt.Errorf("expected limits.%s.region_limit.%s.cpu to be %q, is %q in state", key, regKey,
				strconv.Itoa(*limits[0].RegionLimit.CPU),
				instanceState.Attributes["limits."+key+".region_limit."+regKey+".cpu"])
		}

		client := testProvider.Meta().(ProviderConfig).client
		spec, _, err := client.Quotas().Info(name, nil)
		if err != nil {
			return fmt.Errorf("error reading back quota specification %q: %s", name, err)
		}

		if spec.Name != name {
			return fmt.Errorf("expected name to be %q, is %q in API", name, spec.Name)
		}
		if spec.Description != description {
			return fmt.Errorf("expected description to be %q, is %q in API", description, spec.Description)
		}

		if len(spec.Limits) != len(limits) {
			return fmt.Errorf("expected %d limits, is %d in API", len(limits), len(spec.Limits))
		}
		if spec.Limits[0].Region != limits[0].Region {
			return fmt.Errorf("expected limits[0].Region to be %q, is %q in API", limits[0].Region, spec.Limits[0].Region)
		}
		if *spec.Limits[0].RegionLimit.CPU != *limits[0].RegionLimit.CPU {
			return fmt.Errorf("expected limits[0].RegionLimit.CPU to be %d, is %d in API", *limits[0].RegionLimit.CPU, *spec.Limits[0].RegionLimit.CPU)
		}

		return nil
	}
}

func testResourceQuotaSpecification_checkExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client
		spec, _, err := client.Quotas().Info(name, nil)
		if err != nil {
			return fmt.Errorf("error reading back quota specification %q: %s", name, err)
		}
		if spec == nil {
			return fmt.Errorf("no quota specification returned for %q", name)
		}

		return nil
	}
}

func testResourceQuotaSpecification_checkDestroy(name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client
		spec, _, err := client.Quotas().Info(name, nil)
		if err != nil && strings.Contains(err.Error(), "404") || spec == nil {
			return nil
		} else if err != nil {
			return fmt.Errorf("error checking if quota specification %q exists: %s", name, err.Error())
		}
		return fmt.Errorf("quota specification %q has not been deleted.", name)
	}
}

func testResourceQuotaSpecification_delete(t *testing.T, name string) func() {
	return func() {
		client := testProvider.Meta().(ProviderConfig).client
		_, err := client.Quotas().Delete(name, nil)
		if err != nil {
			t.Fatalf("error deleting quota specification %q: %s", name, err)
		}
	}
}

func testResourceQuotaSpecification_updateConfig(name string) string {
	return fmt.Sprintf(`
resource "nomad_quota_specification" "test" {
  name = "%s"
  description = "An updated Terraform acctest quota specification"
  limits {
	  region = "global"
	  region_limit {
		  cpu = 2400
		  memory_mb = 1900
	  }
  }
}
`, name)
}

func testResourceQuotaSpecification_updateCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const description = "An updated Terraform acctest quota specification"
		cpu := 2400
		mem := 1900
		limits := []*api.QuotaLimit{
			{
				Region: "global",
				RegionLimit: &api.QuotaResources{
					CPU:      &cpu,
					MemoryMB: &mem,
				},
			},
		}

		resourceState := s.Modules[0].Resources["nomad_quota_specification.test"]
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

		if instanceState.Attributes["limits.#"] != "1" {
			return fmt.Errorf("expected limits.# to be %q, is %q in state", "1", instanceState.Attributes["limits.#"])
		}

		var key, regKey string
		for k := range instanceState.Attributes {
			if !strings.HasPrefix(k, "limits.") {
				continue
			}
			parts := strings.Split(k, ".")
			if len(parts) < 4 {
				continue
			}
			if parts[1] == "#" || parts[3] == "#" {
				continue
			}
			key = parts[1]
			regKey = parts[3]
		}

		if instanceState.Attributes["limits."+key+".region"] != limits[0].Region {
			return fmt.Errorf("expected limits.%s.region to be %q, is %q in state", key, limits[0].Region, instanceState.Attributes["limits."+key+".region"])
		}

		if instanceState.Attributes["limits."+key+".region_limit."+regKey+".cpu"] != strconv.Itoa(*limits[0].RegionLimit.CPU) {
			return fmt.Errorf("expected limits.%s.region_limit.%s.cpu to be %q, is %q in state", key, regKey,
				strconv.Itoa(*limits[0].RegionLimit.CPU),
				instanceState.Attributes["limits."+key+".region_limit."+regKey+".cpu"])
		}

		if instanceState.Attributes["limits."+key+".region_limit."+regKey+".memory_mb"] != strconv.Itoa(*limits[0].RegionLimit.MemoryMB) {
			return fmt.Errorf("expected limits.%s.region_limit.%s.memory_mb to be %q, is %q in state", key, regKey,
				strconv.Itoa(*limits[0].RegionLimit.MemoryMB),
				instanceState.Attributes["limits."+key+".region_limit."+regKey+".memory_mb"])
		}

		client := testProvider.Meta().(ProviderConfig).client
		spec, _, err := client.Quotas().Info(name, nil)
		if err != nil {
			return fmt.Errorf("error reading back quota specification %q: %s", name, err)
		}

		if spec.Name != name {
			return fmt.Errorf("expected name to be %q, is %q in API", name, spec.Name)
		}
		if spec.Description != description {
			return fmt.Errorf("expected description to be %q, is %q in API", description, spec.Description)
		}

		if len(spec.Limits) != len(limits) {
			return fmt.Errorf("expected %d limits, is %d in API", len(limits), len(spec.Limits))
		}
		if spec.Limits[0].Region != limits[0].Region {
			return fmt.Errorf("expected limits[0].Region to be %q, is %q in API", limits[0].Region, spec.Limits[0].Region)
		}
		if *spec.Limits[0].RegionLimit.CPU != *limits[0].RegionLimit.CPU {
			return fmt.Errorf("expected limits[0].RegionLimit.CPU to be %d, is %d in API", *limits[0].RegionLimit.CPU, *spec.Limits[0].RegionLimit.CPU)
		}
		if *spec.Limits[0].RegionLimit.MemoryMB != *limits[0].RegionLimit.MemoryMB {
			return fmt.Errorf("expected limits[0].RegionLimit.MemoryMB to be %d, is %d in API", *limits[0].RegionLimit.MemoryMB, *spec.Limits[0].RegionLimit.MemoryMB)
		}

		return nil
	}
}

func testResourceQuotaSpecification_allFieldsConfig(name string) string {
	return fmt.Sprintf(`
resource "nomad_quota_specification" "test" {
  name        = "%s"
  description = "A Terraform acctest quota specification with all fields"
  limits {
    region = "global"
    region_limit {
      cpu           = 2500
      cores         = 4
      memory_mb     = 2048
      memory_max_mb = 4096
      secrets_mb    = 512

      devices {
        name  = "gpu"
        count = 2

        constraints {
          ltarget = "${device.attr.driver_version}"
          rtarget = "450.0"
          operand = ">="
        }

        affinities {
          ltarget = "${device.attr.memory}"
          rtarget = "8GB"
          operand = ">="
          weight  = 80
        }
      }

      numa {
        affinity = "require"
      }

      storage {
        variables_mb    = 100
        host_volumes_mb = 500
      }
    }
  }
}
`, name)
}

func testResourceQuotaSpecification_allFieldsCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const description = "A Terraform acctest quota specification with all fields"
		cpu := 2500
		cores := 4
		mem := 2048
		memMax := 4096
		secretsMB := 512
		devCount := uint64(2)
		affWeight := int8(80)
		limits := []*api.QuotaLimit{
			{
				Region: "global",
				RegionLimit: &api.QuotaResources{
					CPU:         &cpu,
					Cores:       &cores,
					MemoryMB:    &mem,
					MemoryMaxMB: &memMax,
					SecretsMB:   &secretsMB,
					Devices: []*api.RequestedDevice{
						{
							Name:  "gpu",
							Count: &devCount,
							Constraints: []*api.Constraint{
								{
									LTarget: "${device.attr.driver_version}",
									RTarget: "450.0",
									Operand: ">=",
								},
							},
							Affinities: []*api.Affinity{
								{
									LTarget: "${device.attr.memory}",
									RTarget: "8GB",
									Operand: ">=",
									Weight:  &affWeight,
								},
							},
						},
					},
					NUMA: &api.NUMAResource{
						Affinity: "require",
					},
					Storage: &api.QuotaStorageResources{
						VariablesMB:   100,
						HostVolumesMB: 500,
					},
				},
			},
		}

		resourceState := s.Modules[0].Resources["nomad_quota_specification.test"]
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
		spec, _, err := client.Quotas().Info(name, nil)
		if err != nil {
			return fmt.Errorf("error reading back quota specification %q: %s", name, err)
		}

		if spec.Name != name {
			return fmt.Errorf("expected name to be %q, is %q in API", name, spec.Name)
		}
		if spec.Description != description {
			return fmt.Errorf("expected description to be %q, is %q in API", description, spec.Description)
		}
		if len(spec.Limits) != len(limits) {
			return fmt.Errorf("expected %d limits, is %d in API", len(limits), len(spec.Limits))
		}

		rl := spec.Limits[0].RegionLimit
		el := limits[0].RegionLimit

		if *rl.CPU != *el.CPU {
			return fmt.Errorf("expected CPU to be %d, got %d", *el.CPU, *rl.CPU)
		}
		if *rl.Cores != *el.Cores {
			return fmt.Errorf("expected Cores to be %d, got %d", *el.Cores, *rl.Cores)
		}
		if *rl.MemoryMB != *el.MemoryMB {
			return fmt.Errorf("expected MemoryMB to be %d, got %d", *el.MemoryMB, *rl.MemoryMB)
		}
		if *rl.MemoryMaxMB != *el.MemoryMaxMB {
			return fmt.Errorf("expected MemoryMaxMB to be %d, got %d", *el.MemoryMaxMB, *rl.MemoryMaxMB)
		}
		if *rl.SecretsMB != *el.SecretsMB {
			return fmt.Errorf("expected SecretsMB to be %d, got %d", *el.SecretsMB, *rl.SecretsMB)
		}
		if len(rl.Devices) != len(el.Devices) {
			return fmt.Errorf("expected %d devices, got %d", len(el.Devices), len(rl.Devices))
		}
		if rl.Devices[0].Name != el.Devices[0].Name {
			return fmt.Errorf("expected device name to be %q, got %q", el.Devices[0].Name, rl.Devices[0].Name)
		}
		if *rl.Devices[0].Count != *el.Devices[0].Count {
			return fmt.Errorf("expected device count to be %d, got %d", *el.Devices[0].Count, *rl.Devices[0].Count)
		}
		if len(rl.Devices[0].Constraints) != 1 {
			return fmt.Errorf("expected 1 device constraint, got %d", len(rl.Devices[0].Constraints))
		}
		if rl.Devices[0].Constraints[0].LTarget != el.Devices[0].Constraints[0].LTarget {
			return fmt.Errorf("expected constraint LTarget to be %q, got %q", el.Devices[0].Constraints[0].LTarget, rl.Devices[0].Constraints[0].LTarget)
		}
		if rl.Devices[0].Constraints[0].RTarget != el.Devices[0].Constraints[0].RTarget {
			return fmt.Errorf("expected constraint RTarget to be %q, got %q", el.Devices[0].Constraints[0].RTarget, rl.Devices[0].Constraints[0].RTarget)
		}
		if rl.Devices[0].Constraints[0].Operand != el.Devices[0].Constraints[0].Operand {
			return fmt.Errorf("expected constraint Operand to be %q, got %q", el.Devices[0].Constraints[0].Operand, rl.Devices[0].Constraints[0].Operand)
		}
		if len(rl.Devices[0].Affinities) != 1 {
			return fmt.Errorf("expected 1 device affinity, got %d", len(rl.Devices[0].Affinities))
		}
		if rl.Devices[0].Affinities[0].LTarget != el.Devices[0].Affinities[0].LTarget {
			return fmt.Errorf("expected affinity LTarget to be %q, got %q", el.Devices[0].Affinities[0].LTarget, rl.Devices[0].Affinities[0].LTarget)
		}
		if rl.Devices[0].Affinities[0].RTarget != el.Devices[0].Affinities[0].RTarget {
			return fmt.Errorf("expected affinity RTarget to be %q, got %q", el.Devices[0].Affinities[0].RTarget, rl.Devices[0].Affinities[0].RTarget)
		}
		if rl.Devices[0].Affinities[0].Operand != el.Devices[0].Affinities[0].Operand {
			return fmt.Errorf("expected affinity Operand to be %q, got %q", el.Devices[0].Affinities[0].Operand, rl.Devices[0].Affinities[0].Operand)
		}
		if *rl.Devices[0].Affinities[0].Weight != *el.Devices[0].Affinities[0].Weight {
			return fmt.Errorf("expected affinity Weight to be %d, got %d", *el.Devices[0].Affinities[0].Weight, *rl.Devices[0].Affinities[0].Weight)
		}
		if rl.NUMA == nil {
			return errors.New("expected NUMA to be set, got nil")
		}
		if rl.NUMA.Affinity != el.NUMA.Affinity {
			return fmt.Errorf("expected NUMA.Affinity to be %q, got %q", el.NUMA.Affinity, rl.NUMA.Affinity)
		}
		if rl.Storage == nil {
			return errors.New("expected Storage to be set, got nil")
		}
		if rl.Storage.VariablesMB != el.Storage.VariablesMB {
			return fmt.Errorf("expected Storage.VariablesMB to be %d, got %d", el.Storage.VariablesMB, rl.Storage.VariablesMB)
		}
		if rl.Storage.HostVolumesMB != el.Storage.HostVolumesMB {
			return fmt.Errorf("expected Storage.HostVolumesMB to be %d, got %d", el.Storage.HostVolumesMB, rl.Storage.HostVolumesMB)
		}

		return nil
	}
}
