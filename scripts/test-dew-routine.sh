#!/usr/bin/env bash
# E2E cho `agent-cli routine` với server OpenAI-compat giả. Chứng minh trên binary thật:
# routine chạy trong worktree riêng (repo gốc không bị đụng), bị CHẶN git push khi
# không người trực trong khi `run` cùng lệnh đó push được (ca đối chứng), khoá chống
# chạy chồng (mã 75), từ chối --scheduled cho việc có đích và định nghĩa đã đổi,
# huỷ bằng SIGTERM, và `schedule --print` chỉ in.
# Mọi thứ nằm trong thư mục tạm. Dùng: scripts/test-dew-routine.sh
# (DEW_BIN=... để bỏ qua build)
set -uo pipefail
here="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
t="$(mktemp -d /tmp/aclrtn.XXXXXX)"
fake_pid=""
hold_pid=""
cleanup() {
  [ -n "$fake_pid" ] && kill "$fake_pid" 2>/dev/null
  [ -n "$hold_pid" ] && kill "$hold_pid" 2>/dev/null
  [ -n "${KEEP_TMP:-}" ] && echo "kept $t" || rm -rf "$t"
}
trap cleanup EXIT

fail=0
# Không đặt $(...) trong tên check: nó ghi đè $? trước khi check đọc. Đọc quan sát
# vào biến TRƯỚC điều kiện.
check() { if [ "$2" -eq 0 ]; then echo "ok   $1"; else echo "FAIL $1"; fail=1; fi; }
# jget <file> <key>: giá trị key (có thể lồng "a.b") của object JSON trong tệp.
jget() {
  python3 - "$1" "$2" <<'PY'
import json, sys
obj = json.load(open(sys.argv[1]))
for k in sys.argv[2].split("."):
    obj = obj.get(k, "") if isinstance(obj, dict) else ""
print(obj)
PY
}

bin="${DEW_BIN:-$t/agent-cli}"
if [ -z "${DEW_BIN:-}" ]; then
  (cd "$here" && GOMAXPROCS=2 go build -p 2 -o "$bin" .) || { echo "build failed"; exit 1; }
fi

# --- server giả: TOOLCALL tạo e2e-probe; PUSHCALL thử git push; SLOW chờ 30s ----
cat > "$t/fake.py" <<'PY'
import http.server, json, sys, time
class H(http.server.BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers.get('content-length', 0))) or b'{}')
        users = [m for m in body.get('messages', []) if m.get('role') == 'user']
        last = users[-1]['content']
        if isinstance(last, list): last = ''.join(p.get('text', '') for p in last)
        msgs = body.get('messages', [])
        if 'SLOW' in last and body.get('stream'): time.sleep(30)
        text = "TOOLDONE" if msgs and msgs[-1].get('role') == 'tool' else "ECHO"
        usage = {"prompt_tokens": 11, "completion_tokens": 7, "total_tokens": 18}
        c = lambda d, **k: {"id": "1", "object": "chat.completion.chunk", "model": "m", **k}
        if body.get('stream'):
            self.send_response(200); self.send_header('content-type', 'text/event-stream'); self.end_headers()
            ev = lambda d: self.wfile.write(("data: " + json.dumps(d) + "\n\n").encode())
            cmd = None
            if msgs[-1].get('role') != 'tool':
                if 'ALIASCALL' in last: cmd = "git -c alias.p=push p origin HEAD:refs/heads/aliased"
                elif 'PUSHCALL' in last: cmd = "git push origin HEAD:refs/heads/pushed"
                elif 'TOOLCALL' in last: cmd = "touch e2e-probe"
            if cmd:
                call = {"index": 0, "id": "call_1", "type": "function", "function": {"name": "bash", "arguments": json.dumps({"command": cmd, "description": "probe"})}}
                ev(c(0, choices=[{"index": 0, "delta": {"role": "assistant", "tool_calls": [call]}, "finish_reason": None}]))
                ev(c(0, choices=[{"index": 0, "delta": {}, "finish_reason": "tool_calls"}]))
            else:
                ev(c(0, choices=[{"index": 0, "delta": {"role": "assistant", "content": text}, "finish_reason": None}]))
                ev(c(0, choices=[{"index": 0, "delta": {}, "finish_reason": "stop"}]))
            ev(c(0, choices=[], usage=usage))
            self.wfile.write(b"data: [DONE]\n\n")
        else:
            self.send_response(200); self.send_header('content-type', 'application/json'); self.end_headers()
            self.wfile.write(json.dumps({"id": "1", "object": "chat.completion", "model": "m", "choices": [{"index": 0, "message": {"role": "assistant", "content": "Title"}, "finish_reason": "stop"}], "usage": usage}).encode())
