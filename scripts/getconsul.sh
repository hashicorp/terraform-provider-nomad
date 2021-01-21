#!/bin/bash

CONSUL_BINARY=https://releases.hashicorp.com/consul/1.9.1+ent/consul_1.9.1+ent_linux_amd64.zip

curl -L $CONSUL_BINARY > consul.zip
sudo unzip -o consul.zip -d /usr/local/bin
sudo chmod 0755 /usr/local/bin/consul
sudo chown root:root /usr/local/bin/consul

which consul
consul -v
