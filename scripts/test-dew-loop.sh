#!/usr/bin/env bash
# E2E cho `dew loop` với server OpenAI-compat giả, cả đường local lẫn
# client/server. Chứng minh: xanh khi lệnh verify thoát 0, STALLED / MAX_REPAIRS /
# UNMEASURED / CANCELLED đúng lý do và đúng số lượt agent, thiếu --verify thoát 2;
# đường local ghi mỗi request LLM thành dòng llm_request mà `routine stats` tổng hợp được.
# Mọi thứ nằm trong thư mục tạm; không đụng repo hay cấu hình thật.
# Dùng: scripts/test-dew-loop.sh   (DEW_BIN=... để bỏ qua build)
set -uo pipefail
here="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
t="$(mktemp -d /tmp/aclloop.XXXXXX)"
fake_pid=""
bin=""
# server_pids: PID server dew của lượt test này (lọc theo exe và socket,
# không dùng pgrep -f vì mẫu khớp cả dòng lệnh của chính shell đang chạy).
server_pids() {
  local want name p
  [[ -n "$bin" ]] || return 0
  want="$(readlink -f -- "$bin" 2>/dev/null)" || return 0
  [[ -n "$want" ]] || return 0
  name="$(basename -- "$want")"; name="${name:0:15}"
  for p in $(pgrep -x -- "$name" 2>/dev/null); do
    [[ "$(readlink -- "/proc/$p/exe" 2>/dev/null)" == "$want" ]] || continue
    tr '\0' '\n' <"/proc/$p/cmdline" 2>/dev/null | grep -qF -- "$t/s.sock" && echo "$p"
  done
}
cleanup() {
  [ -n "$fake_pid" ] && kill "$fake_pid" 2>/dev/null
  local p; for p in $(server_pids); do kill "$p" 2>/dev/null; done
  [ -n "${KEEP_TMP:-}" ] && echo "kept $t" || rm -rf "$t"
}
trap cleanup EXIT

fail=0
# Không đặt $(...) trong tên check: nó ghi đè $? trước khi check đọc (đọc quan sát
# vào biến TRƯỚC điều kiện). Đã mắc lỗi này một lần: check xanh giả với binary hỏng.
check() { if [ "$2" -eq 0 ]; then echo "ok   $1"; else echo "FAIL $1"; fail=1; fi; }
# jget <file> <key>: in giá trị key; lỗi nếu stdout không đúng 1 object JSON.
jget() {
  python3 - "$1" "$2" <<'PY'
import json, sys
raw = open(sys.argv[1]).read()
obj, end = json.JSONDecoder().raw_decode(raw)
if raw[end:].strip(): sys.exit("stdout holds more than one JSON object")
print(obj.get(sys.argv[2], ""))
PY
}
# last_event <journal> <key>: giá trị key của dòng journal cuối.
last_event() { tail -1 "$1" | python3 -c 'import json,sys; print(json.load(sys.stdin).get(sys.argv[1], ""))' "$2"; }

bin="${DEW_BIN:-$t/dew}"
if [ -z "${DEW_BIN:-}" ]; then
  (cd "$here" && GOMAXPROCS=2 go build -p 2 -o "$bin" .) || { echo "build failed"; exit 1; }
fi

