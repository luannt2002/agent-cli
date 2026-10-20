#!/usr/bin/env bash
# Kịch bản "chỉ Qwen" (Daily D): đi qua MỌI đường gọi LLM (main, tiêu đề, agent con, agentic_fetch, tự
# tóm tắt) với model nhỏ CỐ Ý đặt ở provider khác ("glm"). Server giả đếm request theo provider/vai trò;
# sổ ghi LLM của binary đối chiếu độc lập. Đạt khi glm = 0 và đủ 5 vai trò đều đi qwen.
# Dùng: bash scripts/eval/qwen-only.sh <thư-mục-mới> <binary> [context_window]
set -u
P="$1"; BIN="$2"; CW="${3:-30000}"
R=$(cd "$(dirname "$0")/../.." && pwd)
[ -e "$P" ] && { echo "exists: $P" >&2; exit 2; }
mkdir -p "$P/home" "$P/proj" "$P/log"
cat > "$P/rules.json" <<'JSON'
[{"match": "The previous session was interrupted", "steps": [{"text": "DONE-AFTER-SUMMARY"}]},
 {"match": "QWEN-ONLY", "steps": [
  {"tool": "agent", "args": {"prompt": "list the files in this repo"}},
  {"tool": "agentic_fetch", "args": {"prompt": "find the release notes of tool X"}},
  {"tool": "bash", "args": {"command": "seq 1 6000", "description": "big output 1"}},
  {"tool": "bash", "args": {"command": "seq 6001 12000", "description": "big output 2"}},
  {"tool": "bash", "args": {"command": "seq 12001 18000", "description": "big output 3"}},
  {"text": "DONE-QWEN-ONLY"}]}]
JSON
timeout 150 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
for _ in $(seq 50); do [ -s "$P/port" ] && break; sleep 0.1; done
PORT=$(cat "$P/port")
cat > "$P/proj/crush.json" <<JSON
{"providers":{
  "qwen":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/qwen/v1","api_key":"x","models":[{"id":"qwen3-coder","name":"qwen3-coder","context_window":$CW,"default_max_tokens":1000}]},
  "glm":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/glm/v1","api_key":"y","models":[{"id":"glm-5.3-flash","name":"glm-5.3-flash","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"qwen","model":"qwen3-coder"},"small":{"provider":"glm","model":"glm-5.3-flash"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}}
JSON
(
  cd "$P/proj" && git init -q && echo "# demo" > README.md
  env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/c" XDG_DATA_HOME="$P/home/d" XDG_CACHE_HOME="$P/home/x" \
    CRUSH_GLOBAL_CONFIG="$P/home/cfg" CRUSH_GLOBAL_DATA="$P/home/data" CRUSH_CACHE_DIR="$P/home/cache" \
    CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 CRUSH_CLIENT_SERVER=0 \
    AGENT_CLI_LLM_LEDGER="$P/ledger.jsonl" \
    timeout 120 "$BIN" run --json "QWEN-ONLY exercise every pathway" > "$P/out.json" 2> "$P/err.txt"
  echo $? > "$P/rc"
  # Đường loop/routine (cùng coordinator, chạy agent in-process) với cùng cấu hình hai provider.
  env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/c" XDG_DATA_HOME="$P/home/d" XDG_CACHE_HOME="$P/home/x" \
    CRUSH_GLOBAL_CONFIG="$P/home/cfg" CRUSH_GLOBAL_DATA="$P/home/data" CRUSH_CACHE_DIR="$P/home/cache" \
    CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 CRUSH_CLIENT_SERVER=0 \
    AGENT_CLI_LLM_LEDGER="$P/ledger.jsonl" \
    timeout 120 "$BIN" loop --verify "test -f README.md" "LOOP-PATH make sure the readme exists" > "$P/loop.out" 2> "$P/loop.err"
  echo $? > "$P/loop.rc"
)
sleep 1
python3 -I - "$P" <<'PY'
import json, os, sys
P = sys.argv[1]
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))]
led = [json.loads(l) for l in open(os.path.join(P, "ledger.jsonl"))] if os.path.exists(os.path.join(P, "ledger.jsonl")) else []
def table(rows, key_p, key_r):
    t = {}
    for r in rows:
        t.setdefault(r[key_p], {}).setdefault(r[key_r], 0)
        t[r[key_p]][r[key_r]] += 1
    return t
srv, lg = table(reqs, "provider", "role"), table(led, "provider", "role")
out = json.load(open(os.path.join(P, "out.json"))) if os.path.getsize(os.path.join(P, "out.json")) else {}
need = {"main", "title", "subagent", "agentic_fetch", "summary"}
seen = set(srv.get("qwen", {}))
print("rc=%s text=%r error=%r loop_rc=%s" % (open(os.path.join(P, "rc")).read().strip(), out.get("text", "")[:40], out.get("error", "")[:80], open(os.path.join(P, "loop.rc")).read().strip()))
loop_calls = sum(1 for r in reqs if r["last_user"].find("LOOP-PATH") >= 0 or "LOOP-PATH" in " ".join(r.get("user_texts", [])))
print("loop-path requests:", loop_calls)
print("server  (provider -> role: count):", srv)
print("ledger  (provider -> role: count):", lg)
print("roles on qwen:", sorted(seen), "missing:", sorted(need - seen))
ok = not srv.get("glm") and not lg.get("glm") and need <= seen and loop_calls > 0
print("QWEN-ONLY:", "PASS" if ok else "FAIL")
sys.exit(0 if ok else 1)
PY
