## How to setup a new policy

## Install go packages
```bash
go mod init pkg
go mod tidy
```

## Build your components
``` bash
go build -ldflags "-w -s" -o build/client client.go Lib.go 
go build -ldflags "-w -s" -o build/proxy proxy.go Lib.go loss.go delayConfig.go
go build -ldflags "-w -s" -o build/server server.go Lib.go
```

```bash
PER IL ROUTING  PROXY:
go build -ldflags "-w -s" -o ../build/routingProxy proxy.go probability.go avoidfull.go meccloud.go meccloudLocal.go meccloudRemoteEstimate.go lennonRemoteEstimate.go forwardingConf.go

PER IL DELAY PROXY:
go build -ldflags "-w -s" -o ../build/delayProxy proxy.go
SERVER:
go build -ldflags "-w -s" -o ../build/server server.go

CLIENT:
go build -ldflags "-w -s" -o ../build/client client.go

```

## How to build containers

## Push containers to remote hosts
```bash
./push_image.sh rope-proxy:latest "user@server -p port"
./push_image.sh rope-server:latest "user@server -p port"
./push_image.sh rope-client:latest "user@server -p port"
```

```bash
./push_image.sh rope-server:latest "user@server"
```

## Setup for local testing
Define your configuration– a Docker compose template is available in templates– and run it locally.


## Install go dependecied
```bash
go get github.com/go-zeromq/zmq4
go install github.com/go-zeromq/zmq4
```

## Build go work
```bash
go work init              
go work use ./client      
go work use ./server      
go work use ./util        
go work use ./routing
go work use ./delayProxy
```

## Build go module
```bash
go mod init client
go mod tidy
```
Replace remote dependence with a local one
```bash
go mod edit -replace github.com/paolocastagno/RoPE/RoPE/util=/Users/paolo/Documents/git/RoPE/RoPE/src/util
```

## InfluxDB
Login to local instance
```bash
influx config create -a -n config-name  -u http://localhost:8086 -p example-user:example-password -o example-org
```
Query for downloading data
```bash
# Get log packets
influx query -r 'from(bucket:"measures") |> range(start: -1d) |> filter(fn: (r) => r["_measurement"] == "packet") |> keep(columns: ["_time","_value","eventType","idDevice","idRequest","timestamp"])'
# Get log ping
influx query 'from(bucket:"measures_poa") |> range(start: -1d) |> filter(fn: (r) => r["_measurement"] == "ping") |> keep(columns: ["direction","rtt"])'

# Get client_123's packet generation time
influx query --raw 'from(bucket:"measures_poa") |> range(start: -5d) |> filter(fn: (r) => r["_measurement"] == "packet") |> filter(fn: (r) => r["_value"] == "New Request") |> keep(columns: ["_time","_value","eventType","idDevice","idRequest","timestamp"])'
# Get Server_high packet arrival time
influx query --raw 'from(bucket:"measures_poa") |> range(start: -5d) |> filter(fn: (r) => r["_measurement"] == "packet") |> filter(fn: (r) => r["idDevice"] == "Server_high") |> filter(fn: (r) => r["_value"] == "Recieved packet") |> keep(columns: ["_time","_value","eventType","idDevice","idRequest","timestamp"])'
```
Create a new bucket with a specific token
```bash
influx bucket create -n bkt_name -o bkt_org --token bkt_token
```
Delete all data in a bucket
```bash
influx config create -a -n config  -u http://localhost:8086 --token $TKN -o unito;

TKN=Jcwn7Cf9w46D3z5gfYc8XlD6tQWIng5EJWoXUqW-YxprkG-gMEoFR8Sa3L99xkwY_xXCmeoDxY8S8v7btW9GXw==
for i in {1..9};do influx delete --org unito --token $TKN --bucket measures_poa_${i} --start 1970-01-01T00:00:00Z --stop $(date +"%Y-%m-%dT%H:%M:%SZ"); echo "$i" done
```