# --- server giả: TOOLCALL -> gọi tool bash tạo e2e-probe; SLOW -> chờ 30s -----
cat > "$t/fake.py" <<'PY'
import http.server, json, sys, time
class H(http.server.BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers.get('content-length', 0))) or b'{}')
        users = [m for m in body.get('messages', []) if m.get('role') == 'user']
        last = users[-1]['content']
        if isinstance(last, list): last = ''.join(p.get('text', '') for p in last)
        if body.get('stream') and not last.startswith("Generate a concise title"):
            open(sys.argv[2], 'a').write("turn\n")
        if 'SLOW' in last and body.get('stream'): time.sleep(30)
        msgs = body.get('messages', [])
        text = "ECHO"
        if msgs and msgs[-1].get('role') == 'tool': text = "TOOLDONE"
        usage = {"prompt_tokens": 11, "completion_tokens": 7, "total_tokens": 18}
        c = lambda d, **k: {"id": "1", "object": "chat.completion.chunk", "model": "m", **k}
        if body.get('stream'):
            self.send_response(200); self.send_header('content-type', 'text/event-stream'); self.end_headers()
            # DELAY: độ trễ biết trước tới byte đầu, 600ms cho request sau kết quả tool, 300ms còn lại.
            if 'DELAY' in last: time.sleep(0.6 if msgs and msgs[-1].get('role') == 'tool' else 0.3)
            ev = lambda d: self.wfile.write(("data: " + json.dumps(d) + "\n\n").encode())
            if 'TOOLCALL' in last and msgs[-1].get('role') != 'tool':
                call = {"index": 0, "id": "call_1", "type": "function", "function": {"name": "bash", "arguments": json.dumps({"command": "touch e2e-probe", "description": "probe"})}}
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
python3 "$t/fake.py" "$t/port" "$t/requests.log" >/dev/null 2>&1 &
fake_pid=$!
for _ in $(seq 50); do [ -s "$t/port" ] && break; sleep 0.1; done
[ -s "$t/port" ] || { echo "fake server did not start"; exit 1; }
: > "$t/requests.log"

# --- môi trường cô lập ------------------------------------------------------
mkdir -p "$t/home"
export HOME="$t/home" XDG_CONFIG_HOME="$t/home/c" XDG_DATA_HOME="$t/home/d" XDG_CACHE_HOME="$t/home/x"
export XDG_RUNTIME_DIR="$t/run"; mkdir -p "$t/run" && chmod 700 "$t/run"
export CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_SERVER_IDLE_TIMEOUT=1
cat > "$t/crush.json" <<JSON
{"providers":{"fake":{"type":"openai-compat","base_url":"http://127.0.0.1:$(cat "$t/port")/v1","api_key":"x","models":[{"id":"m","name":"m","context_window":100000,"default_max_tokens":1000}]}},
 "models":{"large":{"provider":"fake","model":"m"},"small":{"provider":"fake","model":"m"}},
 "options":{"disable_provider_auto_update":true,"disable_metrics":true}}
JSON

# mkrepo <tên>: repo git sạch (crush.json đã commit) để dấu vân tay cây ổn định.
mkrepo() {
  local d="$t/$1"; mkdir -p "$d"; cp "$t/crush.json" "$d/"
  (cd "$d" && git init -q && git config user.email t@t && git config user.name t && git add -A && git commit -q -m init)
  echo "$d"
}
turns() { wc -l < "$t/requests.log"; }

