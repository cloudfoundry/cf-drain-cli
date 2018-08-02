#!/bin/bash

set -e

PORT=9999 PROXY_HEALTH_PORT=10000 ./proxy &
echo $! > /tmp/pids
sleep 2

HTTP_PROXY=localhost:9999 ./syslog-forwarder &
echo $! >> /tmp/pids

# Close everything, otherwise the container won't be reset
function kill_everything {
    for pid in $(cat /tmp/pids)
    do
        kill -9 $pid &>/dev/null || true
    done
}

# Watch pids
while true
do
    for pid in $(cat /tmp/pids)
    do
        ps -p $pid &> /dev/null || kill_everything
    done
    sleep 10
done
