#!/bin/bash

export VAULT_TEST_TOKEN=terraform-provider-nomad-token
export VAULT_ADDR=http://localhost:8200

if [ ! -e /tmp/vault-test.pid ]; then
    vault server -dev -dev-root-token-id=$VAULT_TEST_TOKEN > /dev/null 2>&1 &

    VAULT_PID=$!
    echo "Started Vault ($VAULT_PID) on $VAULT_ADDR..."
    echo $VAULT_PID > /tmp/vault-test.pid
fi

if [ ! -e /tmp/nomad-test.pid ]; then
    nomad agent -dev -vault-address=$VAULT_ADDR -vault-token $VAULT_TEST_TOKEN -vault-enabled > /dev/null 2>&1 &
    NOMAD_PID=$!
    echo "Started Nomad ($NOMAD_PID)..."
    echo $NOMAD_PID > /tmp/nomad-test.pid

    # Give some time for the process to initialize
    sleep 5
fi

