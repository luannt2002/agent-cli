#!/usr/bin/env bash
# Chạy một lệnh go theo luật máy: chờ RAM available >= 3000 MB, flock, GOMAXPROCS=2, -mod=readonly, GOPROXY=off.
set -u
for i in $(seq 120); do
  a=$(free -m | awk '/^Mem:/{print $7}')
  [ "$a" -ge 3000 ] && break
  echo "g.sh: available=${a}MB < 3000, waiting" >&2; sleep 5
done
cd "${GODIR:-/home/luannt/application/opensource/agent-cli-p1c}"
exec flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off go "$@"
