#!/usr/bin/env bash
# Chạy bộ đo LLM thật (scripts/eval/run.sh) cho MỘT binary, mọi kịch bản trừ s13-provider-switch (bước 2 gọi
# glm/glm-5.3-flash — provider chưa được người dùng xác nhận là miễn phí). Provider: cấu hình thật của người dùng
# (~/.config/agent-cli, main/qwen3.8 local). Dữ liệu mỗi kịch bản ở <out-root>/<kịch bản>/.
#   bash run-suite.sh <eval-repo> <binary> <out-root-mới>
# <eval-repo>: cây có scripts/eval/ (vd worktree v2-final). Chạy tuần tự (một LLM local).
set -u
repo=$1; bin=$2; root=$3
[ -e "$root" ] && { echo "exists: $root" >&2; exit 2; }
mkdir -p "$root"
for sc in $(python3 -I "$repo/scripts/eval/fixtures.py" list | grep -v '^s13-'); do
  echo "== $sc $(date +%H:%M:%S)"
  bash "$repo/scripts/eval/run.sh" "$sc" "$bin" "$root/$sc" > "$root/$sc.log" 2>&1
  tail -1 "$root/$sc.log"
done
python3 -I "$repo/scripts/eval/score.py" summary "$root/*" > "$root/SUMMARY.md"
cat "$root/SUMMARY.md"
