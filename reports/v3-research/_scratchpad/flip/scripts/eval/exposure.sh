#!/usr/bin/env bash
# Test phơi nhiễm V3 (reports/v3-research/experiments/00_EXPOSURE_TESTS.md) với binary thật + LLM giả
# (scripts/eval/fakellm.py). Mỗi ca khẳng định BẤT BIẾN MONG MUỐN rồi in đúng một dòng
#   T<n>: EXPOSED|NOT_EXPOSED|INCONCLUSIVE — lý do (giá trị thật)
# EXPOSED = phơi nhiễm được xác nhận trên binary này. Đây là bộ ĐO, không phải cổng: thoát 0 khi chạy đủ các
# ca bất kể kết quả; thoát 2 khi sai cách dùng. Bằng chứng thô của từng ca nằm trong <thư-mục>/T<n>/.
# Cô lập như faults.sh: mỗi ca HOME/XDG/CRUSH_GLOBAL_* riêng trong thư mục ca, socket riêng, không LLM thật.
# Socket nằm trong thư mục ca: đường dẫn <thư-mục>/T13/s.sock phải ≤ 107 byte (chọn thư mục ngắn).
# Dùng: bash scripts/eval/exposure.sh <thư-mục-mới> <binary> [T2 T3 T5 T6 T11 T13 T17 T21]
set -u
[ $# -ge 2 ] || { echo "usage: $0 <new-dir> <binary> [cases...]" >&2; exit 2; }
OUT=$1; BIN=$2; shift 2
CASES=${*:-T2 T3 T5 T6 T11 T13 T17 T21}
R=$(cd "$(dirname "$0")/../.." && pwd)
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
[ -x "$BIN" ] || { echo "not executable: $BIN" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd) && BIN=$(cd "$(dirname "$BIN")" && pwd)/$(basename "$BIN")
[ ${#OUT} -le 92 ] || { echo "dir path too long for unix sockets (${#OUT} > 92 bytes): $OUT" >&2; exit 2; }

verdict() { printf '%s: %s — %s\n' "$1" "$2" "$3" | tee -a "$OUT/verdicts.txt"; }

# begin <ca> <luật fakellm JSON> <provider large> [mcp JSON]: dựng thư mục ca, LLM giả, cấu hình global.
begin() {
  P="$OUT/$1"; SOCK="$P/s.sock"; HOSTARG=(); CS=0; EXTRA=()
  mkdir -p "$P/home/cfg" "$P/log"
  printf '%s' "$2" > "$P/rules.json"
  timeout 900 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
  FAKE=$!
  for _ in $(seq 50); do [ -s "$P/port" ] && break; sleep 0.1; done
  PORT=$(cat "$P/port" 2>/dev/null)
  local mcp=""
  [ -n "${4:-}" ] && mcp=",\"mcp\":$4"
  # Ba provider giả trên cùng một LLM giả: request mang tên provider trong đường dẫn (fakellm.py).
  cat > "$P/home/cfg/crush.json" <<JSON
{"providers":{
  "qwen":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/qwen/v1","api_key":"x","models":[{"id":"m-qwen","name":"m-qwen","context_window":100000,"default_max_tokens":1000}]},
  "pp":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/pp/v1","api_key":"x","models":[{"id":"m-pp","name":"m-pp","context_window":100000,"default_max_tokens":1000}]},
  "qq":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/qq/v1","api_key":"x","models":[{"id":"m-qq","name":"m-qq","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"$3","model":"m-$3"},"small":{"provider":"$3","model":"m-$3"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}$mcp}
JSON
}
# ac <thư mục> <args...>: chạy binary với env cô lập. CS=1 + HOSTARG=(--host unix://$SOCK) là chế độ server.
ac() {
  local d=$1; shift
  (cd "$d" && env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/c" XDG_DATA_HOME="$P/home/d" XDG_CACHE_HOME="$P/home/x" \
    CRUSH_GLOBAL_CONFIG="$P/home/cfg" CRUSH_GLOBAL_DATA="$P/home/data" CRUSH_CACHE_DIR="$P/home/cache" \
    CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 CRUSH_CLIENT_SERVER=$CS CRUSH_SERVER_IDLE_TIMEOUT=60 \
    "${EXTRA[@]}" timeout "${TO:-60}" "$BIN" "${HOSTARG[@]}" "$@")
}
server_mode() { CS=1; HOSTARG=(--host "unix://$SOCK"); }
# local_ac: như ac nhưng in-process (đọc DB trực tiếp, không đi qua server).
local_ac() { local CS=0 HOSTARG=(); ac "$@"; }
server_pid() { pgrep -f -- "server --host unix://$SOCK\$" | head -1; }
wait_gone() { for _ in $(seq 150); do kill -0 "$1" 2>/dev/null || return 0; sleep 0.1; done; return 1; }
stop_server() { local pid; pid=$(server_pid); [ -n "$pid" ] && kill -TERM "$pid" && wait_gone "$pid"; }
finish() { stop_server; kill "$FAKE" 2>/dev/null; wait "$FAKE" 2>/dev/null; }
git_repo() { (cd "$1" && git init -q && echo "# $(basename "$1")" > README.md && git add README.md && git -c user.name=t -c user.email=t@t commit -qm init); }
sid_of() { python3 -I -c 'import json,sys
try: print(json.load(sys.stdin).get("session_id",""))
except Exception: print("")'; }
# db_of <gốc> <session id>: crush.db nào dưới gốc chứa session (in đường tương đối theo gốc).
db_of() { python3 -I - "$1" "$2" <<'PY'
import os, sqlite3, sys
root, sid = sys.argv[1], sys.argv[2]
hits = []
for d, _, fs in os.walk(root):
    if "crush.db" in fs:
        p = os.path.join(d, "crush.db")
        try:
            c = sqlite3.connect("file:%s?mode=ro" % p, uri=True)
            if c.execute("select count(*) from sessions where id=?", (sid,)).fetchone()[0]:
                hits.append(os.path.relpath(p, root))
            c.close()
        except Exception as e:
            hits.append("%s(err:%s)" % (os.path.relpath(p, root), e))
print(",".join(hits) or "none")
PY
}

# ---------------------------------------------------------------------------------------------------------
# T2 (C3): server dựng từ thư mục A; client thứ hai ở repo/internal với `-c ..`.
# Bất biến: workspace của client hai là repo (đường tuyệt đối) và session của nó nằm trong repo/.crush/crush.db.
t2() {
  begin T2 '[{"match":"T2-two","steps":[{"sleep":6,"text":"two done"}]}]' qwen
  server_mode
  local W="$P/w/s" two sid ws
  mkdir -p "$W/A" "$W/repo/internal"; git_repo "$W/A"; git_repo "$W/repo"
  ac "$W/A" run --json "T2-one from A" > "$P/one.json" 2> "$P/one.err"
  ac "$W/repo/internal" -c "$W/repo" run --json "T2-two from repo/internal" > "$P/two.json" 2> "$P/two.err" &
  two=$!
  sleep 3
  curl -s --unix-socket "$SOCK" http://x/v1/workspaces > "$P/workspaces.json"
  ac "$W/repo" ps --json > "$P/ps.json" 2>/dev/null
  wait $two; echo $? > "$P/two.rc"
  sid=$(sid_of < "$P/two.json")
  finish
  find "$P/w" -name crush.db | sed "s|$P/||" | sort > "$P/dbs.txt"
  [ -n "$sid" ] || { verdict T2 INCONCLUSIVE "client two returned no session (rc=$(cat "$P/two.rc"), $(tail -c 200 "$P/two.err"))"; return; }
  ws=$(python3 -I -c 'import json,sys
d=json.load(open(sys.argv[1])); print(" ".join("%s|%s" % (w.get("path"), w.get("data_dir","")) for w in d))' "$P/workspaces.json" 2>/dev/null)
  local db; db=$(db_of "$P/w" "$sid")
  echo "workspaces(path|data_dir)=$ws session_db=$db dbs=$(tr '\n' ' ' < "$P/dbs.txt")" > "$P/summary.txt"
  if [ "$db" = "s/repo/.crush/crush.db" ] && ! grep -q '"path":"\.\."' "$P/workspaces.json"; then
    verdict T2 NOT_EXPOSED "session of client two in w/s/repo/.crush/crush.db; workspaces: ${ws//$P\//}"
  else
    verdict T2 EXPOSED "client two (-c .. from w/s/repo/internal, server cwd w/s/A): session stored in w/$db (want w/s/repo/.crush/crush.db); server workspaces path|data_dir: ${ws//$P\//}"
  fi
}

# ---------------------------------------------------------------------------------------------------------
# T3 (C2): mở repo1/sub → repo1 → repo1/sub (thứ tự 1); repo2 → repo2/sub (thứ tự 2), chế độ local.
# Bất biến: data dir của `sub` không phụ thuộc thứ tự mở; mỗi repo một crush.db; session thấy được như nhau từ
# gốc và từ sub.
t3() {
  begin T3 '[]' qwen
  local a b c d e
  mkdir -p "$P/repo1/sub" "$P/repo2/sub"; git_repo "$P/repo1"; git_repo "$P/repo2"
  a=$(ac "$P/repo1" run --json "T3 r1 sub first" 2>>"$P/err" | sid_of)
  b=$(ac "$P/repo1" run --json "T3 r1 root" 2>>"$P/err" | sid_of)
  c=$(ac "$P/repo1/sub" run --json "T3 r1 sub again" 2>>"$P/err" | sid_of)
  d=$(ac "$P/repo2" run --json "T3 r2 root first" 2>>"$P/err" | sid_of)
  e=$(ac "$P/repo2/sub" run --json "T3 r2 sub" 2>>"$P/err" | sid_of)
  ac "$P/repo1" session list --json > "$P/r1-root-list.json" 2>/dev/null
  ac "$P/repo1/sub" session list --json > "$P/r1-sub-list.json" 2>/dev/null
  finish
  [ -n "$a" ] && [ -n "$b" ] && [ -n "$c" ] && [ -n "$d" ] && [ -n "$e" ] || { verdict T3 INCONCLUSIVE "a run returned no session ($(tail -c 200 "$P/err"))"; return; }
  python3 -I - "$P" "$a" "$b" "$c" "$d" "$e" "$(db_of "$P/repo1" "$a")" "$(db_of "$P/repo1" "$b")" "$(db_of "$P/repo1" "$c")" "$(db_of "$P/repo2" "$d")" "$(db_of "$P/repo2" "$e")" > "$P/summary.txt" <<'PY'
import json, os, sys
P, a, b, c, d, e, da, db, dc, dd, de = sys.argv[1:12]
def ids(f):
    try:
        x = json.load(open(f))
    except Exception:
        return set()
    items = x if isinstance(x, list) else x.get("sessions", [])
    return {s.get("uuid") or s.get("id") for s in items}
n1 = sum(1 for dd_, _, fs in os.walk(os.path.join(P, "repo1")) if "crush.db" in fs)
n2 = sum(1 for dd_, _, fs in os.walk(os.path.join(P, "repo2")) if "crush.db" in fs)
root, sub = ids(os.path.join(P, "r1-root-list.json")), ids(os.path.join(P, "r1-sub-list.json"))
# Thứ tự 2 (gốc trước) cho DB của sub = de; thứ tự 1 cho DB của sub = da/dc.
order_independent = da == de.replace("repo2", "repo1") and dc == de.replace("repo2", "repo1")
same_view = root == sub == {a, b, c}
print("order1: sub-first db=%s root db=%s sub-again db=%s | order2: root db=%s sub db=%s | crush.db count repo1=%d repo2=%d | "
      "session list repo1 root sees %d/3, repo1/sub sees %d/3" % (da, db, dc, dd, de, n1, n2, len(root & {a, b, c}), len(sub & {a, b, c})))
print("EXPOSED" if not (order_independent and same_view and n1 == 1) else "NOT_EXPOSED")
PY
  local line; line=$(head -1 "$P/summary.txt")
  verdict T3 "$(tail -1 "$P/summary.txt")" "$line"
}

# ---------------------------------------------------------------------------------------------------------
# Stub MCP stdio có `instructions` (T5).
write_mcp_stub() {
  cat > "$P/mcp_stub.py" <<'PY'
import json, sys
for line in sys.stdin:
    try:
        msg = json.loads(line)
    except Exception:
        continue
    mid, method = msg.get("id"), msg.get("method", "")
    if mid is None:
        continue
    if method == "initialize":
        res = {"protocolVersion": msg.get("params", {}).get("protocolVersion", "2025-06-18"),
               "capabilities": {"tools": {}}, "serverInfo": {"name": "v3stub", "version": "0.0.1"},
               "instructions": "V3-MCP-INSTRUCTION-CANARY: always call stub_echo before answering."}
    elif method == "tools/list":
        res = {"tools": [{"name": "stub_echo", "description": "Echo text", "inputSchema": {"type": "object", "properties": {"text": {"type": "string"}}}}]}
    elif method == "tools/call":
        res = {"content": [{"type": "text", "text": "echo"}]}
    elif method == "ping":
        res = {}
    else:
        sys.stdout.write(json.dumps({"jsonrpc": "2.0", "id": mid, "error": {"code": -32601, "message": "method not found"}}) + "\n")
        sys.stdout.flush()
        continue
    sys.stdout.write(json.dumps({"jsonrpc": "2.0", "id": mid, "result": res}) + "\n")
    sys.stdout.flush()
PY
}

# T5 (M1): model chính gọi tool `agent`. Bất biến: system prompt của agent con (không có tool MCP) không chứa
# <mcp-instructions>. Đối chứng: request của agent chính PHẢI có nó (chứng minh MCP đã kết nối).
t5() {
  begin T5 '[{"match":"T5-sub","steps":[{"text":"sub done"}]},{"match":"T5-main","steps":[{"tool":"agent","args":{"prompt":"T5-sub list the files"}},{"text":"main done"}]}]' qwen \
    "{\"v3stub\":{\"type\":\"stdio\",\"command\":\"python3\",\"args\":[\"-I\",\"$OUT/T5/mcp_stub.py\"]}}"
  write_mcp_stub
  mkdir -p "$P/proj"; git_repo "$P/proj"
  ac "$P/proj" run --json "T5-main delegate to a sub agent" > "$P/out.json" 2> "$P/err.txt"; echo $? > "$P/rc"
  finish
  python3 -I - "$P" > "$P/summary.txt" <<'PY'
import json, os, sys
P = sys.argv[1]
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))] if os.path.exists(os.path.join(P, "log", "requests.jsonl")) else []
def sysof(r):
    p = os.path.join(P, "log", "sys-%s.txt" % r["system_sha"])
    return open(p).read() if os.path.exists(p) else ""
main = [r for r in reqs if r["role"] == "main"]
sub = [r for r in reqs if r["role"] == "subagent"]
main_has = any("<mcp-instructions>" in sysof(r) for r in main)
sub_has = [("<mcp-instructions>" in sysof(r)) for r in sub]
sub_mcp_tools = sorted({t for r in sub for t in r["tools"] if t.startswith("mcp_")})
print("main requests=%d with <mcp-instructions>=%s | subagent requests=%d with <mcp-instructions>=%d, subagent tools=%s, subagent mcp tools=%s" % (
    len(main), main_has, len(sub), sum(sub_has), sorted({t for r in sub for t in r["tools"]}), sub_mcp_tools))
if not sub or not main_has:
    print("INCONCLUSIVE")
elif any(sub_has) and not sub_mcp_tools:
    print("EXPOSED")
else:
    print("NOT_EXPOSED")
PY
  verdict T5 "$(tail -1 "$P/summary.txt")" "$(head -1 "$P/summary.txt") (rc=$(cat "$P/rc"))"
}

# ---------------------------------------------------------------------------------------------------------
# T6 (M2/M3/M4/M5): request của agentic_fetch, plan phi tương tác (plan bật qua API như TUI), agent con.
# Bất biến: system prompt không nhắc tool vắng mặt trong danh sách tool của CHÍNH request đó.
# Đo thêm (báo, không vào phán quyết): todo reminder "use the todos tool" trong messages khi không có `todos`.
t6() {
  begin T6 '[{"match":"T6-fetchsub","steps":[{"text":"fetch summary"}]},{"match":"T6-fetch","steps":[{"tool":"agentic_fetch","args":{"prompt":"T6-fetchsub what is agent-cli"}},{"text":"done"}]},{"match":"T6-sub","steps":[{"text":"sub done"}]},{"match":"T6-agent","steps":[{"tool":"agent","args":{"prompt":"T6-sub list the files"}},{"text":"done"}]},{"match":"T6-plan","steps":[{"text":"plan done"}]}]' qwen
  server_mode
  mkdir -p "$P/proj"; git_repo "$P/proj"
  ac "$P/proj" run --json "T6-fetch research it" > "$P/fetch.json" 2> "$P/fetch.err"
  ac "$P/proj" run --json "T6-agent delegate it" > "$P/agent.json" 2> "$P/agent.err"
  # Bật plan như TUI làm (POST agent/main) trên một workspace rảnh: tạo workspace qua API (giữ 30 s nhờ
  # create grace, không cần run nào đang chạy), rồi `run` gắn vào đúng workspace đó.
  local wsid
  curl -s --unix-socket "$SOCK" -H 'Content-Type: application/json' -d "{\"path\":\"$P/proj\",\"client_id\":\"$(python3 -I -c 'import uuid; print(uuid.uuid4())')\"}" \
    http://x/v1/workspaces > "$P/ws-create.json"
  wsid=$(python3 -I -c 'import json,sys; print(json.load(open(sys.argv[1])).get("id",""))' "$P/ws-create.json" 2>/dev/null)
  curl -s -o "$P/setplan.out" -w '%{http_code}' --unix-socket "$SOCK" -H 'Content-Type: application/json' \
    -d '{"agent_id":"plan"}' "http://x/v1/workspaces/$wsid/agent/main" > "$P/setplan.code"
  ac "$P/proj" run --json "T6-plan make a plan" > "$P/plan.json" 2> "$P/plan.err"
  finish
  python3 -I - "$P" > "$P/summary.txt" <<'PY'
import json, os, re, sys
P = sys.argv[1]
f = os.path.join(P, "log", "requests.jsonl")
reqs = [json.loads(l) for l in open(f)] if os.path.exists(f) else []
def sysof(r):
    p = os.path.join(P, "log", "sys-%s.txt" % r["system_sha"])
    return open(p).read() if os.path.exists(p) else ""
known = set().union(*[set(r["tools"]) for r in reqs]) if reqs else set()
known |= {"todos", "find_references", "question", "agentic_fetch", "agent", "bash", "edit", "multiedit", "write", "view",
          "glob", "grep", "ls", "fetch", "download", "sourcegraph", "web_fetch", "web_search", "job_output", "job_kill",
          "diagnostics", "references", "lsp_restart", "crush_logs"}
known = {t for t in known if not t.startswith("mcp_")}
def mentions(text, name):
    # Tên có "_" là từ riêng; tên là từ thường chỉ tính khi nằm trong `..`/".." hoặc ngay trước chữ tool.
    pats = [r"`%s`" % re.escape(name), r'"%s"' % re.escape(name), r"\b%s\b tool" % re.escape(name)]
    if "_" in name:
        pats.append(r"(?<![\w-])%s(?![\w-])" % re.escape(name))
    for p in pats:
        m = re.search(p, text)
        if m:
            return text[max(0, m.start() - 50):m.end() + 30].replace("\n", " ")
    return None
reminder = "<system_reminder>This is a reminder that your todo list"
rows, exposed = [], False
for role in ("agentic_fetch", "plan", "subagent", "main"):
    rs = [r for r in reqs if r["role"] == role]
    if not rs:
        rows.append("%s: no request" % role)
        continue
    r = rs[0]
    s = sysof(r)
    tools = set(r["tools"])
    absent = {}
    for name in sorted(known - tools):
        ctx = mentions(s, name)
        if ctx:
            absent[name] = ctx
    todo_reminder = any(u.startswith(reminder) for u in r["user_texts"]) and "todos" not in tools
    if role != "main":  # agent chính chỉ để tham khảo (M3), không vào phán quyết T6
        exposed |= bool(absent)
    rows.append("%s: tools=%s | system mentions absent=%s | todo-reminder-without-todos=%s" % (role, sorted(tools), sorted(absent), todo_reminder))
    for name, ctx in sorted(absent.items()):
        rows.append("    %s ← …%s…" % (name, ctx))
print("\n".join(rows))
have = all(any(r["role"] == role for r in reqs) for role in ("agentic_fetch", "plan", "subagent"))
print("EXPOSED" if exposed else ("NOT_EXPOSED" if have else "INCONCLUSIVE"))
PY
  local mentions; mentions=$(grep -E '^(agentic_fetch|plan|subagent):' "$P/summary.txt" | sed 's/tools=\[[^]]*\] | //' | tr '\n' ';')
  verdict T6 "$(tail -1 "$P/summary.txt")" "$mentions set-plan HTTP $(cat "$P/setplan.code")"
}

# ---------------------------------------------------------------------------------------------------------
# T11 (F1): `send` không --wait (LLM giả chậm 20 s), SIGKILL server ngay khi send thoát. Lặp N lần.
# Bất biến: send rc 0 ⇒ tin user đã có trong DB (session show đọc thẳng DB).
t11() {
  begin T11 '[{"match":"T11-trial","steps":[{"sleep":20,"text":"late"}]}]' qwen
  server_mode
  mkdir -p "$P/proj"; git_repo "$P/proj"
  local n=${T11_TRIALS:-8} i pid rc sid lost=0 ok=0 bad=0 control=no delay
  # Lượt cuối là ĐỐI CHỨNG: chờ 2 s rồi mới SIGKILL; tin user phải có trong DB, nếu không thì phép đo hỏng.
  for i in $(seq $((n + 1))); do
    delay=0; [ "$i" -gt "$n" ] && delay=2
    ac "$P/proj" run --json "warm $i" > /dev/null 2>&1
    pid=$(server_pid)
    [ -n "$pid" ] || { echo "trial $i: no server" >> "$P/trials.txt"; bad=$((bad + 1)); continue; }
    ac "$P/proj" send --new --title t11 "T11-trial-$i prompt" > "$P/send-$i.out" 2> "$P/send-$i.err"; rc=$?
    [ "$delay" = 0 ] || sleep "$delay"
    kill -9 "$pid"
    wait_gone "$pid"
    sid=$(tail -1 "$P/send-$i.out")
    local_ac "$P/proj" session show "$sid" --json > "$P/show-$i.json" 2> "$P/show-$i.err"
    if grep -q "T11-trial-$i prompt" "$P/show-$i.json"; then indb=yes; else indb=no; fi
    echo "trial $i (kill after ${delay}s): send rc=$rc session=$sid user_message_in_db=$indb" >> "$P/trials.txt"
    if [ "$delay" != 0 ]; then control=$indb
    elif [ "$rc" = 0 ] && [ "$indb" = no ]; then lost=$((lost + 1)); elif [ "$rc" = 0 ]; then ok=$((ok + 1)); else bad=$((bad + 1)); fi
  done
  finish
  local why="trials=$n killed at once: rc0+in_db=$ok, rc0+missing=$lost, other=$bad; control (kill after 2 s) in_db=$control (details T11/trials.txt)"
  if [ "$control" != yes ]; then verdict T11 INCONCLUSIVE "$why — the control did not find the message, measurement broken"
  elif [ "$lost" -gt 0 ]; then verdict T11 EXPOSED "$why"
  elif [ "$ok" -gt 0 ]; then verdict T11 NOT_EXPOSED "$why; the race window was not hit in $ok trials"
  else verdict T11 INCONCLUSIVE "$why"; fi
}

# ---------------------------------------------------------------------------------------------------------
# T13 (F2): gửi A (LLM giả ngủ 10 s), gửi B vào cùng session (xếp hàng), SIGTERM server.
# Bất biến: DB có B, hoặc có bản ghi huỷ cho B.
t13() {
  begin T13 '[{"match":"T13-A","steps":[{"text":"A done"}]},{"match":"T13-B","steps":[{"text":"B done"}]}]' qwen
  server_mode
  mkdir -p "$P/proj"; git_repo "$P/proj"
  ac "$P/proj" run --json "warm" > /dev/null 2>&1
  local pid sid rcB
  pid=$(server_pid)
  sid=$(ac "$P/proj" send --new --title t13 "T13-A long turn" 2> "$P/sendA.err" | tail -1)
  sleep 1.5
  ac "$P/proj" send -s "$sid" "T13-B queued prompt" > "$P/sendB.out" 2> "$P/sendB.err"; rcB=$?
  sleep 0.5
  ac "$P/proj" ps --json > "$P/ps.json" 2>/dev/null
  [ -n "$pid" ] && kill -TERM "$pid" && wait_gone "$pid"
  local_ac "$P/proj" session show "$sid" --json > "$P/show.json" 2> "$P/show.err"
  finish
  python3 -I - "$P" "$sid" "$rcB" > "$P/summary.txt" <<'PY'
import glob, json, os, sys
P, sid, rcB = sys.argv[1:4]
raw = b"".join(open(f, "rb").read() for f in glob.glob(os.path.join(P, "proj", ".crush", "crush.db*")))
show = open(os.path.join(P, "show.json")).read()
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))]
b_sent = any("T13-B" in r["last_user"] for r in reqs)
a_db, b_db = "T13-A" in show, "T13-B" in show
b_raw = b"T13-B" in raw
print("send B rc=%s | session show: A=%s B=%s | 'T13-B' bytes anywhere in crush.db*=%s | B reached LLM=%s" % (rcB, a_db, b_db, b_raw, b_sent))
if not sid or rcB != "0" or not a_db:
    print("INCONCLUSIVE")
else:
    print("NOT_EXPOSED" if (b_db or b_raw) else "EXPOSED")
PY
  verdict T13 "$(tail -1 "$P/summary.txt")" "$(head -1 "$P/summary.txt")"
}

