resource "nomad_acl_token" "build" {
  type     = "client"
  policies = ["dev"]
}

ephemeral "nomad_acl_token" "build" {
  accessor_id = nomad_acl_token.build.accessor_id
}
