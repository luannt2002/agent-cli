#!/usr/bin/env bash
# Model thuộc workspace, chỉ đổi khi người dùng chọn: tiếp session cũ (`run -s`) không đổi model, ở cả chế độ local
# lẫn qua server, với binary thật và LLM giả (scripts/eval/fakellm.py: request mang tên provider trong đường dẫn).
# Kịch bản: X chạy provider pp; dự án đổi sang qq (crush.json của repo); `run -s X` phải chạy qq, không ghi
# `<dự án>/.crush/crush.json` và in một dòng stderr nêu cả hai model. Đối chứng: `run -s X -m pp/m-pp` chạy pp, và qua
# server thì ghi pp (lựa chọn tường minh), session mới sau đó chạy pp.
# Dùng: bash scripts/eval/session-model.sh <thư-mục-mới, ≤ 70 byte> <binary>   → thoát 0 PASS, 1 FAIL, 2 sai cách dùng.
set -u
[ $# -eq 2 ] || { echo "usage: $0 <new-dir> <binary>" >&2; exit 2; }
OUT=$1; BIN=$2
R=$(cd "$(dirname "$0")/../.." && pwd)
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
[ -x "$BIN" ] || { echo "not executable: $BIN" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd) && BIN=$(cd "$(dirname "$BIN")" && pwd)/$(basename "$BIN")
[ ${#OUT} -le 70 ] || { echo "dir path too long for unix sockets (${#OUT} > 70 bytes): $OUT" >&2; exit 2; }
FAILS=0
check() { if [ "$2" = 0 ]; then echo "ok   $1"; else echo "FAIL $1"; FAILS=$((FAILS + 1)); fi; }

begin() { # begin <ca> <CRUSH_CLIENT_SERVER 0|1>
  P="$OUT/$1"; CS=$2; HOSTARG=()
  mkdir -p "$P/home/.config/dew" "$P/log" "$P/rt" "$P/proj"; chmod 700 "$P/rt"
  echo '[]' > "$P/rules.json"
  timeout 300 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
  FAKE=$!
  for _ in $(seq 50); do [ -s "$P/port" ] && break; sleep 0.1; done
  PORT=$(cat "$P/port")
  cat > "$P/home/.config/dew/crush.json" <<JSON
{"providers":{
  "pp":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/pp/v1","api_key":"x","models":[{"id":"m-pp","name":"m-pp","context_window":100000,"default_max_tokens":1000}]},
  "qq":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/qq/v1","api_key":"x","models":[{"id":"m-qq","name":"m-qq","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"pp","model":"m-pp"},"small":{"provider":"pp","model":"m-pp"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}}
JSON
  (cd "$P/proj" && git init -q && echo x > README.md && git add -A && git -c user.name=t -c user.email=t@t commit -qm init)
  [ "$CS" = 1 ] && HOSTARG=(--host "unix://$P/s.sock")
}
ac() { # ac <lệnh...>: binary trong thư mục dự án, env cô lập
  (cd "$P/proj" && env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/.config" \
    XDG_DATA_HOME="$P/home/.local/share" XDG_CACHE_HOME="$P/home/.cache" CRUSH_GLOBAL_CONFIG="$P/home/.config/dew" \
    CRUSH_GLOBAL_DATA="$P/home/.local/share/dew" CRUSH_CACHE_DIR="$P/home/.cache/dew" DEW_RUNTIME_DIR="$P/rt" \
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
prov() { # prov <nhãn>: provider đã nhận lượt main mang nhãn đó
  python3 -I - "$P/log/requests.jsonl" "$1" <<'PY'
import json, sys
tag = sys.argv[2]
try:
    rows = [json.loads(l) for l in open(sys.argv[1])]
except FileNotFoundError:
    rows = []
print(",".join(sorted({r["provider"] for r in rows if r["role"] == "main" and tag in r["last_user"]})) or "none")
PY
}
models() { # models: khối "models" của <dự án>/.crush/crush.json, "<absent>" khi chưa có tệp
  python3 -I -c 'import json,sys
try: print(json.dumps(json.load(open(sys.argv[1])).get("models", {}), sort_keys=True))
except FileNotFoundError: print("<absent>")' "$P/proj/.crush/crush.json"
}
large_is_pp() { python3 -I -c 'import json,sys; m = json.loads(sys.argv[1]).get("large", {}); sys.exit(0 if (m.get("provider"), m.get("model")) == ("pp", "m-pp") else 1)' "$1" 2>/dev/null; }
project_qq() { printf '%s' '{"models":{"large":{"provider":"qq","model":"m-qq"},"small":{"provider":"qq","model":"m-qq"}}}' > "$P/proj/crush.json"; }
note_ok() { [ "$(grep -c 'last ran on pp/m-pp; continuing on the project model qq/m-qq' "$1")" = 1 ]; }

# --- local (CRUSH_CLIENT_SERVER=0)
begin local 0
x=$(ac run --json "L-X1 on pp" 2>>"$P/err" | sid)
project_qq
ac run --json -s "$x" "L-X2 continue X" > "$P/x2.json" 2> "$P/x2.err"
ac run --json -s "$x" -m pp/m-pp "L-X3 explicit pp" > "$P/x3.json" 2> "$P/x3.err"
echo "local: X1=$(prov L-X1) X2=$(prov L-X2) X3=$(prov L-X3) models=$(models)"
[ -n "$x" ] && [ "$(prov L-X1)" = pp ]; check "local: setup, X1 on pp" $?
[ "$(prov L-X2)" = qq ]; check "local: run -s X runs the project model qq" $?
note_ok "$P/x2.err"; check "local: run -s X prints one note naming pp/m-pp and qq/m-qq" $?
[ "$(models)" = "<absent>" ]; check "local: nothing written to <project>/.crush/crush.json" $?
[ "$(prov L-X3)" = pp ] && ! grep -q 'last ran on' "$P/x3.err"; check "local: run -s X -m pp/m-pp runs pp, no note" $?
finish

# --- server (CRUSH_CLIENT_SERVER=1)
begin server 1
x=$(ac run --json "S-X1 on pp" 2>>"$P/err" | sid)
project_qq
y=$(ac run --json "S-Y1 on project default" 2>>"$P/err" | sid)
before=$(models)
ac run --json -s "$x" "S-X2 continue X" > "$P/x2.json" 2> "$P/x2.err"
after=$(models)
ac send -s "$y" --wait --json "S-Y2 next turn of Y" > "$P/y2.json" 2>>"$P/err"
ac run --json "S-N1 new session" > "$P/n1.json" 2>>"$P/err"
ac run --json -s "$x" -m pp/m-pp "S-X3 explicit pp" > "$P/x3.json" 2> "$P/x3.err"
explicit=$(models)
ac run --json "S-N2 new session after -m" > "$P/n2.json" 2>>"$P/err"
echo "server: X1=$(prov S-X1) Y1=$(prov S-Y1) X2=$(prov S-X2) Y2=$(prov S-Y2) N1=$(prov S-N1) X3=$(prov S-X3) N2=$(prov S-N2)"
echo "server: models before=$before after run -s=$after after run -m=$explicit"
[ -n "$x" ] && [ -n "$y" ] && [ "$(prov S-X1)" = pp ] && [ "$(prov S-Y1)" = qq ]; check "server: setup, X1 on pp, Y1 on qq" $?
[ "$before" = "$after" ]; check "server: run -s X leaves <project>/.crush/crush.json unchanged" $?
[ "$(prov S-X2)" = qq ] && [ "$(prov S-Y2)" = qq ] && [ "$(prov S-N1)" = qq ]; check "server: X2, Y2 (send), N1 all run qq" $?
note_ok "$P/x2.err"; check "server: run -s X prints one note naming pp/m-pp and qq/m-qq" $?
[ "$(prov S-X3)" = pp ] && large_is_pp "$explicit"; check "server: run -s X -m pp/m-pp runs pp and writes it (explicit choice)" $?
[ "$(prov S-N2)" = pp ]; check "server: a new session after -m runs pp" $?
finish

echo "SESSION-MODEL: $([ "$FAILS" = 0 ] && echo PASS || echo "FAIL ($FAILS)")"
[ "$FAILS" = 0 ]