run_suite() { # run_suite <tên> [cờ toàn cục...]
  local mode="$1"; shift
  local o="$t/out-$mode"; mkdir -p "$o"
  local d rc before after reason attempts jp jkind jreason jmode

  # A: agent tạo tệp, verify đỏ -> xanh sau lượt đầu.
  d="$(mkrepo "$mode-green")"
  (cd "$d" && "$bin" "$@" loop --json --name e2e --verify 'test -f e2e-probe' 'TOOLCALL create the probe file') >"$o/a.json" 2>"$o/a.err"; rc=$?
  reason="$(jget "$o/a.json" reason)"; attempts="$(jget "$o/a.json" attempts)"
  [ "$rc" -eq 0 ] && [ "$reason" = GREEN ] && [ "$attempts" = 1 ] && [ -e "$d/e2e-probe" ]; check "$mode: verify xanh sau lượt đầu -> GREEN, rc=0, tệp được tạo (rc=$rc $reason $attempts)" $?
  jp="$(jget "$o/a.json" journal)"; jkind="$(last_event "$jp" kind)"; jreason="$(last_event "$jp" reason)"
  [ -s "$jp" ] && [ "$jkind" = stop ] && [ "$jreason" = GREEN ]; check "$mode: journal có dòng stop GREEN ($jkind $jreason)" $?
  jmode="$(stat -c %a "$jp")"
  [ "$jmode" = 600 ]; check "$mode: journal chỉ chủ đọc, quan sát $jmode" $?

  # B: agent không đổi gì, verify đỏ giống hệt -> STALLED ở lượt thứ hai.
  d="$(mkrepo "$mode-stalled")"
  (cd "$d" && "$bin" "$@" loop --json --verify 'echo "FAIL TestX 12ms"; test -f never' 'do nothing useful') >"$o/b.json" 2>/dev/null; rc=$?
  reason="$(jget "$o/b.json" reason)"; attempts="$(jget "$o/b.json" attempts)"
  [ "$rc" -eq 1 ] && [ "$reason" = STALLED ] && [ "$attempts" = 2 ]; check "$mode: không tiến triển -> STALLED sau 2 lượt, rc=1 (rc=$rc $reason $attempts)" $?

  # C: --max-repairs 0 chỉ chạy đúng một lượt.
  d="$(mkrepo "$mode-max0")"
  (cd "$d" && "$bin" "$@" loop --json --max-repairs 0 --verify 'test -f never' 'x') >"$o/c.json" 2>/dev/null; rc=$?
  reason="$(jget "$o/c.json" reason)"; attempts="$(jget "$o/c.json" attempts)"
  [ "$rc" -eq 1 ] && [ "$reason" = MAX_REPAIRS ] && [ "$attempts" = 1 ]; check "$mode: --max-repairs 0 -> MAX_REPAIRS sau đúng 1 lượt (rc=$rc $reason $attempts)" $?

  # D: verify không chạy được = CHƯA ĐO, không đốt lượt agent nào.
  d="$(mkrepo "$mode-unmeasured")"
  before="$(turns)"
  (cd "$d" && "$bin" "$@" loop --json --verify 'no_such_command_xyz' 'x') >"$o/d.json" 2>/dev/null; rc=$?
  reason="$(jget "$o/d.json" reason)"; attempts="$(jget "$o/d.json" attempts)"; after="$(turns)"
  [ "$rc" -eq 1 ] && [ "$reason" = UNMEASURED ] && [ "$attempts" = 0 ] && [ "$after" = "$before" ]; check "$mode: verify không chạy được -> UNMEASURED, 0 lượt agent (rc=$rc $reason, lượt: $before -> $after)" $?

  # E: thiếu --verify là lỗi dùng lệnh: rc=2, stdout rỗng.
  d="$(mkrepo "$mode-usage")"
  (cd "$d" && "$bin" "$@" loop --json 'x') >"$o/e.out" 2>/dev/null; rc=$?
  [ "$rc" -eq 2 ] && [ ! -s "$o/e.out" ]; check "$mode: thiếu --verify -> rc=2, stdout rỗng (rc=$rc)" $?

  # F: SIGTERM giữa lượt agent -> CANCELLED, vẫn in JSON.
  d="$(mkrepo "$mode-cancel")"
  (cd "$d" && exec "$bin" "$@" loop --json --verify 'test -f never' 'SLOW wait') >"$o/f.json" 2>/dev/null &
  local pf=$!
  sleep 5
  kill -TERM "$pf" 2>/dev/null
  ( sleep 20; kill -9 "$pf" 2>/dev/null ) & local wd=$!
  wait "$pf"; rc=$?
  kill "$wd" 2>/dev/null; wait "$wd" 2>/dev/null
  reason="$(jget "$o/f.json" reason)"
  [ "$rc" -eq 1 ] && [ "$reason" = CANCELLED ]; check "$mode: SIGTERM giữa lượt -> CANCELLED + JSON, rc=1 (rc=$rc $reason)" $?
}

unset CRUSH_CLIENT_SERVER
run_suite local

