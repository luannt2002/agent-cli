#!/usr/bin/env bash
# P0-2b mục 7 (e2e): verify của `routine run` là tiến trình con của lượt không người trực, phải thấy
# AGENT_CLI_UNATTENDED=1 và AGENT_CLI_UNATTENDED_COMMIT = 0 (định nghĩa không --commit) / 1 (có --commit).
# Đối chứng: `loop --verify` (không người trực tắt) trong repo có `.env` đặt cả hai = 1 -> verify thấy unset.
# Binary thật, LLM giả (OpenAI-compat tối giản, chỉ trả lời chữ), HOME/XDG cô lập. Chỉ in giá trị của 2 biến này.
# Dùng: bash p02b-routine-unattended.sh <thư-mục-mới> <binary>
set -u
[ $# -eq 2 ] || { echo "usage: $0 <new-dir> <binary>" >&2; exit 2; }
t=$1; bin=$2
[ -e "$t" ] && { echo "exists: $t" >&2; exit 2; }
mkdir -p "$t/home" "$t/run" && chmod 700 "$t/run" && t=$(cd "$t" && pwd)
cat > "$t/fake.py" <<'PY'
import http.server, json, sys
class H(http.server.BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers.get('content-length', 0))) or b'{}')
        usage = {"prompt_tokens": 11, "completion_tokens": 7, "total_tokens": 18}
        c = lambda **k: {"id": "1", "object": "chat.completion.chunk", "model": "m", **k}
        if body.get('stream'):
            self.send_response(200); self.send_header('content-type', 'text/event-stream'); self.end_headers()
            ev = lambda d: self.wfile.write(("data: " + json.dumps(d) + "\n\n").encode())
            ev(c(choices=[{"index": 0, "delta": {"role": "assistant", "content": "ECHO"}, "finish_reason": None}]))
            ev(c(choices=[{"index": 0, "delta": {}, "finish_reason": "stop"}]))
            ev(c(choices=[], usage=usage)); self.wfile.write(b"data: [DONE]\n\n")
        else:
            self.send_response(200); self.send_header('content-type', 'application/json'); self.end_headers()
            self.wfile.write(json.dumps({"id": "1", "object": "chat.completion", "model": "m", "choices": [{"index": 0, "message": {"role": "assistant", "content": "Title"}, "finish_reason": "stop"}], "usage": usage}).encode())
srv = http.server.ThreadingHTTPServer(('127.0.0.1', 0), H)
open(sys.argv[1], 'w').write(str(srv.server_address[1])); srv.serve_forever()
PY
python3 -I "$t/fake.py" "$t/port" >/dev/null 2>&1 &
fake=$!
trap 'kill $fake 2>/dev/null' EXIT
for _ in $(seq 50); do [ -s "$t/port" ] && break; sleep 0.1; done
E=(env -i PATH=/usr/bin:/bin HOME="$t/home" XDG_CONFIG_HOME="$t/home/c" XDG_DATA_HOME="$t/home/d" XDG_CACHE_HOME="$t/home/x"
   XDG_RUNTIME_DIR="$t/run" CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1 CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 GIT_CEILING_DIRECTORIES="$t")
"${E[@]}" git config --global user.email t@t; "${E[@]}" git config --global user.name t
cfg='{"providers":{"fake":{"type":"openai-compat","base_url":"http://127.0.0.1:'"$(cat "$t/port")"'/v1","api_key":"x","models":[{"id":"m","name":"m","context_window":100000,"default_max_tokens":1000}]}},"models":{"large":{"provider":"fake","model":"m"},"small":{"provider":"fake","model":"m"}},"options":{"disable_provider_auto_update":true,"disable_metrics":true}}'
mkrepo() { mkdir -p "$t/$1" && printf '%s\n' "$cfg" > "$t/$1/crush.json" && (cd "$t/$1" && "${E[@]}" git init -q -b main && "${E[@]}" git add -A && "${E[@]}" git commit -q -m init); }
probe='echo "on=${AGENT_CLI_UNATTENDED:-unset} commit=${AGENT_CLI_UNATTENDED_COMMIT:-unset}" > '
for name in nocommit withcommit; do
  mkrepo "$name"
  extra=(); [ "$name" = withcommit ] && extra=(--commit)
  "${E[@]}" "$bin" routine new "$name" --mode maintenance --repo "$t/$name" --goal "do nothing" --verify "$probe$t/$name.state; true" "${extra[@]}" > "$t/$name.new" 2>&1
  (cd "$t" && "${E[@]}" timeout 120 "$bin" routine run "$name") > "$t/$name.run.out" 2> "$t/$name.run.err"; echo $? > "$t/$name.rc"
done
mkrepo looprepo
printf 'AGENT_CLI_UNATTENDED=1\nAGENT_CLI_UNATTENDED_COMMIT=1\n' > "$t/looprepo/.env"
(cd "$t/looprepo" && "${E[@]}" timeout 120 "$bin" loop --verify "$probe$t/loop.state; false" --max-repairs 0 "do nothing") > "$t/loop.out" 2> "$t/loop.err"; echo $? > "$t/loop.rc"
st() { cat "$1" 2>/dev/null || echo "missing"; }
nc=$(st "$t/nocommit.state"); wc=$(st "$t/withcommit.state"); lp=$(st "$t/loop.state")
echo "routine (no --commit) rc=$(cat "$t/nocommit.rc"): verify sees $nc"
echo "routine (--commit)    rc=$(cat "$t/withcommit.rc"): verify sees $wc"
echo "loop (.env sets both=1) rc=$(cat "$t/loop.rc"): verify sees $lp"
if [ "$nc" = "on=1 commit=0" ] && [ "$wc" = "on=1 commit=1" ] && [ "$lp" = "on=unset commit=unset" ]; then echo "UNATTENDED-VERIFY: PASS"
else echo "UNATTENDED-VERIFY: FAIL"; fi