# ---------------------------------------------------------------------------------------------------------
# T17 (H / run -s): X tạo bằng provider pp; dự án đổi sang qq (crush.json của repo); Y chạy bằng qq;
# `run -s X` qua server. Bất biến: <data>/crush.json không bị ghi model; lượt sau của Y (send, như TUI) và
# session mới vẫn đi qq.
t17() {
  begin T17 '[]' pp
  server_mode
  mkdir -p "$P/proj"; git_repo "$P/proj"
  local x y before after y2 n1
  x=$(ac "$P/proj" run --json "T17-X1 on pp" 2>>"$P/err" | sid_of)
  stop_server
  printf '%s' '{"models":{"large":{"provider":"qq","model":"m-qq"},"small":{"provider":"qq","model":"m-qq"}}}' > "$P/proj/crush.json"
  y=$(ac "$P/proj" run --json "T17-Y1 on project default" 2>>"$P/err" | sid_of)
  before=$(cat "$P/proj/.crush/crush.json" 2>/dev/null || echo "<absent>")
  ac "$P/proj" send -s "$x" --wait --json "T17-X2 continue X" > "$P/x2.json" 2>>"$P/err"
  after=$(cat "$P/proj/.crush/crush.json" 2>/dev/null || echo "<absent>")
  ac "$P/proj" send -s "$y" --wait --json "T17-Y2 next turn of Y" > "$P/y2.json" 2>>"$P/err"
  ac "$P/proj" run --json "T17-N1 brand new session" > "$P/n1.json" 2>>"$P/err"
  finish
  printf '%s\n' "$before" > "$P/data-crush.before.json"; printf '%s\n' "$after" > "$P/data-crush.after.json"
  python3 -I - "$P" "$x" "$y" > "$P/summary.txt" <<'PY'
import json, os, sys
P, x, y = sys.argv[1:4]
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))]
def prov(tag):
    return sorted({r["provider"] for r in reqs if r["role"] == "main" and tag in r["last_user"]}) or ["none"]
