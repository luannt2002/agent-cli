#!/usr/bin/env bash
# P0-2b vòng 2 (#1): luồng hằng ngày `agent-cli` trần trong repo -> Home -> chọn dự án. CLI con giao việc (claude_code)
# của workspace mà Home dựng có nhận biến từ `.env` của repo không? Khung cô lập của scripts/eval/exposure.sh (trích hàm
# bằng sed), binary thật + LLM giả, tmux riêng (không đụng màn hình). Không in giá trị env: CLI giả chỉ ghi TÊN biến kèm
# nguồn của giá trị canary.
#   Env gốc: bật giao việc (allowlist có claude_code, AGENT_CLI_CLAUDE_BIN = CLI giả ngoài repo), export
#   OPENAI_API_KEY=CANARY_ORIGINAL, AGENT_CLI_LAUNCHER = launcher giả (như launcher thật, đặt ở env gốc). Launcher giả
#   thay cho TUI của workspace: exec `<binary> run --json` trong ĐÚNG env mà Home đưa cho nó (LLM giả gọi claude_code,
#   `run` tự duyệt). repo/.env đặt ANTHROPIC_BASE_URL/ANTHROPIC_API_KEY/OPENAI_BASE_URL/OPENAI_API_KEY/CLAUDE_CONFIG_DIR/
#   CODEX_HOME = CANARY_DOTENV (không export).
#   HCL: launcher chạy chế độ local. HCS: launcher chạy chế độ server (--host unix riêng, như launcher thật).
# Bất biến: CLI con không nhận biến nguồn dotenv; OPENAI_API_KEY:original phải có (đối chứng).
# Dùng: bash p02b-home-cli.sh <thư-mục-mới ≤ 70 byte> <binary>
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

home_case() { # home_case <ca> <local|server>
  begin "$1" '[{"match":"HC-","steps":[{"tool":"claude_code","args":{"prompt":"probe","model":"claude-opus-5-5"}},{"text":"done"}]}]' qwen
  mkdir -p "$P/repo"; git_repo "$P/repo"
  printf 'ANTHROPIC_BASE_URL=http://CANARY_DOTENV\nANTHROPIC_API_KEY=CANARY_DOTENV\nOPENAI_BASE_URL=http://CANARY_DOTENV\nOPENAI_API_KEY=CANARY_DOTENV\nCLAUDE_CONFIG_DIR=/CANARY_DOTENV\nCODEX_HOME=/CANARY_DOTENV\n' > "$P/repo/.env"
  cat > "$P/fakeclaude" <<EOF
#!/bin/sh
env | while IFS= read -r line; do
  k=\${line%%=*}; v=\${line#*=}
  case "\$v" in
    *CANARY_DOTENV*) echo "\$k:dotenv";;
    *CANARY_ORIGINAL*) echo "\$k:original";;
  esac
done | sort > "$P/cli.names"
echo '{"result":"ok","session_id":"s1"}'
EOF
  local hostargs="" cs=0
  [ "$2" = server ] && { hostargs="--host unix://$P/s.sock"; cs=1; }
  cat > "$P/launcher" <<EOF
#!/bin/sh
env | while IFS= read -r line; do v=\${line#*=}; case "\$v" in *CANARY_DOTENV*) echo "\${line%%=*}";; esac; done | sort > "$P/launcher-env.dotenv-names"
CRUSH_CLIENT_SERVER=$cs exec "$BIN" $hostargs run --json "HC-$2 probe" > "$P/run.json" 2> "$P/run.err"
EOF
  chmod 755 "$P/fakeclaude" "$P/launcher"
  EXTRA=(AGENT_CLI_ALLOWED_MODELS="qwen:m-qwen claude_code:claude-opus-5-5" AGENT_CLI_DELEGATION=on
    AGENT_CLI_CLAUDE_BIN="$P/fakeclaude" OPENAI_API_KEY=CANARY_ORIGINAL AGENT_CLI_LAUNCHER="$P/launcher")
  local TM="$P/tmux.sock"
  iso "$P/repo" "$TMUX_BIN" -S "$TM" new-session -d -x 160 -y 40 -s h "'$BIN' 2> '$P/home.stderr'; echo \$? > '$P/home.rc'; sleep 3" 2> "$P/tmux.err"
  for _ in $(seq 50); do "$TMUX_BIN" -S "$TM" capture-pane -p -t h 2>/dev/null | grep -q "Open current folder" && break; sleep 0.2; done
  "$TMUX_BIN" -S "$TM" capture-pane -p -t h > "$P/home-screen.txt" 2>/dev/null
  sleep 1
  "$TMUX_BIN" -S "$TM" send-keys -t h Enter; sleep 1
  "$TMUX_BIN" -S "$TM" capture-pane -p -t h > "$P/home-after.txt" 2>/dev/null
  grep -q "New session" "$P/home-after.txt" && "$TMUX_BIN" -S "$TM" send-keys -t h Enter
  for _ in $(seq 200); do [ -s "$P/run.json" ] && break; sleep 0.2; done
  sleep 1
  "$TMUX_BIN" -S "$TM" kill-server 2>/dev/null
  EXTRA=()
  finish
  python3 -I - "$P" "$1" <<'PY' | tee "$P/summary.txt"
import os, sys
P, name = sys.argv[1], sys.argv[2]
rd = lambda f: open(os.path.join(P, f)).read().split() if os.path.exists(os.path.join(P, f)) else None
names, lnames = rd("cli.names"), rd("launcher-env.dotenv-names")
print("%s: workspace env from Home carries .env names=%s | CLI env canary names=%s" % (name, lnames, names))
if names is None or "OPENAI_API_KEY:original" not in names:
    print("%s: INCONCLUSIVE" % name)
else:
    print("%s: %s" % (name, "EXPOSED" if any(n.endswith(":dotenv") for n in names) else "NOT_EXPOSED"))
PY
}
home_case HCL local
home_case HCS server
