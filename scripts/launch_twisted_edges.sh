#!/usr/bin/env bash
#
set -xe

# ./scripts/setup_poa_remote.sh poa_remote 2 '"130.192.212.176:8085"' XJGkvOmlXk98pofGZW6Krnt05-eV9A669E-gcbPJnOgSmd8H8G4MpYZ9_iMB4aR7Y-zC5ysZrbvh99DE1RIr1A== client 60 '"proxy:4040"' 60 '"proxy:4040"' '"130.192.212.176:4343", "193.147.104.34:43001", "130.192.212.176:4444"' "193.147.34:43002"
FLG=${1:?identifier of the experiment}
LAM_BASE=${2:? traffic generated at the clients}
LAM_STEP=${3:? traffic generated at the background}
r_begin=${4:?index to start with}
r_end=${5:?index to end}
begin=${6:?index to start with}
end=${7:?index to end}
sync=${8:?synchronize servers time? (yes/no)}

DHOST_TRN="130.192.212.176"
DHOST_MAD="193.147.104.34"

PATH_TRN="/home/arch/git/RoPE"
PATH_MAD="/home/vincenzo/RoPE"

# command parameters
LOGGER=$DHOST_TRN:8086
TOKEN=Jcwn7Cf9w46D3z5gfYc8XlD6tQWIng5EJWoXUqW-YxprkG-gMEoFR8Sa3L99xkwY_xXCmeoDxY8S8v7btW9GXw==
PDEL_TRN="4444"
PSRV_TRN="4343"
PDEL_MD="43002"
PSRV_MD="43001"

LTY_UPMD="no delay"
LTY_DNMD="7.5ms"

LTY_UPTN="19ms"
LTY_DNTN="30ms"

LTY_UPUPF="0.1ms"
LTY_DNUPF="0.2ms"

# Start LOGGER
ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose up -d LOGGER"

for j in $(seq $r_begin $r_end); do
	if [ $sync = "yes" ]; then
		if (( j == $r_begin )); then
			sudo ntpdate 0.europe.pool.ntp.org
			ssh -p 2280 -t vincenzo@$DHOST_MAD sudo -S ntpdate 0.europe.pool.ntp.org
			ssh -t arch@$DHOST_TRN sudo ntpdate 0.europe.pool.ntp.org
		fi
	fi
	for i in $(seq $begin $end); do
		RPS=$(bc -l <<< $LAM_BASE+$LAM_STEP*$i)
		./scripts/setup_twisted_edges.sh -f $FLG -j $j -i $i -l $LOGGER -T $TOKEN -r $RPS -s $DHOST_MAD -p $PDEL_MD -q $PSRV_MD -S $DHOST_TRN -P $PDEL_TRN -Q $PSRV_TRN -d $LTY_DNMD -U $LTY_UPTN -D $LTY_DNTN -w $LTY_UPUPF -g $LTY_DNUPF
		ssh -t arch@$DHOST_TRN "cd ${PATH_TRN}; ./scripts/setup_poa_remote.sh -f $FLG -j $j -T $APROACH -i $i -l $LOGGER -t $TOKEN -I $cli_id -r $lambda_cli -d $proxy  -R $lambda_bkg -D $proxy -s $srv_trn -s $srv_md0 -s $srv_md1 -n $srv_trn -n $srv_md0 -n $hop_trn -S $srv_md1 -N $srv_md1"
		ssh -p 2280 -t vincenzo@$DHOST_MAD "cd ${PATH_MAD}; ./scripts/setup_poa_remote.sh -f $FLG -j $j -T $APROACH -i $i -l $LOGGER -t $TOKEN -I $cli_id -r $lambda_cli -d $proxy  -R $lambda_bkg -D $proxy -s $srv_trn -s $srv_md0 -s $srv_md1 -n $srv_trn -n $srv_md0 -n $hop_trn -S $srv_md1 -N $srv_md1"
		
		# Start servers
		ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose up -d srv_trn"
		ssh -p 2280 -t vincenzo@$DHOST_MAD "cd $PATH_MAD && docker-compose up -d srv_md"
		# Start routing & delay elements
		ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose up -d hop_trn"
		ssh -p 2280 -t vincenzo@$DHOST_MAD "cd $PATH_MAD && docker-compose up -d hop_md"
		sleep 10
		# Start upfs
		docker-compose up -d upf_trn
		ssh -p 2280 -t vincenzo@$DHOST_MAD "cd $PATH_MAD && docker-compose up -d upf_md"
		sleep 30
		# Start background 
		ssh -p 2280 -t vincenzo@$DHOST_MAD "cd $PATH_MAD && docker-compose up -d bg_md"
		ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose up -d bg_trn"
		# Start client
		docker-compose up -d cli_md
		ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose up -d cli_trn"
		sleep 300
		docker-compose down
		ssh -t arch@$DHOST_TRN "cd $PATH_TRN && docker compose stop cli_trn bg_trn upf_trn hop_trn srv_trn && docker compose rm cli_trn bg_trn upf_trn hop_trn srv_trn"
		ssh -p 2280 -t vincenzo@$DHOST_MAD "cd $PATH_MAD && docker-compose down"
	done
done
