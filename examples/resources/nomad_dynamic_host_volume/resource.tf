resource nomad_dynamic_host_volume "example" {
  name      = "example"
  namespace = "prod"
  plugin_id = "mkdir"

  capacity_max = "12 GiB"
  capacity_min = "1.0 GiB"

  capability {
    access_mode     = "single-node-writer"
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
