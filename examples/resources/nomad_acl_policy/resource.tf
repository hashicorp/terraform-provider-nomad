resource "nomad_acl_policy" "dev" {
  name        = "dev"
  description = "Submit jobs to the dev environment."
  rules_hcl   = file("${path.module}/dev.hcl")
}

resource "nomad_acl_policy" "dev" {
  name        = "dev"
  description = "Submit jobs to the dev environment."

  rules_hcl = <<EOT
namespace "dev" {
  policy = "write"
}
EOT
}
