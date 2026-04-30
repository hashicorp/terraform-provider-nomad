data "nomad_datacenters" "datacenters" {
  prefix            = "prod"
  ignore_down_nodes = true
}
