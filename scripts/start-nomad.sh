#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

set -e

export VAULT_TEST_TOKEN=terraform-provider-nomad-token
export VAULT_ADDR=http://localhost:8200

if [ ! -e /tmp/vault-test.pid ]; then
    vault server -dev -dev-root-token-id=$VAULT_TEST_TOKEN > /tmp/vault.log 2>&1 &

    VAULT_PID=$!
    echo $VAULT_PID > /tmp/vault-test.pid
fi

if [ ! -e /tmp/consul-test.pid ]; then
    consul agent -dev > /tmp/consul.log 2>&1 &

    CONSUL_PID=$!
    echo $CONSUL_PID > /tmp/consul-test.pid
fi

if [ ! -e /tmp/nomad-test.pid ]; then
    cat <<EOF > /tmp/nomad-config.hcl
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
      -vault-address=$VAULT_ADDR \
      -vault-token=$VAULT_TEST_TOKEN \
      -vault-enabled \
      -vault-allow-unauthenticated=false & \
      echo $! > /tmp/nomad-test.pid'

    # Give some time for the process to initialize
    sleep 10

    retries=30
    while [ $retries -ge 0 ]; do
      nomad acl bootstrap -json | jq -r '.SecretID' > /tmp/nomad-test.token && break
      sleep 5
      retries=$(( retries - 1 ))
    done
    NOMAD_TOKEN=$(cat /tmp/nomad-test.token)
    export NOMAD_TOKEN
    if [ -z "$NOMAD_TOKEN" ]; then
      echo "Failed to bootstrap Nomad ACL" 1>&2
      exit 1
    fi

    if [ -z "$GITHUB_ENV" ]; then
      echo "$NOMAD_TOKEN"
    else
      echo "NOMAD_TOKEN=$NOMAD_TOKEN" >> "$GITHUB_ENV"
    fi

    # Run hostpath CSI plugin and wait for it to be healthy.
    nomad job run https://raw.githubusercontent.com/hashicorp/nomad/v1.3.1/demo/csi/hostpath/plugin.nomad 1>&2
    echo "Waiting for hostpath CSI plugin to become healthy" 1>&2
    retries=30
    while [ $retries -ge 0 ]; do
        nomad plugin status hostpath \
            | grep -q "Nodes Healthy        = 1" && break
        sleep 2
        retries=$(( retries - 1 ))
    done
    nomad plugin status hostpath 1>&2
elif [ -e /tmp/nomad-test.token ]; then
    NOMAD_TOKEN=$(cat /tmp/nomad-test.token)
    if [ -z "$GITHUB_ENV" ]; then
      echo "$NOMAD_TOKEN"
    else
      echo "NOMAD_TOKEN=$NOMAD_TOKEN" >> "$GITHUB_ENV"
    fi
fi
