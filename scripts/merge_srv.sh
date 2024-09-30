#!/usr/bin/env bash

# refer to: https://www.unix.com/shell-programming-and-scripting/191029-left-join-using-awk.html

set -xe 

fnm_d=${1:?download file name missing (without number and extension, e.g. file_1 -> file}
fnm_u=${2:?upload file name missing (without number and extension, e.g. file_1 -> file}
fnm_o=${3:?output file name missing}
dir=${4:?input directory missing}
begin=${5:?index to start with}
end=${6:?index to end}

OPTS=""
OPT=""
if [ $(uname) == "Darwin" ]; then
    OPTS=( -i '' -e ) 
    OPT=( -i '' )
else
    OPTS=( -i ) 
    OPT=( -i )
fi

declare -a latency_1
declare -a latency_2

touch $dir/${fnm_o}_rtt.csv && rm $dir/${fnm_o}_rtt.csv

for i in $(seq $begin $end); do
	sed 's/ //g' ${dir}/${fnm_u}_${i}.csv > ${dir}/${fnm_u}_${i}
	sed "${OPTS[@]}" 's/,/ /g' ${dir}/${fnm_u}_${i}
	sed "${OPTS[@]}" 's/New //g' ${dir}/${fnm_u}_${i}
	sed "${OPTS[@]}" 's/  //g' ${dir}/${fnm_u}_${i}
	sed "${OPTS[@]}" 's/\r//' ${dir}/${fnm_u}_${i}
	sed "${OPT[@]}" '1,4d' ${dir}/${fnm_u}_${i}

	sed 's/ //g' ${dir}/${fnm_d}_${i}.csv > ${dir}/${fnm_d}_${i}
	sed "${OPTS[@]}" 's/,/ /g' ${dir}/${fnm_d}_${i}
	sed "${OPTS[@]}" 's/  //g' ${dir}/${fnm_d}_${i}
	sed "${OPTS[@]}" 's/\r//' ${dir}/${fnm_d}_${i}
	sed "${OPT[@]}" '1,4d' ${dir}/${fnm_d}_${i}
	
	# awk 'NR==FNR{ts[$6]=$7;cid[$6]=$5;rid[$6]=$6;next}{print $7 " " $5 " " $6 " " (ts[$6]?ts[$6] " " cid[$6] " " rid[$6]:"missing")}' ${dir}/${fnm_u}${i} ${dir}/${fnm_d}${i} > ${fnm}${i}.csv
	# awk 'NR==FNR{ts[$5]=$6;cid[$5]=$4;next}{print $6 " " $4 " " (ts[$5]?ts[$5] " " cid[$5] : "missing")}' ${dir}/${fnm_u}_${i} ${dir}/${fnm_d}_${i} > ${dir}/${fnm_o}_${i}.csv
	awk 'NR==FNR{ts[$6]=$7;cid[$6]=$5;rid[$6]=$6;next}{print (ts[$6]? $5 " " cid[$6] " " $6 " " rid[$6] " " ($7-ts[$6]) : "missing")}' ${dir}/${fnm_u}_${i} ${dir}/${fnm_d}_${i} > ${dir}/${fnm_o}_${i}.csv
	
	sed "${OPT[@]}" '/missing/d' ${dir}/${fnm_o}_${i}.csv
	latency_1[$((i-1))]=$(awk '{($1==$2)? latency+=$5 : latency+=0; ($1==$2)? cnt++ : cnt+=0  }END{print (cnt!=0)? (1e-6*latency)/cnt : cnt}' ${dir}/${fnm_o}_${i}.csv)
	latency_2[$((i-1))]=$(awk '{($1!=$2)? latency+=$5 : latency+=0; ($1==$2)? cnt++ : cnt+=0 }END{print (cnt!=0)? (1e-6*latency)/cnt : cnt}' ${dir}/${fnm_o}_${i}.csv)

	rm ${dir}/${fnm_u}_${i} ${dir}/${fnm_d}_${i}


	x=$(echo "$i/10" | bc -l)
	
	echo "$x ${latency_1[$((i-1))]} ${latency_2[$((i-1))]}" >> $dir/${fnm_o}_rtt.csv
done
