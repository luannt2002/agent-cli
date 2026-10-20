#!/usr/bin/env bash
# P1b: chạy lệnh go theo luật máy (RAM available >= 3000 MB, flock chung, GOMAXPROCS=2). Dùng: gp1b.sh go <args...>
cd /home/luannt/application/opensource/agent-cli-p1b || exit 99
exec flock /tmp/claude-1000/agent-go.lock bash -c '
  while a=$(free -m | awk "/^Mem:/{print \$7}"); [ "$a" -lt 3000 ]; do echo "gp1b: available=${a}MB < 3000, waiting" >&2; sleep 5; done
  exec env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off "$@"' _ "$@"
