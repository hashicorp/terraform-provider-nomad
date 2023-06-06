#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

if [ -e /tmp/consul-test.pid ]; then
    echo "Stopping consul"
    kill "$(cat /tmp/consul-test.pid)" && rm -f /tmp/consul-test.pid
fi
if [ -e /tmp/vault-test.pid ]; then
    echo "Stopping vault"
    kill "$(cat /tmp/vault-test.pid)" && rm -f /tmp/vault-test.pid
fi
if [ -e /tmp/nomad-test.pid ]; then
    echo "Stopping nomad"
    sudo kill "$(cat /tmp/nomad-test.pid)" && sudo rm -f /tmp/nomad-test.pid
fi
if [ -e /tmp/nomad-debug.pid ]; then
    echo "Stopping nomad debug"
    sudo kill "$(cat /tmp/nomad-debug.pid)" && sudo rm -f /tmp/nomad-debug.pid
fi

rm -f /tmp/nomad-test.token
