resource "nomad_acl_policy" "my_nomad_acl_policy" {
  name        = "my-nomad-acl-policy"
  rules_hcl   = <<EOT
namespace "default" {
  policy       = "read"
  capabilities = ["submit-job"]
}
EOT
}

resource "nomad_acl_role" "my_nomad_acl_role" {
  name        = "my-nomad-acl-role"
  description = "An ACL Role for cluster developers"

  policy {
    name = nomad_acl_policy.my_nomad_acl_policy.name
  }
}
