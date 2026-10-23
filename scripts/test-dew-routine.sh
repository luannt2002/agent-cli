#!/usr/bin/env bash
# E2E cho `dew routine` với server OpenAI-compat giả. Chứng minh trên binary thật:
# routine chạy trong worktree riêng (repo gốc không bị đụng), bị CHẶN git push khi
# không người trực trong khi `run` cùng lệnh đó push được (ca đối chứng), khoá chống
# chạy chồng (mã 75), từ chối --scheduled cho việc có đích và định nghĩa đã đổi,
# huỷ bằng SIGTERM, `schedule --print` chỉ in, `ps` thấy lượt đang chạy, `routine gc`
# gỡ worktree sạch (giữ crush.db và logs, giữ worktree có tệp bị gitignore trừ khi
# --include-ignored; node_modules bị ignore thì dựng lại được, không giữ), mã HTTP của
# provider và model đã phân giải vào báo cáo (routine new không --model đóng băng model mặc
# định vào định nghĩa), giới hạn DEW_ROUTINE_SLOTS đọc từ tệp env, và
# bước dọn đầu lượt (trim cache Go giả tối đa 1 lần/giờ, đĩa thiếu -> 75) cùng `dew clean`,
# `routine list`/`ps` nêu trạng thái theo pid sống (ABANDONED khi đã chết), model, verify;
# DEW_ROUTINE_MIN_MEM_MB (RAM thiếu -> 75); 3 lượt khởi động cùng lúc với DEW_ROUTINE_SLOTS=2
# -> đúng 1 lượt 75; lượt kế của routine khác đóng run ma (pid đã chết) thành ABANDONED; và
# `routine export` ra patch git apply được vào clone sạch ở base; `routine run --json` in một object
# (báo cáo cuối, hoặc refused lock_held / no_slot / low_memory / low_disk với rc 75); model không có
# -> ERROR trước worktree add (không nhánh, không worktree); nhánh của worktree bị gỡ từ bên ngoài được
# gc xoá; `routine rm` từ chối lượt còn việc, --discard gỡ hết trừ báo cáo.
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

