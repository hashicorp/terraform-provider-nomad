// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Testing this resource requires access to a Nomad cluster with CSI plugins
// running. You can follow the instructions in the URL below to get a test
// environment.
//
// https://github.com/hashicorp/nomad/tree/main/demo/csi/hostpath

func TestResourceCSIVolume_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testProviders,
		PreCheck: func() {
			testAccPreCheck(t)
			testCheckCSIPluginAvailable(t, "hostpath-plugin0")
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "nomad_csi_volume" "test" {
  plugin_id    = "hostpath-plugin0"
  volume_id    = "mysql_volume"
  name         = "mysql_volume"
  capacity_min = "10GiB"
  capacity_max = "20GiB"

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
  }

  mount_options {
    fs_type     = "ext4"
	mount_flags = ["ro", "noatime"]
  }

  topology_request {
    required {
      topology {
        segments = {
          rack = "R1"
          "topology.hostpath.csi/node" = "node-0"
        }
      }

      topology {
        segments = {
          rack = "R2"
        }
      }
    }

    preferred {
      topology {
        segments = {
          zone = "us-east-1a"
        }
      }
    }
  }
}
				`,
				Check: func(s *terraform.State) error {
					resourceState := s.Modules[0].Resources["nomad_csi_volume.test"]
					if resourceState == nil {
						return errors.New("resource not found in state")
					}

					instanceState := resourceState.Primary
					if instanceState == nil {
						return errors.New("resource has no primary instance")
					}

					if instanceState.ID != "mysql_volume" {
						return fmt.Errorf("expected ID to be mysql_volume, got %s", instanceState.ID)
					}

					expected := map[string]string{
						"namespace":                     "default",
						"name":                          "mysql_volume",
						"plugin_id":                     "hostpath-plugin0",
						"capacity_min":                  "10 GiB",
						"capacity_min_bytes":            "10737418240",
						"capacity_max":                  "20 GiB",
						"capacity_max_bytes":            "21474836480",
						"capacity":                      "10737418240",
						"mount_options.#":               "1",
						"mount_options.0.mount_flags.#": "2",
						"mount_options.0.mount_flags.0": "ro",
						"mount_options.0.mount_flags.1": "noatime",
						"mount_options.0.fs_type":       "ext4",
						"topology_request.0.required.0.topology.0.segments.rack":                       "R1",
						"topology_request.0.required.0.topology.0.segments.topology.hostpath.csi/node": "node-0",
						"topology_request.0.required.0.topology.1.segments.rack":                       "R2",
						"topology_request.0.preferred.0.topology.0.segments.zone":                      "us-east-1a",
						"capability.#": "1", // capability is a set, so it's hard to infer their indexes.
					}
					for k, v := range expected {
						got := instanceState.Attributes[k]
						if got != v {
							return fmt.Errorf("expected %s to be %s, got %s", k, v, got)
						}
					}

					client := testProvider.Meta().(ProviderConfig).client
					volume, _, err := client.CSIVolumes().Info(instanceState.ID, nil)
					if err != nil {
						return fmt.Errorf("failed to read volume %s: %v", instanceState.ID, err)
					}

					if volume.Name != expected["name"] {
						return fmt.Errorf("expected Name to be %s, got: %s", expected["name"], volume.Name)
					}
					if volume.Namespace != expected["namespace"] {
						return fmt.Errorf("expected Namespace to be %s, got: %s", expected["namespace"], volume.Namespace)
					}
					if volume.PluginID != expected["plugin_id"] {
						return fmt.Errorf("expected PluginID to be %s, got: %s", expected["plugin_id"], volume.PluginID)
					}

					expectedCapacity := int64(10 * 1024 * 1024 * 1024)
					if volume.Capacity != expectedCapacity {
						return fmt.Errorf("expected Capacity to be %d, got: %d", expectedCapacity, volume.Capacity)
					}
					expectedMinCapacity := int64(10 * 1024 * 1024 * 1024)
					if volume.RequestedCapacityMin != expectedMinCapacity {
						return fmt.Errorf("expected RequestedCapacityMin to be %d, got: %d",
							expectedMinCapacity, volume.RequestedCapacityMin)
					}
					expectedMaxCapacity := int64(20 * 1024 * 1024 * 1024)
					if volume.RequestedCapacityMax != expectedMaxCapacity {
						return fmt.Errorf("expected RequestedCapacityMax to be %d, got: %d",
							expectedMaxCapacity, volume.RequestedCapacityMax)
					}

					expectedMountOptions := &api.CSIMountOptions{
						FSType: "ext4",
						// mount flags may contain secrets, so they are not
						// returned by the Nomad API, but check if they are at
						// least set.
						MountFlags: []string{"[REDACTED]"},
					}
					if diff := cmp.Diff(expectedMountOptions, volume.MountOptions); diff != "" {
						t.Errorf("MountOptions mismatch (-want +got):\n%s", diff)
					}

					expectedCapabilities := []*api.CSIVolumeCapability{
						{
							AccessMode:     api.CSIVolumeAccessModeSingleNodeWriter,
							AttachmentMode: api.CSIVolumeAttachmentModeFilesystem,
						},
					}
					if diff := cmp.Diff(expectedCapabilities, volume.RequestedCapabilities); diff != "" {
						t.Errorf("RequestedCapabilities mismatch (-want +got):\n%s", diff)
					}

					expectedTopologyRequest := &api.CSITopologyRequest{
						Required: []*api.CSITopology{
							{
								Segments: map[string]string{
									"topology.hostpath.csi/node": "node-0",
									"rack":                       "R1",
								},
							},
							{
								Segments: map[string]string{
									"rack": "R2",
								},
							},
						},
						Preferred: []*api.CSITopology{
							{
								Segments: map[string]string{
									"zone": "us-east-1a",
								},
							},
						},
					}
					if diff := cmp.Diff(expectedTopologyRequest, volume.RequestedTopologies); diff != "" {
						t.Errorf("RequestedTopologies mismatch (-want +got):\n%s", diff)
					}

					expectedTopologies := []*api.CSITopology{
						nil, // not sure why the hostpath plugin returns this nil topology.
						{
							Segments: map[string]string{
								"topology.hostpath.csi/node": "node-0",
							},
						},
					}
					if diff := cmp.Diff(expectedTopologies, volume.Topologies); diff != "" {
						t.Errorf("Topologies mismatch (-want +got):\n%s", diff)
					}

					return nil
				},
			},
		},

		CheckDestroy: func(s *terraform.State) error {
			for _, s := range s.Modules[0].Resources {
				if s.Type != "nomad_csi_volume" {
					continue
				}
				if s.Primary == nil {
					continue
				}
				client := testProvider.Meta().(ProviderConfig).client
				volume, _, err := client.CSIVolumes().Info(s.Primary.ID, nil)
				if err != nil && strings.Contains(err.Error(), "404") || volume == nil {
					continue
				}
				return fmt.Errorf("volume %q has not been deleted.", volume.ID)
			}
			return nil
		},
	})
}
