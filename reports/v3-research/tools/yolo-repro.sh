#!/usr/bin/env bash
# Tái hiện: dự án mở bằng `agent-cli --yolo` (TUI trong tmux RIÊNG) rồi `agent-cli run` cùng dự án qua server.
# run/send/loop không có cờ --yolo nên luôn gửi YOLO=false. In rc + stderr của run, cho ca --yolo và ca đối chứng.
# Dùng: bash yolo-repro.sh <thư-mục-mới ≤70 byte> <binary>
set -u
OUT=$1; BIN=$2
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT"; OUT=$(cd "$OUT" && pwd)
R=${R:-$(cd "$(dirname "$0")/../../.." && pwd)}  # gốc repo (có scripts/eval/exposure.sh)
eval "$(sed -n '/^verdict()/,/^mcp_cfg()/p' "$R/scripts/eval/exposure.sh")"

one() { # one <tên> <cờ run> <cờ TUI...>
  local name=$1 rflags=$2; shift 2
  begin "$name" '[{"match":"Y-","steps":[{"text":"ok"}]}]' qwen
  mkdir -p "$P/repo"; git_repo "$P/repo"
  server_mode s
  local TS="$P/tmux.sock"
  TO=10 iso "$P/repo" tmux -S "$TS" new-session -d -x 160 -y 45 "$BIN ${HOSTARG[*]} $*; sleep 30"
  for _ in $(seq 100); do [ -S "$SOCK" ] && break; sleep 0.1; done
  sleep 3
  ac "$P/repo" run $rflags --json "Y-run from another terminal" > "$P/run.json" 2> "$P/run.err"; local rc=$?
  echo "[$name] TUI flags='$*' run flags='$rflags' run rc=$rc stdout=$(head -c 200 "$P/run.json" | tr '\n' ' ') stderr=$(head -c 300 "$P/run.err" | tr '\n' ' ')"
  tmux -S "$TS" kill-server 2>/dev/null
  finish
}
one yolo "" --yolo
one ctl ""
one yolo-runyolo --yolo --yolo
one ctl-runyolo --yolo