bin="${DEW_BIN:-$t/dew}"
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
        if 'HTTP400' in last:
            self.send_response(400); self.send_header('content-type', 'application/json'); self.end_headers()
            self.wfile.write(json.dumps({"error": {"message": "no such model", "type": "invalid_request_error"}}).encode())
            return
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
unset CRUSH_CLIENT_SERVER DEW_UNATTENDED DEW_UNATTENDED_COMMIT DEW_ENV_FILE DEW_ROUTINE_SLOTS DEW_GOCACHE_MAX
export GOCACHE="$t/gocache" # bước dọn đầu `routine run` không bao giờ chạm cache Go thật
export DEW_MIN_FREE_GB=0    # máy còn ít hơn 10 GB vẫn chạy được E2E; chỉ ca gác đĩa đặt giá trị
export DEW_RUNTIME_DIR="$t/run" # `ps` chỉ quét socket trong thư mục tạm, không chạm server thật
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
new_out="$(cat "$t/new.out")"; frozen="$(jget "$defs/green.json" model)"
[ "$frozen" = fake/m ] && [[ "$new_out" == *"model: fake/m (no --model: froze the current default large model"*"$defs/green.json"* ]] && [[ "$new_out" != *warning* ]]; check "routine new không --model: đóng băng model mặc định hiện tại vào định nghĩa, nêu model và chỗ đổi (model='$frozen')" $?
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
run1_err="$(cat "$t/run1.err")"; model1="$(jget "$t/show1.json" last_run.model)"
[ "$model1" = fake/m ] && [[ "$run1_err" == *"[routine] green: model fake/m (from the definition)"* ]]; check "báo cáo và stderr nêu model đã phân giải, lấy từ định nghĩa (model='$model1')" $?
[ -n "$wt" ] && [ -e "$wt/e2e-probe" ] && [ ! -e "$repo/e2e-probe" ]; check "agent làm trong worktree ($wt), không phải checkout của người dùng" $?
[ -z "$status_after" ] && [ "$head_before" = "$head_after" ] && [ -n "$branch_listed" ]; check "repo gốc nguyên vẹn (status rỗng, HEAD không đổi), nhánh $branch có mặt" $?
rmode="$(stat -c %a "$(dirname "$(jget "$t/show1.json" last_run.journal)")/report.json" 2>/dev/null)"
[ "$rmode" = 600 ]; check "báo cáo chỉ chủ đọc, nằm ngoài worktree (mode $rmode)" $?
"$bin" routine list >"$t/list1.out" 2>&1
grep -q 'green.*GREEN' "$t/list1.out"; check "routine list hiện lần chạy cuối GREEN" $?
list1="$(grep '^green ' "$t/list1.out")"
"$bin" routine list --json >"$t/list1.json" 2>&1
lj="$(python3 - "$t/list1.json" <<'PY'
import json, sys
e = [r for r in json.load(open(sys.argv[1])) if r["name"] == "green"][0]
print(e.get("state"), e.get("model"), e.get("verify"), e.get("changed_count"), e.get("attempts"), e.get("last_reason"))
PY
)"
[[ "$list1" == *"last: GREEN "*"  model fake/m  "*"  verify PASS  changed 1  attempts 1" ]] && [ "$lj" = "GREEN fake/m PASS 1 1 GREEN" ]; check "routine list (+--json): trạng thái, model, thời lượng, verify, số tệp đổi, số lượt ('$list1' json='$lj')" $?

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
# ---- 3c: provider trả HTTP 400 -> RUN_ERROR nêu mã HTTP, không có verify_exit_code ---
repo5="$(mkrepo http400)"
"$bin" routine new http400 --mode goal --repo "$repo5" --goal "HTTP400 fail on purpose" --verify 'test -f never' --max-repairs 0 >/dev/null 2>&1
(cd "$t" && "$bin" routine run http400) >/dev/null 2>"$t/h400.err"; rc=$?
"$bin" routine show http400 >"$t/show6.json" 2>/dev/null
reason="$(jget "$t/show6.json" last_run.reason)"; detail="$(jget "$t/show6.json" last_run.detail)"; vexit="$(jget "$t/show6.json" last_run.verify_exit_code)"
[ "$rc" -eq 1 ] && [ "$reason" = RUN_ERROR ] && [[ "$detail" == *"HTTP 400"* ]] && [ -z "$vexit" ]; check "provider 400 -> RUN_ERROR, detail nêu HTTP 400, không có verify_exit_code (rc=$rc $reason detail='$detail' vexit='$vexit')" $?
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
# Lượt đang chạy (in-process, không server): báo cáo RUNNING có pid, chưa có finished; ps thấy nó.
run7="$(ls -d "$data/slow/runs/"* | tail -n1)"
fin="$(jget "$run7/report.json" finished)"; rpid="$(jget "$run7/report.json" pid)"
[ -z "$fin" ] && [ "$rpid" = "$ps_" ]; check "báo cáo RUNNING: có pid của lượt, chưa có finished (pid=$rpid/$ps_ finished='$fin')" $?
"$bin" ps --json >"$t/ps.json" 2>"$t/ps.err"; rc=$?
"$bin" ps >"$t/ps.txt" 2>/dev/null
seen="$(python3 - "$t/ps.json" "$ps_" <<'PY'
import json, sys
rows = json.load(open(sys.argv[1]))
print(sum(1 for r in rows if r.get("routine") == "slow" and r.get("pid") == int(sys.argv[2]) and r.get("busy") and "/routines/slow/wt/" in r.get("path", "") and r.get("model") == "fake/m"))
PY
)"
ps_txt="$(cat "$t/ps.txt")"
[ "$rc" -eq 0 ] && [ "$seen" = 1 ] && [[ "$ps_txt" == *ROUTINE*slow*"model fake/m"* ]]; check "ps thấy routine đang chạy: một hàng ROUTINE, path là worktree, pid đúng, model (rc=$rc rows=$seen)" $?
"$bin" routine list >"$t/list7.out" 2>&1
list7="$(grep '^slow ' "$t/list7.out")"
[[ "$list7" == *"last: RUNNING (pid $ps_)  model fake/m  running "* ]]; check "routine list: lượt đang chạy là RUNNING kèm pid, model, thời lượng tới giờ ('$list7')" $?
kill -TERM "$ps_" 2>/dev/null
( sleep 25; kill -9 "$ps_" 2>/dev/null ) & wd=$!
wait "$ps_"; rc=$?
kill "$wd" 2>/dev/null; wait "$wd" 2>/dev/null
"$bin" routine show slow >"$t/show4.json" 2>/dev/null
reason="$(jget "$t/show4.json" last_run.reason)"
[ "$rc" -eq 1 ] && [ "$reason" = CANCELLED ]; check "SIGTERM giữa lượt -> CANCELLED, rc=1, báo cáo vẫn được ghi (rc=$rc $reason)" $?
fin7="$(jget "$run7/report.json" finished)"; reason7="$(jget "$run7/report.json" reason)"
[ "$reason7" = CANCELLED ] && [ -n "$fin7" ]; check "báo cáo trên đĩa của lượt bị SIGTERM: CANCELLED, có finished ($reason7 finished='$fin7')" $?
flock -n "$data/slow/slow.lock" true; rc=$?
[ "$rc" -eq 0 ]; check "sau khi bị huỷ khoá đã được nhả (flock rc=$rc)" $?

# Ghi chú agent để lại mà gitignore che (changed_files không thấy): giữ worktree của lượt CANCELLED qua
# lượt dọn tự động của lượt sau (cây sạch thì lượt mới nhất cũng bị gỡ, Q4), trừ khi --include-ignored.
wt7="$(jget "$run7/report.json" worktree)"
echo 'notes/' >>"$repo3/.git/info/exclude"
mkdir -p "$wt7/notes" && echo findings >"$wt7/notes/n.md"

# ---- 8: SIGKILL giữa lượt -> ABANDONED, không bị che sau báo cáo cũ (CANCELLED ở trên) ---
(cd "$t" && exec "$bin" routine run slow) >/dev/null 2>"$t/slow2.err" &
ps_=$!
sleep 6
kill -9 "$ps_" 2>/dev/null
wait "$ps_" 2>/dev/null
"$bin" routine show slow >"$t/show5.json" 2>/dev/null
reason="$(jget "$t/show5.json" last_run.reason)"
[ "$reason" = ABANDONED ]; check "SIGKILL giữa lượt -> lượt cuối ABANDONED, không hiện báo cáo CANCELLED cũ ($reason)" $?
"$bin" routine list >"$t/list8.out" 2>&1
list8="$(grep '^slow ' "$t/list8.out")"
[[ "$list8" == *"last: ABANDONED "*"verify not run"* ]]; check "routine list: lượt bị SIGKILL (pid đã chết) là ABANDONED, không phải RUNNING ('$list8')" $?
# Tiến trình thay thế (khác danh tính) nhận lại được: khoá đã nhả khi tiến trình chết.
flock -n "$data/slow/slow.lock" true; rc=$?
[ "$rc" -eq 0 ]; check "sau SIGKILL khoá đã nhả, lượt khác chạy được (flock rc=$rc)" $?

