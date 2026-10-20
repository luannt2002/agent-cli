#!/usr/bin/env bash
# E2E T15/D2: run --json với sổ ghi LLM hỏng (/dev/full) và đối chứng tệp ghi được,
# chế độ local và qua server, LLM giả. Dùng: e2e-ledger.sh <thư-mục-mới> <binary> <repo>
set -u
OUT="$1"; BIN="$2"; R="$3"
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT"
case_run() { # case_run <name> <cs 0|1> <ledger path or "FILE">
  local name=$1 cs=$2 led=$3 P="$OUT/$1" host=()
  mkdir -p "$P/home" "$P/proj" "$P/log"
  [ "$led" = FILE ] && led="$P/ledger.jsonl"
  [ "$cs" = 1 ] && host=(--host "unix://$P/s.sock")
  printf '%s' '[{"match":"LEDGER-PROBE","steps":[{"tool":"bash","args":{"command":"echo hi","description":"probe"}},{"text":"DONE-LEDGER"}]}]' > "$P/rules.json"
  timeout 90 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
  for _ in $(seq 50); do [ -s "$P/port" ] && break; sleep 0.1; done
  local port; port=$(cat "$P/port")
  cat > "$P/proj/crush.json" <<JSON
{"providers":{"qwen":{"type":"openai-compat","base_url":"http://127.0.0.1:$port/qwen/v1","api_key":"x","models":[{"id":"qwen3-coder","name":"qwen3-coder","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"qwen","model":"qwen3-coder"},"small":{"provider":"qwen","model":"qwen3-coder"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}}
JSON
  (
    cd "$P/proj" && git init -q && echo "# demo" > README.md
    env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/c" XDG_DATA_HOME="$P/home/d" XDG_CACHE_HOME="$P/home/x" \
      CRUSH_GLOBAL_CONFIG="$P/home/cfg" CRUSH_GLOBAL_DATA="$P/home/data" CRUSH_CACHE_DIR="$P/home/cache" \
      CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 CRUSH_CLIENT_SERVER=$cs CRUSH_SERVER_IDLE_TIMEOUT=3 \
      AGENT_CLI_LLM_LEDGER="$led" \
      timeout 60 "$BIN" "${host[@]}" run --json "LEDGER-PROBE run once" > "$P/out.json" 2> "$P/err.txt"
    echo $? > "$P/rc"
  )
  sleep 4 # tiêu đề chạy nền; server tắt sau idle
}
case_run local-devfull 0 /dev/full
case_run local-file 0 FILE
case_run server-devfull 1 /dev/full
case_run server-file 1 FILE
python3 -I - "$OUT" <<'PY'
import glob, json, os, sys
out = sys.argv[1]
bad = []
print("%-15s %3s %-12s %-44s %4s %6s %s" % ("case", "rc", "text", "ledger_error", "reqs", "ledger", "warn(sid/nosid)"))
for name in sorted(os.listdir(out)):
    P = os.path.join(out, name)
    rc = open(os.path.join(P, "rc")).read().strip()
    try:
        o = json.load(open(os.path.join(P, "out.json")))
    except Exception as e:
        o = {"_err": str(e)}
    reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))] if os.path.exists(os.path.join(P, "log", "requests.jsonl")) else []
    lf = os.path.join(P, "ledger.jsonl")
    led = [json.loads(l) for l in open(lf)] if os.path.exists(lf) else None
    warn_sid = warn_nosid = 0
    for log in glob.glob(os.path.join(P, "**", "crush.log"), recursive=True, include_hidden=True):
        for line in open(log, errors="replace"):
            if "Failed to write the LLM ledger" not in line:
                continue
            try:
                j = json.loads(line)
                sid = j.get("session_id", "")
            except Exception:
                sid = "?"
            if sid:
                warn_sid += 1
            else:
                warn_nosid += 1
    le = o.get("ledger_error")
    print("%-15s %3s %-12s %-44s %4d %6s %d/%d" % (name, rc, repr(o.get("text", ""))[:12], repr(le)[:44], len(reqs),
          "-" if led is None else len(led), warn_sid, warn_nosid))
    if rc != "0" or o.get("text") != "DONE-LEDGER" or o.get("error"):
        bad.append(name + ": turn did not run cleanly")
    if "devfull" in name:
        if not le or "no space left on device" not in le:
            bad.append(name + ": no ledger_error")
        if warn_sid != 1:
            bad.append(name + ": session warnings=%d (want 1 per turn)" % warn_sid)
    else:
        if "ledger_error" in o:
            bad.append(name + ": unexpected ledger_error")
        if led is None or len(led) != len(reqs) or not reqs:
            bad.append(name + ": ledger lines %s != requests %d" % (None if led is None else len(led), len(reqs)))
print("E2E-LEDGER:", "PASS" if not bad else "FAIL " + "; ".join(bad))
sys.exit(1 if bad else 0)
PY
