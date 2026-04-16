# It can sometimes be helpful to wait for a particular plugin to be available
data "nomad_plugin" "ebs" {
  plugin_id        = "aws-ebs0"
  wait_for_healthy = true
}

resource "nomad_csi_volume_registration" "mysql_volume" {
  depends_on = [data.nomad_plugin.ebs]

  plugin_id   = "aws-ebs0"
  volume_id   = "mysql_volume"
  name        = "mysql_volume"
  external_id = module.hashistack.ebs_test_volume_id

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
          rack = "R1"
          zone = "us-east-1a"
        }
      }

      topology {
        segments = {
          rack = "R2"
        }
      }
    }
  }
}
