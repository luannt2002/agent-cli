#!/usr/bin/env bash
# Chạy go trong worktree p2a: chờ RAM available >= 3000 MB, khoá flock chung, giới hạn lõi.
set -u
cd "${GODIR:-/home/luannt/application/opensource/agent-cli-p2a}"
for i in $(seq 120); do
  a=$(awk '/MemAvailable/{print int($2/1024)}' /proc/meminfo)
  [ "$a" -ge 3000 ] && break
  echo "waiting RAM: ${a}MB" >&2; sleep 5
done
exec flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off go "$@"
