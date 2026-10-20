#!/usr/bin/env bash
# Test phơi nhiễm V3 (reports/v3-research/experiments/00_EXPOSURE_TESTS.md) với binary thật + LLM giả
# (scripts/eval/fakellm.py). Mỗi ca khẳng định BẤT BIẾN MONG MUỐN rồi in đúng một dòng
#   T<n>: EXPOSED|NOT_EXPOSED|INCONCLUSIVE|MEASURED — lý do (giá trị thật)
# EXPOSED = phơi nhiễm được xác nhận trên binary này; MEASURED = phép đo không có đúng/sai cứng. Ca nào ra
# EXPOSED đều có ĐỐI CHỨNG chạy trong cùng ca, bỏ đúng điều kiện gây phơi nhiễm, phải ra xanh (ghi trong lý do).
# Đây là bộ ĐO, không phải cổng: thoát 0 khi chạy đủ các ca bất kể kết quả; thoát 2 khi sai cách dùng.
# Bằng chứng thô của từng ca nằm trong <thư-mục>/<ca>/ (summary.txt, log/requests.jsonl, log/sys-*.txt...).
# Cô lập như faults.sh: mỗi ca HOME/XDG/CRUSH_GLOBAL_*/AGENT_CLI_RUNTIME_DIR riêng trong thư mục ca, socket riêng,
# tmux riêng (T4), không LLM thật. Launcher thật (install-agent-cli.sh) cài vào <thư-mục>/_launcher với
# AGENT_CLI_SKIP_BUILD=1 và một shim gọi <binary>.
# Dùng: bash scripts/eval/exposure.sh <thư-mục-mới, ≤ 70 byte> <binary> [ca...]
set -u
[ $# -ge 2 ] || { echo "usage: $0 <new-dir> <binary> [cases...]" >&2; exit 2; }
OUT=$1; BIN=$2; shift 2
CASES=${*:-T1M T2 T3 T4 T5 T6 T9 T11 T13 T17 T19 T20 T21 T23 C5}
R=/home/luannt/application/opensource/agent-cli-p1b
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
[ -x "$BIN" ] || { echo "not executable: $BIN" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd) && BIN=$(cd "$(dirname "$BIN")" && pwd)/$(basename "$BIN")
[ ${#OUT} -le 70 ] || { echo "dir path too long for unix sockets (${#OUT} > 70 bytes): $OUT" >&2; exit 2; }
L="$OUT/_launcher"

verdict() { printf '%s: %s — %s\n' "$1" "$2" "$3" | tee -a "$OUT/verdicts.txt"; }

# Launcher thật, cài một lần cho cả lượt chạy (T2c, T4).
install_launcher() {
  [ -x "$L/inst/bin/agent-cli" ] && return 0
  mkdir -p "$L/fake/scripts" "$L/inst/lib/agent-cli" "$L/home" "$L/rt"
  cp "$R/scripts/install-agent-cli.sh" "$R/scripts/agent-cli-tmux.sh" "$L/fake/scripts/"
  cp "$R/agent-cli.crushrc.example" "$R/agent-cli.env.example" "$L/fake/"
  printf '#!/bin/sh\nexec %s "$@"\n' "$BIN" > "$L/inst/lib/agent-cli/crush"
  chmod 755 "$L/inst/lib/agent-cli/crush"
  env -i PATH=/usr/bin:/bin HOME="$L/home" AGENT_CLI_RUNTIME_DIR="$L/rt" AGENT_CLI_INSTALL_DIR="$L/inst" AGENT_CLI_SKIP_BUILD=1 \
    bash "$L/fake/scripts/install-agent-cli.sh" > "$L/install.log" 2>&1
}

# begin <ca> <luật fakellm JSON> <provider large> [mcp JSON]: thư mục ca, LLM giả, cấu hình global theo đúng
# bố cục của launcher ($HOME/.config/agent-cli, $HOME/.local/share/agent-cli, $HOME/.cache/agent-cli).
begin() {
  P="$OUT/$1"; HOSTARG=(); CS=0; EXTRA=(); XPATH=""
  mkdir -p "$P/home/.config/agent-cli" "$P/log" "$P/rt"; chmod 700 "$P/rt"
  printf '%s' "$2" > "$P/rules.json"
  timeout 900 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
  FAKE=$!
  for _ in $(seq 50); do [ -s "$P/port" ] && break; sleep 0.1; done
  PORT=$(cat "$P/port" 2>/dev/null)
  local mcp=""
  [ -n "${4:-}" ] && mcp=",\"mcp\":$4"
  # Ba provider giả trên cùng một LLM giả: request mang tên provider trong đường dẫn (fakellm.py).
  cat > "$P/home/.config/agent-cli/crush.json" <<JSON
{"providers":{
  "qwen":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/qwen/v1","api_key":"x","models":[{"id":"m-qwen","name":"m-qwen","context_window":100000,"default_max_tokens":1000}]},
  "pp":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/pp/v1","api_key":"x","models":[{"id":"m-pp","name":"m-pp","context_window":100000,"default_max_tokens":1000}]},
  "qq":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/qq/v1","api_key":"x","models":[{"id":"m-qq","name":"m-qq","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"$3","model":"m-$3"},"small":{"provider":"$3","model":"m-$3"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}$mcp}
JSON
}
# iso <thư mục> <lệnh...>: chạy lệnh với env cô lập của ca.
iso() {
  local d=$1; shift
  (cd "$d" && env -i PATH="${XPATH:+$XPATH:}/usr/bin:/bin" HOME="$P/home" XDG_CONFIG_HOME="$P/home/.config" \
    XDG_DATA_HOME="$P/home/.local/share" XDG_CACHE_HOME="$P/home/.cache" CRUSH_GLOBAL_CONFIG="$P/home/.config/agent-cli" \
    CRUSH_GLOBAL_DATA="$P/home/.local/share/agent-cli" CRUSH_CACHE_DIR="$P/home/.cache/agent-cli" AGENT_CLI_RUNTIME_DIR="$P/rt" \
    GIT_CEILING_DIRECTORIES="$OUT" CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 \
    CRUSH_CLIENT_SERVER=$CS CRUSH_SERVER_IDLE_TIMEOUT=60 "${EXTRA[@]}" timeout "${TO:-60}" "$@")
}
ac() { local d=$1; shift; iso "$d" "$BIN" "${HOSTARG[@]}" "$@"; }        # binary trực tiếp
la() { local d=$1; shift; local CS=1; iso "$d" "$L/inst/bin/agent-cli" "$@"; } # qua launcher thật
local_ac() { local CS=0 HOSTARG=(); ac "$@"; }                               # in-process, đọc DB trực tiếp
server_mode() { CS=1; SOCK="$P/${1:-s}.sock"; HOSTARG=(--host "unix://$SOCK"); }
server_pid_of() { pgrep -f -- "server --host unix://$1\$" | head -1; }
server_cwd() { local pid; pid=$(server_pid_of "$1"); [ -n "$pid" ] && readlink "/proc/$pid/cwd"; }
wait_gone() { for _ in $(seq 150); do kill -0 "$1" 2>/dev/null || return 0; sleep 0.1; done; return 1; }
stop_servers() { local pid; for pid in $(pgrep -f -- "server --host unix://$P/"); do kill -TERM "$pid" 2>/dev/null; wait_gone "$pid"; done; }
finish() { stop_servers; kill "$FAKE" 2>/dev/null; wait "$FAKE" 2>/dev/null; }
git_repo() { (cd "$1" && git init -q && echo "# $(basename "$1")" > README.md && git add -A && git -c user.name=t -c user.email=t@t commit -qm init); }
sid_of() { python3 -I -c 'import json,sys
try: print(json.load(sys.stdin).get("session_id",""))
except Exception: print("")'; }
# db_of <gốc> <session id>: crush.db nào dưới gốc chứa session (đường tương đối theo gốc).
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
write_mcp_stub() {
  cat > "$P/mcp_stub.py" <<'PY'
import json, sys
BIG = "".join("line %07d %s\n" % (i, "x" * 48) for i in range(5 * 1024 * 1024 // 62 + 1))[:5 * 1024 * 1024]
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
        obj = {"type": "object", "properties": {}}
        res = {"tools": [{"name": "stub_echo", "description": "Echo text", "inputSchema": obj},
                         {"name": "stub_big", "description": "Return a 5 MiB text", "inputSchema": obj},
                         {"name": "stub_fail", "description": "Always fails", "inputSchema": obj}]}
    elif method == "tools/call":
        name = msg.get("params", {}).get("name")
        if name == "stub_big":
            res = {"content": [{"type": "text", "text": BIG}]}
        elif name == "stub_fail":
            res = {"content": [{"type": "text", "text": "V3-MCP-FAIL boom"}], "isError": True}
        else:
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
mcp_cfg() { printf '{"v3stub":{"type":"stdio","command":"python3","args":["-I","%s"]}}' "$OUT/$1/mcp_stub.py"; }

# ---------------------------------------------------------------------------------------------------------
# T1M (C1, tác hại): repo và repo/sub mở trên CÙNG server (launcher chọn socket theo git root) thành hai workspace
# (khoá repo và repo/sub) dùng chung repo/.crush/crush.db. Session S đang chạy (BUSY) ở workspace repo; một prompt
# cho S gửi từ repo/sub. Bất biến: S bận thì prompt thứ hai phải xếp hàng như khi gửi từ cùng workspace, không
# chạy song song trên cùng session. Đối chứng: prompt thứ hai gửi từ repo (cùng workspace) phải xếp hàng.
t1m() {
  begin T1M '[{"match":"T1M-long","steps":[{"sleep":8,"text":"long done"}]},{"match":"T1M-second","steps":[{"text":"second done"}]}]' qwen
  mkdir -p "$P/repo/sub"; echo x > "$P/repo/sub/a.txt"; git_repo "$P/repo"
  local v dir2 sid
  for v in harm control; do
    server_mode "$v"; dir2="$P/repo/sub"; [ $v = control ] && dir2="$P/repo"
    ac "$P/repo" run --json "warm $v" > /dev/null 2>&1
    sid=$(ac "$P/repo" send --new --title t1m "T1M-long $v" 2>"$P/$v-send1.err" | tail -1)
    sleep 1.5
    ac "$dir2" send -s "$sid" "T1M-second $v" > "$P/$v-send2.out" 2> "$P/$v-send2.err"; echo $? > "$P/$v-send2.rc"
    sleep 1
    ac "$P/repo" ps --json > "$P/$v-ps.json" 2>/dev/null
    curl -s --unix-socket "$SOCK" http://x/v1/workspaces > "$P/$v-workspaces.json"
    sleep 8
    echo "$sid" > "$P/$v-sid"
    stop_servers
  done
  finish
  python3 -I - "$P" > "$P/summary.txt" <<'PY'
import json, os, sys
P = sys.argv[1]
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))]
def ts(tag):
    t = [r["ts"] for r in reqs if r["role"] == "main" and r["last_user"].lstrip().startswith(tag)]
    return min(t) if t else None
dbs = sorted(os.path.relpath(os.path.join(d, "crush.db"), P) for d, _, fs in os.walk(os.path.join(P, "repo")) if "crush.db" in fs)
out, conc = [], {}
for v in ("harm", "control"):
    sid = open(os.path.join(P, v + "-sid")).read().strip()
    t1, t2 = ts("T1M-long " + v), ts("T1M-second " + v)
    ps = json.load(open(os.path.join(P, v + "-ps.json"))) if os.path.getsize(os.path.join(P, v + "-ps.json")) else []
    rows = [(os.path.relpath(r["path"], P), r["busy"]) for r in ps if r.get("session_id") == sid]
    ws = [os.path.relpath(w["path"], P) for w in json.load(open(os.path.join(P, v + "-workspaces.json")))]
    conc[v] = t1 is not None and t2 is not None and t2 < t1 + 7.5
    out.append("%s: second prompt sent %.1fs after the long turn started (long turn lasts 8s) -> %s; ps rows for S (path,busy)=%s; workspaces=%s" % (
        v, (t2 - t1) if (t1 and t2) else float("nan"), "RAN CONCURRENTLY" if conc[v] else "queued", rows, ws))
print("; ".join(out) + "; crush.db under repo=%s" % dbs)
if conc["control"] or ts("T1M-long harm") is None:
    print("INCONCLUSIVE")
else:
    print("EXPOSED" if conc["harm"] else "NOT_EXPOSED")
PY
  verdict T1M "$(tail -1 "$P/summary.txt")" "$(head -1 "$P/summary.txt")"
}

# ---------------------------------------------------------------------------------------------------------
# T2 (C3): client ở w/s/repo/internal chạy `-c ..`. Bất biến: workspace là w/s/repo và session của nó nằm trong
# w/s/repo/.crush/crush.db. Biến thể (mỗi biến thể một cây w/ riêng):
#   a   server dùng chung, do một client khác dựng từ thư mục A không liên quan;
#   b1  client tự dựng server (socket mới), repo chưa có .crush;
#   b2  như b1 nhưng repo đã có .crush (đã mở một lần);
#   c   launcher thật: socket theo git root như install-agent-cli.sh:115-135; server của repo đã được dựng từ gốc repo;
#   ctl đối chứng của a: cùng server dùng chung, client hai dùng `-c <repo tuyệt đối>`.
t2() {
  begin T2 '[{"match":"T2-two","steps":[{"sleep":4,"text":"two done"}]}]' qwen
  install_launcher
  local v W sid ws sock cwd db
  for v in a b1 b2 c ctl; do
    W="$P/$v/w/s"; mkdir -p "$W/A" "$W/repo/internal"; echo x > "$W/repo/internal/x.go"; git_repo "$W/A"; git_repo "$W/repo"
    case $v in
      a | ctl) server_mode "$v"; ac "$W/A" run --json "T2-one from A" > "$P/$v-one.json" 2>&1 ;;
      b1) server_mode "$v" ;;
      b2) server_mode "$v"; local_ac "$W/repo" run --json "T2-one local in repo" > "$P/$v-one.json" 2>&1 ;;
      c) la "$W/repo" run --json "T2-one from repo root" > "$P/$v-one.json" 2>&1 ;;
    esac
    local arg=..; [ $v = ctl ] && arg="$W/repo"
    if [ $v = c ]; then
      la "$W/repo/internal" -c "$arg" run --json "T2-two $v" > "$P/$v-two.json" 2> "$P/$v-two.err" &
    else
      ac "$W/repo/internal" -c "$arg" run --json "T2-two $v" > "$P/$v-two.json" 2> "$P/$v-two.err" &
    fi
    local two=$!
    sleep 2
    sock=$SOCK; [ $v = c ] && sock=$(ls "$P"/rt/agent-cli-*.sock 2>/dev/null | head -1)
    cwd=$(server_cwd "$sock")
    curl -s --unix-socket "$sock" http://x/v1/workspaces > "$P/$v-workspaces.json"
    wait $two
    sid=$(sid_of < "$P/$v-two.json")
    stop_servers
    db=$(db_of "$P/$v" "$sid")
    python3 -I - "$P" "$v" "$cwd" "$db" "$sid" "$W" >> "$P/variants.txt" <<'PY'
import json, os, sys
P, v, cwd, db, sid, W = sys.argv[1:7]
try:
    ws = json.load(open(os.path.join(P, v + "-workspaces.json")))
except Exception:
    ws = []
want = os.path.relpath(os.path.join(W, "repo", ".crush", "crush.db"), os.path.join(P, v))
rows = []
for w in ws:
    path, dd = w.get("path", ""), w.get("data_dir", "")
    root = os.path.normpath(os.path.join(cwd, path)) if cwd else "?"
    droot = os.path.normpath(os.path.join(cwd, dd)) if cwd else "?"
    rows.append("path=%r(-> %s) data_dir=%r(-> %s)" % (path, os.path.relpath(root, P), dd, os.path.relpath(droot, P)))
ok = bool(sid) and db == want
print("%s|%s|server_cwd=%s; workspaces: %s; session db=%s (want %s)" % (
    v, "OK" if ok else ("NO_SESSION" if not sid else "WRONG"), os.path.relpath(cwd, P) if cwd else "?", " / ".join(rows) or "none", db, want))
PY
  done
  finish
  local bad; bad=$(awk -F'|' '$1 != "ctl" && $2 == "WRONG" {printf "%s ", $1}' "$P/variants.txt")
  local ctl; ctl=$(awk -F'|' '$1 == "ctl" {print $2}' "$P/variants.txt")
  cp "$P/variants.txt" "$P/summary.txt"
  if grep -q '|NO_SESSION|' "$P/variants.txt" || [ "$ctl" != OK ]; then
    verdict T2 INCONCLUSIVE "control=$ctl; $(tr '\n' ' ' < "$P/variants.txt")"
  elif [ -n "$bad" ]; then
    verdict T2 EXPOSED "wrong variants: $bad| control (absolute -c)=OK | $(tr '\n' ' ' < "$P/variants.txt")"
  else
    verdict T2 NOT_EXPOSED "$(tr '\n' ' ' < "$P/variants.txt")"
  fi
}