```bash
TKN=Jcwn7Cf9w46D3z5gfYc8XlD6tQWIng5EJWoXUqW-YxprkG-gMEoFR8Sa3L99xkwY_xXCmeoDxY8S8v7btW9GXw==
for i in {1..9};do influx delete --bucket measures_poa_${i} --start 1970-01-01T00:00:00Z --stop $(date +"%Y-%m-%dT%H:%M:%SZ"); echo "done ${i}\n"; done
```
Show authorizations (i.e., find a token)
```bash
influx auth ls
```

Create new buckets
```bash
TKN=Jcwn7Cf9w46D3z5gfYc8XlD6tQWIng5EJWoXUqW-YxprkG-gMEoFR8Sa3L99xkwY_xXCmeoDxY8S8v7btW9GXw==
influx config create -a -n config  -u http://localhost:8086 --token $TKN -o unito;
for i in {1..9}; do influx bucket create -n measures_poa_remote_${i} -o unito --token $TKN; done
```
Export data
Client
```bash
TKN=Jcwn7Cf9w46D3z5gfYc8XlD6tQWIng5EJWoXUqW-YxprkG-gMEoFR8Sa3L99xkwY_xXCmeoDxY8S8v7btW9GXw==;
influx config create -a -n config  -u http://localhost:8086 --token $TKN -o unito;

for i in {1..5};do 
    influx query --raw "from(bucket:\"measures_poa_remote_${i}\") |> range(start: -7d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\") |> filter(fn: (r) => r[\"_value\"] == \"New Request\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" > poa_remote_up_game_${i}.csv;
    influx query --raw "from(bucket:\"measures_poa_remote_${i}\") |> range(start: -7d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\") |> filter(fn: (r) => r[\"_value\"] == \"Response\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" > poa_remote_down_game_${i}.csv;
    echo "${i}"
done;

TKN=Jcwn7Cf9w46D3z5gfYc8XlD6tQWIng5EJWoXUqW-YxprkG-gMEoFR8Sa3L99xkwY_xXCmeoDxY8S8v7btW9GXw==;
influx config create -a -n config  -u http://localhost:8086 --token $TKN -o unito;
for j in {1..3}; do
    for i in {1..5};do 
        influx query --raw "from(bucket:\"measures_poa_remote_${i}\") |> range(start: -7d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\") |> filter(fn: (r) => r[\"_value\"] == \"New Request\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" > poa_remote_up_game_${i}.csv;
        influx query --raw "from(bucket:\"measures_poa_remote_${i}\") |> range(start: -7d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\") |> filter(fn: (r) => r[\"_value\"] == \"Response\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" > poa_remote_down_game_${i}.csv;
        echo "${i}"
    done;
done;
```
srv_trn
```bash
TKN=Jcwn7Cf9w46D3z5gfYc8XlD6tQWIng5EJWoXUqW-YxprkG-gMEoFR8Sa3L99xkwY_xXCmeoDxY8S8v7btW9GXw==;
influx config create -a -n config  -u http://localhost:8086 --token $TKN -o unito;
for i in {1..5};do 
    influx query --raw "from(bucket:\"measures_poa_remote_${i}\") |> range(start: -7d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\") |>  filter(fn: (r) => r[\"idDevice\"] == \"serv_trn\") |> filter(fn: (r) => r[\"eventType\"] == \"dequeued\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" > poa_remote_srvtrn_in_game_${i}.csv;
    influx query --raw "from(bucket:\"measures_poa_remote_${i}\") |> range(start: -7d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\")   |> filter(fn: (r) => r[\"idDevice\"] == \"serv_trn\") |> filter(fn: (r) => r[\"eventType\"] == \"sent\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" > poa_remote_srvtrn_out_game_${i}.csv;
    echo "${i}"
done;
```
srv_md0
```bash
TKN=Jcwn7Cf9w46D3z5gfYc8XlD6tQWIng5EJWoXUqW-YxprkG-gMEoFR8Sa3L99xkwY_xXCmeoDxY8S8v7btW9GXw==;
influx config create -a -n config  -u http://localhost:8086 --token $TKN -o unito;
for i in {1..5};do 
    influx query --raw "from(bucket:\"measures_poa_remote_${i}\") |> range(start: -7d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\") |>  filter(fn: (r) => r[\"idDevice\"] == \"srv_md0\") |> filter(fn: (r) => r[\"eventType\"] == \"dequeued\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" > poa_remote_srvmd0_in_game_${i}.csv;
    influx query --raw "from(bucket:\"measures_poa_remote_${i}\") |> range(start: -7d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\")   |> filter(fn: (r) => r[\"idDevice\"] == \"srv_md0\") |> filter(fn: (r) => r[\"eventType\"] == \"sent\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" > poa_remote_srvmd0_out_game_${i}.csv;
    echo "${i}"
done;
```