srv = http.server.ThreadingHTTPServer(('127.0.0.1', 0), H)
open(sys.argv[1], 'w').write(str(srv.server_address[1]))
srv.serve_forever()
PY
python3 "$t/fake.py" "$t/port" >/dev/null 2>&1 &
fake_pid=$!
for _ in $(seq 50); do [ -s "$t/port" ] && break; sleep 0.1; done
[ -s "$t/port" ] || { echo "fake server did not start"; exit 1; }

# --- môi trường cô lập ------------------------------------------------------
mkdir -p "$t/home"
export HOME="$t/home" XDG_CONFIG_HOME="$t/home/c" XDG_DATA_HOME="$t/home/d" XDG_CACHE_HOME="$t/home/x"
export XDG_RUNTIME_DIR="$t/run"; mkdir -p "$t/run" && chmod 700 "$t/run"
export CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1
unset CRUSH_CLIENT_SERVER DEW_UNATTENDED DEW_UNATTENDED_COMMIT
git config --global user.email t@t; git config --global user.name t
git config --global protocol.file.allow always
cat > "$t/crush.json" <<JSON
{"providers":{"fake":{"type":"openai-compat","base_url":"http://127.0.0.1:$(cat "$t/port")/v1","api_key":"x","models":[{"id":"m","name":"m","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"fake","model":"m"},"small":{"provider":"fake","model":"m"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true}}
JSON

# mkrepo <tên>: repo git có crush.json đã commit và remote origin = repo bare riêng.
mkrepo() {
  local d="$t/$1"; mkdir -p "$d"; cp "$t/crush.json" "$d/"
  git init -q --bare "$t/$1.bare.git"
  (cd "$d" && git init -q -b main && git add -A && git commit -q -m init && git remote add origin "$t/$1.bare.git")
  echo "$d"
}
data="$t/home/d/crush/routines"
defs="$t/home/c/crush/routines"

# ---- 1: routine new / list / thiếu verify ------------------------------------
repo="$(mkrepo green)"
"$bin" routine new green --mode maintenance --repo "$repo" --goal "TOOLCALL create the probe file" --verify 'test -f e2e-probe' >"$t/new.out" 2>&1; rc=$?
fmode="$(stat -c %a "$defs/green.json" 2>/dev/null)"
[ "$rc" -eq 0 ] && [ "$fmode" = 600 ]; check "routine new ghi định nghĩa 0600 (rc=$rc mode=$fmode)" $?
"$bin" routine new noverify --mode maintenance --repo "$repo" --goal x >"$t/nv.out" 2>/dev/null; rc=$?
[ "$rc" -eq 2 ] && [ ! -e "$defs/noverify.json" ]; check "routine new thiếu --verify -> rc=2, không ghi tệp (rc=$rc)" $?
"$bin" routine list >"$t/list0.out" 2>&1
grep -q 'green.*never run' "$t/list0.out"; check "routine list: chưa chạy lần nào" $?

# ---- 2: chạy xanh trong worktree riêng, repo gốc nguyên vẹn -------------------
head_before="$(git -C "$repo" rev-parse HEAD)"
(cd "$t" && "$bin" routine run green) >"$t/run1.out" 2>"$t/run1.err"; rc=$?
"$bin" routine show green >"$t/show1.json" 2>/dev/null
wt="$(jget "$t/show1.json" last_run.worktree)"; branch="$(jget "$t/show1.json" last_run.branch)"; reason="$(jget "$t/show1.json" last_run.reason)"
status_after="$(git -C "$repo" status --porcelain)"; head_after="$(git -C "$repo" rev-parse HEAD)"
branch_listed="$(git -C "$repo" branch --list "routine/green-*")"
[ "$rc" -eq 0 ] && [ "$reason" = GREEN ]; check "routine run -> GREEN, rc=0 (rc=$rc $reason)" $?
[ -n "$wt" ] && [ -e "$wt/e2e-probe" ] && [ ! -e "$repo/e2e-probe" ]; check "agent làm trong worktree ($wt), không phải checkout của người dùng" $?
[ -z "$status_after" ] && [ "$head_before" = "$head_after" ] && [ -n "$branch_listed" ]; check "repo gốc nguyên vẹn (status rỗng, HEAD không đổi), nhánh $branch có mặt" $?
rmode="$(stat -c %a "$(dirname "$(jget "$t/show1.json" last_run.journal)")/report.json" 2>/dev/null)"
[ "$rmode" = 600 ]; check "báo cáo chỉ chủ đọc, nằm ngoài worktree (mode $rmode)" $?
"$bin" routine list >"$t/list1.out" 2>&1
grep -q 'green.*GREEN' "$t/list1.out"; check "routine list hiện lần chạy cuối GREEN" $?

# ---- 3: chặn git push khi không người trực; đối chứng bằng `run` --------------
repo2="$(mkrepo pusher)"
"$bin" routine new pusher --mode goal --repo "$repo2" --goal "PUSHCALL try to publish the work" --verify 'test -f never' --max-repairs 0 >/dev/null 2>&1
(cd "$t" && "$bin" routine run pusher) >"$t/run2.out" 2>"$t/run2.err"; rc=$?
"$bin" routine show pusher >"$t/show2.json" 2>/dev/null
reason="$(jget "$t/show2.json" last_run.reason)"
refs_after_routine="$(git --git-dir="$t/pusher.bare.git" for-each-ref)"
[ "$rc" -eq 1 ] && [ "$reason" = MAX_REPAIRS ] && [ -z "$refs_after_routine" ]; check "routine: agent thử git push -> bị chặn, remote vẫn rỗng (rc=$rc $reason refs='$refs_after_routine')" $?
# Đối chứng: cùng lệnh qua `run` (không chế độ không người trực) PUSH ĐƯỢC. Không có vế
# này thì "remote rỗng" có thể chỉ vì agent không bao giờ gọi git push.
(cd "$repo2" && "$bin" run --json "PUSHCALL try to publish the work") >"$t/ctl.json" 2>/dev/null; rc=$?
refs_after_run="$(git --git-dir="$t/pusher.bare.git" for-each-ref)"
[ "$rc" -eq 0 ] && [[ "$refs_after_run" == *refs/heads/pushed* ]]; check "đối chứng: lệnh run (không chế độ không người trực) cùng lệnh PUSH ĐƯỢC (rc=$rc refs='$refs_after_run')" $?

# ---- 3b: đường vòng alias (`git -c alias.p=push p`) cũng bị chặn; đối chứng bằng run ---
repo4="$(mkrepo aliaser)"
"$bin" routine new aliaser --mode goal --repo "$repo4" --goal "ALIASCALL try to publish through an alias" --verify 'test -f never' --max-repairs 0 >/dev/null 2>&1
(cd "$t" && "$bin" routine run aliaser) >"$t/run3.out" 2>"$t/run3.err"; rc=$?
refs_alias_routine="$(git --git-dir="$t/aliaser.bare.git" for-each-ref)"
[ "$rc" -eq 1 ] && [ -z "$refs_alias_routine" ]; check "routine: git -c alias.p=push p bị chặn, remote vẫn rỗng (rc=$rc refs='$refs_alias_routine')" $?
(cd "$repo4" && "$bin" run --json "ALIASCALL try to publish through an alias") >/dev/null 2>&1; rc=$?
refs_alias_run="$(git --git-dir="$t/aliaser.bare.git" for-each-ref)"
[ "$rc" -eq 0 ] && [[ "$refs_alias_run" == *refs/heads/aliased* ]]; check "đối chứng: cùng lệnh alias qua run PUSH ĐƯỢC (rc=$rc refs='$refs_alias_run')" $?
(cd "$t" && "$bin" routine run green --cwd "$repo") >/dev/null 2>"$t/cwd.err"; rc=$?
cwd_err="$(cat "$t/cwd.err")"
[ "$rc" -eq 2 ] && [[ "$cwd_err" == *worktree* ]]; check "routine run --cwd bị từ chối (rc=$rc)" $?

# ---- 4: khoá chống chạy chồng (mã 75) -----------------------------------------
sleep 1.1
lockfile="$data/green/green.lock"
# exec thay subshell bằng sleep để PID giữ khoá chính là $hold_pid: giết nó thì khoá nhả
# (flock FILE cmd chạy cmd như tiến trình con nên giết flock không nhả khoá).
( exec 9>"$lockfile"; flock -n 9 && exec sleep 20 ) &
hold_pid=$!
sleep 0.5
branches_before="$(git -C "$repo" branch --list "routine/green-*" | wc -l)"
(cd "$t" && "$bin" routine run green) >"$t/busy.out" 2>"$t/busy.err"; rc=$?
branches_after="$(git -C "$repo" branch --list "routine/green-*" | wc -l)"
[ "$rc" -eq 75 ] && [ "$branches_before" = "$branches_after" ]; check "đang giữ khoá -> rc=75, không tạo nhánh (rc=$rc $branches_before->$branches_after)" $?
kill "$hold_pid" 2>/dev/null; wait "$hold_pid" 2>/dev/null; hold_pid=""
sleep 0.3
(cd "$t" && "$bin" routine run green) >"$t/free.out" 2>"$t/free.err"; rc=$?
[ "$rc" -eq 0 ]; check "nhả khoá -> chạy lại được (rc=$rc)" $?

# ---- 5: --scheduled chỉ cho maintenance; ghim hash --------------------------
(cd "$t" && "$bin" routine run pusher --scheduled) >/dev/null 2>"$t/sg.err"; rc=$?
[ "$rc" -eq 2 ]; check "--scheduled với việc có đích (goal) -> rc=2 (rc=$rc)" $?
(cd "$t" && "$bin" routine run green --scheduled --expect-sha "$(printf 'b%.0s' $(seq 64))") >/dev/null 2>"$t/sha.err"; rc=$?
sha_err="$(cat "$t/sha.err")"
[ "$rc" -eq 2 ] && [[ "$sha_err" == *changed* ]]; check "--expect-sha lệch -> rc=2, nêu 'changed' (rc=$rc)" $?

# ---- 6: lịch chỉ IN, ghim hash, từ chối việc có đích -----------------------
python3 - "$defs/green.json" <<'PY'
import json, sys
p = sys.argv[1]; d = json.load(open(p))
d["schedule"] = {"on_calendar": "Mon..Fri 09:00", "timezone": "UTC"}
json.dump(d, open(p, "w"), indent=2)
PY
"$bin" routine show green >"$t/show3.json" 2>/dev/null
sha="$(jget "$t/show3.json" sha256)"
"$bin" routine schedule green --print >"$t/sched.out" 2>"$t/sched.err"; rc=$?
sched="$(cat "$t/sched.out")"
[ "$rc" -eq 0 ] && [[ "$sched" == *"Persistent=true"* ]] && [[ "$sched" == *"routine run green --scheduled --expect-sha $sha"* ]] && [[ "$sched" == *"OnCalendar=Mon..Fri 09:00"* ]]; check "schedule --print in unit systemd có Persistent và hash đã ghim (rc=$rc)" $?
"$bin" routine schedule green >/dev/null 2>&1; rc=$?
[ "$rc" -eq 2 ]; check "schedule không có --print -> rc=2 (không tự cài gì) (rc=$rc)" $?
"$bin" routine schedule pusher --print >/dev/null 2>&1; rc=$?
[ "$rc" -eq 2 ]; check "schedule việc có đích -> rc=2 (rc=$rc)" $?
no_units="$(ls "$t/home/c/systemd" 2>/dev/null)"
[ -z "$no_units" ]; check "không tự cài timer systemd nào (thư mục: '$no_units')" $?
sleep 1.1
(cd "$t" && "$bin" routine run green --scheduled --expect-sha "$sha") >/dev/null 2>"$t/pinned.err"; rc=$?
[ "$rc" -eq 0 ]; check "hash khớp + --scheduled (maintenance) -> chạy được (rc=$rc)" $?

# ---- 7: SIGTERM giữa lượt -> CANCELLED, báo cáo có, khoá nhả ------------------
repo3="$(mkrepo slow)"
"$bin" routine new slow --mode goal --repo "$repo3" --goal "SLOW wait" --verify 'test -f never' >/dev/null 2>&1
(cd "$t" && exec "$bin" routine run slow) >/dev/null 2>"$t/slow.err" &
ps_=$!
sleep 6
kill -TERM "$ps_" 2>/dev/null
( sleep 25; kill -9 "$ps_" 2>/dev/null ) & wd=$!
wait "$ps_"; rc=$?
kill "$wd" 2>/dev/null; wait "$wd" 2>/dev/null
"$bin" routine show slow >"$t/show4.json" 2>/dev/null
reason="$(jget "$t/show4.json" last_run.reason)"
[ "$rc" -eq 1 ] && [ "$reason" = CANCELLED ]; check "SIGTERM giữa lượt -> CANCELLED, rc=1, báo cáo vẫn được ghi (rc=$rc $reason)" $?
flock -n "$data/slow/slow.lock" true; rc=$?
[ "$rc" -eq 0 ]; check "sau khi bị huỷ khoá đã được nhả (flock rc=$rc)" $?

# ---- 8: SIGKILL giữa lượt -> ABANDONED, không bị che sau báo cáo cũ (CANCELLED ở trên) ---
(cd "$t" && exec "$bin" routine run slow) >/dev/null 2>"$t/slow2.err" &
ps_=$!
sleep 6
kill -9 "$ps_" 2>/dev/null
wait "$ps_" 2>/dev/null
"$bin" routine show slow >"$t/show5.json" 2>/dev/null
reason="$(jget "$t/show5.json" last_run.reason)"
[ "$reason" = ABANDONED ]; check "SIGKILL giữa lượt -> lượt cuối ABANDONED, không hiện báo cáo CANCELLED cũ ($reason)" $?
# Tiến trình thay thế (khác danh tính) nhận lại được: khoá đã nhả khi tiến trình chết.
flock -n "$data/slow/slow.lock" true; rc=$?
[ "$rc" -eq 0 ]; check "sau SIGKILL khoá đã nhả, lượt khác chạy được (flock rc=$rc)" $?

if [ "$fail" -eq 0 ]; then echo PASS; else echo "FAILED"; fi
exit "$fail"
