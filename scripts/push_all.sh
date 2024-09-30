#!/bin/bash

./scripts/push_docker.sh

# Push immagini su server spagna
./scripts/push_image.sh rope-proxy:latest "monroe@193.147.104.34 -p 2280"
./scripts/push_image.sh rope-server:latest "monroe@193.147.104.34 -p 2280"
./scripts/push_image.sh rope-client:latest "monroe@193.147.104.34 -p 2280"

# Push immagine server unito
./scripts/push_image.sh rope-server:latest "lorenzo@130.192.212.176"
