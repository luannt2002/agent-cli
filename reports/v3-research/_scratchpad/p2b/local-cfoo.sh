#!/usr/bin/env bash
# Đo nơi crush.db rơi khi gọi -c/-D tương đối (local + server), LLM giả, HOME cô lập.
# Dùng: bash local-cfoo.sh <thư-mục-mới, ngắn> <binary> <repo agent-cli để lấy fakellm.py>
set -u
OUT=$1; BIN=$2; R=$3
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT/home/.config/agent-cli" "$OUT/log" "$OUT/rt"; chmod 700 "$OUT/rt"
OUT=$(cd "$OUT" && pwd)
echo '[]' > "$OUT/rules.json"
timeout 600 python3 -I "$R/scripts/eval/fakellm.py" "$OUT/port" "$OUT/log" "$OUT/rules.json" >/dev/null 2>&1 &
FAKE=$!
for _ in $(seq 50); do [ -s "$OUT/port" ] && break; sleep 0.1; done
PORT=$(cat "$OUT/port")
cat > "$OUT/home/.config/agent-cli/crush.json" <<JSON
{"providers":{"qwen":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/qwen/v1","api_key":"x","models":[{"id":"m-qwen","name":"m-qwen","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"qwen","model":"m-qwen"},"small":{"provider":"qwen","model":"m-qwen"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}}
JSON
iso() { # iso <dir> <CS> <args...>
  local d=$1 cs=$2; shift 2
  (cd "$d" && env -i PATH=/usr/bin:/bin HOME="$OUT/home" XDG_CONFIG_HOME="$OUT/home/.config" \
    XDG_DATA_HOME="$OUT/home/.local/share" XDG_CACHE_HOME="$OUT/home/.cache" CRUSH_GLOBAL_CONFIG="$OUT/home/.config/agent-cli" \
    CRUSH_GLOBAL_DATA="$OUT/home/.local/share/agent-cli" CRUSH_CACHE_DIR="$OUT/home/.cache/agent-cli" AGENT_CLI_RUNTIME_DIR="$OUT/rt" \
    GIT_CEILING_DIRECTORIES="$OUT" CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 \
    CRUSH_CLIENT_SERVER=$cs CRUSH_SERVER_IDLE_TIMEOUT=60 timeout 60 "$BIN" "$@")
}
mk() { mkdir -p "$OUT/$1/r/foo"; (cd "$OUT/$1/r/foo" && git init -q && echo x > README.md && git add -A && git -c user.name=t -c user.email=t@t commit -qm init); }
dbs() { (cd "$OUT/$1" && find . -name crush.db | sed 's|^\./||' | sort | tr '\n' ' '); }
pj() { python3 -I - "$(find "$OUT/home" -name projects.json | head -1)" "$OUT/$1/" <<'PY'
import json, sys
try:
    d = json.load(open(sys.argv[1]))
except Exception as e:
    print("projects.json: %s" % e); sys.exit()
out = []
for p in d.get("projects", []):
    for l in p.get("locations") or [{"data_dir": p.get("data_dir", "")}]:
        dd = l.get("data_dir", "")
        if (l.get("cwd", "") + "/").startswith(sys.argv[2]):
            out.append(dd.replace(sys.argv[2], "<case>/"))
print("projects data_dir=" + (",".join(out) or "none"))
PY
}
row() { printf '%s|rc=%s|db=%s|%s\n' "$1" "$2" "$(dbs "$1")" "$(pj "$1")" | tee -a "$OUT/rows.txt"; }

mk L1; iso "$OUT/L1/r" 0 -c foo run --json "L1 local -c foo" > "$OUT/L1.json" 2>&1; row L1 $?
mk L2; iso "$OUT/L2/r" 0 -c "$OUT/L2/r/foo" run --json "L2 local -c abs" > "$OUT/L2.json" 2>&1; row L2 $?
mk L3; iso "$OUT/L3/r/foo" 0 run --json "L3 local no -c" > "$OUT/L3.json" 2>&1; row L3 $?
iso "$OUT/L1/r" 0 -c foo logs --tail 1 > "$OUT/L4.txt" 2>&1; echo "L4|rc=$?|logs -c foo (local, after L1): $(head -c 160 "$OUT/L4.txt" | tr '\n' ' ')" | tee -a "$OUT/rows.txt"
mk L5; iso "$OUT/L5/r" 0 -c foo -D x run --json "L5 local -c foo -D x" > "$OUT/L5.json" 2>&1; row L5 $?
mk S5; iso "$OUT/S5/r" 1 --host "unix://$OUT/s5.sock" -c foo -D x run --json "S5 server -c foo -D x" > "$OUT/S5.json" 2>&1; rc=$?
for pid in $(pgrep -f -- "server --host unix://$OUT/s5.sock"); do kill -TERM "$pid"; for _ in $(seq 100); do kill -0 "$pid" 2>/dev/null || break; sleep 0.1; done; done
row S5 $rc
kill "$FAKE" 2>/dev/null; wait "$FAKE" 2>/dev/null
exit 0
