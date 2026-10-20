#!/usr/bin/env bash
# Bảng chân trị P2c: data dir + workspace + cảnh báo theo cách mở, local và server.
# Dùng: bash table.sh <thư-mục-mới ≤60 byte> <binary-đo> <binary-gốc (dựng fixture 2 .crush)> <repo agent-cli>
set -u
OUT=$1; BIN=$2; OLD=$3; R=$4
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd -P)
P=$OUT; mkdir -p "$P/home/.config/agent-cli" "$P/log" "$P/rt"; chmod 700 "$P/rt"
echo '[]' > "$P/rules.json"
timeout 900 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" "$P/rules.json" >/dev/null 2>&1 &
FAKE=$!
for _ in $(seq 50); do [ -s "$P/port" ] && break; sleep 0.1; done
PORT=$(cat "$P/port")
cat > "$P/home/.config/agent-cli/crush.json" <<JSON
{"providers":{"qwen":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/qwen/v1","api_key":"x","models":[{"id":"m-qwen","name":"m-qwen","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"qwen","model":"m-qwen"},"small":{"provider":"qwen","model":"m-qwen"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}}
JSON
# run1 <bin> <mode local|server> <sock> <cwd> <nhãn prompt> [cờ...]
run1() {
  local bin=$1 mode=$2 sock=$3 d=$4 tag=$5; shift 5
  local cs=0 host=()
  [ "$mode" = server ] && { cs=1; host=(--host "unix://$sock"); }
  (cd "$d" && env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/.config" XDG_DATA_HOME="$P/home/.local/share" \
    XDG_CACHE_HOME="$P/home/.cache" CRUSH_GLOBAL_CONFIG="$P/home/.config/agent-cli" CRUSH_GLOBAL_DATA="$P/home/.local/share/agent-cli" \
    CRUSH_CACHE_DIR="$P/home/.cache/agent-cli" AGENT_CLI_RUNTIME_DIR="$P/rt" GIT_CEILING_DIRECTORIES="$OUT" CRUSH_DISABLE_METRICS=1 \
    DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 CRUSH_CLIENT_SERVER=$cs CRUSH_SERVER_IDLE_TIMEOUT=60 \
    timeout 60 "$bin" "${host[@]}" "$@" run --json "$tag") > "$P/$tag.json" 2> "$P/$tag.err"
}
stop_servers() { local pid; for pid in $(pgrep -f -- "server --host unix://$P/"); do kill -TERM "$pid"; while kill -0 "$pid" 2>/dev/null; do sleep 0.1; done; done; }
git_repo() { mkdir -p "$1/sub"; (cd "$1" && git init -q -b main && echo x > sub/a.txt && git add -A && git -c user.name=t -c user.email=t@t commit -qm init); }
# row <ca> <mode> <thư-mục-gốc-ca> <tag...>: một dòng bảng cho lượt cuối (tag cuối) của ca.
row() {
  local name=$1 mode=$2 C=$3; shift 3
  python3 -I - "$P" "$name" "$mode" "$C" "$@" >> "$P/table.txt" <<'PY'
import json, os, sqlite3, sys
P, name, mode, C, *tags = sys.argv[1:]
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))]
def wd(tag):
    for r in reqs:
        if r["role"] == "main" and r["last_user"].strip() == tag:
            for line in open(os.path.join(P, "log", "sys-%s.txt" % r["system_sha"])):
                if line.startswith("Working directory:"):
                    return os.path.relpath(line.split(":", 1)[1].strip(), C)
    return "?"
def dbs():
    return sorted(os.path.relpath(os.path.join(d, "crush.db"), C) for d, _, fs in os.walk(C) if "crush.db" in fs)
def db_of(sid):
    hit = []
    for rel in dbs():
        c = sqlite3.connect("file:%s?mode=ro" % os.path.join(C, rel), uri=True)
        if c.execute("select count(*) from sessions where id=?", (sid,)).fetchone()[0]:
            hit.append(rel)
        c.close()
    return ",".join(hit) or "none"
cells = []
for tag in tags:
    try:
        sid = json.load(open(os.path.join(P, tag + ".json"))).get("session_id", "")
    except Exception:
        sid = ""
    err = open(os.path.join(P, tag + ".err")).read()
    warn = "có" if "is not used" in err else "không"
    cells.append("%s: db=%s, workspace=%s, cảnh báo=%s" % (tag.split("-")[-1], db_of(sid) if sid else "NO_SESSION", wd(tag), warn))
print("| %s | %s | %s | %s |" % (name, mode, " ; ".join(cells), ", ".join(dbs())))
PY
}

# Ngữ cảnh của thư mục con: sub/AGENTS.md có dấu SUBCTX-MARK, gốc có ROOTCTX-MARK. Prompt hệ thống có dấu nào?
for mode in local server; do
  m=${mode:0:1}; C=$P/$m-ctx; git_repo "$C/repo"
  echo "SUBCTX-MARK" > "$C/repo/sub/AGENTS.md"; echo "ROOTCTX-MARK" > "$C/repo/AGENTS.md"
  run1 "$BIN" $mode "$P/$m-x.sock" "$C/repo/sub" "$m-ctx-sub"; stop_servers
  python3 -I - "$P" "$m-ctx-sub" "$mode" <<'PY2'
import json, os, sys
P, tag, mode = sys.argv[1:]
for l in open(os.path.join(P, "log", "requests.jsonl")):
    r = json.loads(l)
    if r["role"] == "main" and r["last_user"].strip() == tag:
        t = open(os.path.join(P, "log", "sys-%s.txt" % r["system_sha"])).read()
        print("%s: sub/AGENTS.md trong prompt=%s, AGENTS.md gốc trong prompt=%s" % (mode, "SUBCTX-MARK" in t, "ROOTCTX-MARK" in t))
PY2
done
kill $FAKE 2>/dev/null; wait $FAKE 2>/dev/null