# ---------------------------------------------------------------------------------------------------------
# T3 (C2): mở repo1/sub → repo1 → repo1/sub (thứ tự 1); repo2 → repo2/sub (thứ tự 2, đối chứng).
# Bất biến: data dir của `sub` không phụ thuộc thứ tự mở; mỗi repo một crush.db; gốc và sub thấy cùng tập session.
# Chạy hai chế độ: local (in-process) và server (mỗi repo một socket, như launcher).
t3_run() {
  local mode=$1 a b c d e
  mkdir -p "$P/$mode/repo1/sub" "$P/$mode/repo2/sub"; git_repo "$P/$mode/repo1"; git_repo "$P/$mode/repo2"
  [ "$mode" = server ] && server_mode r1
  a=$(ac "$P/$mode/repo1/sub" run --json "T3 r1 sub first" 2>>"$P/err" | sid_of)
  b=$(ac "$P/$mode/repo1" run --json "T3 r1 root" 2>>"$P/err" | sid_of)
  c=$(ac "$P/$mode/repo1/sub" run --json "T3 r1 sub again" 2>>"$P/err" | sid_of)
  [ "$mode" = server ] && server_mode r2
  d=$(ac "$P/$mode/repo2" run --json "T3 r2 root first" 2>>"$P/err" | sid_of)
  e=$(ac "$P/$mode/repo2/sub" run --json "T3 r2 sub" 2>>"$P/err" | sid_of)
  [ "$mode" = server ] && server_mode r1
  ac "$P/$mode/repo1" session list --json > "$P/$mode-r1-root-list.json" 2>/dev/null
  ac "$P/$mode/repo1/sub" session list --json > "$P/$mode-r1-sub-list.json" 2>/dev/null
  stop_servers
  python3 -I - "$P" "$mode" "$a" "$b" "$c" "$d" "$e" "$(db_of "$P/$mode/repo1" "$a")" "$(db_of "$P/$mode/repo1" "$b")" "$(db_of "$P/$mode/repo1" "$c")" "$(db_of "$P/$mode/repo2" "$d")" "$(db_of "$P/$mode/repo2" "$e")" <<'PY'
import json, os, sys
P, mode, a, b, c, d, e, da, db, dc, dd, de = sys.argv[1:13]
def ids(f):
    try:
        x = json.load(open(f))
    except Exception:
        return set()
    items = x if isinstance(x, list) else x.get("sessions", [])
    return {s.get("uuid") or s.get("id") for s in items}
n1 = sum(1 for _, _, fs in os.walk(os.path.join(P, mode, "repo1")) if "crush.db" in fs)
n2 = sum(1 for _, _, fs in os.walk(os.path.join(P, mode, "repo2")) if "crush.db" in fs)
root, sub = ids(os.path.join(P, mode + "-r1-root-list.json")), ids(os.path.join(P, mode + "-r1-sub-list.json"))
ok_ids = all([a, b, c, d, e])
exposed = not (da == de and dc == de and root == sub == {a, b, c} and n1 == 1)
print("%s|%s|order1 sub-first db=%s root db=%s sub-again db=%s | order2 (control) root db=%s sub db=%s | crush.db count repo1=%d repo2=%d | repo1 root sees %d/3, repo1/sub sees %d/3" % (
    mode, "INCONCLUSIVE" if not ok_ids else ("EXPOSED" if exposed else "NOT_EXPOSED"), da, db, dc, dd, de, n1, n2, len(root & {a, b, c}), len(sub & {a, b, c})))
PY
}
t3() {
  begin T3 '[]' qwen
  { t3_run local; t3_run server; } > "$P/summary.txt"
  finish
  local v=NOT_EXPOSED
  grep -q '|EXPOSED|' "$P/summary.txt" && v=EXPOSED
  grep -q '|INCONCLUSIVE|' "$P/summary.txt" && v=INCONCLUSIVE
  verdict T3 "$v" "$(tr '\n' ' ' < "$P/summary.txt")"
}

