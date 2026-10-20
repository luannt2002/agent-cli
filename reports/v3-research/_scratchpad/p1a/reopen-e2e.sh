#!/usr/bin/env bash
# Kiểm "mở lại session thấy prompt đã ack là huỷ/Interrupted" với binary thật. Dùng lại hàm của
# scripts/eval/exposure.sh (cô lập HOME/XDG/socket, LLM giả). "Mở lại" = server MỚI đọc session qua
# ListSessionMessages (run -s → restoreModelFromSession), là đường settle của TUI; `session show` đọc thô.
# Dùng: reopen-e2e.sh <thư-mục-mới ≤70 byte> <binary> <số lần T11>
set -u
REPO=/home/luannt/application/opensource/agent-cli-p1a
LIB=$(dirname "$0")/exposure-lib.sh
sed '/^for c in \$CASES; do/,$d' "$REPO/scripts/eval/exposure.sh" > "$LIB"
N=${3:-4}
set -- "$1" "$2"
# shellcheck disable=SC1090
source "$LIB"
R=$REPO
dump() { python3 -I -c '
import json,sys
ms=json.load(open(sys.argv[1]))["messages"]
out=[]
for m in ms:
    fin=[p for p in m["parts"] if p.get("type")=="finish"]
    txt="".join(p.get("text","") for p in m["parts"] if p.get("type")=="text")[:22]
    out.append("%s[%s|%s]" % (m["role"][0], txt, ("%s:%s" % (fin[0].get("reason"), fin[0].get("message") or "")) if fin else "UNFINISHED"))
print(" ".join(out))' "$1" 2>/dev/null || echo "unreadable"; }

case_t13() { # $1 = TERM|KILL
  begin "RT13-$1" '[{"match":"T13-A","steps":[{"sleep":10,"text":"A done"}]},{"match":"T13-B","steps":[{"text":"B done"}]},{"match":"reopen","steps":[{"text":"reopen done"}]}]' qwen
  server_mode; mkdir -p "$P/proj"; git_repo "$P/proj"
  ac "$P/proj" run --json warm > /dev/null 2>&1
  local pid sid rcB
  pid=$(server_pid_of "$SOCK")
  sid=$(ac "$P/proj" send --new --title rt13 "T13-A long turn" 2>/dev/null | tail -1)
  sleep 1.5
  ac "$P/proj" send -s "$sid" "T13-B queued prompt" > /dev/null 2>&1; rcB=$?
  sleep 0.5
  kill -"$1" "$pid"; wait_gone "$pid"
  local_ac "$P/proj" session show "$sid" --json > "$P/before.json" 2>/dev/null
  ac "$P/proj" run -s "$sid" --json "reopen check" > "$P/reopen.json" 2> "$P/reopen.err"
  local_ac "$P/proj" session show "$sid" --json > "$P/after.json" 2>/dev/null
  finish
  echo "T13/$1 sendB_rc=$rcB"; echo "  before reopen: $(dump "$P/before.json")"; echo "  after  reopen: $(dump "$P/after.json")"
}

case_t11() { # SIGKILL ngay sau send, N lần
  begin RT11 '[{"match":"T11","steps":[{"sleep":20,"text":"late"}]},{"match":"reopen","steps":[{"text":"reopen done"}]}]' qwen
  server_mode; mkdir -p "$P/proj"; git_repo "$P/proj"
  local i pid sid rc
  for i in $(seq "$N"); do
    ac "$P/proj" run --json "warm $i" > /dev/null 2>&1
    pid=$(server_pid_of "$SOCK")
    sid=$(ac "$P/proj" send --new --title rt11 "T11 trial $i" 2>/dev/null | tail -1); rc=$?
    kill -9 "$pid"; wait_gone "$pid"
    local_ac "$P/proj" session show "$sid" --json > "$P/before-$i.json" 2>/dev/null
    ac "$P/proj" run -s "$sid" --json "reopen check $i" > "$P/reopen-$i.json" 2> "$P/reopen-$i.err"
    local_ac "$P/proj" session show "$sid" --json > "$P/after-$i.json" 2>/dev/null
    pid=$(server_pid_of "$SOCK"); [ -n "$pid" ] && kill -TERM "$pid" && wait_gone "$pid"
    echo "T11/$i send_rc=$rc"; echo "  before reopen: $(dump "$P/before-$i.json")"; echo "  after  reopen: $(dump "$P/after-$i.json")"
  done
  finish
}

case_t13 TERM
case_t13 KILL
case_t11
