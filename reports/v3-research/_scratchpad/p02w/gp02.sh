#!/usr/bin/env bash
# P02: chạy một lệnh go theo luật máy (khoá chung, RAM available >= 3000 MB, GOMAXPROCS=2, offline) trong worktree p02.
cd /home/luannt/application/opensource/agent-cli-p02 || exit 1
exec flock /tmp/claude-1000/agent-go.lock bash -c '
while :; do
  a=$(free -m | awk "/^Mem:/{print \$7}")
  [ "$a" -ge 3000 ] && break
  echo "waiting RAM avail=${a}MB" >&2; sleep 5
done
exec env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off "$@"' _ "$@"
