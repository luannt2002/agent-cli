#!/usr/bin/env bash
# Tải lại mã nguồn ba dự án nghiên cứu V3 ĐÚNG revision đã audit (không lưu trong repo: ~1,9 GB, lịch sử git của
# người khác, Context Mode là ELv2 — chỉ học ý tưởng, không chép mã).
# Dùng: bash fetch-research-sources.sh <thư-mục-đích>   (mặc định trên máy gốc: /home/luannt/application/opensource/v3-src)
set -eu
dst=${1:?thư mục đích}
mkdir -p "$dst"
fetch() { # fetch <tên> <url> <commit> [nhánh phụ cần có] [commit nhánh phụ]
  local name=$1 url=$2 rev=$3 extra=${4:-} extra_rev=${5:-}
  [ -d "$dst/$name/.git" ] || git clone -q "$url" "$dst/$name"
  git -C "$dst/$name" fetch -q origin
  git -C "$dst/$name" checkout -q "$rev"
  if [ -n "$extra" ]; then
    got=$(git -C "$dst/$name" rev-parse --short=8 "origin/$extra" 2>/dev/null || echo "không-có")
    [ "$got" = "$extra_rev" ] || echo "cảnh báo: $name origin/$extra = $got, báo cáo audit ở $extra_rev (nhánh đã tiến; dùng $extra_rev)"
  fi
  echo "$name @ $(git -C "$dst/$name" rev-parse --short HEAD)"
}
fetch context-mode https://github.com/mksglu/context-mode.git 6006a351 v2-design 7a99ece2
fetch claude-mem https://github.com/thedotmack/claude-mem.git 3fae1eb45
fetch hindsight https://github.com/vectorize-io/hindsight.git 5331da2f4