# ---- 9: routine gc: lượt CANCELLED (cây sạch) bị gỡ, lượt mới nhất (bị kill, sạch) cũng gỡ (Q4) ------
run8="$(ls -d "$data/slow/runs/"* | tail -n1)"
wt7="$(jget "$run7/report.json" worktree)"; br7="$(jget "$run7/report.json" branch)"; wt8="$(jget "$run8/report.json" worktree)"
vexit7="$(jget "$run7/report.json" verify_exit_code)"
[ -z "$vexit7" ]; check "lượt CANCELLED không có verify_exit_code (='$vexit7')" $?
"$bin" routine gc slow --dry-run >"$t/gc-ign.out" 2>&1; rc=$?
gc_ign="$(cat "$t/gc-ign.out")"
[ "$rc" -eq 0 ] && [[ "$gc_ign" == *"keep $wt7: 1 ignored path(s) outside .crush ("*"): notes/;"* ]]; check "gc --dry-run liệt kê mục bị gitignore ngoài .crush và giữ worktree (rc=$rc)" $?
"$bin" routine gc slow >"$t/gc-keep.out" 2>&1; rc=$?
gc_keep="$(cat "$t/gc-keep.out")"
[ "$rc" -eq 0 ] && [ -f "$wt7/notes/n.md" ]; check "gc không --include-ignored: tệp bị gitignore còn nguyên (rc=$rc)" $?
[ ! -e "$wt8" ] && [[ "$gc_keep" == *"removed $wt8 "* ]] && [[ "$gc_keep" == *"kept 1;"* ]]; check "lượt mới nhất (bị kill, cây sạch) không còn được giữ vô điều kiện: gc gỡ nó; dòng tổng đếm worktree giữ ($gc_keep)" $?
"$bin" routine gc slow --include-ignored --dry-run >"$t/gc-dry.out" 2>&1; rc=$?
gc_dry="$(cat "$t/gc-dry.out")"
[ "$rc" -eq 0 ] && [[ "$gc_dry" == *"would remove $wt7 "* ]] && [[ "$gc_dry" == *"would remove 1 worktree(s)"* ]] && [ -d "$wt7" ]; check "gc --include-ignored --dry-run in worktree sẽ gỡ và số byte, không gỡ gì (rc=$rc)" $?
"$bin" routine gc slow --include-ignored >"$t/gc.out" 2>&1; rc=$?
gc_out="$(cat "$t/gc.out")"
br7_left="$(git -C "$repo3" branch --list "$br7")"
sdb="$(jget "$run7/report.json" session_db)"; slogs="$(jget "$run7/report.json" session_logs)"
reason8="$(jget "$run8/report.json" reason)"; fin8="$(jget "$run8/report.json" finished)"
[ "$rc" -eq 0 ] && [ ! -e "$wt7" ] && [ -z "$br7_left" ] && [ -f "$run7/report.json" ] && [ -f "$run8/report.json" ]; check "gc gỡ worktree+nhánh sạch, giữ báo cáo (rc=$rc $gc_out)" $?
[ "$sdb" = "$run7/crush.db" ] && [ -s "$sdb" ] && [ "$slogs" = "$run7/logs" ] && [ -d "$slogs" ]; check "crush.db và .crush/logs chuyển ra cạnh report.json trước khi gỡ (session_db='$sdb' session_logs='$slogs')" $?
[ "$reason8" = ABANDONED ] && [ -n "$fin8" ]; check "gc giữ khoá: báo cáo RUNNING của lượt bị kill ghi lại thành ABANDONED trên đĩa ($reason8 finished='$fin8')" $?

# ---- 9b: node_modules bị ignore dựng lại được; ghi chú bị ignore thì giữ worktree ------
repo6="$(mkrepo deps)"
printf 'node_modules/\nnotes/\n' >"$repo6/.gitignore"
git -C "$repo6" add .gitignore && git -C "$repo6" commit -q -m ignore
"$bin" routine new deps --mode goal --repo "$repo6" --model m --goal "nothing to do" --verify 'test -f never' --max-repairs 0 >"$t/newdeps.out" 2>&1
deps_run() { sleep 1.1; (cd "$t" && "$bin" routine run deps) >/dev/null 2>"$t/deps.err"; }
deps_run
rund1="$(ls -d "$data/deps/runs/"* | head -n1)"; wtd1="$(jget "$rund1/report.json" worktree)"
newdeps="$(cat "$t/newdeps.out")"; modeld="$(jget "$rund1/report.json" model)"
[[ "$newdeps" != *"warning: no --model"* ]] && [ "$modeld" = fake/m ]; check "routine new --model m: không cảnh báo; báo cáo ghi model ghim đã phân giải fake/m (model='$modeld')" $?
mkdir -p "$wtd1/web/node_modules/pkg" "$wtd1/notes" && echo x >"$wtd1/web/node_modules/pkg/index.js" && echo findings >"$wtd1/notes/n.md"
deps_run
[ -d "$wtd1" ]; check "tự động: worktree có node_modules và ghi chú bị ignore được giữ" $?
rm -r "$wtd1/notes"
"$bin" routine gc deps --dry-run >"$t/gc-deps.out" 2>&1
gc_deps="$(cat "$t/gc-deps.out")"
[[ "$gc_deps" == *"would remove $wtd1 ("*"drops 1 regenerable ignored path(s)"* ]]; check "gc --dry-run: chỉ còn node_modules -> sẽ gỡ, nêu số mục dựng lại được bị bỏ" $?
deps_run
[ ! -e "$wtd1" ]; check "tự động: chỉ còn node_modules bị ignore -> worktree được gỡ" $?

