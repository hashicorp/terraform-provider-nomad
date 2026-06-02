// Copyright IBM Corp. 2016, 2025
// SPDX-License-Identifier: MPL-2.0

package volumes_test

import (
	"fmt"
	"strings"
	"testing"

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

func TestResourceCSIVolumeRegistration_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories(t),
		PreCheck: func() {
			testutil.TestAccPreCheck(t)
			testCheckCSIPluginAvailable(t, "hostpath-plugin0")
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "nomad_csi_volume" "prereq" {
  plugin_id    = "hostpath-plugin0"
  volume_id    = "mysql_volume_reg_prereq"
  name         = "mysql_volume_reg_prereq"
  capacity_min = "10GiB"
  capacity_max = "20GiB"

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
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

resource "nomad_csi_volume_registration" "test" {
  plugin_id   = "hostpath-plugin0"
  volume_id   = "mysql_volume_reg"
  name        = "mysql_volume_reg"
  external_id = nomad_csi_volume.prereq.volume_id

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
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nomad_csi_volume_registration.test", "name", "mysql_volume_reg"),
					resource.TestCheckResourceAttr("nomad_csi_volume_registration.test", "namespace", "default"),
					resource.TestCheckResourceAttr("nomad_csi_volume_registration.test", "plugin_id", "hostpath-plugin0"),
					resource.TestCheckResourceAttr("nomad_csi_volume_registration.test", "external_id", "mysql_volume_reg_prereq"),
					resource.TestCheckResourceAttr("nomad_csi_volume_registration.test", "mount_options.0.fs_type", "ext4"),
					resource.TestCheckResourceAttr("nomad_csi_volume_registration.test", "deregister_on_destroy", "true"),
					testCSIVolumeRegistrationAPICheck(t, "mysql_volume_reg"),
				),
			},
		},

		CheckDestroy: testCSIVolumeRegistrationCheckDestroy(t),
	})
}

func testCSIVolumeRegistrationAPICheck(t *testing.T, volumeID string) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		providerData := testutil.SDKV2ProviderMeta(t)()
		providerConfig, ok := providerData.(nomad.ProviderConfig)
		must.True(t, ok, must.Sprintf("expected nomad.ProviderConfig, got %T", providerData))
		client := providerConfig.Client()
		volume, _, err := client.CSIVolumes().Info(volumeID, nil)
		must.NoError(t, err, must.Sprintf("failed to read volume %s", volumeID))

		test.Eq(t, "mysql_volume_reg", volume.Name)
		test.Eq(t, "default", volume.Namespace)
		test.Eq(t, "hostpath-plugin0", volume.PluginID)
		test.Eq(t, "mysql_volume_reg_prereq", volume.ExternalID)

		return nil
	}
}

func TestResourceCSIVolumeRegistration_secretsWO(t *testing.T) {
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
				Config: testCSIVolumeRegistrationConfigSecretsWO(`{"key1":"initial_secret"}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nomad_csi_volume_registration.test",
						tfjsonpath.New("secrets_wo_version"),
						knownvalue.Int64Exact(1),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("nomad_csi_volume_registration.test", "secrets_wo"),
				),
			},
			{
				Config: testCSIVolumeRegistrationConfigSecretsWO(`{"key1":"updated_secret"}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nomad_csi_volume_registration.test",
						tfjsonpath.New("secrets_wo_version"),
						knownvalue.Int64Exact(2),
					),
				},
			},
			{
				// Same secret value — version should NOT bump.
				Config: testCSIVolumeRegistrationConfigSecretsWO(`{"key1":"updated_secret"}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("nomad_csi_volume_registration.test",
						tfjsonpath.New("secrets_wo_version"),
						knownvalue.Int64Exact(2),
					),
				},
			},
		},

		CheckDestroy: testCSIVolumeRegistrationCheckDestroy(t),
	})
}

func testCSIVolumeRegistrationConfigSecretsWO(secretsJSON string) string {
	return fmt.Sprintf(`
resource "nomad_csi_volume" "prereq_wo" {
  plugin_id    = "hostpath-plugin0"
  volume_id    = "mysql_volume_reg_wo_prereq"
  name         = "mysql_volume_reg_wo_prereq"
  capacity_min = "10GiB"
  capacity_max = "20GiB"

  capability {
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
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

resource "nomad_csi_volume_registration" "test" {
  plugin_id   = "hostpath-plugin0"
  volume_id   = "mysql_volume_reg_wo"
  name        = "mysql_volume_reg_wo"
  external_id = nomad_csi_volume.prereq_wo.volume_id

  secrets_wo = jsonencode(%s)

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
`, secretsJSON)
}

func testCSIVolumeRegistrationCheckDestroy(t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.Modules[0].Resources {
			if rs.Type != "nomad_csi_volume_registration" {
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
			return fmt.Errorf("volume %q has not been deregistered.", volume.ID)
		}
		return nil
	}
}
