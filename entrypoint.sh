#!/bin/bash

set -eo pipefail

LOCUST=( "./a.out" )
LOCUST+=(--master-host=$LOCUST_MASTER --master-port=$LOCUST_MASTER_PORT --rpc=zeromq --max-rps=$MAX_RPS)
echo "Wait for master"
counter=0
while ! wget -qT5 $LOCUST_MASTER:$LOCUST_MASTER_WEB >/dev/null 2>&1; do
  if [[ "$counter" -gt 1 ]]; then
    echo "Master not found in 30 seconds. Quiting"
    exit 1
  fi
  counter=$((counter+1))
  echo "Waiting for master"
  sleep 5
done
echo "${LOCUST[@]}"

exec ${LOCUST[@]}
