#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

set -e

export NOMAD_TOKEN=00000000-0000-0000-0000-000000000000

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
      echo $! > /tmp/nomad-test.pid'

    # Give some time for the process to initialize
    sleep 10

    retries=30
    while [ $retries -ge 0 ]; do
      echo $NOMAD_TOKEN | nomad acl bootstrap -
      if [ $? -eq 0 ]; then
        break
      fi
      sleep 5
      retries=$(( retries - 1 ))
    done

    # Run hostpath CSI plugin and wait for it to be healthy.
    nomad job run https://raw.githubusercontent.com/hashicorp/nomad/v1.8.0/demo/csi/hostpath/plugin.nomad 1>&2
    echo "Waiting for hostpath CSI plugin to become healthy" 1>&2
    retries=30
    while [ $retries -ge 0 ]; do
        nomad plugin status hostpath \
            | grep -q "Nodes Healthy        = 1" && break
        sleep 2
        retries=$(( retries - 1 ))
    done
    nomad plugin status hostpath 1>&2
fi
