#!/bin/bash

NOMAD_BINARY=https://releases.hashicorp.com/nomad/0.12.2+ent/nomad_0.12.2+ent_linux_amd64.zip

curl -L $NOMAD_BINARY > nomad.zip
sudo unzip -o nomad.zip -d /usr/local/bin
sudo chmod 0755 /usr/local/bin/nomad
sudo chown root:root /usr/local/bin/nomad

which nomad
nomad -v
