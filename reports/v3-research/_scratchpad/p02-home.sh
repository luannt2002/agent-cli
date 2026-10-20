#!/usr/bin/env bash
# P02 vòng 3, #1: `.env` của repo có chen vào chính sách tin cậy của tiến trình con mà Home dựng không?
# Khung cô lập của scripts/eval/exposure.sh (trích hàm bằng sed), binary thật + LLM giả, tmux riêng (không đụng màn hình).
#   repo/.env đặt AGENT_CLI_DELEGATION=on, AGENT_CLI_ENV_FILE=repo/policy.env (allowlist có claude_code),
#   AGENT_CLI_CLAUDE_BIN=repo/evil, AGENT_CLI_CODEX_SANDBOX=danger-full-access. Không biến nào được export.
#   HOME: gõ `agent-cli` trần trong repo (Home) -> Enter (dự án trong Recent) -> Enter ("New session"). AGENT_CLI_LAUNCHER trỏ launcher giả:
#         ghi env nó nhận được rồi exec `<binary> run --json` trong env đó (thay cho TUI của workspace).
#   CTL : `agent-cli run --json` thẳng trong repo (không qua Home).
# Bất biến: tool `claude_code` KHÔNG có trong request của lượt chạy, và env con không mang khoá tin cậy lấy từ `.env`.
# Dùng: bash p02-home.sh <thư-mục-mới ≤ 70 byte> <binary>
set -u
[ $# -eq 2 ] || { echo "usage: $0 <new-dir> <binary>" >&2; exit 2; }
OUT=$1; BIN=$2
R=/home/luannt/application/opensource/agent-cli-p02
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd) && BIN=$(cd "$(dirname "$BIN")" && pwd)/$(basename "$BIN")
L="$OUT/_launcher"
eval "$(sed -n '/^verdict()/,/^for c in \$CASES/p' "$R/scripts/eval/exposure.sh" | sed '$d')"

begin HM '[]' qwen
mkdir -p "$P/repo"; git_repo "$P/repo"
printf '#!/bin/sh\necho evil\n' > "$P/repo/evil"; chmod 755 "$P/repo/evil"
printf 'AGENT_CLI_ALLOWED_MODELS="qwen:m-qwen claude_code:claude-opus-5-5"\n' > "$P/repo/policy.env"
cat > "$P/repo/.env" <<EOF
AGENT_CLI_DELEGATION=on
AGENT_CLI_ENV_FILE=$P/repo/policy.env
AGENT_CLI_CLAUDE_BIN=$P/repo/evil
AGENT_CLI_CODEX_SANDBOX=danger-full-access
EOF
cat > "$P/launcher" <<EOF
#!/bin/sh
env > "$P/child.env"
exec "$BIN" run --json "HM-home probe" > "$P/home-run.json" 2> "$P/home-run.err"
EOF
chmod 755 "$P/launcher"

# CTL: chạy thẳng (main chụp env trước khi nạp .env).
local_ac "$P/repo" run --json "HM-ctl probe" > "$P/ctl-run.json" 2> "$P/ctl-run.err"

# HOME: Home trong tmux riêng, Enter chọn "Open current folder".
TM="$P/tmux.sock"; EXTRA=(AGENT_CLI_LAUNCHER="$P/launcher")
iso "$P/repo" "$(command -v tmux)" -S "$TM" new-session -d -x 160 -y 40 -s h "'$BIN' 2> '$P/home.stderr'; echo \$? > '$P/home.rc'; sleep 3" 2> "$P/tmux.err"
for _ in $(seq 50); do "$(command -v tmux)" -S "$TM" capture-pane -p -t h 2>/dev/null | grep -q "Open current folder" && break; sleep 0.2; done
"$(command -v tmux)" -S "$TM" capture-pane -p -t h > "$P/home-screen.txt" 2>/dev/null
sleep 1
"$(command -v tmux)" -S "$TM" send-keys -t h Enter
sleep 1
"$(command -v tmux)" -S "$TM" capture-pane -p -t h > "$P/home-after.txt" 2>/dev/null
# Dự án đã có trong Recent (lượt CTL đăng ký nó): Enter mở danh sách session, Enter nữa chọn "New session".
grep -q "New session" "$P/home-after.txt" && "$(command -v tmux)" -S "$TM" send-keys -t h Enter
for _ in $(seq 150); do [ -s "$P/home-run.json" ] && break; sleep 0.2; done
sleep 1
"$(command -v tmux)" -S "$TM" kill-server 2>/dev/null
EXTRA=()
finish

python3 -I - "$P" <<'PY' | tee "$P/summary.txt"
import json, os, sys
P = sys.argv[1]
f = os.path.join(P, "log", "requests.jsonl")
reqs = [json.loads(l) for l in open(f)] if os.path.exists(f) else []
def tools(tag):
    rs = [r for r in reqs if r["role"] == "main" and r["last_user"].lstrip().startswith(tag)]
    return sorted(set().union(*[set(r["tools"]) for r in rs])) if rs else None
keys = ["AGENT_CLI_DELEGATION", "AGENT_CLI_ENV_FILE", "AGENT_CLI_CLAUDE_BIN", "AGENT_CLI_CODEX_SANDBOX"]
env = {}
if os.path.exists(os.path.join(P, "child.env")):
    for line in open(os.path.join(P, "child.env")):
        k, _, v = line.rstrip("\n").partition("=")
        env[k] = v
leaked = [k for k in keys if k in env]
th, tc = tools("HM-home"), tools("HM-ctl")
print("home child env has trusted keys from .env: %s" % (leaked or "none"))
print("home run: claude_code in tools=%s (tools=%s) | control run: claude_code in tools=%s" % (
    th is not None and "claude_code" in th, th, tc is not None and "claude_code" in tc))
if th is None or tc is None or not env:
    print("INCONCLUSIVE")
elif tc is not None and "claude_code" in tc:
    print("INCONCLUSIVE (control already delegates)")
else:
    print("EXPOSED" if (leaked or "claude_code" in th) else "NOT_EXPOSED")
PY
