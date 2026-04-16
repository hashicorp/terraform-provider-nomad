data "nomad_regions" "my_regions" {}

data "template_file" "jobs" {
  count    = length(data.nomad_regions.my_regions.regions)
  template = <<EOT
job "foo" {
  datacenters = ["dc1"]
  type        = "service"
  region      = "$${region}"
  # ... rest of your job here
}
EOT
  vars = {
    region = data.nomad_regions.my_regions.regions[count.index]
  }
}

resource "nomad_job" "app" {
  count   = length(data.nomad_regions.my_regions.regions)
  jobspec = data.template_file.jobs[count.index].rendered
}
