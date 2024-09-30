#!/usr/bin/env bash

# ./scripts/run.sh poa client 2 60 "\"proxy:4040\"" "\"logger:8086\"" XJGkvOmlXk98pofGZW6Krnt05-eV9A669E-gcbPJnOgSmd8H8G4MpYZ9_iMB4aR7Y-zC5ysZrbvh99DE1RIr1A==

set -xe

FLG=${1:?flag missing}
ID=${2:?ID missing}
I=${3:?index missing}
RPS=${4:?requests per second missing}
DST=${5:?destinations missing}
LGR=${6:?loger missing}
TKN=${7:?tocken missing}

SRCDIR=$(pwd)/cfg/${FLG}
DIR=$(pwd)/policy


declare -a LAM
declare -a PSL
declare -a PSM
declare -a PSH

SRC=$(pwd)/cfg/${FLG}/routing/lambda
LAM=($(cat $SRC))
SRC=$(pwd)/cfg/${FLG}/routing/psl
PSL=($(cat $SRC))
SRC=$(pwd)/cfg/${FLG}/routing/psm
PSM=($(cat $SRC))
SRC=$(pwd)/cfg/${FLG}/routing/psh
PSH=($(cat $SRC))

OPTS=""
OPT=""
if [ $(uname) == "Darwin" ]; then
    OPTS=( -i '' -e ) 
    OPT=( -i '' )
else
    OPTS=( -i ) 
    OPT=( -i )
fi

# Creating servers' config files
MOD=server
cp ${SRCDIR}/${MOD}/cfg_*.toml $DIR/$MOD/
# Creating client's config file
MOD=client
FNM=config.json
sed "s/BUCKET/measures_${FLG}_${I}/g" $SRCDIR/${MOD}/cfg.json > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/TOCKEN/$TKN/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/DESTINATIONS/$DST/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/ID/$ID/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/RPS/$RPS/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/LOGGER/$LGR/g" $DIR/$MOD/$FNM
cp ${SRCDIR}/${MOD}/app.toml $DIR/$MOD/
# Creating background's config file
FNM=config-bg.json
sed "s/DESTINATIONS/$DST/g" $SRCDIR/${MOD}/cfg.json > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/ID/${ID}-bg/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/RPS/$((RPS*$((I-1))))/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/toml\",/toml\"/g" $DIR/$MOD/$FNM
sed "${OPT[@]}" '9,17d' $DIR/$MOD/$FNM
# Creating routing's config file
MOD=routing
FNM=config.toml
sed "s/PSL/${PSL[$(I-1)]}/g" ${SRCDIR}/${MOD}/cfg.toml > $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/PSM/${PSM[$(I-1)]}/g" $DIR/$MOD/$FNM
sed "${OPTS[@]}" "s/PSH/${PSH[$(I-1)]}/g" $DIR/$MOD/$FNM

# Creating .env file
# InfluxDB
sed "s/TKN/$TKN/g" $SRCDIR/env > ./.env
sed "${OPTS[@]}" "s/BKT/measures_${FLG}_${I}/g" ./.env
# Client & Background
EPATH=$( echo $DIR/client/app.toml | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/CLI_A/${EPATH}/g" ./.env
sed  "${OPTS[@]}" "s/BG_A/${EPATH}/g" ./.env
EPATH=$( echo $DIR/client/config.json | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/CLI_C/${EPATH}/g" ./.env
EPATH=$( echo $DIR/client/config-bg.json | sed 's/\//\\\//g')
sed  "${OPTS[@]}" "s/BG_C/${EPATH}/g" ./.env
# Proxy
EPATH=$( echo $DIR/$MOD/$FNM | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/PXY_C/${EPATH}/g" ./.env
# Server
EPATH=$( echo $DIR/server/cfg_low.toml | sed 's/\//\\\//g')
sed  "${OPTS[@]}" "s/SRV_L_A/${EPATH}/g" ./.env
EPATH=$( echo $DIR/server/cfg_medium.toml | sed 's/\//\\\//g')
sed "${OPTS[@]}" "s/SRV_M_A/${EPATH}/g" ./.env
EPATH=$( echo $DIR/server/cfg_high.toml | sed 's/\//\\\//g')
sed  "${OPTS[@]}" "s/SRV_H_A/${EPATH}/g" ./.env
