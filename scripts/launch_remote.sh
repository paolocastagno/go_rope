#!/usr/bin/env bash
function help()
{
	echo -e "\tUSAGE:n\t\tsim_setup.sh [OPTIONS]\n\tThe script returns the file name used to generate the new NetLogo model."
	echo -e "\tDESCRIPTION:"
	echo -e "\t\t-d duration of the experiment"
	echo -e "\t\t-c path to the NetLogo model experiment configuration file. If -c parameter is not used, the configuration's file name will be used to specify the experiment to run (mandatory)."
	echo -e "\t\t-e name of the experiment to run."
	echo -e "\t\t-o Name of the NetLogo output file. If no name is provided, the name of the experiment is concatenated to the original NetLogo file name."
	exit 1
}
FOLDER="~"
# Read command line input
while getopts d:r:h:p:u:i:f:t: FLG
do
    case "${FLG}" in
        d) DURATION=${OPTARG};;
        r) REQUESTS=${OPTARG};;
        h) HOST=${OPTARG};;
	p) PORT=${OPTARG};;
	u) USR=${OPTARG};;
	i) IMAGES+=(${OPTARG});;
	f) FOLDER=${OPTARG};;
	t) TEMPLATE=${OPTARG};;
	*) help;; 
    esac
done

# If no -h is specified nothing can be done
if [ -z $HOST ];
then 
	help;
else
	if [ ! -z $TEMPLATE ] && [ ! -z $DURATION ] && [ ! -z $REQUESTS ];
	then
		# If both -d and -r are specified
		# Copy the template to /tmp/ directory
		FNM="/tmp/$(basename $TEMPLATE)"
		cp -f $TEMPLATE $FNM
		# Substitue the placeholder for the number of requests per second
		echo -e "Executing:\n\tsed -i 's/<REQUESTSPERSEC>/$REQUESTS/g' $FNM"
		sed -i "s/<REQUESTSPERSEC>/$REQUESTS/g" $FNM
		if [[ $? -ne 0 ]];
		then
			exit $?
		fi
		# Substitue the placeholder for the test duration
		echo -e "Executing:\n\tsed -i 's/<TESTDURATION>/$DURATION/g' $FNM"
		sed -i "s/<TESTDURATION>/$DURATION/g" $FNM
		if [[ $? -ne 0 ]];
		then
			exit $?
		fi
		CMD="scp -B -q"
		if [ ! -z ${PORT+x} ];
		then
			CMD="$CMD -P $PORT"
		fi
		CMD="$CMD $FNM "
		if [ ! -z ${USR+x} ];
		then
			CMD="$CMD$USR@"
		fi
		CMD="$CMD$HOST:$FOLDER/docker-compose.yml"
		echo -e "Executing:\n\t $CMD ..."
		bash -c "$CMD"
		if [[ $? -ne 0 ]];
		then
			exit $?
		fi
	else
		if ( [ -z $DURATION ] && [ ! -z $REQUESTS ] ) || ( [ ! -z $DURATION ] && [ -z $REQUESTS ] );
		then
			help
		fi
	fi
	CMD="ssh -q -t "
	if [ ! -z ${PORT+x} ];
	then
		CMD="$CMD-p $PORT "
	fi
	if [ ! -z ${USR+x} ];
	then
		CMD="$CMD$USR@"
	fi
	# CMD="$CMD$HOST 'cd $FOLDER; docker-compose up --remove-orphans --detach ${IMAGES[@]}'"
	CMD="$CMD$HOST 'cd $FOLDER; docker-compose up --detach ${IMAGES[@]}'"
	echo -e "Executing:\n\t $CMD ..."
	bash -c "$CMD"
	# "$CMD"
	if [[ $? -ne 0 ]];
	then
		exit $?
	fi
fi
