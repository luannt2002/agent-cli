#!/usr/bin/env bash
# Kiểm răng: chạy test mới của launcher trên các bản ĐỘT BIẾN của scripts/agent-cli-tmux.sh (bản sao trong scratchpad,
# không đụng worktree). Mỗi đột biến phải làm test ĐỎ; bản sửa nguyên vẹn phải XANH.
# Dùng: bash mutants.sh <worktree> <thư-mục-nháp-mới>
set -u
W=$1; S=$2
mkdir "$S" || exit 2
L=$W/scripts/agent-cli-tmux.sh
mk() { mkdir -p "$S/$1/scripts"; cp "$W/scripts/test-agent-cli-tmux.sh" "$S/$1/scripts/"; }
mk fixed; cp "$L" "$S/fixed/scripts/agent-cli-tmux.sh"
mk r0; git -C "$W" show HEAD:scripts/agent-cli-tmux.sh > "$S/r0/scripts/agent-cli-tmux.sh"
mk m1; sed 's|\[\[ \$c != "\$abs" && \$c != "\$abs"/\* \]\]|[[ $c != "$abs" ]]|' "$L" > "$S/m1/scripts/agent-cli-tmux.sh"
mk m2; sed 's|ids=\$(find_window "\$abs" all)|ids=$(find_window "$abs")|' "$L" > "$S/m2/scripts/agent-cli-tmux.sh"
mk m3; sed 's|\$c != "\$abs"/\*|$c != "$abs"*|' "$L" > "$S/m3/scripts/agent-cli-tmux.sh"
mk m4; sed 's|index(\$2, d "/") == 1|index($2, d) == 1|' "$L" > "$S/m4/scripts/agent-cli-tmux.sh"
for m in fixed r0 m1 m2 m3 m4; do
  echo "== $m: dòng khác so với bản sửa"
  diff "$L" "$S/$m/scripts/agent-cli-tmux.sh" | grep '^[<>]' || echo "  (không khác)"
  out=$(bash "$S/$m/scripts/test-agent-cli-tmux.sh" 2>&1); rc=$?
  echo "  rc=$rc | $(tail -1 <<< "$out")"
done
