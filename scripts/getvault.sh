#!/bin/bash

set -e

VAULT_VERSION=1.7.2+ent
VAULT_BINARY=https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_amd64.zip

curl -L $VAULT_BINARY > vault.zip
sudo unzip -o vault.zip -d /usr/local/bin
sudo chmod 0755 /usr/local/bin/vault
sudo chown root:root /usr/local/bin/vault

which vault
vault -v