# ---------------------------------------------------------------------------------------------------------
# T4 (C6): `agent-cli-tmux rm <tên> --force` khi worktree còn session BUSY. Launcher thật + agent-cli-tmux thật,
# tmux server riêng (wrapper -S như scripts/test-agent-cli-tmux.sh). Bất biến: rm từ chối khi có session BUSY ở
# BẤT KỲ đâu trong worktree. Đối chứng: session BUSY ở đúng gốc worktree phải bị từ chối.
t4() {
  begin T4 '[{"match":"T4-busy","steps":[{"sleep":12,"text":"busy done"}]}]' qwen
  install_launcher
  mkdir -p "$P/bin" "$P/repo/sub"; echo x > "$P/repo/sub/a.txt"; git_repo "$P/repo"
  printf '#!/bin/sh\nexec %s -S %s "$@"\n' "$(command -v tmux)" "$P/tmux.sock" > "$P/bin/tmux"; chmod +x "$P/bin/tmux"
  XPATH="$L/inst/bin:$P/bin"
  local v wt at busy
  for v in control sub; do
    wt="$P/repo-agents/t4$v"
    git -C "$P/repo" worktree add -q -b "agent/t4$v" "$wt"
    at="$wt"; [ $v = sub ] && at="$wt/sub"
    TO=40 la "$at" run --json "T4-busy $v" > "$P/$v-busy.json" 2> "$P/$v-busy.err" &
    busy=$!
    sleep 3
    la "$P/repo" ps --json --all > "$P/$v-ps.json" 2> "$P/$v-ps.err"
    CS=1 iso "$P/repo" "$L/inst/bin/agent-cli-tmux" rm "t4$v" --repo "$P/repo" --force > "$P/$v-rm.out" 2> "$P/$v-rm.err"; echo $? > "$P/$v-rm.rc"
    [ -e "$wt" ] && echo yes > "$P/$v-wt-exists" || echo no > "$P/$v-wt-exists"
    wait $busy; echo $? > "$P/$v-busy.rc"
    stop_servers
  done
  "$(command -v tmux)" -S "$P/tmux.sock" kill-server 2>/dev/null
  finish
  python3 -I - "$P" > "$P/summary.txt" <<'PY'
import json, os, sys
P = sys.argv[1]
def rd(f):
    p = os.path.join(P, f)
    return open(p).read().strip() if os.path.exists(p) else ""
res = {}
for v in ("control", "sub"):
    try:
        ps = json.loads(rd(v + "-ps.json") or "[]")
    except Exception:
        ps = []
    busy = [os.path.relpath(r["path"], P) for r in ps if r.get("busy")]
    res[v] = (rd(v + "-rm.rc"), rd(v + "-wt-exists"), busy)
    print("%s: busy rows=%s | rm --force rc=%s, worktree still exists=%s, rm stderr=%r | busy run rc=%s out=%r" % (
        v, busy, res[v][0], res[v][1], rd(v + "-rm.err")[-160:], rd(v + "-busy.rc"), rd(v + "-busy.json")[:160]))
ctl_ok = res["control"][0] != "0" and res["control"][1] == "yes" and res["control"][2]
if not ctl_ok or not res["sub"][2]:
    print("INCONCLUSIVE")
else:
    print("EXPOSED" if res["sub"][0] == "0" and res["sub"][1] == "no" else "NOT_EXPOSED")
PY
  verdict T4 "$(tail -1 "$P/summary.txt")" "$(head -2 "$P/summary.txt" | tr '\n' ' ')"
}

