job "tftest" {
  datacenters = ["dc1"]
  type        = "batch"

  reschedule {
    attempts = 0
  }

  parameterized {
    meta_optional = [
      "provider_branch",
      "consul_version",
      "nomad_version",
      "terraform_version",
      "vault_version"
    ]
  }

  meta {
    provider_branch   = "revert-tf-sdk"
    consul_version    = "1.9.1+ent"
    nomad_version     = "1.0.2+ent"
    terraform_version = "0.14.0"
    vault_version     = "1.6.1+ent"
  }

  group "tftest" {
    restart {
      attempts = 0
    }

    network {
      port "nomad" {
        to = 4646
      }

      port "consul" {
        to = 8500
      }
    }

    volume "tftest" {
      type      = "host"
      read_only = false
      source    = "tftest"
    }

    task "cluster" {
      driver = "docker"

      config {
        image = "tftest:0.0.1"
        args  = ["/bin/bash", "-c", "./scripts/start-nomad.sh && export NOMAD_TOKEN=$(cat /tmp/nomad-test.token) && make testacc 2>&1 | tee -a /var/log/tftest/consul-${NOMAD_META_consul_version}-nomad-${NOMAD_META_nomad_version}-terraform-${NOMAD_META_terraform_version}-vault-${NOMAD_META_vault_version}.log"]

        cap_add = [
          "SYS_ADMIN",
        ]

        ports = ["nomad", "consul"]

        volumes = [
          "local/opt:/opt/hashicorp",
          "local/provider:/root/provider",
          "local/bin:/usr/local/bin",
        ]
      }

      volume_mount {
        volume      = "tftest"
        destination = "/var/log/tftest"
        read_only   = false
      }

      resources {
        cpu    = 2000
        memory = 1024
      }

      artifact {
        source      = "https://releases.hashicorp.com/consul/${NOMAD_META_consul_version}/consul_${NOMAD_META_consul_version}_linux_amd64.zip"
        destination = "local/bin"
      }

      artifact {
        source      = "https://releases.hashicorp.com/nomad/${NOMAD_META_nomad_version}/nomad_${NOMAD_META_nomad_version}_linux_amd64.zip"
        destination = "local/bin"
      }

      artifact {
        source      = "https://releases.hashicorp.com/terraform/${NOMAD_META_terraform_version}/terraform_${NOMAD_META_terraform_version}_linux_amd64.zip"
        destination = "local/bin"
      }

      artifact {
        source      = "https://releases.hashicorp.com/vault/${NOMAD_META_vault_version}/vault_${NOMAD_META_vault_version}_linux_amd64.zip"
        destination = "local/bin"
      }

      artifact {
        source      = "git::https://github.com/hashicorp/terraform-provider-nomad?ref=${NOMAD_META_provider_branch}"
        destination = "local/provider"
      }
    }
  }
}
