#!/usr/bin/env bash
# e2e P2d: binary thật + LLM giả (scripts/eval/fakellm.py) + 2 MCP stdio stub + hook PreToolUse tuỳ chọn.
# Dùng: e2e.sh <thư-mục-mới> <binary> <hook 0|1>
set -u
P="$1"; BIN="$2"; HOOK="$3"
S=$(cd "$(dirname "$0")" && pwd)
R=/home/luannt/application/opensource/agent-cli-p2d
[ -e "$P" ] && { echo "exists: $P" >&2; exit 2; }
mkdir -p "$P/home" "$P/proj" "$P/log"
cat > "$P/rules.json" <<'JSON'
[{"match": "P2D-PROBE", "steps": [
  {"tool": "mcp_p2doff_send", "args": {}},
  {"tool": "mcp_p2don_send", "args": {}},
  {"text": "DONE-P2D"}]}]
JSON
timeout 90 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
for _ in $(seq 50); do [ -s "$P/port" ] && break; sleep 0.1; done
PORT=$(cat "$P/port")
hooks='{}'
if [ "$HOOK" = 1 ]; then
  printf '#!/bin/sh\necho "$CRUSH_TOOL_NAME" >> %s/hook.log\n' "$P" > "$P/hook.sh"; chmod +x "$P/hook.sh"
  hooks="{\"PreToolUse\":[{\"command\":\"$P/hook.sh\"}]}"
fi
cat > "$P/proj/crush.json" <<JSON
{"providers":{
  "qwen":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/qwen/v1","api_key":"x","models":[{"id":"qwen3-coder","name":"qwen3-coder","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"qwen","model":"qwen3-coder"},"small":{"provider":"qwen","model":"qwen3-coder"}},
 "mcp":{
  "p2don":{"type":"stdio","command":"python3","args":["-I","$S/mcp_stub.py","p2don","-","$P/mcp.log"]},
  "p2doff":{"type":"stdio","command":"python3","args":["-I","$S/mcp_stub.py","p2doff","$P/proj/.crush/crush.db","$P/mcp.log"]}},
 "hooks":$hooks,
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}}
JSON
python3 -I -c 'import json,sys; json.load(open(sys.argv[1]))' "$P/proj/crush.json" || exit 3
(
  cd "$P/proj" && git init -q && echo "# demo" > README.md
  env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/c" XDG_DATA_HOME="$P/home/d" XDG_CACHE_HOME="$P/home/x" \
    CRUSH_GLOBAL_CONFIG="$P/home/cfg" CRUSH_GLOBAL_DATA="$P/home/data" CRUSH_CACHE_DIR="$P/home/cache" \
    CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 CRUSH_CLIENT_SERVER=0 \
    timeout 60 "$BIN" run --json "P2D-PROBE call the tools" > "$P/out.json" 2> "$P/err.txt"
  echo $? > "$P/rc"
)
python3 -I - "$P" <<'PY'
import json, os, sys
P = sys.argv[1]
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))] if os.path.exists(os.path.join(P, "log", "requests.jsonl")) else []
main = [r for r in reqs if r["role"] == "main"]
rd = lambda f: open(os.path.join(P, f)).read().split("\n") if os.path.exists(os.path.join(P, f)) else []
mcp = [l for l in rd("mcp.log") if l]
hook = [l for l in rd("hook.log") if l]
print("rc=%s main_requests=%d" % (open(os.path.join(P, "rc")).read().strip(), len(main)))
for r in main:
    mt = sorted(t for t in r["tools"] if t.startswith("mcp_"))
    print("  main#%d tool_steps=%d n_tools=%d mcp_tools=%s" % (r["seq"], r["tool_steps"], len(r["tools"]), mt))
print("mcp.log:", mcp)
print("hook.log:", hook)
PY
