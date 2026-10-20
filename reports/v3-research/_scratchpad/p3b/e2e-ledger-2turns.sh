#!/usr/bin/env bash
# Hai lượt trên CÙNG server (tiến trình sống lâu), sổ = /dev/full: mỗi lượt một cảnh báo + ledger_error.
# Dùng: e2e-ledger-2turns.sh <thư-mục-mới> <binary> <repo>
set -u
P="$1"; BIN="$2"; R="$3"
[ -e "$P" ] && { echo "exists: $P" >&2; exit 2; }
mkdir -p "$P/home" "$P/proj" "$P/log"
printf '%s' '[{"match":"LEDGER-PROBE","steps":[{"tool":"bash","args":{"command":"echo hi","description":"probe"}},{"text":"DONE-LEDGER"}]}]' > "$P/rules.json"
timeout 90 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
for _ in $(seq 50); do [ -s "$P/port" ] && break; sleep 0.1; done
port=$(cat "$P/port")
cat > "$P/proj/crush.json" <<JSON
{"providers":{"qwen":{"type":"openai-compat","base_url":"http://127.0.0.1:$port/qwen/v1","api_key":"x","models":[{"id":"qwen3-coder","name":"qwen3-coder","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"qwen","model":"qwen3-coder"},"small":{"provider":"qwen","model":"qwen3-coder"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}}
JSON
ac() {
  (cd "$P/proj" && env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/c" XDG_DATA_HOME="$P/home/d" XDG_CACHE_HOME="$P/home/x" \
    CRUSH_GLOBAL_CONFIG="$P/home/cfg" CRUSH_GLOBAL_DATA="$P/home/data" CRUSH_CACHE_DIR="$P/home/cache" \
    CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 CRUSH_CLIENT_SERVER=1 CRUSH_SERVER_IDLE_TIMEOUT=20 \
    AGENT_CLI_LLM_LEDGER=/dev/full timeout 60 "$BIN" --host "unix://$P/s.sock" "$@")
}
(cd "$P/proj" && git init -q && echo "# demo" > README.md)
ac run --json "LEDGER-PROBE turn one" > "$P/t1.json" 2> "$P/t1.err"; echo $? > "$P/t1.rc"
sid=$(python3 -I -c 'import json,sys; print(json.load(open(sys.argv[1]))["session_id"])' "$P/t1.json")
ac run --json -s "$sid" "LEDGER-PROBE turn two" > "$P/t2.json" 2> "$P/t2.err"; echo $? > "$P/t2.rc"
sleep 2
python3 -I - "$P" <<'PY'
import glob, json, os, sys
P = sys.argv[1]
bad = []
logs = glob.glob(os.path.join(P, "**", "crush.log"), recursive=True, include_hidden=True)
pids = set(); warns = []
for log in logs:
    for line in open(log, errors="replace"):
        if "Failed to write the LLM ledger" in line:
            warns.append(json.loads(line))
for t in ("t1", "t2"):
    o = json.load(open(os.path.join(P, t + ".json")))
    rc = open(os.path.join(P, t + ".rc")).read().strip()
    print(t, "rc=%s text=%r ledger_error=%r session=%s" % (rc, o.get("text"), o.get("ledger_error"), o.get("session_id")))
    if rc != "0" or o.get("text") != "DONE-LEDGER" or "no space left" not in (o.get("ledger_error") or ""):
        bad.append(t)
reqs = sum(1 for _ in open(os.path.join(P, "log", "requests.jsonl")))
sid_warns = [w for w in warns if w.get("session_id")]
print("server logs:", len(logs), "requests:", reqs, "warnings with session:", len(sid_warns), "without:", len(warns) - len(sid_warns))
if len(logs) != 1:
    bad.append("expected one server log (one server process), got %d" % len(logs))
if len(sid_warns) != 2:
    bad.append("want 2 session warnings (one per turn), got %d" % len(sid_warns))
print("E2E-LEDGER-2TURNS:", "PASS" if not bad else "FAIL " + "; ".join(bad))
sys.exit(1 if bad else 0)
PY
