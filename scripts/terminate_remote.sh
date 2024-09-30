#!/usr/bin/env bash
function help()
{
	echo -e "\tUSAGE:n\t\tsim_setup.sh [OPTIONS]\n\tThe script returns the file name used to generate the new NetLogo model."
	echo -e "\tDESCRIPTION:"
	echo -e "\t\t-h host address."
	echo -e "\t\t-p ssh port."
	echo -e "\t\t-u user on the remote machine."
	echo -e "\t\t-i docker image's name."
	exit 1
}
FOLDER="~"
# Read command line input
while getopts d:r:h:p:u:i:f:t: FLG
do
    case "${FLG}" in
        h) HOST=${OPTARG};;
	p) PORT=${OPTARG};;
	u) USR=${OPTARG};;
	i) IMAGES+=(${OPTARG});;
	*) help;; 
    esac
done

# If no -h is specified nothing can be done
if [ -z $HOST ];
then 
	help;
else
	CMD="ssh -q -t "
	if [ ! -z ${PORT+x} ];
	then
		CMD="$CMD-p $PORT "
	fi
	if [ ! -z ${USR+x} ];
	then
		CMD="$CMD$USR@"
	fi
	CMD="$CMD$HOST 'docker stop ${IMAGES[@]}'"
	echo -e "Executing:\n\t $CMD ..."
	bash -c "$CMD"
	# "$CMD"
	if [[ $? -ne 0 ]];
	then
		exit $?
	fi
fi
