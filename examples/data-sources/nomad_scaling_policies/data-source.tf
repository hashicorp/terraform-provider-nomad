data "nomad_scaling_policies" "example" {
  job_id = "webapp"
  type   = "horizontal"
}
