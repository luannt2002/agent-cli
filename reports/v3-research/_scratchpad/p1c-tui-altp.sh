#!/usr/bin/env bash
# TUI thật trong tmux socket RIÊNG (không đụng màn hình người dùng), server mode, LLM giả.
# 1) session X chạy pp; dự án đổi sang qq; mở TUI bằng `-s X`: thanh trạng thái phải báo, <data>/crush.json không có models.
# 2) alt+m (danh sách model của provider đang chọn, qq), chọn m-qq, Enter: <data>/crush.json phải có models.large = qq
#    (lựa chọn tường minh vẫn ghi, kể cả khi trùng model đang có từ crush.json của repo).
# Dùng: bash p1c-tui.sh <repo agent-cli> <thư-mục-mới ≤ 70 byte> <binary>
set -u
R=$1; OUT=$2; BIN=$3
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd)
P=$OUT; FAILS=0
check() { if [ "$2" = 0 ]; then echo "PASS $1"; else echo "FAIL $1"; FAILS=$((FAILS + 1)); fi; }
mkdir -p "$P/home/.config/agent-cli" "$P/log" "$P/rt" "$P/proj"; chmod 700 "$P/rt"
echo '[]' > "$P/rules.json"
timeout 600 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
FAKE=$!
for _ in $(seq 50); do [ -s "$P/port" ] && break; sleep 0.1; done
PORT=$(cat "$P/port")
cat > "$P/home/.config/agent-cli/crush.json" <<JSON
{"providers":{
  "pp":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/pp/v1","api_key":"x","models":[{"id":"m-pp","name":"m-pp","context_window":100000,"default_max_tokens":1000}]},
  "qq":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/qq/v1","api_key":"x","models":[{"id":"m-qq","name":"m-qq","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"pp","model":"m-pp"},"small":{"provider":"pp","model":"m-pp"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}}
JSON
(cd "$P/proj" && git init -q && echo x > README.md && echo "# t" > AGENTS.md && git add -A && git -c user.name=t -c user.email=t@t commit -qm init)
ENVS=(env -i PATH=/usr/bin:/bin TERM=xterm-256color HOME="$P/home" XDG_CONFIG_HOME="$P/home/.config"
  XDG_DATA_HOME="$P/home/.local/share" XDG_CACHE_HOME="$P/home/.cache" CRUSH_GLOBAL_CONFIG="$P/home/.config/agent-cli"
  CRUSH_GLOBAL_DATA="$P/home/.local/share/agent-cli" CRUSH_CACHE_DIR="$P/home/.cache/agent-cli" AGENT_CLI_RUNTIME_DIR="$P/rt"
  GIT_CEILING_DIRECTORIES="$OUT" CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1
  CRUSH_CLIENT_SERVER=1 CRUSH_SERVER_IDLE_TIMEOUT=60)
HOST=(--host "unix://$P/s.sock")
x=$(cd "$P/proj" && "${ENVS[@]}" timeout 60 "$BIN" "${HOST[@]}" run --json "T-X1 on pp" 2>>"$P/err" | python3 -I -c 'import json,sys; print(json.load(sys.stdin)["session_id"])')
printf '%s' '{"models":{"large":{"provider":"qq","model":"m-qq"},"small":{"provider":"qq","model":"m-qq"}}}' > "$P/proj/crush.json"
models() { python3 -I -c 'import json,sys
try: print(json.dumps(json.load(open(sys.argv[1])).get("models", {}), sort_keys=True))
except FileNotFoundError: print("<absent>")' "$P/proj/.crush/crush.json"; }

TM=(tmux -S "$P/tmux.sock" -f /dev/null)
printf '#!/bin/sh\ncd %q && exec %s %q --host %q -s %q\n' "$P/proj" "$(printf '%q ' "${ENVS[@]}")" "$BIN" "unix://$P/s.sock" "$x" > "$P/tui.sh"
"${TM[@]}" new-session -d -s t -x 160 -y 45 "sh $P/tui.sh"
seen=""
for _ in $(seq 60); do
  sleep 0.25
  "${TM[@]}" capture-pane -p -t t > "$P/pane1.txt" 2>/dev/null
  grep -q "last ran on" "$P/pane1.txt" && { seen=1; break; }
done
echo "pane: $(grep -o 'This session[^│]*' "$P/pane1.txt" | head -1)"
[ -n "$x" ] && [ -n "$seen" ]; check "TUI -s X shows the model note in the status bar" $?
after_open=$(models)
echo "after open: data-models=$after_open"
[ "$after_open" = "<absent>" ]; check "TUI -s X writes no model to <data>/crush.json" $?

"${TM[@]}" send-keys -t t M-p
sleep 1
"${TM[@]}" send-keys -t t -l ""
sleep 0.5
"${TM[@]}" capture-pane -p -t t > "$P/pane2.txt"
"${TM[@]}" send-keys -t t Enter
picked=""
for _ in $(seq 40); do
  sleep 0.25
  m=$(models)
  [[ $m == *'"large": {'*'"model": "m-qq", "provider": "qq"}'* ]] && { picked=1; break; }
done
"${TM[@]}" capture-pane -p -t t > "$P/pane3.txt"
echo "after alt+m: data-models=$(models)"
[ -n "$picked" ]; check "alt+m picking m-qq writes models.large=qq (explicit choice)" $?

"${TM[@]}" kill-server 2>/dev/null
for pid in $(pgrep -f -- "server --host unix://$P/"); do kill -TERM "$pid" 2>/dev/null; done
kill "$FAKE" 2>/dev/null; wait "$FAKE" 2>/dev/null
echo "fails=$FAILS"
[ "$FAILS" = 0 ]
