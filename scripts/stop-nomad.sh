#!/bin/bash

[ -e /tmp/vault-test.pid ] && echo "Stopping vault" && kill $(cat /tmp/vault-test.pid)
rm -f /tmp/vault-test.pid
[ -e /tmp/nomad-test.pid ] && echo "Stopping nomad" && kill $(cat /tmp/nomad-test.pid)
rm -f /tmp/nomad-test.pid