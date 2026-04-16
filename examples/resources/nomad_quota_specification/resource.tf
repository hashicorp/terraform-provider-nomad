resource "nomad_quota_specification" "prod_api" {
  name        = "prod-api"
  description = "Production instances of backend API servers"

  limits {
    region = "global"

    region_limit {
      cpu          = 2400
      cores        = 4
      memory_mb    = 1200
      memory_max_mb = 2400

      devices {
        name  = "nvidia/gpu"
        count = 2
      }

      node_pools {
        node_pool     = "batch"
        cpu           = 800
        cores         = 2
        memory_mb     = 1024
        memory_max_mb = 2048

        devices {
          name  = "fpga"
          count = 1
        }

        storage {
          variables_mb    = 25
          host_volumes_mb = 50
        }
      }

      storage {
        variables_mb    = 500
        host_volumes_mb = 1000
      }
    }
  }
}