# ---- 10: DEW_ROUTINE_SLOTS trong tệp env (lượt theo lịch không qua crushrc) ------------
mkdir -p "$t/home/.config/dew" "$data/slots"
echo 'DEW_ROUTINE_SLOTS=1' >"$t/home/.config/dew/env"
( exec 9>"$data/slots/0.lock"; flock -n 9 && exec sleep 30 ) &
hold_pid=$!
sleep 0.5
(cd "$t" && "$bin" routine run green) >/dev/null 2>"$t/slots.err"; rc=$?
slots_err="$(cat "$t/slots.err")"
[ "$rc" -eq 75 ] && [[ "$slots_err" == *"DEW_ROUTINE_SLOTS=1"* ]]; check "hết chỗ (DEW_ROUTINE_SLOTS=1 từ tệp env) -> rc=75, nêu tên biến (rc=$rc)" $?
(cd "$t" && DEW_ROUTINE_SLOTS=2 "$bin" routine run green) >/dev/null 2>"$t/slots2.err"; rc=$?
[ "$rc" -eq 0 ]; check "biến export thắng tệp env: DEW_ROUTINE_SLOTS=2 còn chỗ thứ hai -> chạy (rc=$rc)" $?
kill "$hold_pid" 2>/dev/null; wait "$hold_pid" 2>/dev/null; hold_pid=""

# ---- 11: dọn dẹp: trim cache Go giả đầu lượt (1 lần/giờ), đĩa thiếu -> 75, dew clean ------
# Hai mục 600 MiB là tệp thưa (không tốn đĩa); trần nhỏ nhất là 1 GB, trim về 80% = 0,8 GB.
mkcache() {
  rm -rf "$GOCACHE" && mkdir -p "$GOCACHE"
  echo 'This directory holds cached build artifacts from the Go build system.' >"$GOCACHE/README"
  local i h
  for i in 1 2; do
    h="$(printf '%s' "$i" | sha256sum | cut -c1-64)"
    mkdir -p "$GOCACHE/${h:0:2}" && truncate -s 600M "$GOCACHE/${h:0:2}/$h-d" && touch -d "$((i + 2)) days ago" "$GOCACHE/${h:0:2}/$h-d"
  done
}
entries() { find "$GOCACHE" -name '*-d' | wc -l; }
mkcache; rm -f "$data/.gocache-trim.last"
sleep 1.1
(cd "$t" && DEW_GOCACHE_MAX=1 "$bin" routine run green) >/dev/null 2>"$t/trim.err"; rc=$?
trim_err="$(cat "$t/trim.err")"; left="$(entries)"
[ "$rc" -eq 0 ] && [ "$left" = 1 ] && [[ "$trim_err" == *"[routine] Go build cache 1.3 GB was over DEW_GOCACHE_MAX (1.0 GB): removed 1 old entries"* ]]; check "cache Go vượt DEW_GOCACHE_MAX -> trim mục cũ nhất ở đầu lượt, một dòng stderr (rc=$rc còn $left mục)" $?
mkcache
sleep 1.1
(cd "$t" && DEW_GOCACHE_MAX=1 "$bin" routine run green) >/dev/null 2>"$t/trim2.err"; rc=$?
trim2_err="$(cat "$t/trim2.err")"; left="$(entries)"
[ "$rc" -eq 0 ] && [ "$left" = 2 ] && [[ "$trim2_err" != *"Go build cache"* ]]; check "lượt sau trong cùng giờ không quét lại (rc=$rc còn $left mục)" $?
branches_before="$(git -C "$repo" branch --list "routine/green-*" | wc -l)"
sleep 1.1
(cd "$t" && DEW_MIN_FREE_GB=999999 "$bin" routine run green) >/dev/null 2>"$t/disk.err"; rc=$?
disk_err="$(cat "$t/disk.err")"; branches_after="$(git -C "$repo" branch --list "routine/green-*" | wc -l)"
[ "$rc" -eq 75 ] && [[ "$disk_err" == *"dew clean"* ]] && [[ "$disk_err" == *"DEW_MIN_FREE_GB=999999"* ]] && [ "$branches_before" = "$branches_after" ]; check "đĩa còn ít hơn DEW_MIN_FREE_GB -> rc=75, nêu dew clean, không tạo nhánh (rc=$rc)" $?
(cd "$t" && DEW_GOCACHE_MAX=20GB "$bin" routine run green) >/dev/null 2>"$t/badenv.err"; rc=$?
badenv_err="$(cat "$t/badenv.err")"
(cd "$t" && DEW_MIN_FREE_GB=-1 "$bin" routine run green) >/dev/null 2>&1; rc2=$?
[ "$rc" -eq 2 ] && [ "$rc2" -eq 2 ] && [[ "$badenv_err" == *DEW_GOCACHE_MAX* ]]; check "giá trị sai của DEW_GOCACHE_MAX / DEW_MIN_FREE_GB -> rc=2 (rc=$rc/$rc2)" $?
# dew clean chỉ thấy git trên PATH: pnpm/pip/docker/go của máy không bao giờ bị gọi.
mkdir -p "$t/gitonly" && ln -sf "$(command -v git)" "$t/gitonly/git"
(cd "$t" && PATH="$t/gitonly" DEW_GOCACHE_MAX=1 "$bin" clean) >"$t/clean.out" 2>&1; rc=$?
clean_out="$(cat "$t/clean.out")"; left="$(entries)"
[ "$rc" -eq 0 ] && [ "$left" = 2 ] && [[ "$clean_out" == *"SAFE   Go build cache"*"run: dew clean --apply"* ]] && [[ "$clean_out" == *"Nothing was deleted"* ]]; check "dew clean in bảng SAFE kèm lệnh, không xoá gì (rc=$rc còn $left mục)" $?
(cd "$t" && PATH="$t/gitonly" DEW_GOCACHE_MAX=1 "$bin" clean --apply) >"$t/clean2.out" 2>&1; rc=$?
clean2_out="$(cat "$t/clean2.out")"; left="$(entries)"
[ "$rc" -eq 0 ] && [ "$left" = 1 ] && [[ "$clean2_out" == *"Go build cache: removed 1 old entries"* ]]; check "dew clean --apply trim cache Go về 80% trần (rc=$rc còn $left mục)" $?
(cd "$t" && PATH="$t/gitonly" DEW_GOCACHE_MAX=x "$bin" clean) >/dev/null 2>&1; rc=$?
[ "$rc" -eq 2 ]; check "dew clean với DEW_GOCACHE_MAX sai -> rc=2 (rc=$rc)" $?

