#!/usr/bin/env bash

set -xe 

fnm=${1:?file name missing (without number and extension, e.g. poa_gaming_1 -> poa_gaming}
dir=${2:?input/output directory}
begin=${3:?index to start with}
end=${4:?index to end}

declare -a lambda
declare -a latency_1
declare -a latency_2

for i in $(seq $begin $end); do
	f=${dir}/${fnm}_${i}
	awk '{print ($2==$4)? $1-$3 " " 0.0 : 0.0 " " $1-$3 }' ${f}.csv > $f
	res=($(awk 'BEGIN{latency_1=0; latency_2=0}{latency_1+=$1; latency_2+=$2}END{print latency_1/NR, latency_2/NR}' $f))
	latency_1[$i]=${res[0]}
	latency_2[$i]=${res[1]}
	lambda[$i]=$(echo "$i/10" | bc -l)
	rm $f
done
echo "${lambda[@]}" > $dir/$fnm.csv
echo "${latency_1[@]}" >> $dir/$fnm.csv
echo "${latency_2[@]}" >> $dir/$fnm.csv
