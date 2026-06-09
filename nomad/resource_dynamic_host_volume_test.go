// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/shoenig/test/must"
	"github.com/shoenig/test/wait"
)

const testResourceNameDynamicHostVolume = "nomad_dynamic_host_volume.test"
const minVersionDHV = "1.10.0"

func TestDynamicHostVolumeCapacityDiffSuppress(t *testing.T) {
	testCases := []struct {
		name     string
		oldValue string
		newValue string
		expect   bool
	}{
		{name: "identical strings", oldValue: "1GiB", newValue: "1GiB", expect: true},
		{name: "equivalent representations", oldValue: "1GiB", newValue: "1.0 GiB", expect: true},
		{name: "equivalent binary/decimal formatting", oldValue: "2048MiB", newValue: "2.0 GiB", expect: true},
		{name: "different byte values", oldValue: "1GiB", newValue: "2GiB", expect: false},
		{name: "invalid new value", oldValue: "1GiB", newValue: "bogus", expect: false},
		{name: "empty values", oldValue: "", newValue: "", expect: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := dynamicHostVolumeCapacityDiffSuppress("", tc.oldValue, tc.newValue, nil)
			must.Eq(t, tc.expect, actual)
		})
	}
}

func TestResourceDynamicHostVolume_import(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, minVersionDHV) },
		Steps: []resource.TestStep{
			{
				Config: testResourceDynamicHostVolume_config(name),
				Check: testResourceDynamicHostVolume_check(
					testResourceNameDynamicHostVolume, name),
			},
			{
				ResourceName:      testResourceNameDynamicHostVolume,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					id, err := testResourceDynamicHostVolume_getStateID(state,
						"nomad_dynamic_host_volume.test")
					return id, err
				},
			},
		},
		CheckDestroy: testResourceDynamicHostVolume_checkDestroy(
			testResourceNameDynamicHostVolume),
	})
}

func TestResourceDynamicHostVolume_refresh(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, minVersionDHV) },
		Steps: []resource.TestStep{
			{
				Config: testResourceDynamicHostVolume_config(name),
				Check: testResourceDynamicHostVolume_check(
					testResourceNameDynamicHostVolume, name),
			},
			{
				// causes volume to be deleted, exercising the "exists" function
				PreConfig: testResourceDynamicHostVolume_delete(t, name),
				Config:    testResourceDynamicHostVolume_config(name),
			},
		},
		CheckDestroy: testResourceDynamicHostVolume_checkDestroy(
			testResourceNameDynamicHostVolume),
	})
}

func TestResourceDynamicHostVolume_update(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, minVersionDHV) },
		Steps: []resource.TestStep{
			{
				Config: testResourceDynamicHostVolume_config(name),
				Check: testResourceDynamicHostVolume_check(
					testResourceNameDynamicHostVolume, name),
			},
			{
				Config: testResourceDynamicHostVolume_update(name),
				Check: testResourceDynamicHostVolume_update_check(
					testResourceNameDynamicHostVolume, name),
			},
		},
		CheckDestroy: testResourceDynamicHostVolume_checkDestroy(
			testResourceNameDynamicHostVolume),
	})
}

// TestResourceDynamicHostVolume_updateSameCapacityDifferentFormat verifies that
// changing only the string representation of capacity (e.g. "1.0 GiB" -> "1GiB")
// does not trigger an update when the byte values are equivalent.
func TestResourceDynamicHostVolume_updateSameCapacityDifferentFormat(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, minVersionDHV) },
		Steps: []resource.TestStep{
			{
				Config: testResourceDynamicHostVolume_capacityConfig(name, "1.0 GiB", "12 GiB"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(testResourceNameDynamicHostVolume, "name", name),
					resource.TestCheckResourceAttr(testResourceNameDynamicHostVolume, "capacity_min_bytes", "1073741824"),
					resource.TestCheckResourceAttr(testResourceNameDynamicHostVolume, "capacity_max_bytes", "12884901888"),
				),
			},
			{
				// Same bytes, different string format — should be a no-op
				Config:   testResourceDynamicHostVolume_capacityConfig(name, "1GiB", "12GiB"),
				PlanOnly: true,
			},
			{
				// Another equivalent format — should also be a no-op
				Config:   testResourceDynamicHostVolume_capacityConfig(name, "1024MiB", "12288MiB"),
				PlanOnly: true,
			},
		},
		CheckDestroy: testResourceDynamicHostVolume_checkDestroy(
			testResourceNameDynamicHostVolume),
	})
}

