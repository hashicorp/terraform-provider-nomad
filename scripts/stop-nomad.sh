#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

if [ -e /tmp/nomad-test.pid ]; then
    echo "Stopping nomad"
    sudo kill "$(cat /tmp/nomad-test.pid)" && sudo rm -f /tmp/nomad-test.pid
fi

rm -f /tmp/nomad-test.token
