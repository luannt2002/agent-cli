#!/usr/bin/env bash
# P0-2b đối chứng: server dựng với env GỐC vẫn tự nạp `.env` của cwd (main), nên tool bash của agent (chế độ
# server) vẫn thấy biến thường của `.env` như trước. LLM giả gọi bash ghi tên biến (không giá trị) nếu giá trị
# là canary. Bất biến: PLAIN_SV có trong env của tool bash ở chế độ server.
# Dùng: bash p02b-sv.sh <thư-mục-mới ≤ 70 byte> <binary>
set -u
[ $# -eq 2 ] || { echo "usage: $0 <new-dir> <binary>" >&2; exit 2; }
OUT=$1; BIN=$2
R=/home/luannt/application/opensource/agent-cli-p02b
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd) && BIN=$(cd "$(dirname "$BIN")" && pwd)/$(basename "$BIN")
L="$OUT/_launcher"
eval "$(sed -n '/^verdict()/,/^for c in \$CASES/p' "$R/scripts/eval/exposure.sh" | sed '$d')"
begin SV '[{"match":"SV-","steps":[{"tool":"bash","args":{"command":"printenv PLAIN_SV | grep -q CANARY_DOTENV && echo PLAIN_SV:dotenv > seen.txt; true","description":"probe"}},{"text":"done"}]}]' qwen
mkdir -p "$P/repo"; git_repo "$P/repo"
printf 'PLAIN_SV=CANARY_DOTENV\n' > "$P/repo/.env"
server_mode
ac "$P/repo" run --json "SV-server probe" > "$P/server-run.json" 2> "$P/server-run.err"; echo $? > "$P/server-run.rc"
finish
if grep -q "PLAIN_SV:dotenv" "$P/repo/seen.txt" 2>/dev/null; then echo "SV: server bash tool sees plain .env variable: PASS"
else echo "SV: FAIL (rc=$(cat "$P/server-run.rc"), seen=$(cat "$P/repo/seen.txt" 2>/dev/null))"; fi | tee "$P/summary.txt"