# ---------------------------------------------------------------------------------------------------------
# T5 (M1): model chính gọi tool `agent`. Bất biến: system prompt của agent con (không có tool MCP) không chứa
# <mcp-instructions>. Đối chứng dương: agent chính có nó (MCP đã kết nối). Đối chứng bỏ điều kiện (T5ctl): không cấu
# hình MCP thì agent con không có thẻ đó.
t5_one() {
  local name=$1 with_mcp=$2
  begin "$name" '[{"match":"T5-sub","steps":[{"text":"sub done"}]},{"match":"T5-main","steps":[{"tool":"agent","args":{"prompt":"T5-sub list the files"}},{"text":"main done"}]}]' qwen \
    "$([ "$with_mcp" = 1 ] && mcp_cfg "$name")"
  write_mcp_stub
  mkdir -p "$P/proj"; git_repo "$P/proj"
  local_ac "$P/proj" run --json "T5-main delegate to a sub agent" > "$P/out.json" 2> "$P/err.txt"; echo $? > "$P/rc"
  finish
  python3 -I - "$P" > "$P/summary.txt" <<'PY'
import json, os, sys
P = sys.argv[1]
f = os.path.join(P, "log", "requests.jsonl")
reqs = [json.loads(l) for l in open(f)] if os.path.exists(f) else []
def sysof(r):
    p = os.path.join(P, "log", "sys-%s.txt" % r["system_sha"])
    return open(p).read() if os.path.exists(p) else ""
main = [r for r in reqs if r["role"] == "main"]
sub = [r for r in reqs if r["role"] == "subagent"]
print("main=%d (mcp-instructions in %d, sys %s) subagent=%d (mcp-instructions in %d, sys %s, bytes %s, mcp tools %s)" % (
    len(main), sum("<mcp-instructions>" in sysof(r) for r in main), sorted({"sys-%s.txt" % r["system_sha"] for r in main}),
    len(sub), sum("<mcp-instructions>" in sysof(r) for r in sub), sorted({"sys-%s.txt" % r["system_sha"] for r in sub}),
    sorted({r["system_bytes"] for r in sub}), sorted({t for r in sub for t in r["tools"] if t.startswith("mcp_")})))
PY
}
t5() {
  local a b
  t5_one T5 1; a=$(cat "$OUT/T5/summary.txt")
  t5_one T5ctl 0; b=$(cat "$OUT/T5ctl/summary.txt")
  if ! grep -q "main=[1-9].*(mcp-instructions in [1-9]" <<< "$a" || ! grep -q "subagent=[1-9]" <<< "$a"; then
    verdict T5 INCONCLUSIVE "with MCP: $a | control: $b"
  elif grep -q "subagent=[1-9]* (mcp-instructions in [1-9].*mcp tools \[\])" <<< "$a" && grep -q "subagent=[1-9]* (mcp-instructions in 0" <<< "$b"; then
    verdict T5 EXPOSED "with MCP: $a | control without MCP: $b"
  else
    verdict T5 NOT_EXPOSED "with MCP: $a | control without MCP: $b"
  fi
}

# ---------------------------------------------------------------------------------------------------------
# T6 (M2/M3/M4): request của agentic_fetch, agent con, plan phi tương tác (plan bật qua API như TUI) và agent chính.
# Bất biến: context gửi model (system prompt và todo reminder trong messages) không chỉ dẫn dùng tool vắng mặt trong
# danh sách tool của CHÍNH request đó. Phán quyết gồm: todo reminder gửi agentic_fetch, `find_references` ở agent
# chính, `question` ở plan phi tương tác (tổ hợp hiếm: chỉ khi TUI để workspace ở plan mà `run`/`send` gửi vào).
t6() {
  begin T6 '[{"match":"T6-fetchsub","steps":[{"text":"fetch summary"}]},{"match":"T6-fetch","steps":[{"tool":"agentic_fetch","args":{"prompt":"T6-fetchsub what is agent-cli"}},{"text":"done"}]},{"match":"T6-sub","steps":[{"text":"sub done"}]},{"match":"T6-agent","steps":[{"tool":"agent","args":{"prompt":"T6-sub list the files"}},{"text":"done"}]},{"match":"T6-plan","steps":[{"text":"plan done"}]}]' qwen
  server_mode
  mkdir -p "$P/proj"; git_repo "$P/proj"
  ac "$P/proj" run --json "T6-fetch research it" > "$P/fetch.json" 2> "$P/fetch.err"
  ac "$P/proj" run --json "T6-agent delegate it" > "$P/agent.json" 2> "$P/agent.err"
  # Bật plan như TUI (POST agent/main) trên workspace rảnh: tạo workspace qua API (create grace 30 s), rồi `run` gắn vào.
  local wsid
  curl -s --unix-socket "$SOCK" -H 'Content-Type: application/json' \
    -d "{\"path\":\"$P/proj\",\"client_id\":\"$(python3 -I -c 'import uuid; print(uuid.uuid4())')\"}" http://x/v1/workspaces > "$P/ws-create.json"
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
    # Tên có "_" là từ riêng; tên là từ thường chỉ tính khi nằm trong `..`/".." , ngay trước chữ tool, hoặc trong
    # dòng "Your available tools are:".
    pats = [r"`%s`" % re.escape(name), r'"%s"' % re.escape(name), r"\b%s\b tool" % re.escape(name),
            r"available tools are:[^\n]*\b%s\b" % re.escape(name)]
    if "_" in name:
        pats.append(r"(?<![\w-])%s(?![\w-])" % re.escape(name))
    for p in pats:
        m = re.search(p, text)
        if m:
            line = text.count("\n", 0, m.start()) + 1
            return "sys line %d: …%s…" % (line, text[max(0, m.end() - 70):m.end() + 20].replace("\n", " "))
    return None
reminder = "<system_reminder>This is a reminder that your todo list"
findings, rows = [], []
for role in ("agentic_fetch", "subagent", "plan", "main"):
    rs = [r for r in reqs if r["role"] == role]
    if not rs:
        rows.append("%s: no request" % role)
        continue
    r = rs[0]
    s, tools = sysof(r), set(r["tools"])
    absent = {n: c for n in sorted(known - tools) for c in [mentions(s, n)] if c}
    todo = any(u.startswith(reminder) for u in r["user_texts"]) and "todos" not in tools
    rows.append("%s (sys-%s.txt): tools=%s | absent tools named in system prompt=%s | todo-reminder-without-todos=%s" % (
        role, r["system_sha"], sorted(tools), sorted(absent), todo))
    for n, c in sorted(absent.items()):
        rows.append("    %s ← %s" % (n, c))
    if role == "agentic_fetch" and todo:
        findings.append("agentic_fetch gets the todo reminder without `todos`")
    if role == "main" and "find_references" in absent:
        findings.append("main names `find_references` (no such tool)")
    if role == "plan" and "question" in absent:
        findings.append("non-interactive plan names `question` (rare combination)")
    if role in ("agentic_fetch", "subagent") and absent:
        findings.append("%s system prompt names %s" % (role, sorted(absent)))
print("\n".join(rows))
print("FINDINGS: " + ("; ".join(findings) or "none"))
have = all(any(r["role"] == role for r in reqs) for role in ("agentic_fetch", "subagent", "plan", "main"))
print("EXPOSED" if findings else ("NOT_EXPOSED" if have else "INCONCLUSIVE"))
PY
  verdict T6 "$(tail -1 "$P/summary.txt")" "$(grep '^FINDINGS' "$P/summary.txt"); green row: $(grep '^subagent' "$P/summary.txt" | sed 's/tools=\[[^]]*\] | //'); set-plan HTTP $(cat "$P/setplan.code")"
}

