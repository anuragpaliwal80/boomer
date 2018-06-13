#!/bin/bash

set -eo pipefail

LOCUST=( "./a.out" )
LOCUST+=(--master-host=$LOCUST_MASTER --master-port=$LOCUST_MASTER_PORT --rpc=zeromq --max-rps=$MAX_RPS)
echo "wait for master"
while ! wget -qT5 $LOCUST_MASTER:$LOCUST_MASTER_WEB >/dev/null 2>&1; do
  echo "Waiting for master"
  sleep 5
done
echo "${LOCUST[@]}"

exec ${LOCUST[@]}
