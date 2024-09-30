#!/usr/bin/env bash
#
set -xe

# ./scripts/setup_poa_remote.sh poa_remote 2 '"130.192.212.176:8085"' XJGkvOmlXk98pofGZW6Krnt05-eV9A669E-gcbPJnOgSmd8H8G4MpYZ9_iMB4aR7Y-zC5ysZrbvh99DE1RIr1A== client 60 '"proxy:4040"' 60 '"proxy:4040"' '"130.192.212.176:4343", "193.147.104.34:43001", "130.192.212.176:4444"' "193.147.34:43002"
flg_exp=${1:?identifier of the experiment}
type=${2:? type of solution approach}
lambda_cli=${3:? traffic generated at the clients}
lambda_bkg=${4:? traffic generated at the background}
r_begin=${5:?index to start with}
r_end=${6:?index to end}
begin=${7:?index to start with}
end=${8:?index to end}
sync=${9:?synchronize servers time? (yes/no)}

DHOST_TRN="130.192.212.176"
DHOST_MAD="193.147.104.34"

PATH_TRN="/home/arch/git/RoPE"
PATH_MAD="/home/vincenzo/RoPE"

# command parameters
logger=$DHOST_TRN:8086
token=Jcwn7Cf9w46D3z5gfYc8XlD6tQWIng5EJWoXUqW-YxprkG-gMEoFR8Sa3L99xkwY_xXCmeoDxY8S8v7btW9GXw==
cli_id='client'
proxy='"proxy:4040"'
srv_trn=$DHOST_TRN:4343
hop_trn=$DHOST_TRN:4444
srv_md0=$DHOST_MAD:43001
srv_md1=$DHOST_MAD:43002

# ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose up -d logger" 
# ssh -t arch@$DHOST_TRN "cd $PATH_TRN && ./scripts/influx_connect.sh $flg_exp $type $r_begin $r_end $begin $end $token"

for j in $(seq $r_begin $r_end); do
	if [ $sync = "yes" ]; then
		if (( j == $r_begin )); then
			sudo ntpdate 0.europe.pool.ntp.org
			ssh -p 2280 -t vincenzo@$DHOST_MAD sudo -S ntpdate 0.europe.pool.ntp.org
			ssh -t arch@$DHOST_TRN sudo ntpdate 0.europe.pool.ntp.org
		fi
	fi
	for i in $(seq $begin $end); do
		./scripts/setup_poa_remote.sh -f $flg_exp -T $type -j $j -i $i -l $logger -t $token -I $cli_id -r $lambda_cli -d $proxy  -R $lambda_bkg -D $proxy -s $srv_trn -s $srv_md0 -s $srv_md1 -n $srv_trn -n $srv_md0 -n $hop_trn -S $srv_md1 -N $srv_md1
		ssh -t arch@$DHOST_TRN "cd ${PATH_TRN}; ./scripts/setup_poa_remote.sh -f $flg_exp -j $j -T $type -i $i -l $logger -t $token -I $cli_id -r $lambda_cli -d $proxy  -R $lambda_bkg -D $proxy -s $srv_trn -s $srv_md0 -s $srv_md1 -n $srv_trn -n $srv_md0 -n $hop_trn -S $srv_md1 -N $srv_md1"
		ssh -p 2280 -t vincenzo@$DHOST_MAD "cd ${PATH_MAD}; ./scripts/setup_poa_remote.sh -f $flg_exp -j $j -T $type -i $i -l $logger -t $token -I $cli_id -r $lambda_cli -d $proxy  -R $lambda_bkg -D $proxy -s $srv_trn -s $srv_md0 -s $srv_md1 -n $srv_trn -n $srv_md0 -n $hop_trn -S $srv_md1 -N $srv_md1"
		# Start logger
		ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose up -d logger"
		# ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose exec logger influx config create -a -n config  -u http://localhost:8086 --token $token -o unito"
		# ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose exec logger influx bucket create -n measures_${flg_exp}_${type}_${j}_${i} --token $token"
		sleep 10
		ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose up -d srv_trn"
		ssh -p 2280 -t vincenzo@$DHOST_MAD "cd $PATH_MAD && docker-compose up -d srv_md0 srv_md1"
		sleep 10
		ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose up -d hop"
		sleep 10
		docker-compose up -d proxy
		sleep 30
		if (( $i > 2 )); then
			docker-compose up -d cli_tn0 cli_tn1 background
		else
			if (( $i == 1 )); then
				docker-compose up -d cli_tn0
			else
				docker-compose up -d cli_tn0 cli_tn1
			fi
		fi
		sleep 240
		docker-compose down
		ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose down"
		ssh -p 2280 -t vincenzo@$DHOST_MAD "cd $PATH_MAD && docker-compose down"
	done
done
