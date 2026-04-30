resource "nomad_acl_token" "dakota" {
  name     = "Dakota"
  type     = "client"
  policies = ["dev", "qa"]
}

resource "nomad_acl_token" "dakota" {
  name     = "Dakota"
  type     = "client"
  policies = ["dev", "qa"]
  global   = true
}

resource "nomad_acl_token" "iman" {
  name = "Iman"
  type = "management"
}

resource "nomad_acl_token" "token" {
  type     = "client"
  policies = ["dev"]
}

output "nomad_token" {
  value = nomad_acl_token.token.secret_id
}
