set -u
OUT=$1; BIN=$2; R=/home/luannt/application/opensource/agent-cli-p02
mkdir -p "$OUT"; L="$OUT/_launcher"
eval "$(sed -n '/^verdict()/,/^for c in \$CASES/p' "$R/scripts/eval/exposure.sh" | sed '$d')"
begin DB '[]' qwen
mkdir -p "$P/repo"; git_repo "$P/repo"
printf '#!/bin/sh\nenv > "%s/child2.env"\n' "$P" > "$P/l2"; chmod 755 "$P/l2"
TM2="$P/t.sock"; EXTRA=(AGENT_CLI_LAUNCHER="$P/l2")
iso "$P/repo" "$(command -v tmux)" -S "$TM2" new-session -d -x 160 -y 40 -s e "'$BIN' --cwd '$P/repo' 2> '$P/tui.stderr'; echo \$? > '$P/tui.rc'; sleep 5"
sleep 6
"$(command -v tmux)" -S "$TM2" capture-pane -p -t e | grep -v "^\s*$" | tail -8
"$(command -v tmux)" -S "$TM2" send-keys -t e M-w
sleep 3
echo "--- after M-w"; "$(command -v tmux)" -S "$TM2" capture-pane -p -t e | grep -v "^\s*$" | tail -8
ls "$P"; cat "$P/tui.rc" 2>/dev/null
"$(command -v tmux)" -S "$TM2" kill-server 2>/dev/null; finish
