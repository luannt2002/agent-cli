#!/usr/bin/env bash
# Tiêm lỗi provider vào binary thật, LLM giả. Mỗi ca: model chính ở provider "qwen", model nhỏ cố ý
# đặt ở provider "glm" — lời gọi ẩn nào lọt sang provider khác đều hiện ở cột glm (đếm phía server).
# Dùng: bash scripts/eval/faults.sh <thư-mục-mới> <binary>
set -u
OUT="$1"; BIN="$2"
R=$(cd "$(dirname "$0")/../.." && pwd)
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT"

# name | luật cho qwen (JSON) | ghi chú
run_case() {
  local name=$1 rules=$2 P="$OUT/$1"
  mkdir -p "$P/home" "$P/proj" "$P/log"
  printf '%s' "$rules" > "$P/rules.json"
  timeout 90 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
  for _ in $(seq 50); do [ -s "$P/port" ] && break; sleep 0.1; done
  local port qport cw=100000
  port=$(cat "$P/port"); qport=$port
  [ "$name" = provider-down ] && qport=1 # không ai nghe ở cổng 1: connection refused
  [ "$name" = summary-loop ] && cw=30000  # đầy lại sau vài bước để tự tóm tắt
  cat > "$P/proj/crush.json" <<JSON
{"providers":{
  "qwen":{"type":"openai-compat","base_url":"http://127.0.0.1:$qport/qwen/v1","api_key":"x","models":[{"id":"qwen3-coder","name":"qwen3-coder","context_window":$cw,"default_max_tokens":1000}]},
  "glm":{"type":"openai-compat","base_url":"http://127.0.0.1:$port/glm/v1","api_key":"y","models":[{"id":"glm-5.3-flash","name":"glm-5.3-flash","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"qwen","model":"qwen3-coder"},"small":{"provider":"glm","model":"glm-5.3-flash"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}}
JSON
  (
    cd "$P/proj" && git init -q && echo "# demo" > README.md
    env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/c" XDG_DATA_HOME="$P/home/d" XDG_CACHE_HOME="$P/home/x" \
      CRUSH_GLOBAL_CONFIG="$P/home/cfg" CRUSH_GLOBAL_DATA="$P/home/data" CRUSH_CACHE_DIR="$P/home/cache" \
      CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 CRUSH_CLIENT_SERVER=0 \
      timeout 40 "$BIN" run --json "fault probe $name" > "$P/out.json" 2> "$P/err.txt"
    echo $? > "$P/rc"
  )
  sleep 1
}

run_case ok '[]'
run_case provider-down '[]'
run_case http-500 '[{"match":"fault","steps":[{"error":500,"message":"upstream exploded"}]}]'
run_case http-400 '[{"match":"fault","steps":[{"error":400,"message":"bad request"}]}]'
run_case empty-reply '[{"match":"fault","steps":[{"text":""}]}]'
run_case malformed-stream '[{"match":"fault","steps":[{"raw":"data: {not json\n\n"}]}]'
run_case truncated-stream '[{"match":"fault","steps":[{"raw":"data: {\"id\":\"c1\",\"object\":\"chat.completion.chunk\",\"model\":\"qwen3-coder\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"half an ans\"},\"finish_reason\":null}]}\n\n"}]}]'
run_case hang '[{"match":"fault","steps":[{"sleep":60}]}]'
# Model làm lại việc từ đầu sau mỗi lần tự tóm tắt (lời nhắc làm tiếp vẫn chứa prompt gốc).
run_case summary-loop '[{"match":"fault","steps":[{"tool":"bash","args":{"command":"seq 1 6000","description":"a"}},{"tool":"bash","args":{"command":"seq 6001 12000","description":"b"}},{"tool":"bash","args":{"command":"seq 12001 18000","description":"c"}},{"text":"never reached"}]}]'

python3 -I - "$OUT" <<'PY'
import json, os, sys
out = sys.argv[1]
print("%-17s %4s %-5s %-34s %s" % ("case", "rc", "glm", "error (out.json)", "requests qwen by role"))
for name in sorted(os.listdir(out)):
    P = os.path.join(out, name)
    rc = open(os.path.join(P, "rc")).read().strip() if os.path.exists(os.path.join(P, "rc")) else "?"
    try:
        o = json.load(open(os.path.join(P, "out.json")))
    except Exception:
        o = {}
    reqs = []
    f = os.path.join(P, "log", "requests.jsonl")
    if os.path.exists(f):
        reqs = [json.loads(l) for l in open(f)]
    glm = sum(1 for r in reqs if r["provider"] == "glm")
    roles = {}
    for r in reqs:
        if r["provider"] == "qwen":
            roles[r["role"]] = roles.get(r["role"], 0) + 1
    err = (o.get("error") or ("text=" + repr(o.get("text", ""))[:20] if o else "no JSON"))[:34]
    print("%-17s %4s %-5d %-34s %s" % (name, rc, glm, err, roles))
PY
