#!/usr/bin/env bash
# Tạo patch P1c: thay đổi đã theo dõi (git diff HEAD) + 2 tệp test mới; loại 6 tệp phơi nhiễm `??`.
set -eu
W=/home/luannt/application/opensource/agent-cli-p1c
OUTP=$1
cd "$W"
{
  git diff HEAD
  for f in internal/app/session_model_test.go internal/cmd/run_session_model_test.go scripts/eval/session-model.sh; do
    git diff --no-index /dev/null "$f" || true
  done
} > "$OUTP"
