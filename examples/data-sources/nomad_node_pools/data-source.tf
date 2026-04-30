data "nomad_node_pools" "prod" {
  filter = "Meta.env == \"prod\""
}
