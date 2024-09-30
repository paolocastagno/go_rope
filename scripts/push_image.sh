#!/usr/bin/env bash

# Example: ./push_image.sh rope-server lorenzo@130.192.212.176

set -xe

dir=$(cd -P -- "$(dirname -- "$0")" && pwd -P)

IMAGE=${1:?image name missing}
DHOST=${2:?destinanion missing}

docker save $IMAGE | bzip2 | ssh $DHOST 'bunzip2 | docker load'