# TC (đối chứng P1b, chỉ ở bản sao scratch): MCP trả chữ NHỎ, thành công. Bất biến: tool_result giữ nguyên "echo",
# is_error=false, không có tệp spill nào.
tc() {
  begin TC '[{"match":"TC-mcp","steps":[{"tool":"mcp_v3stub_stub_echo","args":{}},{"text":"done"}]}]' qwen "$(mcp_cfg TC)"
  write_mcp_stub
  mkdir -p "$P/proj"; git_repo "$P/proj"
  local m; m=$(local_ac "$P/proj" run --json "TC-mcp small ok" 2>>"$P/err" | sid_of)
  local_ac "$P/proj" session show "$m" --json > "$P/mcp-show.json" 2>/dev/null
  local spills; spills=$(ls "$P"/proj/.crush/shell-output 2>/dev/null | wc -l)
  finish
  python3 -I - "$P" "$spills" > "$P/summary.txt" <<'PY2'
import json, os, sys
P, spills = sys.argv[1], int(sys.argv[2])
try:
    d = json.load(open(os.path.join(P, "mcp-show.json")))
    r = [(p.get("name"), bool(p.get("is_error")), p.get("content")) for m in d["messages"] for p in m["parts"] if p.get("type") == "tool_result"]
except Exception:
    r = []
print("MCP tool_result (name, is_error, content)=%s | spill files=%d" % (r, spills))
mcp = [x for x in r if x[0] and x[0].startswith("mcp_")]
print("INCONCLUSIVE" if not mcp else ("OK" if mcp[0][1] is False and mcp[0][2] == "echo" and spills == 0 else "BROKEN"))
PY2
  verdict TC "$(tail -1 "$P/summary.txt")" "$(head -1 "$P/summary.txt")"
}

# ---------------------------------------------------------------------------------------------------------
# T9 (A): MCP trả 5 MiB văn bản (local mode). Bất biến: kết quả tool MCP có trần như bash (request kế tiếp không
# mang nguyên 5 MiB), hoặc được spill ra tệp. Đối chứng: bash in 5 MiB thì bị cắt + spill (cùng thước đo).
t9() {
  begin T9 '[{"match":"T9-mcp","steps":[{"tool":"mcp_v3stub_stub_big","args":{}},{"text":"done"}]},{"match":"T9-bash","steps":[{"tool":"bash","args":{"command":"head -c 5242880 /dev/zero | tr \"\\\\0\" x","description":"big"}},{"text":"done"}]}]' qwen "$(mcp_cfg T9)"
  write_mcp_stub
  mkdir -p "$P/proj"; git_repo "$P/proj"
  local_ac "$P/proj" run --json "T9-warm" > /dev/null 2>&1
  local s0 s1 s2
  s0=$(du -cb "$P"/proj/.crush/crush.db* 2>/dev/null | tail -1 | cut -f1)
  local_ac "$P/proj" run --json "T9-bash big output" > "$P/bash.json" 2> "$P/bash.err"
  s1=$(du -cb "$P"/proj/.crush/crush.db* 2>/dev/null | tail -1 | cut -f1)
  TO=120 local_ac "$P/proj" run --json "T9-mcp big result" > "$P/mcp.json" 2> "$P/mcp.err"
  s2=$(du -cb "$P"/proj/.crush/crush.db* 2>/dev/null | tail -1 | cut -f1)
  local spills; spills=$(ls "$P"/proj/.crush/shell-output 2>/dev/null | wc -l)
  local mid; mid=$(sid_of < "$P/mcp.json")
  local_ac "$P/proj" session show "$mid" --json > "$P/mcp-show.json" 2>/dev/null
  finish
  python3 -I - "$P" "$s0" "$s1" "$s2" "$spills" > "$P/summary.txt" <<'PY'
import json, os, sys
P, s0, s1, s2, spills = sys.argv[1:6]
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))]
def growth(tag):
    rs = [r for r in reqs if r["role"] == "main" and r["last_user"].lstrip().startswith(tag)]
    a = [r["request_bytes"] for r in rs if r["tool_steps"] == 0]
    b = [r["request_bytes"] for r in rs if r["tool_steps"] == 1]
    return (b[0] - a[0]) if a and b else None
try:
    show = json.load(open(os.path.join(P, "mcp-show.json")))
    tr = [p for m in show["messages"] for p in m["parts"] if p.get("type") == "tool_result"]
    clen = [len(p.get("content", "")) for p in tr]
except Exception:
    clen = []
gb, gm = growth("T9-bash"), growth("T9-mcp")
db_b, db_m = int(s1 or 0) - int(s0 or 0), int(s2 or 0) - int(s1 or 0)
print("bash 5 MiB (control): next request +%s bytes, crush.db* +%d bytes | MCP 5 MiB: next request +%s bytes, crush.db* +%d bytes, stored tool_result chars=%s | spill files=%s" % (
    gb, db_b, gm, db_m, clen, spills))
if gb is None or gm is None:
    print("INCONCLUSIVE")
elif gm >= 1 << 20 and gb < 1 << 20:
    print("EXPOSED")
else:
    print("NOT_EXPOSED")
PY
  verdict T9 "$(tail -1 "$P/summary.txt")" "$(head -1 "$P/summary.txt")"
}

# ---------------------------------------------------------------------------------------------------------
# T11 (F1): `send` không --wait (LLM giả chậm 20 s), SIGKILL server ngay khi send thoát. Lặp N lần.
# Bất biến: send rc 0 ⇒ tin user đã có trong DB. Lượt cuối là ĐỐI CHỨNG: kill sau 2 s, tin phải có.
t11() {
  begin T11 '[{"match":"T11-trial","steps":[{"sleep":20,"text":"late"}]}]' qwen
  server_mode
  mkdir -p "$P/proj"; git_repo "$P/proj"
  local n=${T11_TRIALS:-8} i pid rc sid lost=0 ok=0 bad=0 control=no delay indb
  for i in $(seq $((n + 1))); do
    delay=0; [ "$i" -gt "$n" ] && delay=2
    ac "$P/proj" run --json "warm $i" > /dev/null 2>&1
    pid=$(server_pid_of "$SOCK")
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
  local why="trials=$n killed at once: rc0+in_db=$ok, rc0+missing=$lost, other=$bad; control (kill after 2 s) in_db=$control (T11/trials.txt)"
  echo "$why" > "$P/summary.txt"
  if [ "$control" != yes ]; then verdict T11 INCONCLUSIVE "$why — control did not find the message"
  elif [ "$lost" -gt 0 ]; then verdict T11 EXPOSED "$why"
  elif [ "$ok" -gt 0 ]; then verdict T11 NOT_EXPOSED "$why"
  else verdict T11 INCONCLUSIVE "$why"; fi
}

# ---------------------------------------------------------------------------------------------------------
# T13 (F2): gửi A, gửi B vào cùng session, SIGTERM server. Bất biến: DB có B, hoặc có bản ghi huỷ cho B.
# Ca chính: A ngủ 10 s nên B xếp hàng. Đối chứng (T13ctl): A trả ngay nên B không phải xếp hàng.
t13_one() {
  local name=$1 asleep=$2
  begin "$name" "[{\"match\":\"T13-A\",\"steps\":[{\"sleep\":$asleep,\"text\":\"A done\"}]},{\"match\":\"T13-B\",\"steps\":[{\"text\":\"B done\"}]}]" qwen
  server_mode
  mkdir -p "$P/proj"; git_repo "$P/proj"
  ac "$P/proj" run --json "warm" > /dev/null 2>&1
  local pid sid rcB
  pid=$(server_pid_of "$SOCK")
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
try:
    msgs = json.loads(show)["messages"]
    parts = ["%s:%s" % (m["role"], ",".join((p.get("type") + "=" + (p.get("reason") or p.get("text") or "")[:20]) for p in m["parts"])) for m in msgs]
except Exception:
    parts = []
print("send B rc=%s | messages=%s | 'T13-B' bytes in crush.db*=%s | B reached LLM=%s" % (rcB, parts, b"T13-B" in raw, b_sent))
print("NOT_EXPOSED" if ("T13-B" in show or b"T13-B" in raw) else ("EXPOSED" if sid and rcB == "0" and "T13-A" in show else "INCONCLUSIVE"))
PY
}
t13() {
  t13_one T13 10; local a v; a=$(head -1 "$OUT/T13/summary.txt"); v=$(tail -1 "$OUT/T13/summary.txt")
  t13_one T13ctl 0; local b w; b=$(head -1 "$OUT/T13ctl/summary.txt"); w=$(tail -1 "$OUT/T13ctl/summary.txt")
  [ "$v" = EXPOSED ] && [ "$w" != NOT_EXPOSED ] && v=INCONCLUSIVE
  verdict T13 "$v" "queued B: $a | control (A instant, B not queued) -> $w: $b"
}