# ---- 12: RAM tối thiểu; trần song song khi 3 lượt khởi động cùng lúc; run ma của routine khác ----
rm -f "$t/home/.config/dew/env" # bỏ DEW_ROUTINE_SLOTS=1 của ca 10
branches_before="$(git -C "$repo" branch --list "routine/green-*" | wc -l)"
(cd "$t" && DEW_ROUTINE_MIN_MEM_MB=99999999 "$bin" routine run green) >/dev/null 2>"$t/mem.err"; rc=$?
mem_err="$(tr -s '[:space:]' ' ' <"$t/mem.err")"; branches_after="$(git -C "$repo" branch --list "routine/green-*" | wc -l)"
[ "$rc" -eq 75 ] && [[ "$mem_err" == *"MB of memory available (MemAvailable), DEW_ROUTINE_MIN_MEM_MB=99999999"* ]] && [ "$branches_before" = "$branches_after" ]; check "RAM trống ít hơn DEW_ROUTINE_MIN_MEM_MB -> rc=75, nêu số đo và tên biến, không tạo nhánh (rc=$rc)" $?
(cd "$t" && DEW_ROUTINE_MIN_MEM_MB=3GB "$bin" routine run green) >/dev/null 2>"$t/mem2.err"; rc=$?
mem2_err="$(cat "$t/mem2.err")"
[ "$rc" -eq 2 ] && [[ "$mem2_err" == *DEW_ROUTINE_MIN_MEM_MB* ]]; check "DEW_ROUTINE_MIN_MEM_MB sai -> rc=2 (rc=$rc)" $?

declare -A par
for n in par1 par2 par3; do
  r="$(mkrepo "$n")"
  "$bin" routine new "$n" --mode goal --repo "$r" --goal "SLOW wait" --verify 'test -f never' --max-repairs 0 >/dev/null 2>&1
done
for n in par1 par2 par3; do
  (cd "$t" && DEW_ROUTINE_SLOTS=2 exec "$bin" routine run "$n") >/dev/null 2>"$t/$n.err" &
  par[$n]=$!
done
refused=""
for _ in $(seq 150); do
  for n in par1 par2 par3; do kill -0 "${par[$n]}" 2>/dev/null || refused="$n"; done
  [ -n "$refused" ] && break
  sleep 0.1
done
live=()
for n in par1 par2 par3; do [ "$n" != "$refused" ] && live+=("$n"); done
rc=-1; [ -n "$refused" ] && { wait "${par[$refused]}"; rc=$?; }
# fang ngắt dòng thông báo lỗi dài (và viết hoa chữ đầu): gộp khoảng trắng trước khi so.
refused_err="$(tr -s '[:space:]' ' ' <"$t/${refused:-none}.err" 2>/dev/null)"
alive=0; for n in "${live[@]}"; do kill -0 "${par[$n]}" 2>/dev/null && alive=$((alive + 1)); done
[ "$rc" -eq 75 ] && [[ "$refused_err" == *"2 routine slots are in use (DEW_ROUTINE_SLOTS=2"* ]] && [ "$alive" = 2 ]; check "3 lượt khởi động cùng lúc, DEW_ROUTINE_SLOTS=2 -> đúng 1 lượt thoát 75, 2 lượt chạy (refused=$refused rc=$rc alive=$alive err='$refused_err')" $?

# Lượt ma: SIGKILL một lượt đang chạy; lượt `routine run` kế tiếp của routine KHÁC đóng nó thành
# ABANDONED, lượt còn sống thì không đụng.
ghost="${live[0]}"; keep="${live[1]}"
for _ in $(seq 100); do
  ok=0
  for n in "$ghost" "$keep"; do
    f="$(ls -d "$data/$n/runs/"* 2>/dev/null | tail -n1)/report.json"
    [ -f "$f" ] && [ "$(jget "$f" pid)" = "${par[$n]}" ] && ok=$((ok + 1))
  done
  [ "$ok" = 2 ] && break
  sleep 0.1
