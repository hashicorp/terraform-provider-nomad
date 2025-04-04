// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const testResourceNameDynamicHostVolumeRegister = "nomad_dynamic_host_volume_registration.test"

func TestResourceDynamicHostVolumeRegistration_import(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	testAccPreCheck(t) // required to configure provider to get node ID for test config
	nodeID := testResourceDynamicHostVolume_getNodeID(t)

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testCheckMinVersion(t, minVersionDHV) },
		Steps: []resource.TestStep{
			{
				Config: testResourceDynamicHostVolumeRegistration_config(name, nodeID),
				Check: testResourceDynamicHostVolumeRegistration_check(
					testResourceNameDynamicHostVolumeRegister, name),
			},
			{
				ResourceName:      testResourceNameDynamicHostVolumeRegister,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: testResourceDynamicHostVolume_checkDestroy(
			testResourceNameDynamicHostVolumeRegister),
	})
}

func TestResourceDynamicHostVolumeRegistration_refresh(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	testAccPreCheck(t) // required to configure provider to get node ID for test config
	nodeID := testResourceDynamicHostVolume_getNodeID(t)

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testCheckMinVersion(t, minVersionDHV) },
		Steps: []resource.TestStep{
			{
				Config: testResourceDynamicHostVolumeRegistration_config(name, nodeID),
				Check: testResourceDynamicHostVolumeRegistration_check(
					testResourceNameDynamicHostVolumeRegister, name),
			},
			{
				// causes volume to be deleted, exercising the "exists" function
				PreConfig: testResourceDynamicHostVolume_delete(t, name),
				Config:    testResourceDynamicHostVolumeRegistration_config(name, nodeID),
			},
		},
		CheckDestroy: testResourceDynamicHostVolume_checkDestroy(
			testResourceNameDynamicHostVolumeRegister),
	})
}

func TestResourceDynamicHostVolumeRegistration_update(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-nomad-test")
	testAccPreCheck(t) // required to configure provider to get node ID for test config
	nodeID := testResourceDynamicHostVolume_getNodeID(t)

	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testCheckMinVersion(t, minVersionDHV) },
		Steps: []resource.TestStep{
			{
				Config: testResourceDynamicHostVolumeRegistration_config(name, nodeID),
				Check: testResourceDynamicHostVolumeRegistration_check(
					testResourceNameDynamicHostVolumeRegister, name),
			},
			{
				Config: testResourceDynamicHostVolumeRegistration_update(name, nodeID),
				Check: testResourceDynamicHostVolumeRegistration_update_check(
					testResourceNameDynamicHostVolumeRegister, name),
			},
		},
		CheckDestroy: testResourceDynamicHostVolume_checkDestroy(
			testResourceNameDynamicHostVolumeRegister),
	})
}

func testResourceDynamicHostVolumeRegistration_config(name, nodeID string) string {
	return fmt.Sprintf(`
resource "nomad_dynamic_host_volume_registration" "test" {
  name      = "%s"
  capacity  = "12 GiB"
  node_id   = "%s"
  host_path = "/tmp" # not really a good path but lets tests run

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
  }

  capability {
    access_mode     = "single-node-reader-only"
    attachment_mode = "file-system"
  }

  parameters = {
    some_key = "some_value"
  }
}
`, name, nodeID)
}

func testResourceDynamicHostVolumeRegistration_update(name, nodeID string) string {
	return fmt.Sprintf(`
resource "nomad_dynamic_host_volume_registration" "test" {
  name      = "%s"
  capacity  = "15 GiB"
  node_id   = "%s"
  host_path = "/tmp" # not really a good path but lets tests run

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
  }

  parameters = {
    some_key     = "some_other_value"
    some_new_key = "some_new_value"
  }
}
`, name, nodeID)
}

func testResourceDynamicHostVolumeRegistration_check(resourceName, name string) resource.TestCheckFunc {
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
			return fmt.Errorf("dynamic host volume %q not registerd", id)
		}
		if vol.Name != name {
			return fmt.Errorf("wrong name: %s", name)
		}
		if vol.Parameters["some_key"] != "some_value" {
			return fmt.Errorf("parameters not set: %#v", vol.Parameters)
		}
		if len(vol.RequestedCapabilities) != 2 ||
			vol.RequestedCapabilities[0].AccessMode != "single-node-writer" {
			return fmt.Errorf("capabilities not set: %#v", vol.RequestedCapabilities)
		}
		if vol.CapacityBytes != 12884901888 {
			return fmt.Errorf("requested capacity not set: %v", vol.CapacityBytes)
		}

		return nil
	}
}

func testResourceDynamicHostVolumeRegistration_update_check(resourceName, name string) resource.TestCheckFunc {
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
			return fmt.Errorf("dynamic host volume %q not registerd", id)
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
		if vol.CapacityBytes != 16106127360 {
			return fmt.Errorf(
				"requested capacity_max not updated: %v", vol.CapacityBytes)
		}

		return nil
	}
}

func testResourceDynamicHostVolume_getNodeID(t *testing.T) string {
	client := testProvider.Meta().(ProviderConfig).client

	nodes, _, err := client.Nodes().List(nil)
	if err != nil {
		t.Fatalf("unexpected error when listing nodes: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatalf("no nodes")
	}
	return nodes[0].ID
}
