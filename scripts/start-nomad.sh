#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

set -ex

NOMAD_TOKEN=00000000-0000-0000-0000-000000000000

function bootstrap_nomad_acl() {
  echo "building nomad terraform provider" 1>&2
  bash -c 'make'

  cat <<EOF >/tmp/nomad-bootstrap.tf
provider "nomad" {
  address = "http://localhost:4646"
}
resource "nomad_acl_bootstrap" "token" {
  # bootstrap_token = "${NOMAD_TOKEN}"
}
output "token" {
  value = nomad_acl_bootstrap.token.bootstrap_token
  sensitive = true
}
EOF

  export TF_CLI_CONFIG_FILE=/tmp/.terraformrc
  cat <<EOF >/tmp/.terraformrc
provider_installation {
  dev_overrides {
    "hashicorp/nomad" = "${GOPATH}/bin/"
  }

  direct {}
}
EOF

  echo "Bootstrapping Nomad ACL system" 1>&2
  export TF_LOG=debug
  cd /tmp && terraform apply -auto-approve
  terraform output -raw token | tee /tmp/nomad-token.txt 1>&2
}

if [ ! -e /tmp/nomad-test.pid ]; then
  cat <<EOF >/tmp/nomad-config.hcl
log_file = "/tmp/nomad.log"

plugin "docker" {
  config {
    allow_privileged = true
  }
}
EOF

  sudo -Eb bash -c 'nomad agent -dev -acl-enabled \
      -data-dir=/tmp/nomad/data \
      -config=/tmp/nomad-config.hcl \
      echo $! > /tmp/nomad-test.pid'

  # Give some time for the process to initialize
  sleep 10

fi

retries=3
while [ $retries -ge 0 ]; do
  if bootstrap_nomad_acl; then
    break
  fi
  sleep 5
  retries=$((retries - 1))
done

export NOMAD_TOKEN
# Run hostpath CSI plugin and wait for it to be healthy.
nomad job run https://raw.githubusercontent.com/hashicorp/nomad/v1.8.0/demo/csi/hostpath/plugin.nomad 1>&2
echo "Waiting for hostpath CSI plugin to become healthy" 1>&2
retries=30
while [ $retries -ge 0 ]; do
  nomad plugin status hostpath |
    grep -q "Nodes Healthy        = 1" && break
  sleep 2
  retries=$((retries - 1))
done
nomad plugin status hostpath 1>&2
