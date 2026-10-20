#!/usr/bin/env bash
# P0-2b e2e: `.env` của repo có lái được tiến trình con không? Khung cô lập của scripts/eval/exposure.sh (trích
# hàm bằng sed), binary thật + LLM giả, tmux riêng (không đụng màn hình). Không in giá trị env: CLI giả chỉ ghi
# TÊN biến kèm nguồn của giá trị canary.
#   CLI: env gốc bật giao việc (allowlist có claude_code, AGENT_CLI_CLAUDE_BIN = CLI giả ngoài repo) và export
#        OPENAI_API_KEY=CANARY_ORIGINAL; repo/.env đặt ANTHROPIC_BASE_URL/ANTHROPIC_API_KEY/OPENAI_BASE_URL/
#        OPENAI_API_KEY/CLAUDE_CONFIG_DIR/CODEX_HOME = CANARY_DOTENV. LLM giả gọi tool claude_code; `run --json` tự
#        duyệt. Chạy chế độ local rồi chế độ server (--host unix). Bất biến: CLI con không nhận biến nào có nguồn
#        dotenv; OPENAI_API_KEY:original phải có (đối chứng: giá trị người dùng export vẫn tới CLI con).
#   LN : chạy binary trần (không launcher), mở TUI thẳng trong repo có `.env` đặt AGENT_CLI_LAUNCHER=repo/evil;
#        Alt+W -> execHome -> Home mới -> Enter (dự án) [-> Enter "New session"]. Bất biến: Home mới không chạy
#        repo/evil; đối chứng: Home chạy được workspace (`<binary> --cwd repo`).
# Dùng: bash p02b-e2e.sh <thư-mục-mới ≤ 70 byte> <binary>
set -u
[ $# -eq 2 ] || { echo "usage: $0 <new-dir> <binary>" >&2; exit 2; }
OUT=$1; BIN=$2
R=/home/luannt/application/opensource/agent-cli-p02b
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd) && BIN=$(cd "$(dirname "$BIN")" && pwd)/$(basename "$BIN")
[ ${#OUT} -le 70 ] || { echo "dir path too long for unix sockets: $OUT" >&2; exit 2; }
L="$OUT/_launcher"
eval "$(sed -n '/^verdict()/,/^for c in \$CASES/p' "$R/scripts/eval/exposure.sh" | sed '$d')"
TMUX_BIN=$(command -v tmux)

# ---------- CLI ----------
begin CL '[{"match":"CL-","steps":[{"tool":"claude_code","args":{"prompt":"probe","model":"claude-opus-5-5"}},{"text":"done"}]}]' qwen
mkdir -p "$P/repo"; git_repo "$P/repo"
cat > "$P/repo/.env" <<'EOF'
ANTHROPIC_BASE_URL=http://CANARY_DOTENV
ANTHROPIC_API_KEY=CANARY_DOTENV
OPENAI_BASE_URL=http://CANARY_DOTENV
OPENAI_API_KEY=CANARY_DOTENV
CLAUDE_CONFIG_DIR=/CANARY_DOTENV
CODEX_HOME=/CANARY_DOTENV
EOF
cat > "$P/fakeclaude" <<EOF
#!/bin/sh
env | while IFS= read -r line; do
  k=\${line%%=*}; v=\${line#*=}
  case "\$v" in
    *CANARY_DOTENV*) echo "\$k:dotenv";;
    *CANARY_ORIGINAL*) echo "\$k:original";;
  esac
done | sort >> "$P/cli.names"
echo '{"result":"ok","session_id":"s1"}'
EOF
chmod 755 "$P/fakeclaude"
EXTRA=(AGENT_CLI_ALLOWED_MODELS="qwen:m-qwen claude_code:claude-opus-5-5" AGENT_CLI_DELEGATION=on
  AGENT_CLI_CLAUDE_BIN="$P/fakeclaude" OPENAI_API_KEY=CANARY_ORIGINAL)
local_ac "$P/repo" run --json "CL-local probe" > "$P/local-run.json" 2> "$P/local-run.err"; echo $? > "$P/local-run.rc"
[ -f "$P/cli.names" ] && mv "$P/cli.names" "$P/local.names"
server_mode
ac "$P/repo" run --json "CL-server probe" > "$P/server-run.json" 2> "$P/server-run.err"; echo $? > "$P/server-run.rc"
[ -f "$P/cli.names" ] && mv "$P/cli.names" "$P/server.names"
EXTRA=()
finish
python3 -I - "$P" <<'PY' | tee "$P/summary.txt"
import os, sys
P = sys.argv[1]
bad = []
for mode in ("local", "server"):
    f = os.path.join(P, mode + ".names")
    names = open(f).read().split() if os.path.exists(f) else None
    rc = open(os.path.join(P, mode + "-run.rc")).read().strip()
    if names is None:
        print("%s: CLI not run (rc=%s) -> INCONCLUSIVE" % (mode, rc)); bad.append("inconclusive"); continue
    dot = [n for n in names if n.endswith(":dotenv")]
    print("%s: rc=%s cli env canary names=%s" % (mode, rc, names))
    if "OPENAI_API_KEY:original" not in names:
        bad.append("inconclusive")
    elif dot:
        bad.append("exposed")
v = "INCONCLUSIVE" if "inconclusive" in bad else ("EXPOSED" if bad else "NOT_EXPOSED")
print("CLI:", v)
PY

# ---------- LN ----------
begin LN '[]' qwen
mkdir -p "$P/repo"; git_repo "$P/repo"
printf '#!/bin/sh\necho ran > "%s/evil.ran"\nsleep 2\n' "$P" > "$P/repo/evil"; chmod 755 "$P/repo/evil"
printf 'AGENT_CLI_LAUNCHER=%s/repo/evil\n' "$P" > "$P/repo/.env"
TM="$P/tmux.sock"
iso "$P/repo" "$TMUX_BIN" -S "$TM" new-session -d -x 160 -y 40 -s e "'$BIN' --cwd '$P/repo' 2> '$P/tui.stderr'; echo \$? > '$P/tui.rc'; sleep 3" 2> "$P/tmux.err"
for _ in $(seq 50); do [ "$("$TMUX_BIN" -S "$TM" capture-pane -p -t e 2>/dev/null | grep -c .)" -ge 5 ] && break; sleep 0.2; done
sleep 2
"$TMUX_BIN" -S "$TM" capture-pane -p -t e > "$P/tui-screen.txt" 2>/dev/null
if grep -q "initialize now" "$P/tui-screen.txt"; then
  "$TMUX_BIN" -S "$TM" send-keys -t e Right; sleep 0.5
  "$TMUX_BIN" -S "$TM" send-keys -t e Enter; sleep 1.5
fi
"$TMUX_BIN" -S "$TM" send-keys -t e M-w
for _ in $(seq 50); do "$TMUX_BIN" -S "$TM" capture-pane -p -t e 2>/dev/null | grep -q "Open current folder\|New session" && break; sleep 0.2; done
"$TMUX_BIN" -S "$TM" capture-pane -p -t e > "$P/home-screen.txt" 2>/dev/null
sleep 1
"$TMUX_BIN" -S "$TM" send-keys -t e Enter; sleep 1.5
"$TMUX_BIN" -S "$TM" capture-pane -p -t e > "$P/home-after.txt" 2>/dev/null
grep -q "New session" "$P/home-after.txt" && { "$TMUX_BIN" -S "$TM" send-keys -t e Enter; sleep 1.5; }
for _ in $(seq 25); do
  [ -e "$P/evil.ran" ] && break
  pgrep -f -- "--cwd $P/repo" > "$P/child.pids" 2>/dev/null && [ -s "$P/child.pids" ] && break
  sleep 0.2
done
sleep 1
"$TMUX_BIN" -S "$TM" capture-pane -p -t e > "$P/final-screen.txt" 2>/dev/null
pgrep -f -- "--cwd $P/repo" > "$P/child.pids" 2>/dev/null
"$TMUX_BIN" -S "$TM" kill-server 2>/dev/null
finish
{
  home=$(grep -c "Open current folder\|New session" "$P/home-screen.txt")
  if [ -e "$P/evil.ran" ]; then echo "LN: Home after Alt+W ran repo/evil (launcher from .env): EXPOSED"
  elif [ "$home" -gt 0 ] && [ -s "$P/child.pids" ]; then echo "LN: Home after Alt+W started '<binary> --cwd repo', not repo/evil: NOT_EXPOSED"
  else echo "LN: INCONCLUSIVE (home screen seen=$home, child started=$([ -s "$P/child.pids" ] && echo yes || echo no))"; fi
} | tee "$P/summary.txt"
