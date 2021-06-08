#!/bin/bash

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
    nomad agent -dev -acl-enabled -vault-address=$VAULT_ADDR -vault-token $VAULT_TEST_TOKEN -vault-enabled -vault-allow-unauthenticated=false > /tmp/nomad.log 2>&1 &
    NOMAD_PID=$!
    echo $NOMAD_PID > /tmp/nomad-test.pid

    # Give some time for the process to initialize
    sleep 10

    http --ignore-stdin POST http://localhost:4646/v1/acl/bootstrap | jq -r '.SecretID' > /tmp/nomad-test.token
    NOMAD_TOKEN=$(cat /tmp/nomad-test.token)
    echo "NOMAD_TOKEN=$(echo $NOMAD_TOKEN)" >> $GITHUB_ENV
elif [ -e /tmp/nomad-test.token ]; then 
  NOMAD_TOKEN=$(cat /tmp/nomad-test.token)
  echo "NOMAD_TOKEN=$(echo $NOMAD_TOKEN)" >> $GITHUB_ENV
fi
