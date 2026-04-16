resource "nomad_node_pool" "dev" {
  name        = "dev"
  description = "Nodes for the development environment."

  meta = {
    department = "Engineering"
    env        = "dev"
  }
}
