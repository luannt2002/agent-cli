#!/bin/bash
# go qua flock + chờ RAM >= 3000 MB (luật máy).
while :; do
  a=$(awk '/MemAvailable/{print int($2/1024)}' /proc/meminfo)
  [ "$a" -ge 3000 ] && break
  sleep 5
done
exec flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off go "$@"
