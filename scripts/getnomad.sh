#!/bin/bash

NOMAD_BINARY=https://releases.hashicorp.com/nomad/1.0.4+ent/nomad_1.0.4+ent_linux_amd64.zip

curl -L $NOMAD_BINARY > nomad.zip
sudo unzip -o nomad.zip -d /usr/local/bin
sudo chmod 0755 /usr/local/bin/nomad
sudo chown root:root /usr/local/bin/nomad

which nomad
nomad -v
