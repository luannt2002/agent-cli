#!/usr/bin/env bash
# Chạy MỘT kịch bản của bộ đo hệ thống (scripts/eval/fixtures.py) và ghi số đo ra metrics.json.
#
#   bash scripts/eval/run.sh <scenario> <binary> <out-dir>
#
# Provider: cấu hình THẬT của bạn (CRUSH_GLOBAL_CONFIG mặc định ~/.config/agent-cli), skill thật trong HOME,
# nhưng dữ liệu (global data, cache, DB dự án) nằm trong <out-dir>, chạy in-process (không đụng server đang
# chạy). Sổ ghi LLM (DEW_LLM_LEDGER) bật để đếm lời gọi/token THẬT; binary không có sổ ghi thì cột
# token để trống. Không in secret: chỉ ghi kết quả JSON của agent-cli và số đo.
#   EVAL_SMALL_MODEL   provider/model dùng cho bước có "models": "small" (kịch bản đổi model)
#   EVAL_STEP_TIMEOUT  giây tối đa mỗi bước (mặc định 900)
set -u
here="$(cd "$(dirname "$0")" && pwd)"
sc="$1"; bin="$2"; out="$3"
[ -e "$out" ] && { echo "out dir exists: $out" >&2; exit 2; }
mkdir -p "$out"
python3 -I "$here/fixtures.py" make "$sc" "$out" || exit 2
repo="$out/repo"
(cd "$repo" && git init -q && git add -A && git -c user.email=eval@local -c user.name=eval commit -qm "fixture $sc")

run_env=(env -i PATH=/usr/local/bin:/usr/bin:/bin HOME="$HOME"
  CRUSH_GLOBAL_CONFIG="${EVAL_GLOBAL_CONFIG:-$HOME/.config/agent-cli}"
  CRUSH_GLOBAL_DATA="$out/gdata" CRUSH_CACHE_DIR="$out/cache"
  CRUSH_CLIENT_SERVER=0 CRUSH_DISABLE_METRICS=1 DO_NOT_TRACK=1
  CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1 CRUSH_DISABLE_UPDATE_CHECK=1
  DEW_LLM_LEDGER="$out/ledger.jsonl")

nsteps=$(python3 -I -c 'import json,sys; print(len(json.load(open(sys.argv[1]))["steps"]))' "$out/scenario.json")
sid=""
for ((i = 0; i < nsteps; i++)); do
  prompt=$(python3 -I -c 'import json,sys; print(json.load(open(sys.argv[1]))["steps"][int(sys.argv[2])])' "$out/scenario.json" "$i")
  model=$(python3 -I -c 'import json,sys; m=json.load(open(sys.argv[1])).get("models"); print((m or [None]*99)[int(sys.argv[2])] or "")' "$out/scenario.json" "$i")
  args=(run --json -q)
  [ -n "$sid" ] && args+=(-s "$sid")
  [ "$model" = small ] && [ -n "${EVAL_SMALL_MODEL:-}" ] && args+=(-m "$EVAL_SMALL_MODEL")
  start=$(date +%s.%N)
  (cd "$repo" && timeout "${EVAL_STEP_TIMEOUT:-900}" "${run_env[@]}" "$bin" "${args[@]}" "$prompt") >"$out/step-$i.json" 2>"$out/step-$i.err"
  rc=$?
  end=$(date +%s.%N)
  echo "{\"rc\": $rc, \"seconds\": $(python3 -I -c "print(round($end-$start,1))")}" >"$out/step-$i.meta"
  sid=$(python3 -I -c 'import json,sys
try: print(json.load(open(sys.argv[1])).get("session_id",""))
except Exception: print("")' "$out/step-$i.json")
done

verify=$(python3 -I -c 'import json,sys; print(json.load(open(sys.argv[1]))["verify"] or "")' "$out/scenario.json")
vrc=""
if [ -n "$verify" ]; then
  (cd "$repo" && timeout 600 bash -c "set -o pipefail; $verify") >"$out/verify.out" 2>&1
  vrc=$?
fi
[ -n "$sid" ] && (cd "$repo" && "${run_env[@]}" "$bin" session show "$sid" --json) >"$out/session.json" 2>/dev/null
(cd "$repo" && git status --porcelain && git diff --stat HEAD) >"$out/changes.txt" 2>&1

python3 -I "$here/score.py" "$out" "$bin" "${vrc:-}"
