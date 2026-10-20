#!/usr/bin/env bash
# Sổ ghi LLM ghi lỗi phải thấy được (T15): binary thật, LLM giả, cả chạy local lẫn qua server
# (CRUSH_CLIENT_SERVER=1, socket riêng trong thư mục ca). Sổ = /dev/full (mọi lần ghi lỗi ENOSPC):
# mỗi lượt vẫn chạy xong và thoát 0, `run --json` có `ledger_error`, log có đúng một WARN gắn
# session mỗi lượt; qua server chạy hai lượt trên cùng server (`run` rồi `run -s`). Đối chứng sổ là
# tệp thường: không có `ledger_error`, số dòng sổ = số request server giả nhận.
# Thoát 1 khi một điều trên sai.
# Dùng: bash scripts/eval/ledger.sh <thư-mục-mới> <binary>
set -u
OUT="$1"; BIN="$2"
R=$(cd "$(dirname "$0")/../.." && pwd)
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT"

# run_case <tên> <0|1: qua server> <sổ: /dev/full hoặc FILE>
run_case() {
  local name=$1 cs=$2 led=$3 P="$OUT/$1" host=() turns=1
  mkdir -p "$P/home" "$P/proj" "$P/log"
  [ "$led" = FILE ] && led="$P/ledger.jsonl"
  [ "$cs" = 1 ] && host=(--host "unix://$P/s.sock") && turns=2
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
    ac() {
      env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/c" XDG_DATA_HOME="$P/home/d" XDG_CACHE_HOME="$P/home/x" \
        CRUSH_GLOBAL_CONFIG="$P/home/cfg" CRUSH_GLOBAL_DATA="$P/home/data" CRUSH_CACHE_DIR="$P/home/cache" \
        CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 CRUSH_CLIENT_SERVER=$cs CRUSH_SERVER_IDLE_TIMEOUT=5 \
        DEW_LLM_LEDGER="$led" timeout 60 "$BIN" "${host[@]}" "$@"
    }
    ac run --json "LEDGER-PROBE turn 1" > "$P/out1.json" 2> "$P/err1.txt"
    echo $? > "$P/rc1"
    if [ "$turns" = 2 ]; then
      sid=$(python3 -I -c 'import json,sys; print(json.load(open(sys.argv[1])).get("session_id",""))' "$P/out1.json" 2>/dev/null)
      ac run --json -s "$sid" "LEDGER-PROBE turn 2" > "$P/out2.json" 2> "$P/err2.txt"
      echo $? > "$P/rc2"
    fi
  )
  echo "$turns" > "$P/turns"
  sleep 7 # tiêu đề chạy nền; server tự tắt sau idle
}

run_case local-devfull 0 /dev/full
run_case local-file 0 FILE
run_case server-devfull 1 /dev/full
run_case server-file 1 FILE

python3 -I - "$OUT" <<'PY'
import glob, json, os, sys
out = sys.argv[1]
bad = []
print("%-15s %-5s %-6s %-30s %4s %6s %s" % ("case", "rc", "text", "ledger_error", "reqs", "ledger", "warn session/none"))
for name in sorted(os.listdir(out)):
    P = os.path.join(out, name)
    turns = int(open(os.path.join(P, "turns")).read())
    rcs, texts, lerrs = [], [], []
    for i in range(1, turns + 1):
        rcs.append(open(os.path.join(P, "rc%d" % i)).read().strip())
        try:
            o = json.load(open(os.path.join(P, "out%d.json" % i)))
        except Exception:
            o = {}
        texts.append(o.get("text"))
        lerrs.append(o.get("ledger_error"))
        if rcs[-1] != "0" or o.get("text") != "DONE-LEDGER" or o.get("error"):
            bad.append("%s turn %d: did not run cleanly (rc=%s, error=%r)" % (name, i, rcs[-1], o.get("error")))
    f = os.path.join(P, "log", "requests.jsonl")
    reqs = sum(1 for _ in open(f)) if os.path.exists(f) else 0
    lf = os.path.join(P, "ledger.jsonl")
    led = sum(1 for _ in open(lf)) if os.path.exists(lf) else None
    warn_sid = warn_none = 0
    for log in glob.glob(os.path.join(P, "**", "crush.log"), recursive=True, include_hidden=True):
        for line in open(log, errors="replace"):
            if "Failed to write the LLM ledger" not in line:
                continue
            try:
                sid = json.loads(line).get("session_id", "")
            except Exception:
                sid = "?"
            if sid:
                warn_sid += 1
            else:
                warn_none += 1
    print("%-15s %-5s %-6s %-30s %4d %6s %d/%d" % (name, ",".join(rcs), "ok" if all(t == "DONE-LEDGER" for t in texts) else "BAD",
          ",".join("-" if e is None else e[:14] for e in lerrs), reqs, "-" if led is None else led, warn_sid, warn_none))
    if not reqs:
        bad.append(name + ": no request reached the fake LLM")
    if "devfull" in name:
        if not all(e and "no space left on device" in e for e in lerrs):
            bad.append(name + ": ledger_error missing in a turn")
        if warn_sid != turns:
            bad.append(name + ": %d warnings with a session for %d turns (want one per turn)" % (warn_sid, turns))
    else:
        if any(e is not None for e in lerrs):
            bad.append(name + ": ledger_error although the ledger is writable")
        if led != reqs:
            bad.append(name + ": ledger lines %s != requests %d" % (led, reqs))
print("LEDGER:", "PASS" if not bad else "FAIL " + "; ".join(bad))
sys.exit(1 if bad else 0)
PY
