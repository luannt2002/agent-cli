#!/usr/bin/env bash
# P02 vòng 3, #2: server khác build đang bận. Lệnh chạy model (`run`) phải bị TỪ CHỐI rõ ràng (server bản khác có thể
# không so chính sách tin cậy); lệnh không prompt (`jobs`) vẫn dùng server đó. Đối chứng: client cùng build với server
# gắn vào bình thường. Khung cô lập của scripts/eval/exposure.sh, LLM giả.
# Dùng: bash p02-stale.sh <thư-mục-mới ≤ 70 byte> <binary server (bản cũ)> <binary client (bản mới)>
set -u
[ $# -eq 3 ] || { echo "usage: $0 <new-dir> <server-binary> <client-binary>" >&2; exit 2; }
OUT=$1; OLD=$(cd "$(dirname "$2")" && pwd)/$(basename "$2"); NEW=$(cd "$(dirname "$3")" && pwd)/$(basename "$3")
R=/home/luannt/application/opensource/agent-cli-p02
[ -e "$OUT" ] && { echo "exists: $OUT" >&2; exit 2; }
mkdir -p "$OUT" && OUT=$(cd "$OUT" && pwd); L="$OUT/_launcher"; BIN=$OLD
eval "$(sed -n '/^verdict()/,/^for c in \$CASES/p' "$R/scripts/eval/exposure.sh" | sed '$d')"
begin ST '[{"match":"ST-hold","steps":[{"sleep":8,"text":"held"}]}]' qwen
mkdir -p "$P/proj"; git_repo "$P/proj"
server_mode st
BIN=$OLD; ac "$P/proj" run --json "ST-hold old server" > "$P/hold.json" 2>&1 & hold=$!
sleep 2.5
BIN=$NEW; ac "$P/proj" run --json "ST-new run" > "$P/new-run.json" 2> "$P/new-run.err"; echo $? > "$P/new-run.rc"
BIN=$NEW; ac "$P/proj" jobs > "$P/new-jobs.out" 2>&1; echo $? > "$P/new-jobs.rc"
BIN=$OLD; ac "$P/proj" run --json "ST-same run" > "$P/same-run.json" 2> "$P/same-run.err"; echo $? > "$P/same-run.rc"
wait $hold; stop_servers; finish
python3 -I - "$P" <<'PY' | tee "$P/summary.txt"
import json, os, sys
P = sys.argv[1]
reqs = [json.loads(l) for l in open(os.path.join(P, "log", "requests.jsonl"))]
prov = lambda tag: sorted({r["provider"] for r in reqs if r["role"] == "main" and r["last_user"].lstrip().startswith(tag)}) or ["none"]
rd = lambda f: open(os.path.join(P, f), errors="replace").read().strip()
def err(f):
    try:
        return json.load(open(os.path.join(P, f))).get("error") or ""
    except Exception as e:
        return "no JSON (%s)" % e
e = err("new-run.json")
ok_run = rd("new-run.rc") != "0" and "another build and is busy" in e and prov("ST-new") == ["none"]
ok_jobs = rd("new-jobs.rc") == "0" and "busy" not in rd("new-jobs.out")
ok_same = rd("same-run.rc") == "0" and prov("ST-same") == ["qwen"]
print("new-build run on busy old server: %s rc=%s provider=%s err=%r" % ("PASS" if ok_run else "FAIL", rd("new-run.rc"), prov("ST-new"), e[:300]))
print("new-build jobs on busy old server: %s rc=%s out=%r" % ("PASS" if ok_jobs else "FAIL", rd("new-jobs.rc"), rd("new-jobs.out")[-120:]))
print("same-build run (control): %s rc=%s provider=%s" % ("PASS" if ok_same else "FAIL", rd("same-run.rc"), prov("ST-same")))
print("STALE:", "PASS" if ok_run and ok_jobs and ok_same else "FAIL")
PY