# ---------------------------------------------------------------------------------------------------------
# T17 (H): X tạo bằng provider pp; dự án đổi sang qq (crush.json của repo); Y chạy bằng qq; tiếp X qua server.
# Bất biến: <data>/crush.json không bị ghi model; lượt sau của Y (send, như TUI) và session mới vẫn đi qq.
# Ca chính tiếp X bằng `run -s X`; đối chứng (T17ctl) tiếp X bằng `send -s X --wait` (không khôi phục model).
t17_one() {
  local name=$1 how=$2
  begin "$name" '[]' pp
  server_mode
  mkdir -p "$P/proj"; git_repo "$P/proj"
  local x y before after
  x=$(ac "$P/proj" run --json "T17-X1 on pp" 2>>"$P/err" | sid_of)
  stop_servers
  printf '%s' '{"models":{"large":{"provider":"qq","model":"m-qq"},"small":{"provider":"qq","model":"m-qq"}}}' > "$P/proj/crush.json"
  y=$(ac "$P/proj" run --json "T17-Y1 on project default" 2>>"$P/err" | sid_of)
  cat "$P/proj/.crush/crush.json" > "$P/data-crush.before.json" 2>/dev/null || echo "<absent>" > "$P/data-crush.before.json"
  if [ "$how" = run ]; then ac "$P/proj" run --json -s "$x" "T17-X2 continue X" > "$P/x2.json" 2>>"$P/err"
  else ac "$P/proj" send -s "$x" --wait --json "T17-X2 continue X" > "$P/x2.json" 2>>"$P/err"; fi
  cat "$P/proj/.crush/crush.json" > "$P/data-crush.after.json" 2>/dev/null || echo "<absent>" > "$P/data-crush.after.json"
  ac "$P/proj" send -s "$y" --wait --json "T17-Y2 next turn of Y" > "$P/y2.json" 2>>"$P/err"
  ac "$P/proj" run --json "T17-N1 brand new session" > "$P/n1.json" 2>>"$P/err"
  finish
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
print("X1->%s Y1->%s X2->%s Y2->%s N1->%s | <data>/crush.json before: %s | after: models=%s" % (
    prov("T17-X1"), prov("T17-Y1"), prov("T17-X2"), prov("T17-Y2"), prov("T17-N1"), before[:60], json.dumps(models)))
if not (x and y) or prov("T17-X1") != ["pp"] or prov("T17-Y1") != ["qq"]:
    print("INCONCLUSIVE")
elif before == after and prov("T17-Y2") == ["qq"] and prov("T17-N1") == ["qq"]:
    print("NOT_EXPOSED")
else:
    print("EXPOSED")
PY
}
t17() {
  t17_one T17 run; local a v; a=$(head -1 "$OUT/T17/summary.txt"); v=$(tail -1 "$OUT/T17/summary.txt")
  t17_one T17ctl send; local b w; b=$(head -1 "$OUT/T17ctl/summary.txt"); w=$(tail -1 "$OUT/T17ctl/summary.txt")
  [ "$v" = EXPOSED ] && [ "$w" != NOT_EXPOSED ] && v=INCONCLUSIVE
  verdict T17 "$v" "run -s X: $a | control send -s X -> $w: $b"
}

# ---------------------------------------------------------------------------------------------------------
# T19 (F8): tool MCP trả isError=true. Bất biến: tool_result lưu trong DB có is_error=true như tool nội bộ lỗi.
# Đối chứng: `view` tệp không tồn tại (tool nội bộ lỗi) có is_error=true (chứng minh trường được ghi và đọc được).
t19() {
  begin T19 '[{"match":"T19-mcp","steps":[{"tool":"mcp_v3stub_stub_fail","args":{}},{"text":"done"}]},{"match":"T19-view","steps":[{"tool":"view","args":{"file_path":"no-such-file.txt"}},{"text":"done"}]}]' qwen "$(mcp_cfg T19)"
  write_mcp_stub
  mkdir -p "$P/proj"; git_repo "$P/proj"
  local m v
  m=$(local_ac "$P/proj" run --json "T19-mcp call the failing tool" 2>>"$P/err" | sid_of)
  v=$(local_ac "$P/proj" run --json "T19-view read a missing file" 2>>"$P/err" | sid_of)
  local_ac "$P/proj" session show "$m" --json > "$P/mcp-show.json" 2>/dev/null
  local_ac "$P/proj" session show "$v" --json > "$P/view-show.json" 2>/dev/null
  finish
  python3 -I - "$P" > "$P/summary.txt" <<'PY'
import json, os, sys
P = sys.argv[1]
def results(f):
    try:
        d = json.load(open(os.path.join(P, f)))
    except Exception:
        return []
    return [(p.get("name"), bool(p.get("is_error")), (p.get("content") or "")[:60]) for m in d["messages"] for p in m["parts"] if p.get("type") == "tool_result"]
mr, vr = results("mcp-show.json"), results("view-show.json")
print("MCP tool_result (name, is_error, content)=%s | control view tool_result=%s" % (mr, vr))
mcp = [r for r in mr if r[0] and r[0].startswith("mcp_")]
ctl = [r for r in vr if r[0] == "view"]
if not mcp or not ctl or not ctl[0][1]:
    print("INCONCLUSIVE")
else:
    print("NOT_EXPOSED" if mcp[0][1] else "EXPOSED")
PY
  verdict T19 "$(tail -1 "$P/summary.txt")" "$(head -1 "$P/summary.txt")"
}

