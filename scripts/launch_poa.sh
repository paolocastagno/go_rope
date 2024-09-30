#!/usr/bin/env bash
#
set -xe

step=60
tocken=HvcTi0wcDLWGRQ9btL9U3dBTN6hjZT0Jde719_uWpQkA8o0KKcr3cuUTDAtXp3doVxkPPmIGk9HagdgCE9CFEg==


for i in {1..9}; do
	./scripts/setup_poa.sh poa client $i $step "\"proxy:4040\"" "\"logger:8086\""  $tocken
	source ./.env
	docker compose up -d logger 
	sleep 20
	docker compose up -d serverlow servermedium serverhigh
        sleep 10
	docker compose up -d proxy
	sleep 40
	if (( $i > 1 )); then
		docker compose up -d client background
	else
		docker compose up -d client
	fi
	sleep 360
	docker compose down	
done
