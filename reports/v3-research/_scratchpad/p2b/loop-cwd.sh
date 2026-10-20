#!/usr/bin/env bash
# Đo `loop -c ..` gọi từ repo/internal: workspace của agent (tool bash ghi `pwd`, dòng "Working directory" trong
# system prompt, crush.db chứa session) phải là repo, trùng thư mục verify chạy. LLM giả, HOME cô lập.
# Bất biến: OK khi tệp agent-pwd.txt nằm ở repo/, nội dung = repo, prompt nói repo, DB ở repo/.crush.
# Dùng: bash loop-cwd.sh <thư-mục-mới, ngắn> <binary> <repo agent-cli để lấy fakellm.py>
set -u
OUT=$1; BIN=$2; R=$3
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT/home/.config/agent-cli" "$OUT/log" "$OUT/rt"; chmod 700 "$OUT/rt"
OUT=$(cd "$OUT" && pwd)
cat > "$OUT/rules.json" <<'JSON'
[{"match":"LOOPCWD","steps":[{"tool":"bash","args":{"command":"pwd > agent-pwd.txt","description":"pwd"}},{"text":"done"}]}]
JSON
timeout 600 python3 -I "$R/scripts/eval/fakellm.py" "$OUT/port" "$OUT/log" "$OUT/rules.json" >/dev/null 2>&1 &
FAKE=$!
for _ in $(seq 50); do [ -s "$OUT/port" ] && break; sleep 0.1; done
PORT=$(cat "$OUT/port")
cat > "$OUT/home/.config/agent-cli/crush.json" <<JSON
{"providers":{"qwen":{"type":"openai-compat","base_url":"http://127.0.0.1:$PORT/qwen/v1","api_key":"x","models":[{"id":"m-qwen","name":"m-qwen","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"qwen","model":"m-qwen"},"small":{"provider":"qwen","model":"m-qwen"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true,"disable_default_providers":true}}
JSON
iso() { # iso <dir> <CS> <args...>
  local d=$1 cs=$2; shift 2
  (cd "$d" && env -i PATH=/usr/bin:/bin HOME="$OUT/home" XDG_CONFIG_HOME="$OUT/home/.config" \
    XDG_DATA_HOME="$OUT/home/.local/share" XDG_CACHE_HOME="$OUT/home/.cache" CRUSH_GLOBAL_CONFIG="$OUT/home/.config/agent-cli" \
    CRUSH_GLOBAL_DATA="$OUT/home/.local/share/agent-cli" CRUSH_CACHE_DIR="$OUT/home/.cache/agent-cli" AGENT_CLI_RUNTIME_DIR="$OUT/rt" \
    GIT_CEILING_DIRECTORIES="$OUT" CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 \
    CRUSH_CLIENT_SERVER=$cs CRUSH_SERVER_IDLE_TIMEOUT=60 timeout 90 "$BIN" "$@")
}
mk() { mkdir -p "$OUT/$1/w/repo/internal"; (cd "$OUT/$1/w/repo" && git init -q && echo x > README.md && echo y > internal/x.go && git add -A && git -c user.name=t -c user.email=t@t commit -qm init); }
stop() { local pid; for pid in $(pgrep -f -- "server --host unix://$OUT/$1.sock"); do kill -TERM "$pid"; for _ in $(seq 100); do kill -0 "$pid" 2>/dev/null || break; sleep 0.1; done; done; }
row() { # row <ca> <rc>
  python3 -I - "$OUT" "$1" "$2" <<'PY' | tee -a "$OUT/rows.txt"
import glob, json, os, sqlite3, sys
OUT, case, rc = sys.argv[1:4]
root = os.path.join(OUT, case)
repo = os.path.join(root, "w", "repo")
try:
    j = json.load(open(os.path.join(OUT, case + ".json")))
except Exception:
    j = {}
sids = j.get("session_ids") or []
pwdf = sorted(os.path.relpath(p, root) for p in glob.glob(os.path.join(root, "**", "agent-pwd.txt"), recursive=True))
pwdv = [open(os.path.join(root, p)).read().strip().replace(root, "<ca>") for p in pwdf]
wd = set()
for f in glob.glob(os.path.join(OUT, "log", "sys-*.txt")):
    for line in open(f, errors="replace"):
        if line.startswith("Working directory:") and root + "/" in line + "/":
            wd.add(line.split(":", 1)[1].strip().replace(root, "<ca>"))
dbs = []
for d, _, fs in os.walk(root):
    if "crush.db" in fs:
        p = os.path.join(d, "crush.db")
        c = sqlite3.connect("file:%s?mode=ro" % p, uri=True)
        if any(c.execute("select count(*) from sessions where id=?", (s,)).fetchone()[0] for s in sids):
            dbs.append(os.path.relpath(p, root))
        c.close()
ok = pwdf == ["w/repo/agent-pwd.txt"] and pwdv == ["<ca>/w/repo"] and wd == {"<ca>/w/repo"} and dbs == ["w/repo/.crush/crush.db"]
print("%s|%s|rc=%s reason=%s|agent-pwd.txt at %s = %s|prompt Working directory %s|session db %s" % (
    case, "OK" if ok else "WRONG", rc, j.get("reason"), pwdf or "none", pwdv, sorted(wd) or "none", dbs or "none"))
PY
}
VER=(--verify 'test -f agent-pwd.txt')
mk LS; iso "$OUT/LS/w/repo/internal" 1 --host "unix://$OUT/LS.sock" -c .. loop --json "${VER[@]}" 'LOOPCWD write pwd' > "$OUT/LS.json" 2> "$OUT/LS.err"; rc=$?; stop LS; row LS $rc
mk LL; iso "$OUT/LL/w/repo/internal" 0 -c .. loop --json "${VER[@]}" 'LOOPCWD write pwd' > "$OUT/LL.json" 2> "$OUT/LL.err"; row LL $?
mk CS; iso "$OUT/CS/w/repo/internal" 1 --host "unix://$OUT/CS.sock" -c "$OUT/CS/w/repo" loop --json "${VER[@]}" 'LOOPCWD write pwd' > "$OUT/CS.json" 2> "$OUT/CS.err"; rc=$?; stop CS; row CS $rc
mk CL; iso "$OUT/CL/w/repo/internal" 0 -c "$OUT/CL/w/repo" loop --json "${VER[@]}" 'LOOPCWD write pwd' > "$OUT/CL.json" 2> "$OUT/CL.err"; row CL $?
kill "$FAKE" 2>/dev/null; wait "$FAKE" 2>/dev/null
bad=$(awk -F'|' '$1 ~ /^L/ && $2 != "OK" {printf "%s ", $1}' "$OUT/rows.txt")
ctl=$(awk -F'|' '$1 ~ /^C/ && $2 != "OK" {printf "%s ", $1}' "$OUT/rows.txt")
if [ -n "$ctl" ]; then echo "LOOPCWD: INCONCLUSIVE (control wrong: $ctl)"; elif [ -n "$bad" ]; then echo "LOOPCWD: EXPOSED ($bad)"; else echo "LOOPCWD: NOT_EXPOSED"; fi
exit 0
