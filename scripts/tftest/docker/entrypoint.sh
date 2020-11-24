#!/usr/bin/env bash

set -e

terraform --version
nomad --version
consul --version
vault --version

exec "$@"
