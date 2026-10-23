#!/usr/bin/env bash
# Cổng phát hành: so test theo TÊN với một bản nền, vet (chỉ dòng mới), vet Windows, race, build, 4 E2E.
# Dùng: scripts/release-gates.sh <ref-nền, vd v0.3.2> [thư-mục-ra]
# Máy dùng chung: đặt GO_LOCK=<tệp khoá> để mọi lệnh go chạy qua flock; GOFLAGS/GOPROXY lấy từ env.
# Thoát 0 khi mọi cổng PASS, 1 khi có cổng FAIL, 2 khi sai cách dùng.
set -u
base_ref="${1:-}"
[ -n "$base_ref" ] || { echo "usage: $0 <base-ref> [out-dir]" >&2; exit 2; }
root="$(git rev-parse --show-toplevel)" || exit 2
out="${2:-$(mktemp -d)}"
mkdir -p "$out/bin"
cd "$root"
export GOMAXPROCS="${GOMAXPROCS:-2}"
lock=()
[ -n "${GO_LOCK:-}" ] && lock=(flock "$GO_LOCK")
gorun() { "${lock[@]}" go "$@"; }
fails=0
gate() { # gate <tên> <PASS|FAIL> [chi tiết]
  echo "$2 $1${3:+: $3}"
  [ "$2" = PASS ] || fails=$((fails + 1))
}

# Test an ninh của grep/glob chạy với rg thật; thiếu rg thì chúng skip và cổng xanh giả.
security_tests=(TestGrepHostileArgsWithRealRg TestGrepPatternIsNeverARipgrepFlag TestGlobPrefixBypassAttempts)
if ! type -P rg >/dev/null; then
  gate "ripgrep" FAIL "no rg binary on PATH: ${security_tests[*]} would skip"
  exit 1
fi

wt="$out/base"
git worktree add -q --detach "$wt" "$base_ref" || { gate "base worktree" FAIL "$base_ref"; exit 1; }
trap 'git -C "$root" worktree remove --force "$wt" 2>/dev/null' EXIT
(cd "$wt" && CGO_ENABLED=0 gorun test -json -count=1 -p 2 ./... >"$out/before.json" 2>"$out/before.err")
(CGO_ENABLED=0 gorun test -json -count=1 -p 2 ./... >"$out/after.json" 2>"$out/after.err")
python3 -I "$root/reports/v3-research/tools/cmp.py" "$out/before.json" "$out/after.json" | tee "$out/cmp.txt"
fail_after=$(sed -n 's/^== FAIL after: //p' "$out/cmp.txt")
lost=$(sed -n 's/^== lost (before only): //p' "$out/cmp.txt")
names=$(sed -n 's/^names: before=[0-9]* after=//p' "$out/cmp.txt")
if [ "${fail_after:-x}" = 0 ] && [ "${lost:-x}" = 0 ] && [ "${names:-0}" -gt 0 ]; then
  gate "tests by name" PASS "$names names"
else
  gate "tests by name" FAIL "fail=$fail_after lost=$lost names=$names"
fi
skipped=$(python3 -I -c '
import json, sys
must = set(sys.argv[2:])
seen = {}
for line in open(sys.argv[1]):
    try: e = json.loads(line)
    except Exception: continue
    if e.get("Test") in must and e.get("Action") in ("pass", "fail", "skip"):
        seen[e["Test"]] = e["Action"]
print(" ".join(t for t in sorted(must) if seen.get(t) != "pass"))' "$out/after.json" "${security_tests[@]}")
[ -z "$skipped" ] && gate "security tests ran" PASS || gate "security tests ran" FAIL "not passed: $skipped"

(cd "$wt" && CGO_ENABLED=0 gorun vet ./... >"$out/vet-base.out" 2>&1)
CGO_ENABLED=0 gorun vet ./... >"$out/vet.out" 2>&1
new_vet=$(grep -vxF -f <(sed "s#$wt/##g" "$out/vet-base.out") <(sed "s#$root/##g" "$out/vet.out") | grep -v '^#' || true)
[ -z "$new_vet" ] && gate "vet (new lines)" PASS || gate "vet (new lines)" FAIL "$new_vet"

fork_pkgs=(./internal/cmd/... ./internal/loop/... ./internal/routine/... ./internal/shell/... ./internal/execpolicy/... ./internal/housekeep/... ./internal/projects/...)
CGO_ENABLED=0 GOOS=windows gorun vet "${fork_pkgs[@]}" >"$out/vetw.out" 2>&1 && gate "vet windows" PASS || gate "vet windows" FAIL "see $out/vetw.out"

race_pkgs=(./internal/backend/... ./internal/agent/... ./internal/cmd/... ./internal/ui/... ./internal/workspace/... ./internal/projects/...
  ./internal/permission/... ./internal/server/... ./internal/loop/... ./internal/routine/... ./internal/shell/...
  ./internal/execpolicy/... ./internal/housekeep/... ./internal/config/...)
CGO_ENABLED=1 gorun test -race -count=1 -p 2 "${race_pkgs[@]}" >"$out/race.out" 2>&1
rc=$?
races=$(grep -c 'WARNING: DATA RACE' "$out/race.out")
[ $rc = 0 ] && [ "$races" = 0 ] && gate "race" PASS || gate "race" FAIL "rc=$rc races=$races"

if CGO_ENABLED=0 gorun build -p 2 -o "$out/bin/dew" .; then
  gate "build" PASS
  for e in routine loop run-json wrapper; do
    DEW_BIN="$out/bin/dew" bash "scripts/test-dew-$e.sh" >"$out/e2e-$e.out" 2>&1
    rc=$?
    ok=$(grep -c '^ok ' "$out/e2e-$e.out")
    bad=$(grep -c '^FAIL' "$out/e2e-$e.out")
    [ $rc = 0 ] && [ "$bad" = 0 ] && gate "e2e $e" PASS "$ok ok" || gate "e2e $e" FAIL "rc=$rc fail=$bad"
  done
else
  gate "build" FAIL
fi

echo "output: $out"
[ $fails = 0 ]
