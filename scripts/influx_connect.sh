#!/usr/bin/env bash

set -xe
flg_exp=${1:?identifier of the experiment}
type=${2:? type of solution approach}
r_begin=${3:?index to start with}
r_end=${4:?index to end}
begin=${5:?index to start with}
end=${6:?index to end}

token=${7:?influx auth token}

docker compose exec logger influx config create -a -n config  -u http://localhost:8086 --token $token -o unito || true
for j in $(seq $r_begin $r_end); do
	for i in $(seq $begin $end); do
		docker compose exec logger influx bucket create -n measures_${flg_exp}_${type}_${j}_${i} --token $token || true
	done
done
