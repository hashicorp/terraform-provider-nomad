// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package nomad

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/shoenig/test"
	"github.com/shoenig/test/must"
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

/* unit tests */

func TestCSIErrIsRetryable(t *testing.T) {
	cases := []struct {
		err         string
		isRetryable bool
	}{
		{
			"requested capacity is bad",
			false,
		},
		{
			"LimitBytes cannot be less than other things",
			false,
		},
		{
			"anything else",
			true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.err, func(t *testing.T) {
			result := csiErrIsRetryable(errors.New(tc.err))
			if result != tc.isRetryable {
				t.Errorf("expect: %v, got: %v", tc.isRetryable, result)
			}
		})
	}
}

func TestCapacityStateFunc(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"5", "5 B"},
		{"1kb", "1000 B"},
		{"10kib", "10 KiB"},
		{"5.5 Gib", "5.5 GiB"},
		{"5GB", "4.7 GiB"},
		{"ugh", "ugh"}, // validation happens elsewhere
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			result := capacityStateFunc(tc.in)
			if result != tc.out {
				t.Errorf("expect: %v; got: %v", tc.out, result)
			}
		})
	}
}

func TestCapacityValidate(t *testing.T) {
	cases := []struct {
		in, err string
	}{
		{"5", ""},
		{"5GB", ""},
		{"5 TiB", ""},
		{"nope", `unable to parse "nope"`},
		{"", `unable to parse ""`},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			diags := capacityValidate(tc.in, cty.Path{})
			if tc.err == "" {
				must.Len(t, 0, diags)
			} else {
				must.Len(t, 1, diags)
				d := diags[0]
				must.NoError(t, d.Validate(), must.Sprintf("invalid diag: %+v", d))
				test.Eq(t, diag.Error, d.Severity, test.Sprint("expect Error severity"))
				test.Eq(t, d.Summary, "invalid capacity")
				test.StrContains(t, d.Detail, tc.err)
			}
		})
	}
}

func TestParseCapacity(t *testing.T) {
	cases := []struct {
		name                 string
		capacity             int    // calculated
		inMin, inMax         string // human input
		expectMin, expectMax uint64
		errs                 []string
	}{
		{
			name: "all zeros",
		},
		{
			name:      "good",
			capacity:  7000,
			inMin:     "5kb",
			expectMin: 5000,
			inMax:     "10kb",
			expectMax: 10000,
		},
		{
			name:      "max less than min",
			capacity:  1000,
			inMin:     "5kb",
			expectMin: 5000,
			inMax:     "2kb",
			expectMax: 2000,
			errs:      []string{"capacity_max (2kb) less than capacity_min (5kb)"},
		},
		{
			name:      "max less than current",
			capacity:  7000,
			inMin:     "5kb",
			expectMin: 5000,
			inMax:     "5kb",
			expectMax: 5000,
			errs:      []string{"capacity_max (5kb) less than current real capacity (6.8 KiB)"},
		},
		{
			name:      "max less than current and min",
			capacity:  7000,
			inMin:     "6kb",
			expectMin: 6000,
			inMax:     "5kb",
			expectMax: 5000,
			errs: []string{
				"capacity_max (5kb) less than capacity_min (6kb)",
				"capacity_max (5kb) less than current real capacity (6.8 KiB)",
			},
		},
	}

	// calling both of these is basically a test that their schemas are the same,
	// at least regarding capacity/min/max.
	for name, resourceMethod := range map[string]func() *schema.Resource{
		"csi_volume":              resourceCSIVolume,
		"csi_volume_registration": resourceCSIVolumeRegistration,
	} {
		for _, tc := range cases {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				res := resourceMethod()
				d := res.TestResourceData()

				must.NoError(t, d.Set("capacity", tc.capacity))
				if tc.inMin != "" {
					must.NoError(t, d.Set("capacity_min", tc.inMin))
				}
				if tc.inMax != "" {
					must.NoError(t, d.Set("capacity_max", tc.inMax))
				}

				cmin, cmax, diags := parseCapacity(d)
				test.Eq(t, tc.expectMin, cmin)
				test.Eq(t, tc.expectMax, cmax)
				if len(tc.errs) > 0 {
					var gotErrs []string
					for _, err := range diags {
						test.Eq(t, "invalid capacity value(s)", err.Summary)
						gotErrs = append(gotErrs, err.Detail)
					}
					test.SliceEqFunc(t, tc.errs, gotErrs, func(a, b string) bool {
						return a == b
					})
				}
			})
		}
	}
}

func TestCheckCapacity(t *testing.T) {
	cases := []struct {
		cap, cMin  uint64
		expectWarn bool
	}{
		// no error if either value is zero
		{0, 0, false},
		{0, 1, false},
		{1, 0, false},
		// capacity may be greater than min, ok.
		{2, 1, false},
		// they may be equal, good.
		{1, 1, false},
		// min greater than cap means an expand should have happened
		// on newer nomad versions.
		{1, 2, true},
	}
	for _, tc := range cases {
		name := fmt.Sprintf("%d-%d-%v", tc.cap, tc.cMin, tc.expectWarn)
		t.Run(name, func(t *testing.T) {
			diags := checkCapacity(tc.cap, tc.cMin)
			if tc.expectWarn {
				must.Len(t, 1, diags)
				d := diags[0]
				test.Eq(t, diag.Warning, d.Severity)
				test.Eq(t, "capacity out of requested range", d.Summary)
				test.StrContains(t, d.Detail, "expand operation may not have occurred")
			} else {
				test.Len(t, 0, diags)
			}
		})
	}
}
