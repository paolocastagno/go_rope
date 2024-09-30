#!/usr/bin/env bash

set -xe

dir=$(cd -P -- "$(dirname -- "$0")" && pwd -P)

docker-compose up --remove-orphans $@