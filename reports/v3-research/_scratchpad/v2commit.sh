#!/usr/bin/env bash
# Áp một nhóm tệp của patch V2 lên worktree v2-final, build + biên dịch test, rồi commit.
# Dùng: v2commit.sh "<message>" <include-pattern>...
set -euo pipefail
W=/home/luannt/application/opensource/agent-cli-v2
EV=/home/luannt/application/opensource/agent-cli/reports/system-evolution/evidence
P=$EV/takeover-2026-10-19/takeover-final.patch
msg=$1; shift
inc=(); for p in "$@"; do inc+=(--include="$p"); done
cd "$W"
git apply "${inc[@]}" "$P"
if [[ "${APPLY_FOLLOWUP:-0}" == 1 ]]; then git apply "$EV/followup-2026-10-19/crushrc-followup.patch"; fi
git add -A
echo "files staged: $(git diff --cached --name-only | wc -l)"
git diff --cached --name-only | sed 's/^/  /'
avail=$(free -m | awk '/^Mem:/{print $7}')
(( avail >= 3000 )) || { echo "RAM trống ${avail}MB < 3000, dừng"; exit 3; }
export GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off CGO_ENABLED=0 GOEXPERIMENT=greenteagc
flock /tmp/claude-1000/agent-go.lock go build -p 2 ./...
flock /tmp/claude-1000/agent-go.lock go test -p 2 -count=1 -run '^$' ./... > /tmp/claude-1000/v2commit-testcompile.log 2>&1 || { tail -30 /tmp/claude-1000/v2commit-testcompile.log; exit 4; }
echo "build + test-compile: ok"
git commit -q -F - <<EOF
$msg

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git log -1 --format='%h %s'
