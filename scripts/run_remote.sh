#!/usr/bin/env bash

# Example: ./run_remote.sh rope-server lorenzo@130.192.212.176 --help

set -xe

# dir=$(cd -P -- "$(dirname -- "$0")" && pwd -P)

SERVICE=${1:?service name missing}
DHOST=${2:?destinanion missing}

# Lauunch service detached
ssh -t $DHOST docker compose up -d $IMAGE