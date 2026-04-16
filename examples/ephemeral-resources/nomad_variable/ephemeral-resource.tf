resource "nomad_variable" "example" {
  path = "some/path/of/your/choosing"

  items_wo = jsonencode({
    example_key = "example_value"
  })

  items_wo_version = 1
}

ephemeral "nomad_variable" "example" {
  path = nomad_variable.example.path
}

resource "some_resource" "example" {
  secret_value_wo         = ephemeral.nomad_variable.example.items.example_key
  secret_value_wo_version = 1
}
