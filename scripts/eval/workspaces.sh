#!/usr/bin/env bash
# Daily E (nhiều workspace A → B → A → B) và F (nhiều session trong một workspace) với binary thật, LLM giả.
# Kiểm bằng chính request mà model nhận: context file của dự án khác không lọt vào, lịch sử của session
# khác không lọt vào, `session list` mỗi dự án chỉ có session của nó, `-s` nối đúng session.
# Dùng: bash scripts/eval/workspaces.sh <thư-mục-mới> <binary>
set -u
P="$1"; BIN="$2"
R=$(cd "$(dirname "$0")/../.." && pwd)
[ -e "$P" ] && { echo "exists: $P" >&2; exit 2; }
mkdir -p "$P/home" "$P/log" "$P/A" "$P/B"
timeout 120 python3 -I "$R/scripts/eval/fakellm.py" "$P/port" "$P/log" >/dev/null 2>&1 &
for _ in $(seq 50); do [ -s "$P/port" ] && break; sleep 0.1; done
PORT=$(cat "$P/port")
for w in A B; do
  (cd "$P/$w" && git init -q && echo "# project $w" > README.md && echo "Project rule: PROJECT-$w-MARKER" > AGENTS.md)
  cat > "$P/$w/crush.json" <<JSON
{"providers":{"qwen":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/qwen/v1","api_key":"x","models":[{"id":"q","name":"q","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"qwen","model":"q"},"small":{"provider":"qwen","model":"q"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}}
JSON
done
ac() { # ac <dir> <args...>  → stdout JSON của dew
  local d=$1; shift
  (cd "$d" && env -i PATH=/usr/bin:/bin HOME="$P/home" XDG_CONFIG_HOME="$P/home/c" XDG_DATA_HOME="$P/home/d" XDG_CACHE_HOME="$P/home/x" \
    CRUSH_GLOBAL_CONFIG="$P/home/cfg" CRUSH_GLOBAL_DATA="$P/home/data" CRUSH_CACHE_DIR="$P/home/cache" \
    CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 CRUSH_CLIENT_SERVER=0 \
    timeout 60 "$BIN" "$@")
}
sid() { python3 -I -c 'import json,sys; print(json.load(sys.stdin).get("session_id",""))'; }
A1=$(ac "$P/A" run --json "E-A1 first task in A" | sid)
B1=$(ac "$P/B" run --json "E-B1 first task in B" | sid)
A2=$(ac "$P/A" run --json -s "$A1" "E-A2 continue in A" | sid)
B2=$(ac "$P/B" run --json -s "$B1" "E-B2 continue in B" | sid)
# F: hai session xen kẽ trong cùng workspace A
X1=$(ac "$P/A" run --json "F-X1 feature work" | sid)
Y1=$(ac "$P/A" run --json "F-Y1 debug work" | sid)
X2=$(ac "$P/A" run --json -s "$X1" "F-X2 continue feature" | sid)
ac "$P/A" session list --json > "$P/A-sessions.json" 2>/dev/null
ac "$P/B" session list --json > "$P/B-sessions.json" 2>/dev/null
sleep 1
python3 -I - "$P" "$A1" "$B1" "$A2" "$X1" "$Y1" "$X2" "$B2" <<'PY'
import json, os, sys
P, A1, B1, A2, X1, Y1, X2, B2 = sys.argv[1:9]
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))]
mains = [r for r in reqs if r["role"] == "main"]
def sysof(r):
    return open(os.path.join(P, "log", "sys-%s.txt" % r["system_sha"])).read()
def req_for(tag):
    # Prompt có thể mang dòng trống từ template; nhãn fixture không phụ thuộc padding đó.
    return [r for r in mains if r["last_user"].lstrip().startswith(tag)]
checks = []
def check(name, ok):
    checks.append(ok)
    print(("ok   " if ok else "FAIL ") + name)
check("all seven runs returned a session id", all([A1, B1, A2, X1, Y1, X2, B2]))
for tag, mine, other in [("E-A1", "PROJECT-A-MARKER", "PROJECT-B-MARKER"), ("E-B1", "PROJECT-B-MARKER", "PROJECT-A-MARKER"), ("E-A2", "PROJECT-A-MARKER", "PROJECT-B-MARKER"), ("E-B2", "PROJECT-B-MARKER", "PROJECT-A-MARKER")]:
    rs = req_for(tag)
    check("%s: model saw its own context file and not the other project's" % tag,
          bool(rs) and all(mine in sysof(r) and other not in sysof(r) for r in rs))
check("E-A2 resumed A1 (history has 3 user/assistant turns)", bool(req_for("E-A2")) and req_for("E-A2")[0]["n_messages"] >= 4)
check("A2 kept A1's session id, B2 kept B1's (A -> B -> A -> B)", A2 == A1 and B2 == B1)
b2 = req_for("E-B2")
check("E-B2 request carries B1 history and nothing from A", bool(b2) and any(u.startswith("E-B1") for u in b2[0]["user_texts"]) and not any(u.startswith(("E-A", "F-")) for u in b2[0]["user_texts"]))
check("X2 kept X1's session id, Y1 is a different session", X2 == X1 and Y1 != X1)
x2 = req_for("F-X2")
check("F-X2 request carries X1 history but not Y1's", bool(x2) and any(u.startswith("F-X1") for u in x2[0]["user_texts"]) and not any(u.startswith("F-Y1") for u in x2[0]["user_texts"]))
a2 = req_for("E-A2")
check("E-A2 request carries A1 history and nothing from B", bool(a2) and any(u.startswith("E-A1") for u in a2[0]["user_texts"]) and not any(u.startswith("E-B1") for u in a2[0]["user_texts"]))
def ids(f):
    try:
        data = json.load(open(f))
    except Exception:
        return set()
    items = data if isinstance(data, list) else data.get("sessions", [])
    return {s.get("uuid") or s.get("id") for s in items}  # session list: id rút gọn, uuid = session_id của run
a, b = ids(os.path.join(P, "A-sessions.json")), ids(os.path.join(P, "B-sessions.json"))
check("session list of A has A1, X1, Y1 and not B1", {A1, X1, Y1} <= a and B1 not in a)
check("session list of B has only B1", b == {B1})
print("WORKSPACES:", "PASS" if all(checks) else "FAIL")
sys.exit(0 if all(checks) else 1)
PY