func TestResourceDynamicHostVolume_updateCapacityChange(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t); testCheckMinVersion(t, minVersionDHV) },
		Steps: []resource.TestStep{
			{
				Config: testResourceDynamicHostVolume_capacityConfig(name, "1GiB", "10GiB"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(testResourceNameDynamicHostVolume, "capacity_min_bytes", "1073741824"),
					resource.TestCheckResourceAttr(testResourceNameDynamicHostVolume, "capacity_max_bytes", "10737418240"),
				),
			},
			{
				// Increase capacity
				Config: testResourceDynamicHostVolume_capacityConfig(name, "2GiB", "20GiB"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(testResourceNameDynamicHostVolume, "capacity_min_bytes", "2147483648"),
					resource.TestCheckResourceAttr(testResourceNameDynamicHostVolume, "capacity_max_bytes", "21474836480"),
				),
			},
			{
				// Revert capacity back to original
				Config: testResourceDynamicHostVolume_capacityConfig(name, "1GiB", "10GiB"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(testResourceNameDynamicHostVolume, "capacity_min_bytes", "1073741824"),
					resource.TestCheckResourceAttr(testResourceNameDynamicHostVolume, "capacity_max_bytes", "10737418240"),
				),
			},
		},
		CheckDestroy: testResourceDynamicHostVolume_checkDestroy(
			testResourceNameDynamicHostVolume),
	})
}

func testResourceDynamicHostVolume_config(name string) string {
	return fmt.Sprintf(`
resource "nomad_dynamic_host_volume" "test" {
  name      = "%s"
  plugin_id = "mkdir"

  capacity_max = "12 GiB"
  capacity_min = "1.0 GiB"

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
  }

  capability {
    access_mode     = "single-node-multi-writer"
    attachment_mode = "file-system"
  }

  constraint {
    attribute = "$${attr.kernel.name}"
    value     = "linux"
  }

  parameters = {
    mode = "0700"
  }

}
`, name)
}

func testResourceDynamicHostVolume_capacityConfig(name, capacityMin, capacityMax string) string {
	return fmt.Sprintf(`
resource "nomad_dynamic_host_volume" "test" {
  name      = "%s"
  plugin_id = "mkdir"

  capacity_min = "%s"
  capacity_max = "%s"

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
  }

  constraint {
    attribute = "$${attr.kernel.name}"
    value     = "linux"
  }
}
`, name, capacityMin, capacityMax)
}

func testResourceDynamicHostVolume_update(name string) string {
	return fmt.Sprintf(`
resource "nomad_dynamic_host_volume" "test" {
  name      = "%s"
  plugin_id = "mkdir"

  capacity_max = "15 GiB"
  capacity_min = "2.0 GiB"

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
  }

  parameters = {
    mode = "0711"
  }

  constraint {
    attribute = "$${attr.kernel.name}"
    value     = "linux"
  }

}
`, name)
}

func testResourceDynamicHostVolume_check(resourceName, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client
		id, err := testResourceDynamicHostVolume_getID(s, resourceName)
		if err != nil {
			return err
		}
		vol, err := getDynamicHostVolume(client, "default", id)
		if err != nil {
			return fmt.Errorf("error checking for dynamic host volume %q: %w", id, err)
		}
		if vol == nil {
			return fmt.Errorf("dynamic host volume %q not created", id)
		}
		if vol.Name != name {
			return fmt.Errorf("wrong name: %s", name)
		}
		if vol.Parameters["mode"] != "0700" {
			return fmt.Errorf("parameters not set: %#v", vol.Parameters)
		}
		if len(vol.Constraints) != 1 || vol.Constraints[0].LTarget != "${attr.kernel.name}" {
			return fmt.Errorf("constraint not set: %#v", vol.Constraints)
		}
		if len(vol.RequestedCapabilities) != 2 ||
			vol.RequestedCapabilities[0].AccessMode != "single-node-writer" ||
			vol.RequestedCapabilities[1].AccessMode != "single-node-multi-writer" {
			return fmt.Errorf("capabilities not set: %#v", vol.RequestedCapabilities)
		}
		if vol.RequestedCapacityMaxBytes != 12884901888 {
			return fmt.Errorf("requested capacity_max not set: %v", vol.RequestedCapacityMaxBytes)
		}

		return nil
	}
}

