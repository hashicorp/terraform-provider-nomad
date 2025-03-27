// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/shoenig/test/must"
	"github.com/shoenig/test/wait"
)

const testResourceNameDynamicHostVolume = "nomad_dynamic_host_volume.test"
const minVersionDHV = "1.10.0"

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
    access_mode     = "single-node-reader-only"
    attachment_mode = "file-system"
  }

  constraint {
    attribute = "$${attr.kernel.name}"
    value     = "linux"
  }

  parameters = {
    some_key = "some_value"
  }

}
`, name)
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
    some_key     = "some_other_value"
    some_new_key = "some_new_value"
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
		if vol.Parameters["some_key"] != "some_value" {
			return fmt.Errorf("parameters not set: %#v", vol.Parameters)
		}
		if len(vol.Constraints) != 1 || vol.Constraints[0].LTarget != "${attr.kernel.name}" {
			return fmt.Errorf("constraint not set: %#v", vol.Constraints)
		}
		if len(vol.RequestedCapabilities) != 2 ||
			vol.RequestedCapabilities[0].AccessMode != "single-node-writer" {
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
		if vol.Parameters["some_key"] != "some_other_value" {
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
