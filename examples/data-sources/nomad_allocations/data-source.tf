data "nomad_allocations" "example" {
  filter = "JobID == \"example\""
}
