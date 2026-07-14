// Copyright IBM Corp. 2024, 2026
// SPDX-License-Identifier: MPL-2.0

package volumes_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-nomad/internal/framework/provider/testutil"
	"github.com/hashicorp/terraform-provider-nomad/nomad"
	"github.com/shoenig/test"
	"github.com/shoenig/test/must"
)

// Testing this resource requires access to a Nomad cluster with CSI plugins
// running. You can follow the instructions in the URL below to get a test
// environment.
//
// https://github.com/hashicorp/nomad/tree/main/demo/csi/hostpath

func testCheckCSIPluginAvailable(t *testing.T, pluginID string) {
	t.Helper()
	providerData := testutil.SDKV2ProviderMeta(t)()
	providerConfig, ok := providerData.(nomad.ProviderConfig)
	must.True(t, ok, must.Sprintf("expected nomad.ProviderConfig, got %T", providerData))
	client := providerConfig.Client()
	plugins, _, err := client.CSIPlugins().List(nil)
	if err != nil {
		t.Skipf("failed to detect CSI plugin %s: %v", pluginID, err)
	}
	if len(plugins) == 0 {
		t.Skipf("no CSI plugins available")
	}

	for _, plugin := range plugins {
		if plugin.ID == pluginID {
			if plugin.ControllersHealthy == 0 {
				t.Skipf("CSI plugin %s has 0 healthy controllers", pluginID)
			}
			if plugin.NodesHealthy == 0 {
				t.Skipf("CSI plugin %s has 0 healthy nodes", pluginID)
			}
			return
		}
	}
}

func TestResourceCSIVolume_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
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

					test.Eq(t, "mysql_volume", instanceState.ID)

					providerData := testutil.SDKV2ProviderMeta(t)()
					providerConfig, ok := providerData.(nomad.ProviderConfig)
					must.True(t, ok, must.Sprintf("expected nomad.ProviderConfig, got %T", providerData))
					client := providerConfig.Client()
					volume, _, err := client.CSIVolumes().Info(instanceState.ID, nil)
					must.NoError(t, err, must.Sprintf("failed to read volume %s", instanceState.ID))

					test.Eq(t, "mysql_volume", volume.Name)
					test.Eq(t, "default", volume.Namespace)
					test.Eq(t, "hostpath-plugin0", volume.PluginID)

					expectedCapacity := int64(10 * 1024 * 1024 * 1024)
					test.Eq(t, expectedCapacity, volume.Capacity)
					test.Eq(t, expectedCapacity, volume.RequestedCapacityMin)
					expectedMaxCapacity := int64(20 * 1024 * 1024 * 1024)
					test.Eq(t, expectedMaxCapacity, volume.RequestedCapacityMax)

					expectedMountOptions := &api.CSIMountOptions{
						FSType: "ext4",
						// mount flags may contain secrets, so they are not
						// returned by the Nomad API, but check if they are at
						// least set.
						MountFlags: []string{"[REDACTED]"},
					}
					must.Eq(t, expectedMountOptions, volume.MountOptions)

					expectedCapabilities := []*api.CSIVolumeCapability{
						{
							AccessMode:     api.CSIVolumeAccessModeSingleNodeWriter,
							AttachmentMode: api.CSIVolumeAttachmentModeFilesystem,
						},
					}
					must.Eq(t, expectedCapabilities, volume.RequestedCapabilities)

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
					must.Eq(t, expectedTopologyRequest, volume.RequestedTopologies)

					expectedTopologies := []*api.CSITopology{
						nil, // not sure why the hostpath plugin returns this nil topology.
						{
							Segments: map[string]string{
								"topology.hostpath.csi/node": "node-0",
							},
						},
					}
					must.Eq(t, expectedTopologies, volume.Topologies)

					return nil
				},
			},
		},

		CheckDestroy: testCSIVolumeCheckDestroy(t),
	})
}