func testResourceDynamicHostVolume_update_check(resourceName, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client
		id, err := testResourceDynamicHostVolume_getID(s, resourceName)
		if err != nil {
			return err
		}
		vol, err := getDynamicHostVolume(client, "default", id)
		if err != nil {
			return fmt.Errorf("error checking for dynamic host volume %q: %w", id, err)
		}
		if vol == nil {
			return fmt.Errorf("dynamic host volume %q not created", id)
		}
		if vol.Name != name {
			return fmt.Errorf("wrong name: %s", name)
		}
		if vol.Parameters["mode"] != "0711" {
			return fmt.Errorf("parameters not updated: %#v", vol.Parameters)
		}
		if len(vol.RequestedCapabilities) != 1 ||
			vol.RequestedCapabilities[0].AccessMode != "single-node-writer" {
			return fmt.Errorf("capabilities not updated: %#v", vol.RequestedCapabilities)
		}
		if vol.RequestedCapacityMaxBytes != 16106127360 {
			return fmt.Errorf(
				"requested capacity_max not updated: %v", vol.RequestedCapacityMaxBytes)
		}
		if vol.RequestedCapacityMinBytes != 2147483648 {
			return fmt.Errorf(
				"requested capacity_min not updated: %v", vol.RequestedCapacityMinBytes)
		}

		return nil
	}
}

func testResourceDynamicHostVolume_getStateID(s *terraform.State, resourceName string) (string, error) {
	resourceState := s.Modules[0].Resources[resourceName]
	if resourceState == nil {
		return "", errors.New("resource not found in state")
	}

	instanceState := resourceState.Primary
	if instanceState == nil {
		return "", errors.New("resource has no primary instance")
	}
	ns := instanceState.Attributes["namespace"]

	return instanceState.ID + "@" + ns, nil
}

func testResourceDynamicHostVolume_getID(s *terraform.State, resourceName string) (string, error) {
	resourceState := s.Modules[0].Resources[resourceName]
	if resourceState == nil {
		return "", errors.New("resource not found in state")
	}

	instanceState := resourceState.Primary
	if instanceState == nil {
		return "", errors.New("resource has no primary instance")
	}
	return instanceState.ID, nil
}

func testResourceDynamicHostVolume_checkDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testProvider.Meta().(ProviderConfig).client
		id, err := testResourceDynamicHostVolume_getID(s, resourceName)
		if err != nil {
			return err
		}
		vol, err := getDynamicHostVolume(client, "default", id)
		if err != nil {
			return fmt.Errorf("error checking for dynamic host volume %q: %w", id, err)
		}
		if vol != nil {
			return fmt.Errorf("volume %q not deleted", id)
		}

		return nil
	}
}

func testResourceDynamicHostVolume_delete(t *testing.T, name string) func() {
	return func() {
		client := testProvider.Meta().(ProviderConfig).client
		vols, _, err := client.HostVolumes().List(nil, nil)
		if err != nil {
			t.Fatalf("unexpected error when listing volumes: %v", err)
		}

		var nodeID string
		for _, vol := range vols {
			if vol.Name == name {
				nodeID = vol.NodeID
				_, _, err := client.HostVolumes().Delete(
					&api.HostVolumeDeleteRequest{ID: vol.ID},
					&api.WriteOptions{Namespace: vol.Namespace},
				)
				if err != nil {
					t.Fatalf("error deleting volume: %v", err)
				}
				break
			}
		}

		must.Wait(t, wait.InitialSuccess(wait.ErrorFunc(
			func() error {
				node, _, err := client.Nodes().Info(nodeID, nil)
				if err != nil {
					return fmt.Errorf("could not query node: %w", err)
				}
				if _, hasVol := node.HostVolumes[name]; hasVol {
					return fmt.Errorf("node has not removed fingerprint yet")
				}
				return nil
			}),
			wait.Gap(100*time.Millisecond),
			wait.Timeout(10*time.Second),
		))
	}
}
