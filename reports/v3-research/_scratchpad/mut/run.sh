#!/usr/bin/env bash
# run.sh <tên mutant> <tệp> <python: chuỗi cũ> <chuỗi mới> <gói> <-run regex> [count]
set -u
S=/tmp/claude-1000/-home-luannt-application/1ba1a44b-c74c-4a9d-b148-f58af7166a87/scratchpad
cd /home/luannt/application/opensource/agent-cli-p02
name=$1 file=$2 old=$3 new=$4 pkg=$5 re=$6 count=${7:-1}
python3 -I - "$file" "$old" "$new" <<'PY' || exit 3
import sys
p, old, new = sys.argv[1:4]
s = open(p).read()
assert s.count(old) == 1, "pattern count %d" % s.count(old)
open(p, "w").write(s.replace(old, new))
PY
$S/g.sh go test -count=$count -p 2 $pkg -run "$re" > $S/mut/$name.txt 2>&1; rc=$?
cp $S/mut/$(basename $file) $file
sha256sum -c --quiet $S/mut/orig.sha || { echo "RESTORE FAILED"; exit 4; }
echo "$name rc=$rc fails=$(grep -c -- '--- FAIL' $S/mut/$name.txt)"
grep -- '--- FAIL' $S/mut/$name.txt | sed 's/ (.*//' | sort | uniq -c | head -12
