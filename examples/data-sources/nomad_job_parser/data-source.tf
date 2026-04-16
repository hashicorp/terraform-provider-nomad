data "nomad_job_parser" "my_job" {
  hcl = file("${path.module}/jobspec.hcl")
  canonicalize = false
}

data "nomad_job_parser" "my_job" {
  hcl = file("${path.module}/jobspec.hcl")

  variables = <<EOT
datacenter = "dc1"
image      = "nginx:latest"
EOT
}