# ---------------------------------------------------------------------------------------------------------
# T20 (S1, S4, S7): 10 client `run` cùng lúc. Pha lạnh: chưa có server, lặp T20_TRIALS lần (mặc định 10). Pha ấm:
# server do script dựng sẵn (giữ stderr để thấy panic), lặp T20_WARM lần (mặc định 10). Mỗi lần socket/server mới.
# Bất biến: đúng một server khởi động mỗi lần, 10/10 rc 0, server không panic. Đối chứng: 10 client chạy LẦN LƯỢT.
t20() {
  begin T20 '[]' qwen
  mkdir -p "$P/proj"; git_repo "$P/proj"
  local k i pids
  for k in $(seq "${T20_TRIALS:-10}"); do
    server_mode "c$k"; pids=()
    for i in $(seq 10); do
      ( ac "$P/proj" run --json "T20 trial $k client $i" > "$P/c$k-out-$i.json" 2> "$P/c$k-err-$i.txt"; echo $? > "$P/c$k-rc-$i" ) &
      pids+=($!)
    done
    wait "${pids[@]}"
    stop_servers
  done
  # Server ấm do chính script dựng (stderr riêng, không bị lần dựng lại ghi đè), 10 `run` cùng lúc, lặp T20_WARM lần.
  for k in $(seq "${T20_WARM:-10}"); do
    server_mode "w$k"; pids=()
    TO=120 iso "$P/proj" "$BIN" server --host "unix://$SOCK" > "$P/w$k-server.out" 2> "$P/w$k-server.err" &
    for _ in $(seq 50); do [ -S "$SOCK" ] && break; sleep 0.1; done
    for i in $(seq "${T20_N:-10}"); do
      ( ac "$P/proj" run --json "T20 warm $k client $i" > "$P/w$k-out-$i.json" 2> "$P/w$k-err-$i.txt"; echo $? > "$P/w$k-rc-$i" ) &
      pids+=($!)
    done
    wait "${pids[@]}"
    stop_servers
  done
  server_mode seq
  for i in $(seq 10); do
    ac "$P/proj" run --json "T20 sequential client $i" > "$P/seq-out-$i.json" 2> "$P/seq-err-$i.txt"; echo $? > "$P/seq-rc-$i"
  done
  finish
  T20_N=${T20_N:-10} python3 -I - "$P" > "$P/summary.txt" <<'PY'
import glob, os, sys
P = sys.argv[1]
def trial(t):
    rcs = [open(os.path.join(P, "%s-rc-%d" % (t, i))).read().strip() for i in range(1, 11)]
    d = glob.glob(os.path.join(P, "home", ".cache", "agent-cli", "server-*%s.sock" % t))
    log = open(os.path.join(d[0], "crush.log")).read() if d else ""
    err = open(os.path.join(d[0], "stderr.log")).read() if d else ""
    panic = next((l for l in err.splitlines() if l.startswith("panic:")), "")
    where = next((l.strip() for l in err.splitlines() if "coordinator.go" in l), "")
    return rcs.count("0"), log.count("Starting agent-cli server"), panic, where
rows, exposed = [], False
bad = []
trials = sorted({f.split("-rc-")[0].rsplit("/", 1)[-1] for f in glob.glob(os.path.join(P, "c*-rc-1"))}, key=lambda t: int(t[1:]))
for t in trials:
    ok, starts, panic, where = trial(t)
    if ok != 10 or starts != 1 or panic:
        bad.append("%s: rc0=%d/10 server starts=%d panic=%r at %s" % (t, ok, starts, panic, where))
exposed = bool(bad)
rows.append("concurrent trials=%d, failed=%d" % (len(trials), len(bad)))
rows.extend(bad)
warm = sorted({f.split("-rc-")[0].rsplit("/", 1)[-1] for f in glob.glob(os.path.join(P, "w*-rc-1"))}, key=lambda t: int(t[1:]))
wbad = []
for t in warm:
    rcs = [open(f).read().strip() for f in glob.glob(os.path.join(P, t + "-rc-*"))]
    err = open(os.path.join(P, t + "-server.err")).read()
    panic = next((l for l in err.splitlines() if l.startswith("panic:")), "")
    where = next((l.strip() for l in err.splitlines() if "coordinator.go" in l), "")
    if panic or rcs.count("0") != len(rcs):
        wbad.append("%s: rc0=%d/%d panic=%r at %s" % (t, rcs.count("0"), len(rcs), panic, where))
exposed |= bool(wbad)
rows.append("warm server (own stderr), %s concurrent run per trial: trials=%d, failed=%d" % (os.environ.get("T20_N", "10"), len(warm), len(wbad)))
rows.extend(wbad)
ok, starts, panic, _ = trial("seq")
rows.append("control sequential: rc0=%d/10 server starts=%d panic=%r" % (ok, starts, panic))
print(" | ".join(rows))
ctl = ok == 10 and not panic
print("INCONCLUSIVE" if not ctl else ("EXPOSED" if exposed else "NOT_EXPOSED"))
PY
  verdict T20 "$(tail -1 "$P/summary.txt")" "$(head -1 "$P/summary.txt")"
}

# ---------------------------------------------------------------------------------------------------------
# T21 (§33, MEASURED): model chạy `env` (+ output dài để có spill) và `view .env`; loop (local mode) có verify in
# env. Đếm số tệp (ngoài .env gốc và log của LLM giả) còn giữ secret giả. Ba lượt: server mặc định, server với `-d`
# của client, local với `-d`. T21c đo quyền crush.db khi .crush có sẵn với 0755.
canary_scan() { python3 -I - "$1" <<'PY'
import os, sys
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
        b = open(p, "rb").read()
        nf, ne = b.count(b"SECRET_V3_CANARY_FILE_7f3a"), b.count(b"SECRET_V3_CANARY_ENV_9c1d")
        if nf or ne:
            hits.append((os.path.relpath(p, P), nf, ne, oct(os.stat(p).st_mode & 0o777)))
def kind(p):
    for k in ("crush.db", "shell-output", "loops", "crush.log", "stdout.log", "stderr.log"):
        if k in p:
            return k
    return "other"
for h in sorted(hits):
    print("  %-62s file_canary=%d env_canary=%d mode=%s [%s]" % (h[0], h[1], h[2], h[3], kind(h[0])))
k = [kind(h[0]) for h in hits]
print("files_holding_canary=%d (crush.db*=%d shell-output=%d logs=%d journal=%d other=%d)" % (
    len(hits), k.count("crush.db"), k.count("shell-output"), sum(k.count(x) for x in ("crush.log", "stdout.log", "stderr.log")),
    k.count("loops"), k.count("other")))
PY
}
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
  canary_scan "$P" > "$P/summary.txt"
  echo "$(tail -1 "$P/summary.txt") (run rc=$(cat "$P/rc"))"
}
t21c() {
  begin T21c '[]' qwen
  server_mode
  mkdir -p "$P/proj"; git_repo "$P/proj"; mkdir -m 0755 "$P/proj/.crush"
  ac "$P/proj" run --json "T21c perms" > "$P/out.json" 2> "$P/err.txt"
  { echo "umask(test shell)=$(umask)"; stat -c '%n %a' "$P"/proj/.crush "$P"/proj/.crush/crush.db* "$P"/proj/.crush/logs 2>/dev/null | sed "s|$P/||"; } > "$P/summary.txt"
  finish
  tr '\n' ' ' < "$P/summary.txt"
}
t21() {
  local a b c d
  a=$(t21_one T21 "" server)
  b=$(t21_one T21d -d server)
  c=$(t21_one T21l -d local)
  d=$(t21c)
  verdict T21 MEASURED "server: $a | server -d: $b | local -d: $c | T21c pre-existing .crush 0755: $d (per-file lists: T21*/summary.txt; loop runs in local mode)"
  t21b
}

# T21b: `.env` của thư mục đứng khi chạy được nạp vào env tiến trình (main.go:18-22). Bất biến: secret trong `.env`
# của repo không có trong output của lệnh `env` do model chạy. Đối chứng: đứng ngoài repo rồi `-c proj` (main.go chỉ
# nạp `.env` của cwd lúc khởi động) thì không có.
t21b() {
  begin T21b '[{"match":"T21b-go","steps":[{"tool":"bash","args":{"command":"env","description":"print env"}},{"text":"done"}]}]' qwen
  mkdir -p "$P/proj"; git_repo "$P/proj"
  printf 'API_TOKEN=SECRET_V3_CANARY_FILE_7f3a\n' > "$P/proj/.env"
  local s1 s2
  server_mode main
  s1=$(ac "$P/proj" run --json "T21b-go from inside the repo" 2>>"$P/err" | sid_of)
  server_mode ctl
  s2=$(ac "$P" -c "$P/proj" run --json "T21b-go from outside with -c" 2>>"$P/err" | sid_of)
  local_ac "$P/proj" session show "$s1" --json > "$P/main-show.json" 2>/dev/null
  local_ac "$P/proj" session show "$s2" --json > "$P/ctl-show.json" 2>/dev/null
  finish
  python3 -I - "$P" > "$P/summary.txt" <<'PY'
import json, os, sys
P = sys.argv[1]
def env_out(f):
    try:
        d = json.load(open(os.path.join(P, f)))
    except Exception:
        return None
    outs = [p.get("content") or "" for m in d["messages"] for p in m["parts"] if p.get("type") == "tool_result" and p.get("name") == "bash"]
    return outs[0] if outs else None
a, b = env_out("main-show.json"), env_out("ctl-show.json")
fa = a is not None and "SECRET_V3_CANARY_FILE_7f3a" in a
fb = b is not None and "SECRET_V3_CANARY_FILE_7f3a" in b
print("`env` run by the model, started inside repo: .env canary present=%s | control (started outside, -c repo): present=%s | documented: README.md:79-80" % (fa, fb))
print("INCONCLUSIVE" if a is None or b is None else ("EXPOSED" if fa and not fb else ("NOT_EXPOSED" if not fa else "INCONCLUSIVE")))
PY
  verdict T21b "$(tail -1 "$P/summary.txt")" "$(head -1 "$P/summary.txt") (by design per README; the -c path differs from cd)"
}

