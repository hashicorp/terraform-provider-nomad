#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


set -e

NOMAD_VERSION='1.7.5'
if [[ -n "$NOMAD_LICENSE" || -n "$NOMAD_LICENSE_PATH" ]]; then
    NOMAD_VERSION=${NOMAD_VERSION}+ent
fi
NOMAD_BINARY=https://releases.hashicorp.com/nomad/${NOMAD_VERSION}/nomad_${NOMAD_VERSION}_linux_amd64.zip

curl -L $NOMAD_BINARY > nomad.zip
sudo unzip -o nomad.zip -d /usr/local/bin
sudo chmod 0755 /usr/local/bin/nomad
sudo chown root:root /usr/local/bin/nomad

which nomad
nomad -v
