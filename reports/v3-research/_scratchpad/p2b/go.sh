#!/usr/bin/env bash
# Bọc go theo luật máy: chờ RAM available >= 3000 MB, khoá chung, GOMAXPROCS=2.
for _ in $(seq 600); do
  a=$(awk '/MemAvailable/ {print int($2/1024)}' /proc/meminfo)
  [ "$a" -ge 3000 ] && break
  sleep 2
done
exec flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off CGO_ENABLED=${CGO_ENABLED:-0} GOEXPERIMENT=greenteagc go "$@"
