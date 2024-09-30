#!/usr/bin/env bash
function help()
{
	echo -e "\tUSAGE:\n\t\tsim_setup.sh [OPTIONS]\n\tThe script generates the docker-compose .yml configuration file."
	echo -e "\tDESCRIPTION:"
	echo -e "\t\t-t docker-compose template filename."
	echo -e "\t\t-d Experiment duration."
	echo -e "\t\t-r Number of requests/sec generated."
	echo -e "\t\t-c Filename of the proxy's configuration."
	echo -e "\t\t-f Output filename (default docker-compose.yml."
	exit 1
}
FILE="$PWD/docker-compose.yml"
# Read command line input
while getopts t:d:r:i:c:f:h FLG
do
    case "${FLG}" in
	t) TEMPLATE=${OPTARG};;
        d) DURATION=${OPTARG};;
        r) REQUESTS=${OPTARG};;
	c) CFGFILE=${OPTARG};;
	f) FILE=${OPTARG};;
	h) help;; 
	*) help;; 
    esac
done


if [ ! -z $TEMPLATE ] && [ -f "$TEMPLATE" ];
then
	# Copy the template to /tmp/ directory
	FNM="/tmp/$(basename $TEMPLATE)"
	cp -f $TEMPLATE $FNM
       	if [ ! -z $DURATION ] && [ ! -z $REQUESTS ];
	then
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
	fi
       	if [ ! -z $FILE ];
	then
		# Substitue the placeholder for the proxy configuration file
		echo -e "Executing:\n\tsed -i 's/<CONFIGFILE>/$CFGFILE/g' $FNM"
		sed -i "s|<CONFIGFILE>|$CFGFILE|g" $FNM
		if [[ $? -ne 0 ]];
		then
			exit $?
		fi
	fi
	if [ -f "$FILE" ];
	then
		rm $FILE
	fi
	echo -e "Executing\n\tmv $FNM $FILE"
	mv $FNM $FILE
else
	help
fi
