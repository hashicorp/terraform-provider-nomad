data "nomad_namespaces" "namespaces" {
}

resource "nomad_acl_policy" "namespace" {
  count = "${length(data.nomad_namespaces.namespaces.namespaces)}"
  name = "namespace-${data.nomad_namespaces.namespaces[count.index]}"
  description = "Write to the namespace ${data.nomad_namespaces.namespaces[count.index]}"
  rules_hcl = <<EOT
namespace "${data.nomad_namespaces.namespaces[count.index]}" {
  policy = "write"
}
EOT
}