func TestResourceCSIVolume_secretsWO(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testCheckCSIPluginAvailable(t, "hostpath-plugin0")
		},
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testCSIVolumeConfigSecretsWO(`{
					key1 = "initial_secret"
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nomad_csi_volume.test",
						tfjsonpath.New("secrets_wo_version"),
						knownvalue.Int64Exact(1),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("nomad_csi_volume.test", "secrets_wo"),
					testCSIVolumeAPICheck(t, "mysql_volume_wo"),
				),
			},
			{
				Config: testCSIVolumeConfigSecretsWO(`{
					key1 = "updated_secret"
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nomad_csi_volume.test",
						tfjsonpath.New("secrets_wo_version"),
						knownvalue.Int64Exact(2),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("nomad_csi_volume.test", "secrets_wo"),
				),
			},
			{
				// Same secret value, different unrelated field — version should NOT bump.
				Config: testCSIVolumeConfigSecretsWO(`{
					key1 = "updated_secret"
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nomad_csi_volume.test",
						tfjsonpath.New("secrets_wo_version"),
						knownvalue.Int64Exact(2),
					),
				},
			},
		},

		CheckDestroy: testCSIVolumeCheckDestroy(t),
	})
}

func testCSIVolumeConfigSecretsWO(secretsMap string) string {
	return fmt.Sprintf(`
resource "nomad_csi_volume" "test" {
  plugin_id    = "hostpath-plugin0"
  volume_id    = "mysql_volume_wo"
  name         = "mysql_volume_wo"
  capacity_min = "10GiB"
  capacity_max = "20GiB"

  secrets_wo = %s

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
  }

  mount_options {
    fs_type = "ext4"
  }

  topology_request {
    required {
      topology {
        segments = {
          "topology.hostpath.csi/node" = "node-0"
        }
      }
    }
  }
}
`, secretsMap)
}

func testCSIVolumeAPICheck(t *testing.T, volumeID string) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		providerData := testutil.SDKV2ProviderMeta(t)()
		providerConfig, ok := providerData.(nomad.ProviderConfig)
		must.True(t, ok, must.Sprintf("expected nomad.ProviderConfig, got %T", providerData))
		client := providerConfig.Client()
		volume, _, err := client.CSIVolumes().Info(volumeID, nil)
		must.NoError(t, err, must.Sprintf("failed to read volume %s", volumeID))
		test.Eq(t, volumeID, volume.Name)
		return nil
	}
}

func TestResourceCSIVolume_mountFlagsWO(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testCheckCSIPluginAvailable(t, "hostpath-plugin0")
		},
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testCSIVolumeConfigMountFlagsWO(`["ro", "noatime"]`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nomad_csi_volume.test",
						tfjsonpath.New("mount_options").AtSliceIndex(0).AtMapKey("mount_flags_wo_version"),
						knownvalue.Int64Exact(1),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("nomad_csi_volume.test", "mount_options.0.mount_flags_wo"),
					testCSIVolumeAPICheck(t, "mysql_volume_mfwo"),
				),
			},
			{
				Config: testCSIVolumeConfigMountFlagsWO(`["ro", "noatime", "nosuid"]`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nomad_csi_volume.test",
						tfjsonpath.New("mount_options").AtSliceIndex(0).AtMapKey("mount_flags_wo_version"),
						knownvalue.Int64Exact(2),
					),
				},
			},
			{
				// Same flags — version should NOT bump.
				Config: testCSIVolumeConfigMountFlagsWO(`["ro", "noatime", "nosuid"]`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nomad_csi_volume.test",
						tfjsonpath.New("mount_options").AtSliceIndex(0).AtMapKey("mount_flags_wo_version"),
						knownvalue.Int64Exact(2),
					),
				},
			},
		},
		CheckDestroy: testCSIVolumeCheckDestroy(t),
	})
}

func testCSIVolumeConfigMountFlagsWO(flags string) string {
	return fmt.Sprintf(`
resource "nomad_csi_volume" "test" {
  plugin_id    = "hostpath-plugin0"
  volume_id    = "mysql_volume_mfwo"
  name         = "mysql_volume_mfwo"
  capacity_min = "10GiB"
  capacity_max = "20GiB"

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
  }

  mount_options {
    fs_type        = "ext4"
    mount_flags_wo = %s
  }

  topology_request {
    required {
      topology {
        segments = {
          "topology.hostpath.csi/node" = "node-0"
        }
      }
    }
  }
}
`, flags)
}

func TestResourceCSIVolume_updateInPlace(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testCheckCSIPluginAvailable(t, "hostpath-plugin0")
		},
		Steps: []resource.TestStep{
			{
				Config: testCSIVolumeConfigUpdate("mysql_volume_update", "10GiB", "20GiB"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_csi_volume.test", "name", "mysql_volume_update"),
					resource.TestCheckResourceAttr("nomad_csi_volume.test", "capacity_min", "10GiB"),
				),
			},
			{
				// Update capacity (in-place update)
				Config: testCSIVolumeConfigUpdate("mysql_volume_update", "11GiB", "21GiB"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_csi_volume.test", "capacity_min", "11GiB"),
					resource.TestCheckResourceAttr("nomad_csi_volume.test", "capacity_max", "21GiB"),
				),
			},
		},
		CheckDestroy: testCSIVolumeCheckDestroy(t),
	})
}

func TestResourceCSIVolume_secretsWOExplicitVersion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testCheckCSIPluginAvailable(t, "hostpath-plugin0")
		},
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		Steps: []resource.TestStep{
			{
				// Create with explicit version
				Config: testCSIVolumeConfigSecretsWOExplicitVersion(`{
					key1 = "secret_v1"
				}`, 1),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nomad_csi_volume.test",
						tfjsonpath.New("secrets_wo_version"),
						knownvalue.Int64Exact(1),
					),
				},
			},
			{
				// Change secret AND bump version — should trigger update
				Config: testCSIVolumeConfigSecretsWOExplicitVersion(`{
					key1 = "secret_v2"
				}`, 2),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nomad_csi_volume.test",
						tfjsonpath.New("secrets_wo_version"),
						knownvalue.Int64Exact(2),
					),
				},
			},
		},
		CheckDestroy: testCSIVolumeCheckDestroy(t),
	})
}

func TestResourceCSIVolume_replacementTopology(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testCheckCSIPluginAvailable(t, "hostpath-plugin0")
		},
		Steps: []resource.TestStep{
			{
				Config: testCSIVolumeConfigTopology("mysql_volume_topo", `{
					"topology.hostpath.csi/node" = "node-0"
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_csi_volume.test", "volume_id", "mysql_volume_topo"),
				),
			},
			{
				// Changing topology segments triggers replacement
				Config: testCSIVolumeConfigTopology("mysql_volume_topo", `{
					"topology.hostpath.csi/node" = "node-0"
					rack = "R1"
				}`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_csi_volume.test", "volume_id", "mysql_volume_topo"),
				),
			},
		},
		CheckDestroy: testCSIVolumeCheckDestroy(t),
	})
}

