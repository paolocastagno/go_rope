#!/usr/bin/env bash

# Example: ./stop_remote.sh rope-server lorenzo@130.192.212.176

set -xe

# dir=$(cd -P -- "$(dirname -- "$0")" && pwd -P)

DHOST=${2:?destinanion missing}

ssh -t $DHOST docker compose down