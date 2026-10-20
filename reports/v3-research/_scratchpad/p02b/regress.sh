#!/usr/bin/env bash
# Hồi quy P0-2b với một binary: faults, qwen-only, workspaces, run-json, exposure C5, Home e2e của P02.
# Mỗi script ghi vào thư mục mới dưới /home/luannt/.cache/v3ev/; in rc và dòng kết luận.
# Dùng: bash regress.sh <tiền tố tên thư mục> <binary>
set -u
PFX=$1; BIN=$2
W=/home/luannt/application/opensource/agent-cli-p02b
V=/home/luannt/.cache/v3ev
S=$(cd "$(dirname "$0")" && pwd)
LOG=$S/regress-$PFX.txt
: > "$LOG"
step() {
  local name=$1; shift
  local start=$SECONDS
  "$@" > "$S/regress-$PFX-$name.out" 2>&1
  local rc=$?
  echo "$name rc=$rc ($((SECONDS - start))s)" | tee -a "$LOG"
  tail -4 "$S/regress-$PFX-$name.out" | sed 's/^/    /' | tee -a "$LOG"
}
cd "$W" || exit 1
step faults timeout 1500 bash scripts/eval/faults.sh "$V/$PFX-faults" "$BIN"
step qwen timeout 1500 bash scripts/eval/qwen-only.sh "$V/$PFX-qwen" "$BIN"
step workspaces timeout 1500 bash scripts/eval/workspaces.sh "$V/$PFX-ws" "$BIN"
step runjson env AGENT_CLI_BIN="$BIN" timeout 1500 bash scripts/test-agent-cli-run-json.sh
step c5 timeout 1500 bash scripts/eval/exposure.sh "$V/$PFX-c5" "$BIN" C5
step home timeout 900 bash /tmp/claude-1000/-home-luannt-application/1ba1a44b-c74c-4a9d-b148-f58af7166a87/scratchpad/p02w/p02-home.sh "$V/$PFX-home" "$BIN"
echo DONE | tee -a "$LOG"
