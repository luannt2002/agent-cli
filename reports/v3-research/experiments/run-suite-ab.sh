#!/usr/bin/env bash
# Eval LLM thật XEN KẼ hai binary theo từng kịch bản (A rồi B, cùng kịch bản), để nhiễu tải máy rơi đều cả hai.
# Bỏ s13-provider-switch (gọi glm — chưa xác nhận miễn phí). Provider: cấu hình thật của người dùng.
#   bash run-suite-ab.sh <eval-repo> <binA> <binB> <out-root-mới>
set -u
repo=$1; A=$2; B=$3; root=$4
[ -e "$root" ] && { echo "exists: $root" >&2; exit 2; }
mkdir -p "$root/A" "$root/B"
for sc in $(python3 -I "$repo/scripts/eval/fixtures.py" list | grep -v '^s13-'); do
  for side in A B; do
    bin=$A; [ $side = B ] && bin=$B
    echo "== $sc $side $(date +%H:%M:%S)"
    bash "$repo/scripts/eval/run.sh" "$sc" "$bin" "$root/$side/$sc" > "$root/$side/$sc.log" 2>&1
    tail -1 "$root/$side/$sc.log"
  done
done
echo "== done $(date +%H:%M:%S)"
python3 -I "$(dirname "$0")/compare-suite.py" "$root/A" "$root/B" | tee "$root/COMPARE.md"
