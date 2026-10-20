#!/usr/bin/env bash
# go qua khoá chung + chờ RAM available >= 3000 MB.
set -u
cd /home/luannt/application/opensource/agent-cli-p3b
exec flock /tmp/claude-1000/agent-go.lock bash -c '
for i in $(seq 1 120); do
  a=$(awk "/MemAvailable/{print int(\$2/1024)}" /proc/meminfo)
  [ "$a" -ge 3000 ] && break
  echo "waiting RAM avail=${a}MB" >&2; sleep 5
done
GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off exec go "$@"' _ "$@"
