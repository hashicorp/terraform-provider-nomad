resource "nomad_namespace" "dev" {
  name        = "dev"
  description = "Shared development environment."
  quota       = "dev"
  meta        = {
    owner = "John Doe"
    foo   = "bar"
  }
}

resource "nomad_quota_specification" "web_team" {
  name        = "web-team"
  description = "web team quota"

  limits {
    region = "global"

    region_limit {
      cpu       = 1000
      memory_mb = 256
    }
  }
}

resource "nomad_namespace" "web" {
  name        = "web"
  description = "Web team production environment."
  quota       = nomad_quota_specification.web_team.name
}