func testCSIVolumeConfigUpdate(volumeID, capMin, capMax string) string {
	return fmt.Sprintf(`
resource "nomad_csi_volume" "test" {
  plugin_id    = "hostpath-plugin0"
  volume_id    = %q
  name         = %q
  capacity_min = %q
  capacity_max = %q

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
  }

  mount_options {
    fs_type = "ext4"
  }

  topology_request {
    required {
      topology {
        segments = {
          "topology.hostpath.csi/node" = "node-0"
        }
      }
    }
  }
}
`, volumeID, volumeID, capMin, capMax)
}

func testCSIVolumeConfigSecretsWOExplicitVersion(secretsMap string, version int) string {
	return fmt.Sprintf(`
resource "nomad_csi_volume" "test" {
  plugin_id    = "hostpath-plugin0"
  volume_id    = "mysql_volume_wo_ver"
  name         = "mysql_volume_wo_ver"
  capacity_min = "10GiB"
  capacity_max = "20GiB"

  secrets_wo         = %s
  secrets_wo_version = %d

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
  }

  mount_options {
    fs_type = "ext4"
  }

  topology_request {
    required {
      topology {
        segments = {
          "topology.hostpath.csi/node" = "node-0"
        }
      }
    }
  }
}
`, secretsMap, version)
}

func testCSIVolumeConfigTopology(volumeID, segments string) string {
	return fmt.Sprintf(`
resource "nomad_csi_volume" "test" {
  plugin_id    = "hostpath-plugin0"
  volume_id    = %q
  name         = %q
  capacity_min = "10GiB"
  capacity_max = "20GiB"

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
  }

  mount_options {
    fs_type = "ext4"
  }

  topology_request {
    required {
      topology {
        segments = %s
      }
    }
  }
}
`, volumeID, volumeID, segments)
}

func testCSIVolumeCheckDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.Modules[0].Resources {
			if rs.Type != "nomad_csi_volume" {
				continue
			}
			if rs.Primary == nil {
				continue
			}
			providerData := testutil.SDKV2ProviderMeta(t)()
			providerConfig, ok := providerData.(nomad.ProviderConfig)
			must.True(t, ok, must.Sprintf("expected nomad.ProviderConfig, got %T", providerData))
			client := providerConfig.Client()
			volume, _, err := client.CSIVolumes().Info(rs.Primary.ID, nil)
			if err != nil && strings.Contains(err.Error(), "404") || volume == nil {
				continue
			}
			return fmt.Errorf("volume %q has not been deleted.", volume.ID)
		}
		return nil
	}
}