srv_md1
```bash
TKN=Jcwn7Cf9w46D3z5gfYc8XlD6tQWIng5EJWoXUqW-YxprkG-gMEoFR8Sa3L99xkwY_xXCmeoDxY8S8v7btW9GXw==;
influx config create -a -n config  -u http://localhost:8086 --token $TKN -o unito;
for i in {1..5};do 
    influx query --raw "from(bucket:\"measures_poa_remote_${i}\") |> range(start: -7d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\") |>  filter(fn: (r) => r[\"idDevice\"] == \"srv_md1\") |> filter(fn: (r) => r[\"eventType\"] == \"dequeued\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" > poa_remote_srvmd1_in_game_${i}.csv;
    influx query --raw "from(bucket:\"measures_poa_remote_${i}\") |> range(start: -7d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\")   |> filter(fn: (r) => r[\"idDevice\"] == \"srv_md1\") |> filter(fn: (r) => r[\"eventType\"] == \"sent\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" > poa_remote_srvmd1_out_game_${i}.csv;
    echo "${i}"
done;
```

Proxy
```bash
TKN=Jcwn7Cf9w46D3z5gfYc8XlD6tQWIng5EJWoXUqW-YxprkG-gMEoFR8Sa3L99xkwY_xXCmeoDxY8S8v7btW9GXw==;
influx config create -a -n config  -u http://localhost:8086 --token $TKN -o unito;
for i in {1..9};do 
    influx query --raw "from(bucket:\"measures_poa_remote_${i}\") |> range(start: -7d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\") |>  filter(fn: (r) => r[\"idDevice\"] == \"RoutingProxy\") |> filter(fn: (r) => r[\"eventType\"] == \"received\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" > poa_remote_pxy_in_game_${i}.csv;
    influx query --raw "from(bucket:\"measures_poa_remote_${i}\") |> range(start: -7d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\")   |> filter(fn: (r) => r[\"idDevice\"] == \"RoutingProxy\") |> filter(fn: (r) => r[\"eventType\"] == \"sent\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" > poa_remote_pxy_out_game_${i}.csv;
    echo "${i}"
done;
```

```bash
cat scenario2_md1 | awk 'BEGIN{}(NR>1){print $3}END{}' > git/RoPE/cfg/poa/routing/psl
cat scenario2_md1 | awk 'BEGIN{}(NR>1){print $4}END{}' > git/RoPE/cfg/poa/routing/psm
cat scenario2_md1 | awk 'BEGIN{}(NR>1){print $5}END{}' > git/RoPE/cfg/poa/routing/psh

cat scenario2_md1 | awk 'BEGIN{}(NR>1){print $7}END{}' > git/RoPE/cfg/poa/routing/psl
cat scenario2_md1 | awk 'BEGIN{}(NR>1){print $8}END{}' > git/RoPE/cfg/poa/routing/psm
cat scenario2_md1 | awk 'BEGIN{}(NR>1){print $9}END{}' > git/RoPE/cfg/poa/routing/psh

cat scenario2_md1 | awk 'BEGIN{}(NR>1){print $1*1000}END{}' > git/RoPE/cfg/poa/routing/lambda
```