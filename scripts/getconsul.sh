#!/bin/bash

set -e

CONSUL_VERSION=1.9.5+ent
CONSUL_BINARY=https://releases.hashicorp.com/consul/${CONSUL_VERSION}/consul_${CONSUL_VERSION}_linux_amd64.zip

curl -L $CONSUL_BINARY > consul.zip
sudo unzip -o consul.zip -d /usr/local/bin
sudo chmod 0755 /usr/local/bin/consul
sudo chown root:root /usr/local/bin/consul

which consul
consul -v
