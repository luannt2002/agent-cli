#!/usr/bin/env bash
# Chạy go qua khoá chung, chỉ khi RAM available >= 3000 MB.
set -u
cd "${GODIR:-/home/luannt/application/opensource/agent-cli-p2d}"
exec flock /tmp/claude-1000/agent-go.lock bash -c '
  for i in $(seq 120); do a=$(awk "/MemAvailable/{print int(\$2/1024)}" /proc/meminfo); [ "$a" -ge 3000 ] && break; sleep 5; done
  echo "[g.sh] MemAvailable=${a}MB" >&2
  [ "$a" -ge 3000 ] || { echo "[g.sh] RAM thấp, bỏ" >&2; exit 99; }
  GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off exec go "$@"' _ "$@"
