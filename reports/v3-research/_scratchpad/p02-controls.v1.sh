#!/usr/bin/env bash
# Đối chứng e2e cho P02 (bản sửa chính sách tin cậy), dựng lại khung C5 bằng CHÍNH các hàm của
# scripts/eval/exposure.sh (trích bằng sed, không sửa tệp đó). Binary thật + LLM giả, cô lập như exposure.sh.
# Mỗi kịch bản: client đầu giữ một lượt 8s trên server riêng, client sau gắn vào cùng workspace.
#   S1 same    : cả hai AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen, dự án chọn qwen  -> client sau CHẠY (qwen)
#   S2 plain   : khác ở biến thường (FOO=bar), dự án chọn pp                    -> client sau CHẠY (pp)
#   S3 dotenv  : `.env` của dự án đặt allowlist qwen (không export), chọn pp    -> client sau CHẠY (pp), không bị từ chối
#   S4 allow   : client sau export allowlist qwen (= C5 allow)                  -> client sau BỊ TỪ CHỐI, 0 request
#   S5 yolo    : client đầu là TUI --yolo (tmux riêng), client sau `run`        -> client sau BỊ TỪ CHỐI (--yolo)
# Dùng: bash p02-controls.sh <thư-mục-mới ≤ 70 byte> <binary>
set -u
[ $# -eq 2 ] || { echo "usage: $0 <new-dir> <binary>" >&2; exit 2; }
OUT=$1; BIN=$2
R=/home/luannt/application/opensource/agent-cli-p02
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd) && BIN=$(cd "$(dirname "$BIN")" && pwd)/$(basename "$BIN")
L="$OUT/_launcher"
eval "$(sed -n '/^verdict()/,/^for c in \$CASES/p' "$R/scripts/eval/exposure.sh" | sed '$d')"

begin CTL '[{"match":"CTL-hold","steps":[{"sleep":8,"text":"held"}]}]' qwen
for d in s1 s2 s3 s4 s5; do mkdir -p "$P/$d"; git_repo "$P/$d"; done
printf '%s' '{"models":{"large":{"provider":"qwen","model":"m-qwen"},"small":{"provider":"qwen","model":"m-qwen"}}}' > "$P/s1/crush.json"
for d in s2 s3 s4 s5; do
  printf '%s' '{"models":{"large":{"provider":"pp","model":"m-pp"},"small":{"provider":"pp","model":"m-pp"}}}' > "$P/$d/crush.json"
done

# pair <thư mục> <tên> <env client đầu> <env client sau>: env là chuỗi "K=V K=V" (rỗng = không thêm).
pair() {
  local d=$1 n=$2 hold
  server_mode "$n"; read -r -a EXTRA <<< "$3"
  ac "$P/$d" run --json "CTL-hold $n" > "$P/$n-hold.json" 2>&1 & hold=$!
  sleep 2.5
  [ "$n" = s3 ] && printf 'AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen\nAGENT_CLI_DELEGATION=on\n' > "$P/$d/.env"
  read -r -a EXTRA <<< "$4"
  ac "$P/$d" run --json "CTL-$n second" > "$P/$n-two.json" 2> "$P/$n-two.err"; echo $? > "$P/$n-two.rc"
  wait $hold; stop_servers; EXTRA=()
}
pair s1 s1 "AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen" "AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen"
pair s2 s2 "" "FOO=bar"
pair s3 s3 "" ""
pair s4 s4 "" "AGENT_CLI_ALLOWED_MODELS=qwen:m-qwen"

# S5: TUI --yolo trong tmux riêng (socket riêng, không đụng màn hình), rồi `run` cùng dự án.
server_mode s5
TMUX_SOCK="$P/tmux.sock"
iso "$P/s5" "$(command -v tmux)" -S "$TMUX_SOCK" new-session -d -x 200 -y 50 -s y "$BIN --host unix://$SOCK --yolo" 2> "$P/s5-tmux.err"
for _ in $(seq 100); do [ -S "$SOCK" ] && break; sleep 0.1; done
sleep 3
ac "$P/s5" run --json "CTL-s5 second" > "$P/s5-two.json" 2> "$P/s5-two.err"; echo $? > "$P/s5-two.rc"
"$(command -v tmux)" -S "$TMUX_SOCK" kill-server 2>/dev/null
stop_servers
finish

python3 -I - "$P" <<'PY' | tee "$P/summary.txt"
import json, os, sys
P = sys.argv[1]
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))]
def prov(tag):
    return sorted({r["provider"] for r in reqs if r["role"] == "main" and r["last_user"].lstrip().startswith(tag)}) or ["none"]
def err(n):
    try:
        return json.load(open(os.path.join(P, n + "-two.json"))).get("error") or ""
    except Exception as e:
        return "no JSON (%s)" % e
rc = lambda n: open(os.path.join(P, n + "-two.rc")).read().strip()
want = {  # n: (provider mong đợi, có bị từ chối không, chuỗi phải có trong lỗi)
    "s1": (["qwen"], False, ""),
    "s2": (["pp"], False, ""),
    "s3": (["pp"], False, ""),
    "s4": (["none"], True, "AGENT_CLI_ALLOWED_MODELS"),
    "s5": (["none"], True, "--yolo"),
}
ok_all = True
for n, (wp, refused, key) in want.items():
    e, p = err(n), prov("CTL-%s second" % n)
    is_ref = "trust policy differs" in e
    ok = p == wp and is_ref == refused and (key in e) and ("qwen:m-qwen" not in e)
    ok_all &= ok
    print("%s: %s rc=%s provider=%s refused=%s err=%r" % (n, "PASS" if ok else "FAIL", rc(n), p, is_ref, e[:400]))
print("CONTROLS:", "PASS" if ok_all else "FAIL")
PY
