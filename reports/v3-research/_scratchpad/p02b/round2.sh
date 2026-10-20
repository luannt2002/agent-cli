#!/usr/bin/env bash
# P0-2b vòng 2: build binary cuối, chạy toàn bộ e2e + hồi quy, rồi suite theo tên và -race trên bản chép của worktree.
# Kết quả tóm tắt ở $S/round2.txt. Dùng: bash round2.sh
set -u
S=$(cd "$(dirname "$0")" && pwd)
W=/home/luannt/application/opensource/agent-cli-p02b
V=/home/luannt/.cache/v3ev
B=$V/p02b-bin/agent-cli-r2
OUT=$S/round2.txt
: > "$OUT"
log() { echo "$*" | tee -a "$OUT"; }
cd "$W" || exit 1
"$S/gp.sh" env CGO_ENABLED=0 go build -p 2 -o "$B" . || { log "BUILD FAIL"; exit 1; }
log "binary: $B ($("$B" --version))"
bash "$S/p02b-home-cli.sh" "$V/p02b-hc-r2" "$B" 2>&1 | grep -E '^HC' | tee -a "$OUT"
bash "$S/p02b-e2e.sh" "$V/p02b-e-r2" "$B" 2>&1 | grep -E '^(CLI|LN|local|server):' | tee -a "$OUT"
bash "$S/p02b-sv.sh" "$V/p02b-sv-r2" "$B" 2>&1 | tail -1 | tee -a "$OUT"
bash "$S/p02b-routine-unattended.sh" "$V/p02b-ru-r2" "$B" 2>&1 | tail -4 | tee -a "$OUT"
bash "$S/regress.sh" p02b-rr2 "$B" > /dev/null 2>&1
grep -E '^[a-z0-9]+ rc=|PASS|NOT_EXPOSED|ALLOW=' "$S/regress-p02b-rr2.txt" | cut -c1-160 | tee -a "$OUT"
for s in routine loop; do
  start=$SECONDS
  AGENT_CLI_BIN="$B" timeout 1500 bash "scripts/test-agent-cli-$s.sh" > "$S/regress-p02b-rr2-$s.out" 2>&1
  log "$s rc=$? ($((SECONDS - start))s) $(tail -1 "$S/regress-p02b-rr2-$s.out")"
done
mkdir -p "$S/after-r2"
tar --exclude=./.git -cf - . | tar -xf - -C "$S/after-r2"
(cd "$S" && GPDIR="$S/after-r2" "$S/gp.sh" go test -json -count=1 -p 2 ./... > "$S/after-r2.json" 2> "$S/after-r2.err"); log "suite rc=$?"
python3 -I "$S/../p02w/cmp.py" "$S/base.json" "$S/after-r2.json" | grep -v 'TestV3Exposure' | tee -a "$OUT"
(cd "$S" && GPDIR="$S/after-r2" "$S/gp.sh" go test -race -count=1 -p 2 ./internal/cmd/... ./internal/backend/... ./internal/workspace/... ./internal/agent/tools/provider/... ./internal/execpolicy/... > "$S/race-r2.txt" 2>&1); log "race rc=$? DATA_RACE=$(grep -c 'DATA RACE' "$S/race-r2.txt") nonV3fails=$(grep -E '^\s*--- FAIL' "$S/race-r2.txt" | grep -vc V3Exposure)"
grep -E '^(ok|FAIL)\s' "$S/race-r2.txt" | tee -a "$OUT"
log "ROUND2 DONE"
