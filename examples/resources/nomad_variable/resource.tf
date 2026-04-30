resource "nomad_variable" "example" {
  path  = "some/path/of/your/choosing"
  items = {
    example_key = "example_value"
  }
}

resource "nomad_variable" "example" {
  path = "some/path/of/your/choosing"

  items_wo = jsonencode({
    example_key = "example_value"
  })

  items_wo_version = 1
}

resource "nomad_namespace" "example" {
  name        = "example"
  description = "Example namespace."
}

resource "nomad_variable" "example" {
  path      = "some/path/of/your/choosing"
  namespace = nomad_namespace.example.name
  items     = {
    example_key = "example_value"
  }
}
