resource "nomad_job" "app" {
  jobspec = file("${path.module}/jobspec.hcl")
}

resource "nomad_job" "app" {
  jobspec = <<EOT
job "foo" {
  datacenters = ["dc1"]
  type        = "service"
  group "foo" {
    task "foo" {
      driver = "raw_exec"
      config {
        command = "/bin/sleep"
        args    = ["1"]
      }

      resources {
        cpu    = 20
        memory = 10
      }

      logs {
        max_files     = 3
        max_file_size = 10
      }
    }
  }
}
EOT
}

resource "nomad_job" "app" {
  jobspec = file("${path.module}/jobspec.json")
  json    = true
}

resource "nomad_job" "app" {
  hcl2 {
    vars = {
      "restart_attempts" = "5",
      "datacenters"      = "[\"dc1\", \"dc2\"]",
    }
  }

  jobspec = <<EOT
variable "datacenters" {
  type = list(string)
}

variable "restart_attempts" {
  type = number
}

job "foo-hcl2" {
  datacenters = var.datacenters

  restart {
    attempts = var.restart_attempts
    ...
  }
  ...
}
EOT
}

resource "random_pet" "random_dc" {}

# This resource will fail to plan because random_pet.random_dc.id is unknown.
resource "nomad_job" "job_with_hcl2" {
  jobspec = <<EOT
variable "datacenter" {
  type = string
}

job "example" {
  datacenters = [var.datacenter]
  ...
}
EOT

  hcl2 {
    vars = {
      datacenter = random_pet.random_dc.id
    }
  }
}

# This will work since Terraform will provide a fully rendered jobspec once it
# knows the value of random_pet.random_dc.id.
resource "nomad_job" "job_with_hcl2" {
  jobspec = <<EOT
job "example" {
  datacenters = ["${random_pet.random_dc.id}"]
  ...
}
EOT
}

resource "nomad_job" "app" {
  jobspec = file("${path.module}/jobspec.hcl")

  hcl2 {
    allow_fs = true
  }
}

# main.tf

data "local_file" "index_html" {
  filename = "${path.module}/index.html"
}

resource "nomad_job" "nginx" {
  jobspec = templatefile("${path.module}/nginx.nomad.tpl", {
    index_html = data.local_file.index_html.content
  })
}

# nginx.nomad.tpl

job "nginx" {
...
      template {
        data        = <<EOF
${index_html}
EOF
        destination = "local/www/index.html"
      }
...
}
