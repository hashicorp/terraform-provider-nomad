#!/bin/bash

VAULT_BINARY=https://releases.hashicorp.com/vault/1.5.0+ent/vault_1.5.0+ent_linux_amd64.zip

curl -L $VAULT_BINARY > vault.zip
sudo unzip -o vault.zip -d /usr/local/bin
sudo chmod 0755 /usr/local/bin/vault
sudo chown root:root /usr/local/bin/vault

which vault
vault -v
