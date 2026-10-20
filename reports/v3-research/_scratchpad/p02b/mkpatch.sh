#!/usr/bin/env bash
# Patch P0-2b = git diff v3-poc-p02-policy (tệp tracked) + tệp test mới chưa theo dõi, LOẠI TRỪ 6 tệp test phơi
# nhiễm *_v3exposure_test.go và scripts/eval/exposure.sh. Không đụng index git. Kiểm: áp lên bản xuất của
# v3-poc-p02-policy ra thư mục tạm, rồi so cây kết quả với worktree (trừ tệp loại trừ) = 0 khác.
# Dùng: bash mkpatch.sh <tệp patch đích> <thư mục kiểm mới>
set -eu
OUT=$1; CHK=$2
W=/home/luannt/application/opensource/agent-cli-p02b
cd "$W"
excluded() { case "$1" in *_v3exposure_test.go|scripts/eval/exposure.sh) return 0;; esac; return 1; }
{
  git diff v3-poc-p02-policy --
  git ls-files --others --exclude-standard | sort | while read -r f; do
    excluded "$f" && continue
    git diff --no-index -- /dev/null "$f" || true
  done
} > "$OUT"
sha256sum "$OUT" | sed "s|  .*/|  |" > "$OUT.sha256"
# Kiểm áp được và đủ.
[ -e "$CHK" ] && { echo "exists: $CHK" >&2; exit 2; }
mkdir -p "$CHK"
git archive v3-poc-p02-policy | tar -x -C "$CHK"
(cd "$CHK" && git apply --whitespace=error-all "$OUT")
diffs=0
while read -r f; do
  excluded "$f" && continue
  if [ -e "$f" ]; then cmp -s "$f" "$CHK/$f" || { echo "DIFF $f"; diffs=$((diffs+1)); }
  fi
done < <(git ls-files; git ls-files --others --exclude-standard)
echo "files in patch: $(grep -c '^diff --git' "$OUT")  tree mismatches vs worktree: $diffs"
grep '^diff --git' "$OUT" | sed 's/^diff --git a\///; s/ b\/.*//'
cat "$OUT.sha256"
