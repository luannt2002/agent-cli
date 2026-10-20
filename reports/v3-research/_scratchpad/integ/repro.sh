#!/usr/bin/env bash
# Repro một lần (không vào repo): tin giữ chỗ của P1-a nằm sau lời trả lời thật
# thì dòng báo model của P1-c (`run -s`) phải dựa vào lời trả lời thật.
# Binary thật + LLM giả (scripts/eval/fakellm.py); tin giữ chỗ được chèn thẳng
# vào DB đúng hình dạng recordPrompt ghi (user + assistant provider
# agent-cli-pending, model = ID tin user), một đóng (canceled), một còn mở.
# Dùng: bash repro.sh <thư-mục-mới ≤70 byte> <binary> <repo>  → 0 PASS, 1 FAIL.
set -u
OUT=$1; BIN=$2; R=$3
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd)
FAILS=0
check() { if [ "$2" = 0 ]; then echo "ok   $1"; else echo "FAIL $1"; FAILS=$((FAILS + 1)); fi; }
begin() { # begin <ca> <CRUSH_CLIENT_SERVER 0|1>
  P="$OUT/$1"; CS=$2; HOSTARG=()
  mkdir -p "$P/home/.config/agent-cli" "$P/log" "$P/rt" "$P/proj"; chmod 700 "$P/rt"
  echo '[]' > "$P/rules.json"
  timeout 300 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
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
  (cd "$P/proj" && git init -q && echo x > README.md && git add -A && git -c user.name=t -c user.email=t@t commit -qm init)
  [ "$CS" = 1 ] && HOSTARG=(--host "unix://$P/s.sock")
}
ac() {
  (cd "$P/proj" && env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/.config" \
    XDG_DATA_HOME="$P/home/.local/share" XDG_CACHE_HOME="$P/home/.cache" CRUSH_GLOBAL_CONFIG="$P/home/.config/agent-cli" \
    CRUSH_GLOBAL_DATA="$P/home/.local/share/agent-cli" CRUSH_CACHE_DIR="$P/home/.cache/agent-cli" AGENT_CLI_RUNTIME_DIR="$P/rt" \
    GIT_CEILING_DIRECTORIES="$OUT" CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 \
    CRUSH_CLIENT_SERVER=$CS CRUSH_SERVER_IDLE_TIMEOUT=60 timeout 60 "$BIN" "${HOSTARG[@]}" "$@")
}
finish() {
  local pid; for pid in $(pgrep -f -- "server --host unix://$P/s.sock\$"); do kill -TERM "$pid" 2>/dev/null; done
  kill "$FAKE" 2>/dev/null; wait "$FAKE" 2>/dev/null
}
sid() { python3 -I -c 'import json,sys
try: print(json.load(sys.stdin).get("session_id",""))
except Exception: print("")'; }
project_qq() { printf '%s' '{"models":{"large":{"provider":"qq","model":"m-qq"},"small":{"provider":"qq","model":"m-qq"}}}' > "$P/proj/crush.json"; }
inject() { # inject <session>: u2 + tin giữ chỗ đã đóng (canceled), rồi u3 + tin giữ chỗ còn mở
  python3 -I - "$P/proj/.crush/crush.db" "$1" <<'PY'
import json, sqlite3, sys, time, uuid
db, sid = sys.argv[1], sys.argv[2]
c = sqlite3.connect(db, timeout=10)
t = int(time.time()) + 5
def row(role, parts, model, provider, at):
    i = str(uuid.uuid4())
    c.execute("INSERT INTO messages (id, session_id, role, parts, model, provider, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?)",
              (i, sid, role, json.dumps(parts), model, provider, at, at))
    return i
for k, closed in ((0, True), (2, False)):
    u = row("user", [{"type": "text", "data": {"text": "queued prompt"}}, {"type": "finish", "data": {"reason": "stop", "time": t + k}}], "", None, t + k)
    parts = [{"type": "finish", "data": {"reason": "canceled", "time": t + k + 1, "message": "User canceled request"}}] if closed else []
    row("assistant", parts, u, "agent-cli-pending", t + k + 1)
c.commit()
print(c.execute("SELECT role, provider, model FROM messages WHERE session_id = ? ORDER BY created_at", (sid,)).fetchall())
PY
}
nopending() { ! grep -q 'agent-cli-pending' "$1"; }

for mode in local:0 server:1; do
  name=${mode%%:*}; begin "$name" "${mode##*:}"
  x=$(ac run --json "R-X1 on pp" 2>>"$P/err" | sid)
  project_qq
  y=$(ac run --json "R-Y1 on qq" 2>>"$P/err" | sid)
  [ -n "$x" ] && [ -n "$y" ]; check "$name: setup X on pp, Y on qq" $?
  inject "$x"; inject "$y"
  ac run --json -s "$x" "R-X2 continue X" > "$P/x2.json" 2> "$P/x2.err"
  ac run --json -s "$y" "R-Y2 continue Y" > "$P/y2.json" 2> "$P/y2.err"
  echo "$name: X2 stderr: $(grep 'last ran on' "$P/x2.err")"
  echo "$name: Y2 stderr: $(grep 'last ran on' "$P/y2.err")"
  [ "$(grep -c 'last ran on pp/m-pp; continuing on the project model qq/m-qq' "$P/x2.err")" = 1 ] && nopending "$P/x2.err"
  check "$name: X (pp reply, then placeholders) notes pp/m-pp once, no agent-cli-pending" $?
  ! grep -q 'last ran on' "$P/y2.err"
  check "$name: Y (qq reply, then placeholders) prints no note" $?
  finish
done
echo "REPRO: $([ "$FAILS" = 0 ] && echo PASS || echo "FAIL ($FAILS)")"
[ "$FAILS" = 0 ]
