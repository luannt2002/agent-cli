#!/usr/bin/env bash
# mut.sh <tên> <tệp tương đối> <chuỗi cũ> <chuỗi mới> <gói> <-run regex> [count]: áp mutant, chạy test, hoàn nguyên, kiểm sha.
set -u
W=/tmp/claude-1000/-home-luannt-application/1ba1a44b-c74c-4a9d-b148-f58af7166a87/scratchpad/p02w
cd /home/luannt/application/opensource/agent-cli-p02 || exit 1
name=$1 file=$2 old=$3 new=$4 pkg=$5 re=$6 count=${7:-1}
cp "$file" "$W/mut/$name.bak"; before=$(sha256sum < "$file")
python3 -I - "$file" "$old" "$new" <<'PY' || { echo "$name: pattern not found"; exit 3; }
import sys
p, old, new = sys.argv[1:4]
s = open(p).read()
assert s.count(old) == 1, "pattern count %d" % s.count(old)
open(p, "w").write(s.replace(old, new))
PY
"$W/gp02.sh" go test -count="$count" -p 2 "$pkg" -run "$re" > "$W/mut/$name.txt" 2>&1; rc=$?
cp "$W/mut/$name.bak" "$file"
[ "$(sha256sum < "$file")" = "$before" ] || { echo "$name: RESTORE FAILED"; exit 4; }
echo "$name rc=$rc fails=$(grep -c -- '--- FAIL' "$W/mut/$name.txt") build_failed=$(grep -c 'build failed' "$W/mut/$name.txt")"
grep -- '--- FAIL' "$W/mut/$name.txt" | sed 's/ (.*//' | sort | uniq -c | head -8
