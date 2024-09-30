#!/bin/bash  

bucket=measures_poa

curl http://localhost:8086/api/v2/query?org=unito -XPOST  \
-H 'Authorization: Token $TOKEN' \
-H 'Accept: application/csv'   \
-H 'Content-type: application/vnd.flux'   \
-d "from(bucket:\"$bucket\") |> range(start: -1d) |> filter(fn: (r) => r[\"_measurement\"] == \"packet\") |> keep(columns: [\"_time\",\"_value\",\"eventType\",\"idDevice\",\"idRequest\",\"timestamp\"])" >> "$bucket.csv"
