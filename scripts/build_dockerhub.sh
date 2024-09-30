#!/usr/bin/env bash

set -xe

dir=$(cd -P -- "$(dirname -- "$0")" && pwd -P)

# Set optional docker tag in first parameter
TAG=${1:-latest}
echo $TAG

GIT_COMMIT="$TAG-$(git rev-list -1 HEAD)"

DOCKER_BUILDKIT=1 docker build . --build-arg GIT_COMMIT=$GIT_COMMIT --build-arg binary=client -t paolocastagno/rope-client:$TAG
# DOCKER_BUILDKIT=1 docker build . --build-arg GIT_COMMIT=$GIT_COMMIT --build-arg binary=delayProxy -t rope-delay:$TAG
docker push paolocastagno/rope-client:$TAG

DOCKER_BUILDKIT=1 docker build . --build-arg GIT_COMMIT=$GIT_COMMIT --build-arg binary=routing -t paolocastagno/rope-proxy:$TAG
docker push paolocastagno/rope-proxy:$TAG

DOCKER_BUILDKIT=1 docker build . --build-arg GIT_COMMIT=$GIT_COMMIT --build-arg binary=server -t paolocastagno/rope-server:$TAG
docker push paolocastagno/rope-server:$TAG