done
ghost_report="$(ls -d "$data/$ghost/runs/"* | tail -n1)/report.json"; keep_report="$(ls -d "$data/$keep/runs/"* | tail -n1)/report.json"
kill -9 "${par[$ghost]}"; wait "${par[$ghost]}" 2>/dev/null
before_ghost="$(jget "$ghost_report" reason)"
(cd "$t" && DEW_ROUTINE_SLOTS=2 "$bin" routine run green) >/dev/null 2>"$t/ghost.err"; rc=$?
ghost_err="$(cat "$t/ghost.err")"; after_ghost="$(jget "$ghost_report" reason)"; fin_ghost="$(jget "$ghost_report" finished)"; after_keep="$(jget "$keep_report" reason)"
[ "$rc" -eq 0 ] && [ "$before_ghost" = RUNNING ] && [ "$after_ghost" = ABANDONED ] && [ -n "$fin_ghost" ] && [ "$after_keep" = RUNNING ] && [[ "$ghost_err" == *"closed $(dirname "$ghost_report") as ABANDONED"* ]]; check "lượt kế của routine khác đóng run ma thành ABANDONED (có finished), lượt còn sống vẫn RUNNING (rc=$rc $ghost: $before_ghost->$after_ghost, $keep: $after_keep)" $?
"$bin" routine list >"$t/list12.out" 2>&1
list_keep="$(grep "^$keep " "$t/list12.out")"; list_ghost="$(grep "^$ghost " "$t/list12.out")"
[[ "$list_keep" == *"last: RUNNING (pid ${par[$keep]})"* ]] && [[ "$list_ghost" == *"last: ABANDONED "* ]]; check "routine list: lượt sống RUNNING kèm pid, lượt ma ABANDONED" $?
kill -TERM "${par[$keep]}" 2>/dev/null
( sleep 25; kill -9 "${par[$keep]}" 2>/dev/null ) & wd=$!
wait "${par[$keep]}" 2>/dev/null
kill "$wd" 2>/dev/null; wait "$wd" 2>/dev/null

# ---- 13: routine export: một patch (sửa + tệp mới; tệp bị ignore chỉ khi --include-ignored) ------
"$bin" routine show green >"$t/show13.json" 2>/dev/null
wt13="$(jget "$t/show13.json" last_run.worktree)"; base13="$(jget "$t/show13.json" last_run.base_sha)"
echo '*.secret' >>"$repo/.git/info/exclude"
echo changed >>"$wt13/crush.json"
echo token >"$wt13/x.secret"
st_before="$(git -C "$wt13" status --porcelain)"
"$bin" routine export green -o "$t/green.patch" >"$t/export.out" 2>"$t/export.err"; rc=$?
export_out="$(cat "$t/export.out")"; st_after="$(git -C "$wt13" status --porcelain)"
sum13="$(sha256sum "$t/green.patch" | cut -c1-64)"
[ "$rc" -eq 0 ] && [[ "$export_out" == *$'M\tcrush.json'* ]] && [[ "$export_out" == *$'A\te2e-probe'* ]] && [[ "$export_out" != *x.secret* ]] && [[ "$export_out" == *"$sum13  $t/green.patch"* ]] && [ "$st_before" = "$st_after" ]; check "routine export -o: liệt kê tệp sửa + tệp mới (không có tệp bị ignore), in sha256 đúng, không đổi status của worktree (rc=$rc)" $?
treesum() { (cd "$1" && git ls-files -z -co --exclude-standard | xargs -0 -r sha256sum 2>/dev/null | sort); }
git clone -q "$repo" "$t/clean13" && git -C "$t/clean13" checkout -q --detach "$base13"
git -C "$t/clean13" apply "$t/green.patch"; arc=$?
want13="$(treesum "$wt13")"; got13="$(treesum "$t/clean13")"
[ "$arc" -eq 0 ] && [ -n "$want13" ] && [ "$want13" = "$got13" ] && [ ! -e "$t/clean13/x.secret" ]; check "patch git apply được vào clone sạch ở base, ra đúng cây của worktree, không có tệp bị ignore (apply rc=$arc)" $?
"$bin" routine export green --include-ignored >"$t/green-ign.patch" 2>"$t/export-ign.err"; rc=$?
git clone -q "$repo" "$t/clean13b" && git -C "$t/clean13b" checkout -q --detach "$base13"
git -C "$t/clean13b" apply "$t/green-ign.patch"; arc=$?
got13b="$(treesum "$t/clean13b" | grep -v ' x.secret$')"; secret="$(cat "$t/clean13b/x.secret" 2>/dev/null)"
export_ign_err="$(cat "$t/export-ign.err")"
[ "$rc" -eq 0 ] && [ "$arc" -eq 0 ] && [ "$secret" = token ] && [ "$want13" = "$got13b" ] && [[ "$export_ign_err" == *$'A\tx.secret'* ]] && [ ! -e "$t/clean13b/.crush" ]; check "--include-ignored (patch ra stdout, danh sách ra stderr): có cả tệp bị ignore, không có .crush (rc=$rc apply rc=$arc)" $?
stamp7="$(basename "$run7")"
"$bin" routine export slow --run "$stamp7" >/dev/null 2>"$t/export-gone.err"; rc=$?
gone_err="$(cat "$t/export-gone.err")"
[ "$rc" -eq 1 ] && [[ "$gone_err" == *"is gone"* ]] && [[ "$gone_err" == *"$run7/crush.db"* ]]; check "export lượt đã bị gc gỡ worktree -> rc=1, nói rõ, chỉ chỗ crush.db còn giữ (rc=$rc)" $?
# report.json bị sửa: base_sha dạng cờ không bao giờ tới git (git nhận --output và ghi đè tệp).
run13="$(dirname "$(jget "$t/show13.json" last_run.journal)")"
python3 - "$run13/report.json" "--output=$t/pwned" <<'PY'
import json, sys
p = sys.argv[1]; d = json.load(open(p)); d["base_sha"] = sys.argv[2]
json.dump(d, open(p, "w"), indent=2)
PY
"$bin" routine export green -o "$t/tampered.patch" >/dev/null 2>"$t/export-tamper.err"; rc=$?
tamper_err="$(tr -s '[:space:]' ' ' <"$t/export-tamper.err")"
[ "$rc" -eq 1 ] && [[ "$tamper_err" == *"base_sha"* ]] && [ ! -e "$t/pwned" ] && [ ! -e "$t/tampered.patch" ]; check "report.json có base_sha dạng cờ -> rc=1, từ chối, không tệp nào bị ghi (rc=$rc)" $?

