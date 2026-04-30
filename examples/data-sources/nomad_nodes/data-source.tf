data "nomad_nodes" "all" {}

data "nomad_nodes" "ready" {
  filter = "Status == \"ready\""
}

data "nomad_nodes" "with_details" {
  os        = true
  resources = true
}
