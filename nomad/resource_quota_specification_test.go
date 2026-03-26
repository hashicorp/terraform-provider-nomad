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
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

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
				ImportStateVerifyIgnore: []string{
					"limits.0.region_limit.0.cores",
					"limits.0.region_limit.0.memory_mb",
					"limits.0.region_limit.0.memory_max_mb",
					"limits.0.region_limit.0.secrets_mb",
				},
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
      }

			node_pools {
				node_pool     = "batch"
				cpu           = 800
				cores         = 2
				memory_mb     = 1024
				memory_max_mb = 2048
				secrets_mb    = 64

				devices {
					name  = "fpga"
					count = 1
				}

				storage {
					variables_mb    = 25
					host_volumes_mb = 50
				}
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
		nodePoolCPU := 800
		nodePoolCores := 2
		nodePoolMem := 1024
		nodePoolMemMax := 2048
		nodePoolSecrets := 64
		nodePoolDevCount := uint64(1)
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
						},
					},
					Storage: &api.QuotaStorageResources{
						VariablesMB:   100,
						HostVolumesMB: 500,
					},
					NodePools: []*api.NodePoolLimit{
						{
							NodePool:    "batch",
							CPU:         &nodePoolCPU,
							Cores:       &nodePoolCores,
							MemoryMB:    &nodePoolMem,
							MemoryMaxMB: &nodePoolMemMax,
							SecretsMB:   &nodePoolSecrets,
							Devices: []*api.RequestedDevice{
								{
									Name:  "fpga",
									Count: &nodePoolDevCount,
								},
							},
							Storage: &api.QuotaStorageResources{
								VariablesMB:   25,
								HostVolumesMB: 50,
							},
						},
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
		if rl.Storage == nil {
			return errors.New("expected Storage to be set, got nil")
		}
		if rl.Storage.VariablesMB != el.Storage.VariablesMB {
			return fmt.Errorf("expected Storage.VariablesMB to be %d, got %d", el.Storage.VariablesMB, rl.Storage.VariablesMB)
		}
		if rl.Storage.HostVolumesMB != el.Storage.HostVolumesMB {
			return fmt.Errorf("expected Storage.HostVolumesMB to be %d, got %d", el.Storage.HostVolumesMB, rl.Storage.HostVolumesMB)
		}
		if len(rl.NodePools) != len(el.NodePools) {
			return fmt.Errorf("expected %d node pools, got %d", len(el.NodePools), len(rl.NodePools))
		}
		if rl.NodePools[0].NodePool != el.NodePools[0].NodePool {
			return fmt.Errorf("expected node_pool to be %q, got %q", el.NodePools[0].NodePool, rl.NodePools[0].NodePool)
		}
		if *rl.NodePools[0].CPU != *el.NodePools[0].CPU {
			return fmt.Errorf("expected node_pool CPU to be %d, got %d", *el.NodePools[0].CPU, *rl.NodePools[0].CPU)
		}
		if *rl.NodePools[0].Cores != *el.NodePools[0].Cores {
			return fmt.Errorf("expected node_pool Cores to be %d, got %d", *el.NodePools[0].Cores, *rl.NodePools[0].Cores)
		}
		if *rl.NodePools[0].MemoryMB != *el.NodePools[0].MemoryMB {
			return fmt.Errorf("expected node_pool MemoryMB to be %d, got %d", *el.NodePools[0].MemoryMB, *rl.NodePools[0].MemoryMB)
		}
		if *rl.NodePools[0].MemoryMaxMB != *el.NodePools[0].MemoryMaxMB {
			return fmt.Errorf("expected node_pool MemoryMaxMB to be %d, got %d", *el.NodePools[0].MemoryMaxMB, *rl.NodePools[0].MemoryMaxMB)
		}
		if *rl.NodePools[0].SecretsMB != *el.NodePools[0].SecretsMB {
			return fmt.Errorf("expected node_pool SecretsMB to be %d, got %d", *el.NodePools[0].SecretsMB, *rl.NodePools[0].SecretsMB)
		}
		if len(rl.NodePools[0].Devices) != len(el.NodePools[0].Devices) {
			return fmt.Errorf("expected %d node_pool devices, got %d", len(el.NodePools[0].Devices), len(rl.NodePools[0].Devices))
		}
		if rl.NodePools[0].Devices[0].Name != el.NodePools[0].Devices[0].Name {
			return fmt.Errorf("expected node_pool device name to be %q, got %q", el.NodePools[0].Devices[0].Name, rl.NodePools[0].Devices[0].Name)
		}
		if *rl.NodePools[0].Devices[0].Count != *el.NodePools[0].Devices[0].Count {
			return fmt.Errorf("expected node_pool device count to be %d, got %d", *el.NodePools[0].Devices[0].Count, *rl.NodePools[0].Devices[0].Count)
		}
		if rl.NodePools[0].Storage == nil {
			return errors.New("expected node_pool Storage to be set, got nil")
		}
		if rl.NodePools[0].Storage.VariablesMB != el.NodePools[0].Storage.VariablesMB {
			return fmt.Errorf("expected node_pool Storage.VariablesMB to be %d, got %d", el.NodePools[0].Storage.VariablesMB, rl.NodePools[0].Storage.VariablesMB)
		}
		if rl.NodePools[0].Storage.HostVolumesMB != el.NodePools[0].Storage.HostVolumesMB {
			return fmt.Errorf("expected node_pool Storage.HostVolumesMB to be %d, got %d", el.NodePools[0].Storage.HostVolumesMB, rl.NodePools[0].Storage.HostVolumesMB)
		}

		return nil
	}
}