# ---- 14: routine run --json: MỘT object trên stdout (báo cáo cuối, hoặc loại từ chối, rc 75) ----
# jone <tệp>: "refused routine reason" của đúng MỘT object JSON trong tệp; "BAD ..." nếu không phải.
jone() {
  python3 - "$1" <<'PY'
import json, sys
try:
    o = json.loads(open(sys.argv[1]).read())
except Exception as e:
    print("BAD", e); sys.exit()
print(o.get("refused", "-"), o.get("routine", "-"), o.get("reason", "-"))
PY
}
sleep 1.1
(cd "$t" && "$bin" routine run green --json) >"$t/j-green.out" 2>/dev/null; rc=$?
jg="$(jone "$t/j-green.out")"; jg_wt="$(jget "$t/j-green.out" worktree)"
jg_disk="$(jget "$(ls -d "$data/green/runs/"* | tail -n1)/report.json" worktree)"
[ "$rc" -eq 0 ] && [ "$jg" = "- green GREEN" ] && [ -n "$jg_wt" ] && [ "$jg_wt" = "$jg_disk" ]; check "routine run --json xong: một object là báo cáo cuối, khớp report.json (rc=$rc '$jg')" $?
# Hai lượt cùng routine: lượt đang chạy giữ khoá, lượt thứ hai từ chối lock_held.
(cd "$t" && DEW_ROUTINE_SLOTS=1 exec "$bin" routine run slow) >/dev/null 2>"$t/j-slow.err" &
jslow=$!
for _ in $(seq 100); do
  f="$(ls -d "$data/slow/runs/"* 2>/dev/null | tail -n1)/report.json"
  [ -f "$f" ] && [ "$(jget "$f" pid)" = "$jslow" ] && break
  sleep 0.1
done
(cd "$t" && "$bin" routine run slow --json) >"$t/j-lock.out" 2>/dev/null; rc=$?
jl="$(jone "$t/j-lock.out")"; jl_detail="$(jget "$t/j-lock.out" detail)"
[ "$rc" -eq 75 ] && [ "$jl" = "lock_held slow -" ] && [[ "$jl_detail" == *"already running"* ]]; check "hai lượt cùng routine: lượt thứ hai rc=75, --json in refused lock_held (rc=$rc '$jl')" $?
# DEW_ROUTINE_SLOTS=1, routine khác đang giữ chỗ duy nhất: no_slot.
(cd "$t" && DEW_ROUTINE_SLOTS=1 "$bin" routine run green --json) >"$t/j-slot.out" 2>/dev/null; rc=$?
js="$(jone "$t/j-slot.out")"
[ "$rc" -eq 75 ] && [ "$js" = "no_slot green -" ]; check "DEW_ROUTINE_SLOTS=1 với hai routine: routine thứ hai rc=75, --json in refused no_slot (rc=$rc '$js')" $?
kill -TERM "$jslow" 2>/dev/null
( sleep 25; kill -9 "$jslow" 2>/dev/null ) & wd=$!
wait "$jslow" 2>/dev/null
kill "$wd" 2>/dev/null; wait "$wd" 2>/dev/null
(cd "$t" && DEW_ROUTINE_MIN_MEM_MB=99999999 "$bin" routine run green --json) >"$t/j-mem.out" 2>/dev/null; rc=$?
(cd "$t" && DEW_MIN_FREE_GB=999999 "$bin" routine run green --json) >"$t/j-disk.out" 2>/dev/null; rc2=$?
jm="$(jone "$t/j-mem.out")"; jd="$(jone "$t/j-disk.out")"
[ "$rc" -eq 75 ] && [ "$rc2" -eq 75 ] && [ "$jm" = "low_memory green -" ] && [ "$jd" = "low_disk green -" ]; check "RAM/đĩa thiếu: rc=75, --json in refused low_memory / low_disk (rc=$rc/$rc2 '$jm' '$jd')" $?

# ---- 15: kiểm cấu hình trước worktree add; nhánh của worktree bị gỡ từ bên ngoài; routine rm -------
repo15="$(mkrepo life)"
"$bin" routine new nomodel --mode goal --repo "$repo15" --model fake/nope --goal "nothing to do" --verify 'test -f never' --max-repairs 0 >/dev/null 2>&1
(cd "$t" && "$bin" routine run nomodel) >/dev/null 2>"$t/nomodel.err"; rc=$?
"$bin" routine show nomodel >"$t/show15.json" 2>/dev/null
reason="$(jget "$t/show15.json" last_run.reason)"; detail="$(jget "$t/show15.json" last_run.detail)"
nm_branches="$(git -C "$repo15" branch --list 'routine/*')"; nm_wt="$(ls "$data/nomodel/wt" 2>/dev/null)"
[ "$rc" -eq 1 ] && [ "$reason" = ERROR ] && [[ "$detail" == *'large model "fake/nope" not found'* ]] && [ -z "$nm_branches" ] && [ -z "$nm_wt" ]; check "model không tồn tại -> ERROR trước worktree add: không nhánh, không worktree (rc=$rc $reason detail='$detail')" $?

