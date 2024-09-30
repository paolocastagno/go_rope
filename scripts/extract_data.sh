#!/usr/bin/env bash
#
set -xe

flg=${1:?identifier of the experiment}
type=${2:? type of solution approach}
r_begin=${3:?index to start with}
r_end=${4:?index to end}
begin=${5:?index to start with}
end=${6:?index to end}
# tkn=${7:?influx token}

buket=measures_$flg_exp

cli=(client srv_md0 srv_md1 serv_trn hop proxy) 
dwn=(received sent sent sent received sent)
up=(sent received received received sent received)
for i in $(seq 0 ${#cli[@]}); do
	echo "Extract data for ${cli[$i]}"
	docker compose exec logger /retrieve.sh -f $flg -t down -s $type -d ${cli[$i]} -e ${dwn[$i]} -n $r_begin -x $r_end -N $begin -X $end -v /res -o res
	docker compose exec logger /retrieve.sh -f $flg -t up   -s $type -d ${cli[$i]} -e ${up[$i]}  -n $r_begin -x $r_end -N $begin -X $end -v /res -o res
done;