# ---------------------------------------------------------------------------------------------------------
# T23 (F13): `session delete` (CLI, launcher mặc định CRUSH_CLIENT_SERVER=1) khi server đang chạy chính session đó.
# Bất biến: từ chối khi session bận, hoặc sau khi lượt kết thúc DB không có tin mồ côi của session đã xoá.
# Đối chứng: xoá một session rảnh thì không còn tin mồ côi.
t23() {
  begin T23 '[{"match":"T23-long","steps":[{"sleep":6,"text":"long done"}]}]' qwen
  server_mode
  mkdir -p "$P/proj"; git_repo "$P/proj"
  ac "$P/proj" run --json "warm" > /dev/null 2>&1
  local sid ctl
  sid=$(ac "$P/proj" send --new --title t23 "T23-long busy" 2>"$P/send.err" | tail -1)
  sleep 1.5
  ac "$P/proj" session delete "$sid" --json > "$P/del.json" 2> "$P/del.err"; echo $? > "$P/del.rc"
  sleep 7
  ctl=$(ac "$P/proj" run --json "T23-ctl idle" 2>/dev/null | sid_of)
  ac "$P/proj" session delete "$ctl" --json > "$P/delctl.json" 2> "$P/delctl.err"; echo $? > "$P/delctl.rc"
  finish
  python3 -I - "$P" "$sid" "$ctl" > "$P/summary.txt" <<'PY'
import os, sqlite3, sys
P, sid, ctl = sys.argv[1:4]
c = sqlite3.connect("file:%s?mode=ro" % os.path.join(P, "proj", ".crush", "crush.db"), uri=True)
def state(s):
    row = c.execute("select count(*) from sessions where id=?", (s,)).fetchone()[0]
    msgs = c.execute("select role from messages where session_id=?", (s,)).fetchall()
    return row, [m[0] for m in msgs]
orph = c.execute("select count(*) from messages where session_id not in (select id from sessions)").fetchone()[0]
rd = lambda f: open(os.path.join(P, f)).read().strip()
(r1, m1), (r2, m2) = state(sid), state(ctl)
print("busy delete rc=%s (%s) -> session row=%d, messages left=%s | control idle delete rc=%s -> row=%d, messages left=%s | orphan messages in DB=%d" % (
    rd("del.rc"), (rd("del.err") or rd("del.json"))[:80], r1, m1, rd("delctl.rc"), r2, m2, orph))
if rd("del.rc") != "0":
    print("NOT_EXPOSED")
elif rd("delctl.rc") != "0" or m2:
    print("INCONCLUSIVE")
else:
    print("EXPOSED" if (m1 and not r1) else "NOT_EXPOSED")
PY
  verdict T23 "$(tail -1 "$P/summary.txt")" "$(head -1 "$P/summary.txt")"
}

# ---------------------------------------------------------------------------------------------------------
# C5/S10: hai client cùng workspace (cùng server), client đầu đang giữ một lượt chạy. Bất biến: lượt của client sau
# chạy theo env và allowlist provider của CHÍNH nó (hoặc server từ chối rõ ràng). Đối chứng: client sau tự dựng server.
#   env:   base_url của provider dùng $V3_ROUTE; client đầu c1, client sau c2 (LLM giả ghi tên provider = route).
#   allow: client đầu không allowlist; client sau AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen, dự án chọn pp/m-pp.
c5() {
  begin C5 '[{"match":"C5-hold","steps":[{"sleep":8,"text":"held"}]}]' qwen
  mkdir -p "$P/envp" "$P/allowp"; git_repo "$P/envp"; git_repo "$P/allowp"
  printf '%s' "{\"providers\":{\"envp\":{\"type\":\"openai-compat\",\"base_url\":\"http://127.0.0.1:$PORT/\$V3_ROUTE/v1\",\"api_key\":\"x\",\"models\":[{\"id\":\"m-envp\",\"name\":\"m-envp\",\"context_window\":100000,\"default_max_tokens\":1000}]}},\"models\":{\"large\":{\"provider\":\"envp\",\"model\":\"m-envp\"},\"small\":{\"provider\":\"envp\",\"model\":\"m-envp\"}}}" > "$P/envp/crush.json"
  printf '%s' '{"models":{"large":{"provider":"pp","model":"m-pp"},"small":{"provider":"pp","model":"m-pp"}}}' > "$P/allowp/crush.json"
  local hold
  # env, hai client
  server_mode env2; EXTRA=(V3_ROUTE=c1)
  ac "$P/envp" run --json "C5-hold one" > "$P/env-hold.json" 2>&1 & hold=$!
  sleep 2.5; EXTRA=(V3_ROUTE=c2)
  ac "$P/envp" run --json "C5-env second client" > "$P/env-two.json" 2> "$P/env-two.err"
  wait $hold; stop_servers
  # env, đối chứng: client sau một mình
  server_mode envctl; EXTRA=(V3_ROUTE=c2)
  ac "$P/envp" run --json "C5-env control alone" > "$P/env-ctl.json" 2> "$P/env-ctl.err"; stop_servers
  # allowlist, hai client
  server_mode allow2; EXTRA=()
  ac "$P/allowp" run --json "C5-hold allow one" > "$P/allow-hold.json" 2>&1 & hold=$!
  sleep 2.5; EXTRA=(AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen)
  ac "$P/allowp" run --json "C5-allow second client" > "$P/allow-two.json" 2> "$P/allow-two.err"
  wait $hold; stop_servers
  # allowlist, đối chứng
  server_mode allowctl; EXTRA=(AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen)
  ac "$P/allowp" run --json "C5-allow control alone" > "$P/allow-ctl.json" 2> "$P/allow-ctl.err"; stop_servers
  EXTRA=()
  finish
  python3 -I - "$P" > "$P/summary.txt" <<'PY'
import json, os, sys
P = sys.argv[1]
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))]
def prov(tag):
    return sorted({r["provider"] for r in reqs if r["role"] == "main" and r["last_user"].lstrip().startswith(tag)}) or ["none"]
def err(f):
    try:
        return (json.load(open(os.path.join(P, f))).get("error") or "")[:90]
    except Exception:
        return "no JSON"
e2, ec, a2, actl = prov("C5-env second"), prov("C5-env control"), prov("C5-allow second"), prov("C5-allow control")
print("env: second client (V3_ROUTE=c2) -> %s, control alone -> %s | allowlist qwen-only: second client -> %s err=%r, control alone -> %s err=%r | holders: %s %s" % (
    e2, ec, a2, err("allow-two.json"), actl, err("allow-ctl.json"), prov("C5-hold one"), prov("C5-hold allow")))
env_exp = ec == ["c2"] and e2 != ["c2"]
allow_exp = actl == ["none"] and a2 == ["pp"]
ctl_ok = ec == ["c2"] and actl == ["none"]
print("ENV=%s ALLOW=%s" % ("EXPOSED" if env_exp else "ok", "EXPOSED" if allow_exp else "ok"))
print("INCONCLUSIVE" if not ctl_ok else ("EXPOSED" if env_exp or allow_exp else "NOT_EXPOSED"))
PY
  verdict C5 "$(tail -1 "$P/summary.txt")" "$(sed -n 2p "$P/summary.txt") — $(head -1 "$P/summary.txt")"
}

for c in $CASES; do
  case "$c" in
    T1M) t1m ;; T2) t2 ;; T3) t3 ;; T4) t4 ;; T5) t5 ;; T6) t6 ;; T9) t9 ;; T11) t11 ;; T13) t13 ;; T17) t17 ;;
    T19) t19 ;; TC) tc ;; T20) t20 ;; T21) t21 ;; T23) t23 ;; C5) c5 ;;
    *) verdict "$c" INCONCLUSIVE "unknown case" ;;
  esac
done
echo "EXPOSURE: ran $(wc -l < "$OUT/verdicts.txt") verdict line(s); verdicts in $OUT/verdicts.txt"
exit 0