"$bin" routine new life --mode goal --repo "$repo15" --goal "nothing to do" --verify 'test -f never' --max-repairs 0 >/dev/null 2>&1
(cd "$t" && "$bin" routine run life) >/dev/null 2>&1
run15="$(ls -d "$data/life/runs/"* | tail -n1)"; wt15="$(jget "$run15/report.json" worktree)"; br15="$(jget "$run15/report.json" branch)"
rm -rf "$wt15" # gỡ từ bên ngoài: mục quản trị trong .git/worktrees và nhánh còn lại
"$bin" routine gc life --dry-run >"$t/gc15d.out" 2>&1
gc15d="$(cat "$t/gc15d.out")"; br15_dry="$(git -C "$repo15" branch --list "$br15")"
[[ "$gc15d" == *"would delete branch $br15"* ]] && [ -n "$br15_dry" ]; check "gc --dry-run: nhánh của worktree đã bị gỡ từ bên ngoài sẽ được xoá, chưa xoá gì ('$gc15d')" $?
"$bin" routine gc life >"$t/gc15.out" 2>&1; rc=$?
gc15="$(cat "$t/gc15.out")"; br15_left="$(git -C "$repo15" branch --list "$br15")"
wt15_listed="$(git -C "$repo15" worktree list --porcelain | grep -c "^worktree $wt15\$")"; gone15="$(jget "$run15/report.json" worktree_gone)"
[ "$rc" -eq 0 ] && [[ "$gc15" == *"deleted branch $br15"* ]] && [ -z "$br15_left" ] && [ "$wt15_listed" = 0 ] && [ -n "$gone15" ]; check "gc xoá nhánh (không commit ngoài base) và mục quản trị của worktree đã mất, ghi worktree_gone (rc=$rc listed=$wt15_listed)" $?

# Worktree đã mất mà HEAD của nó là commit tách rời (người duyệt checkout --detach rồi commit): mục quản trị
# là thứ duy nhất còn giữ commit đó, gc không gỡ nó; nhánh của lượt (không commit ngoài base) vẫn xoá.
sleep 1.1
(cd "$t" && "$bin" routine run life) >/dev/null 2>&1
rund="$(ls -d "$data/life/runs/"* | tail -n1)"; wtd="$(jget "$rund/report.json" worktree)"; brd="$(jget "$rund/report.json" branch)"
git -C "$wtd" checkout -q --detach && echo d >"$wtd/d.txt" && git -C "$wtd" add d.txt && git -C "$wtd" commit -q -m detached
shad="$(git -C "$wtd" rev-parse HEAD)"
rm -rf "$wtd"
"$bin" routine gc life >"$t/gc15h.out" 2>&1; rc=$?
gc15h="$(cat "$t/gc15h.out")"; reach="$(git -C "$repo15" rev-list --all | grep -c "^$shad\$")"; brd_left="$(git -C "$repo15" branch --list "$brd")"
[ "$rc" -eq 0 ] && [ "$reach" = 1 ] && [ -z "$brd_left" ] && [[ "$gc15h" == *"its worktree entry"*"stays"* ]]; check "worktree đã mất có commit trên HEAD tách rời: gc giữ mục quản trị (commit còn với tới), vẫn xoá nhánh rỗng (rc=$rc reach=$reach)" $?

sleep 1.1
(cd "$t" && "$bin" routine run life) >/dev/null 2>&1 # lượt sạch
sleep 1.1
python3 - "$defs/life.json" <<'PY2'
import json, sys
p = sys.argv[1]; d = json.load(open(p))
d["goal"] = "TOOLCALL create the probe file"
json.dump(d, open(p, "w"), indent=2)
PY2
(cd "$t" && "$bin" routine run life) >/dev/null 2>&1 # lượt để lại e2e-probe (việc chưa commit)
run15w="$(ls -d "$data/life/runs/"* | tail -n1)"; wt15w="$(jget "$run15w/report.json" worktree)"
"$bin" routine rm life >"$t/rm1.out" 2>&1; rc=$?
rm1="$(cat "$t/rm1.out")"
[ "$rc" -eq 1 ] && [[ "$rm1" == *"holds work: $wt15w: uncommitted changes"* ]] && [[ "$rm1" == *"dew routine export life --run $(basename "$run15w")"* ]] && [ -d "$wt15w" ] && [ -f "$defs/life.json" ]; check "routine rm: lượt còn việc -> rc=1, liệt kê, gợi ý export, không gỡ gì (rc=$rc)" $?
"$bin" routine rm life --discard >"$t/rm2.out" 2>&1; rc=$?
rm2="$(cat "$t/rm2.out")"; life_branches="$(git -C "$repo15" branch --list 'routine/life-*')"; life_wts="$(ls "$data/life/wt" 2>/dev/null)"
nreports="$(ls "$data/life/runs/"*/report.json | wc -l)"
[ "$rc" -eq 0 ] && [ ! -e "$defs/life.json" ] && [ -z "$life_branches" ] && [ -z "$life_wts" ] && [ "$nreports" -ge 3 ] && [[ "$rm2" == *"removed routine life"* ]]; check "routine rm --discard: gỡ worktree, nhánh và định nghĩa; giữ $nreports báo cáo (rc=$rc)" $?

if [ "$fail" -eq 0 ]; then echo PASS; else echo "FAILED"; fi
exit "$fail"
