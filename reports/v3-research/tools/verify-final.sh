#!/usr/bin/env bash
# Kiểm cơ học toàn bộ cho một commit ứng viên V3, trong worktree RIÊNG (không đụng worktree ứng viên).
# Dùng: bash verify-final.sh <commit> <out-dir-mới>
# Biến môi trường (mặc định theo bố cục máy luannt-pc):
#   SRC  = worktree bất kỳ của repo agent-cli (để `git worktree add`)
#   WT   = đường worktree riêng cho lượt kiểm (tạo nếu chưa có)
#   TOOLS= thư mục chứa cmp.py (mặc định: thư mục của script này)
#   BASE = tệp `go test -json` của v2-final để so theo tên (chạy trước: git worktree add … v2-final; go test -json ./... > BASE)
#   EV   = thư mục gốc cho dữ liệu cổng eval (đường ngắn: socket unix ≤ 108 byte)
# Kết quả: <out>/SUMMARY.txt (mỗi bước một dòng rc), log từng bước trong <out>/.
set -u
commit=$1; out=$2
[ -e "$out" ] && { echo "exists: $out" >&2; exit 2; }
mkdir -p "$out"
TOOLS=${TOOLS:-$(cd "$(dirname "$0")" && pwd)}
SRC=${SRC:-/home/luannt/application/opensource/agent-cli}
WT=${WT:-$(dirname "$SRC")/agent-cli-vfinal}
BASE=${BASE:-$HOME/.cache/v3ev/final/v2-suite.json}
EV=${EV:-$HOME/.cache/v3ev}
LOCK=${LOCK:-${TMPDIR:-/tmp}/agent-go.lock}
GOENV="GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off CGO_ENABLED=0 GOEXPERIMENT=greenteagc"
sum() { echo "$1 rc=$2" | tee -a "$out/SUMMARY.txt"; }
waitram() { while [ "$(free -m | awk '/Mem/{print $7}')" -lt 3000 ]; do sleep 20; done; }

# Worktree riêng ở đúng commit (tạo mới hoặc dời về commit; cây phải sạch ngoài tệp phơi nhiễm đã chép).
if [ -d "$WT" ]; then
  git -C "$WT" checkout -q --detach "$commit" || { sum worktree 1; exit 1; }
else
  git -C "$SRC" worktree add -q --detach "$WT" "$commit" || { sum worktree 1; exit 1; }
fi
cd "$WT" || exit 1
echo "HEAD $(git rev-parse --short HEAD)" | tee -a "$out/SUMMARY.txt"

waitram
flock $LOCK env $GOENV go build -p 2 -o "$out/bin" . > "$out/build.log" 2>&1; sum build $?
"$out/bin" --version | tee -a "$out/SUMMARY.txt"

# 1. Toàn kho theo tên, so với v2-final (baseline chạy riêng: /home/luannt/.cache/v3ev/final/v2-suite.json).
waitram
flock $LOCK env $GOENV GOMEMLIMIT=3GiB go test -json -count=1 -p 2 ./... > "$out/suite.json" 2> "$out/suite.err"; sum suite $?
python3 -I "$TOOLS/cmp.py" "$BASE" "$out/suite.json" > "$out/suite-compare.txt" 2>&1; sum suite-compare $?

# 2. -race: danh sách P5 của CLAUDE.md + mọi package Go mà ứng viên đã chạm so với v2-final.
pk=$( { printf '%s\n' ./internal/backend/... ./internal/agent/... ./internal/cmd/... ./internal/ui/... ./internal/workspace/... \
          ./internal/projects/... ./internal/permission/... ./internal/server/...;
        git diff --name-only 7cba42ff "$commit" -- '*.go' | xargs -r -n1 dirname | sed 's|^|./|'; } | sort -u)
echo "$pk" > "$out/race-pkgs.txt"
waitram
flock $LOCK env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off CGO_ENABLED=1 GOEXPERIMENT=greenteagc GOMEMLIMIT=3GiB \
  go test -race -count=1 -p 2 $pk > "$out/race.log" 2>&1; sum race $?
grep -c "WARNING: DATA RACE" "$out/race.log" | sed 's/^/data-race-warnings=/' | tee -a "$out/SUMMARY.txt"

# 3. E2E binary thật, LLM giả.
for s in run-json loop routine tmux wrapper; do
  AGENT_CLI_BIN="$out/bin" bash "scripts/test-agent-cli-$s.sh" > "$out/e2e-$s.log" 2>&1; sum "e2e-$s" $?
done

# 4. Cổng eval (thư mục ca ngắn: socket ≤ 108 byte).
E=$EV/vf-$(basename "$out")
for g in faults qwen-only workspaces session-model ledger; do
  bash "scripts/eval/$g.sh" "$E-$g" "$out/bin" > "$out/gate-$g.log" 2>&1; sum "gate-$g" $?
done

# 5. Phơi nhiễm: chép tạm 5 tệp unit + exposure.sh từ nhánh v3-exposure, chạy, gỡ.
files=$(git -C "$SRC" diff --name-only v2-final v3-exposure)
for f in $files; do mkdir -p "$(dirname "$f")"; git -C "$SRC" show "v3-exposure:$f" > "$f"; done
chmod +x scripts/eval/exposure.sh
waitram
flock $LOCK env $GOENV go test -count=1 -p 2 -v -run 'TestV3Exposure' ./internal/backend/ ./internal/agent/ ./internal/agent/tools/ \
  ./internal/permission/ ./internal/shell/ > "$out/exposure-unit.log" 2>&1; sum exposure-unit $?
bash scripts/eval/exposure.sh "$E-exp" "$out/bin" > "$out/exposure-bin.log" 2>&1; sum exposure-bin $?
for f in $files; do rm -f "$f"; done
git status --short | sed 's/^/dirty: /' | tee -a "$out/SUMMARY.txt"
grep -E "^(T[0-9]+[A-Za-z]*|C[0-9]+):" "$out/exposure-bin.log" | tee -a "$out/SUMMARY.txt"
grep -E "^(--- (PASS|FAIL)|=== RUN)" "$out/exposure-unit.log" | grep -- '---' | tee -a "$out/SUMMARY.txt"
echo DONE | tee -a "$out/SUMMARY.txt"
