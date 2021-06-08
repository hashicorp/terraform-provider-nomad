#!/bin/bash

set -e

NOMAD_VERSION=1.1.0+ent
NOMAD_BINARY=https://releases.hashicorp.com/nomad/${NOMAD_VERSION}/nomad_${NOMAD_VERSION}_linux_amd64.zip

curl -L $NOMAD_BINARY > nomad.zip
sudo unzip -o nomad.zip -d /usr/local/bin
sudo chmod 0755 /usr/local/bin/nomad
sudo chown root:root /usr/local/bin/nomad

which nomad
nomad -v