# H (local): mỗi request LLM của lượt là một dòng llm_request hợp lệ trong journal, và
# `routine stats` đọc journal đó ra TTFT p50/p95 khớp độ trễ biết trước của server giả: 300ms
# (request gọi tool, tiêu đề nếu có) và 600ms (request sau kết quả tool); dung sai +400ms. Báo
# cáo routine viết tay, trỏ vào journal thật của lần chạy loop này.
d="$(mkrepo local-delay)"
(cd "$d" && "$bin" loop --json --name e2e-delay --verify 'test -f e2e-probe' 'TOOLCALL DELAY create the probe file') >"$t/h.json" 2>"$t/h.err"; rc=$?
jp="$(jget "$t/h.json" journal)"
python3 - "$jp" >"$t/h.check" 2>&1 <<'PY'
import json, sys
reqs = [e for e in map(json.loads, open(sys.argv[1])) if e["kind"] == "llm_request"]
assert len(reqs) >= 2, reqs
assert all(e["status"] == "ok" and e["attempt"] == 1 and e.get("prompt_tokens") == 11 and e.get("completion_tokens") == 7 for e in reqs), reqs
print(len(reqs), "requests")
PY
rc_j=$?
hcheck="$(tail -1 "$t/h.check")"
[ "$rc" -eq 0 ] && [ "$rc_j" -eq 0 ]; check "local: journal JSONL hợp lệ, mỗi request LLM một dòng llm_request ok + token (rc=$rc; $hcheck)" $?
run="$t/home/d/crush/routines/e2e-delay/runs/20261021-000000"; mkdir -p "$run"
python3 - "$jp" "$run/report.json" <<'PY'
import json, sys
ev = [json.loads(l) for l in open(sys.argv[1])]
json.dump({"routine": "e2e-delay", "started": ev[0]["time"], "finished": ev[-1]["time"], "reason": "GREEN", "journal": sys.argv[1]}, open(sys.argv[2], "w"))
PY
"$bin" routine stats e2e-delay --json >"$t/h-stats.json" 2>"$t/h-stats.err"; rc=$?
stats="$(python3 -c 'import json,sys; s=json.load(open(sys.argv[1]))[0]; print(s["requests"], s["ttft_p50_ms"], s["ttft_p95_ms"], s["total_p50_ms"], s["total_p95_ms"])' "$t/h-stats.json" 2>&1)"
read -r n p50 p95 tp50 tp95 <<<"$stats"
[ "$rc" -eq 0 ] && [ "${n:-0}" -ge 2 ] && [ "${p50:-0}" -ge 300 ] && [ "$p50" -lt 700 ] && [ "${p95:-0}" -ge 600 ] && [ "$p95" -lt 1000 ] && [ "${tp95:-0}" -ge "$p95" ]
check "local: routine stats ra TTFT p50 ∈ [300,700) ms, p95 ∈ [600,1000) ms từ độ trễ đã biết (rc=$rc: requests ttft_p50 ttft_p95 total_p50 total_p95 = $stats)" $?
"$bin" routine stats >"$t/h-stats.txt" 2>&1; cat "$t/h-stats.txt"
export CRUSH_CLIENT_SERVER=1
run_suite clientserver --host "unix://$t/s.sock"

# G (client/server): mỗi lượt đóng luồng SSE của nó khi xong, và server gỡ workspace
# khi client không còn luồng nào quá 10s. Verify chạy 12s giữa hai lượt thì lượt sửa
# vẫn phải chạy được, không RUN_ERROR "workspace not found". Verify tự đếm số lần
# chạy: lần 3 (sau lượt sửa) mới xanh.
d="$(mkrepo clientserver-slowverify)"
(cd "$d" && "$bin" --host "unix://$t/s.sock" loop --json --max-repairs 1 \
  --verify 'sleep 12; n=$(( $(cat .vcount 2>/dev/null || echo 0) + 1 )); echo "$n" > .vcount; [ "$n" -ge 3 ]' 'x') >"$t/g.json" 2>"$t/g.err"; rc=$?
reason="$(jget "$t/g.json" reason)"; attempts="$(jget "$t/g.json" attempts)"; detail="$(jget "$t/g.json" detail)"
[ "$rc" -eq 0 ] && [ "$reason" = GREEN ] && [ "$attempts" = 2 ]; check "clientserver: verify 12s giữa hai lượt -> lượt sửa vẫn chạy, GREEN sau 2 lượt (rc=$rc $reason $attempts $detail)" $?

for _ in $(seq 30); do [ -z "$(server_pids)" ] && break; sleep 0.5; done
[ -z "$(server_pids)" ]; check "không còn server process" $?

if [ "$fail" -eq 0 ]; then echo PASS; else echo "FAILED"; fi
exit "$fail"
