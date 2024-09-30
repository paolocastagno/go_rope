#!/usr/bin/env bash

# https://github.com/marketplace/actions/build-and-push-docker-images

set -xe

dir=$(cd -P -- "$(dirname -- "$0")" && pwd -P)

# Set optional docker tag in first parameter
TAG=${1:-latest}

USERNAME_DOCKER_HUB="paolocastagno"

docker tag rope-server:$TAG $USERNAME_DOCKER_HUB/rope-server:$TAG
docker tag rope-client:$TAG $USERNAME_DOCKER_HUB/rope-client:$TAG
docker tag rope-proxy:$TAG $USERNAME_DOCKER_HUB/rope-proxy:$TAG

docker push $USERNAME_DOCKER_HUB/rope-server:$TAG
docker push $USERNAME_DOCKER_HUB/rope-client:$TAG
docker push $USERNAME_DOCKER_HUB/rope-proxy:$TAG