before = open(os.path.join(P, "data-crush.before.json")).read().strip()
after = open(os.path.join(P, "data-crush.after.json")).read().strip()
try:
    models = json.loads(after).get("models", {}) if after != "<absent>" else {}
except Exception:
    models = {"unparsable": after[:80]}
print("X1->%s Y1->%s X2->%s Y2->%s N1->%s | <data>/crush.json before run -s X: %s | after: models=%s" % (
    prov("T17-X1"), prov("T17-Y1"), prov("T17-X2"), prov("T17-Y2"), prov("T17-N1"), before[:60], json.dumps(models)))
if not (x and y) or prov("T17-X1") != ["pp"] or prov("T17-Y1") != ["qq"]:
    print("INCONCLUSIVE")
elif before == after and prov("T17-Y2") == ["qq"] and prov("T17-N1") == ["qq"]:
    print("NOT_EXPOSED")
else:
    print("EXPOSED")
PY
  verdict T17 "$(tail -1 "$P/summary.txt")" "$(head -1 "$P/summary.txt")"
}

# ---------------------------------------------------------------------------------------------------------
# T21 (§33): model chạy `env` (+ output dài để có spill) và `view .env`; loop có verify in env. Đo: số tệp
# (ngoài .env gốc và log của LLM giả) còn giữ secret giả. Không có đúng/sai cứng: báo số. Ba lượt, mỗi lượt
# một thư mục ca riêng: server mặc định, server với `-d` của client, local với `-d`.
t21_one() {
  local name=$1 dbg=$2 mode=$3
  begin "$name" '[{"match":"T21-go","steps":[{"tool":"bash","args":{"command":"env; seq 1 40000","description":"print env"}},{"tool":"view","args":{"file_path":".env"}},{"text":"done"}]}]' qwen
  [ "$mode" = server ] && server_mode
  mkdir -p "$P/proj"; git_repo "$P/proj"
  printf 'API_TOKEN=SECRET_V3_CANARY_FILE_7f3a\n' > "$P/proj/.env"
  EXTRA=(SECRET_V3_CANARY_ENV=SECRET_V3_CANARY_ENV_9c1d)
  ac "$P/proj" $dbg run --json "T21-go inspect the environment" > "$P/out.json" 2> "$P/err.txt"; echo $? > "$P/rc"
  TO=90 local_ac "$P/proj" loop --json --max-repairs 0 --verify 'env | grep SECRET_V3_CANARY' "T21-loop check" > "$P/loop.json" 2> "$P/loop.err"
  finish
  python3 -I - "$P" > "$P/summary.txt" <<'PY'
import json, os, sys
P = sys.argv[1]
skip = {os.path.join(P, "proj", ".env"), os.path.join(P, "rules.json"), os.path.join(P, "summary.txt")}
hits = []
for d, _, fs in os.walk(P):
    if d.startswith(os.path.join(P, "log")):
        continue  # phía provider giả, không phải bề mặt lưu của agent-cli
    for fn in fs:
        p = os.path.join(d, fn)
        if p in skip or os.path.islink(p) or not os.path.isfile(p):
            continue
        try:
            b = open(p, "rb").read()
        except Exception:
            continue
        nf, ne = b.count(b"SECRET_V3_CANARY_FILE_7f3a"), b.count(b"SECRET_V3_CANARY_ENV_9c1d")
        if nf or ne:
            hits.append((os.path.relpath(p, P), nf, ne, oct(os.stat(p).st_mode & 0o777)))
def kind(p):
    for k in ("crush.db", "shell-output", "loops", "crush.log", "stdout.log", "stderr.log", "out.json", "loop.json"):
        if k in p:
            return k
    return "other"
for h in sorted(hits):
    print("  %-70s file_canary=%d env_canary=%d mode=%s [%s]" % (h[0], h[1], h[2], h[3], kind(h[0])))
print("files_holding_canary=%d (crush.db*=%d shell-output=%d logs=%d journal=%d other=%d)" % (
    len(hits), sum(kind(h[0]) == "crush.db" for h in hits), sum(kind(h[0]) == "shell-output" for h in hits),
    sum(kind(h[0]) in ("crush.log", "stdout.log", "stderr.log") for h in hits), sum(kind(h[0]) == "loops" for h in hits),
    sum(kind(h[0]) not in ("crush.db", "shell-output", "crush.log", "stdout.log", "stderr.log", "loops") for h in hits)))
PY
  echo "$(tail -1 "$P/summary.txt") (run rc=$(cat "$P/rc"))"
}
t21() {
  local a b c
  a=$(t21_one T21 "" server)
  b=$(t21_one T21d -d server)
  c=$(t21_one T21l -d local)
  if grep -q "files_holding_canary=0 " <<< "$a"; then verdict T21 NOT_EXPOSED "server: $a | server -d: $b | local -d: $c"
  else verdict T21 EXPOSED "measurement, not a gate — server: $a | server -d: $b | local -d: $c (per-file lists: T21*/summary.txt)"; fi
}

for c in $CASES; do
  case "$c" in
    T2) t2 ;; T3) t3 ;; T5) t5 ;; T6) t6 ;; T11) t11 ;; T13) t13 ;; T17) t17 ;; T21) t21 ;;
    *) verdict "$c" INCONCLUSIVE "unknown case" ;;
  esac
done
echo "EXPOSURE: ran $(wc -l < "$OUT/verdicts.txt") case(s); verdicts in $OUT/verdicts.txt"
exit 0
