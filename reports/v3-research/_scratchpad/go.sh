#!/usr/bin/env bash
# Chạy lệnh go theo luật máy: chờ RAM available >= 3000 MB, flock, GOMAXPROCS=2, -mod=readonly, GOPROXY=off.
for i in $(seq 120); do
  avail=$(free -m | awk '/^Mem:/{print $7}')
  [ "$avail" -ge 3000 ] && break
  echo "go.sh: available=${avail}MB < 3000, chờ..." >&2; sleep 5
done
[ "$avail" -ge 3000 ] || { echo "go.sh: RAM không đủ" >&2; exit 99; }
cd /home/luannt/application/opensource/agent-cli-v3 || exit 98
exec flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off CGO_ENABLED=0 GOEXPERIMENT=greenteagc go "$@"
