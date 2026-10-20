#!/usr/bin/env bash
# Kiểm răng: mỗi mutant hoàn đúng một cơ chế, chạy test của cơ chế đó, rồi chép lại tệp gốc.
set -u
W=/home/luannt/application/opensource/agent-cli-p2c
S=/tmp/claude-1000/-home-luannt-application/1ba1a44b-c74c-4a9d-b148-f58af7166a87/scratchpad/p2c
cd "$W"
gotest() { flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off CGO_ENABLED=0 GOEXPERIMENT=greenteagc go test -p 2 -count=1 "$@" 2>&1 | grep -E "^(--- FAIL|ok|FAIL)|Error:|expected|actual|Messages" | head -12; }
# mutate <tệp> <cũ> <mới>: thay đúng một chỗ, dừng nếu không khớp.
mutate() { python3 -I - "$1" "$2" "$3" <<'PY'
import sys
p, old, new = sys.argv[1:]
s = open(p).read()
assert s.count(old) == 1, ("no unique match", p, old)
open(p, "w").write(s.replace(old, new))
PY
}
restore() { cp -p "$S/bak/$1" "$W/$1"; }

echo "== M1 fsext: đi lên theo chữ (bỏ canonicalize)"
mutate internal/fsext/lookup.go $'\tcwd = canonicalize(cwd)\n' ''
gotest -run 'TestLookupBounded' ./internal/fsext/
gotest -run 'TestStrayDataDirs' ./internal/config/
restore internal/fsext/lookup.go

echo "== M2 gitutil: findGitDir theo chữ (bỏ EvalSymlinks)"
mutate internal/gitutil/branch.go $'\tif resolved, err := filepath.EvalSymlinks(dir); err == nil {\n\t\tdir = resolved\n\t}\n' ''
gotest -run 'TestCurrentBranch' ./internal/gitutil/
restore internal/gitutil/branch.go

echo "== M3 config: data dir = .crush gần nhất (cơ chế cũ)"
mutate internal/config/load.go $'\t\tc.Options.DataDirectory = filepath.Join(projectBoundary(workingDir), defaultDataDirectory)\n' $'\t\tif path, ok := fsext.LookupClosestBounded(workingDir, projectBoundary(workingDir), defaultDataDirectory); ok {\n\t\t\tc.Options.DataDirectory = path\n\t\t} else {\n\t\t\tc.Options.DataDirectory = defaultDataDirectory\n\t\t}\n'
gotest -run 'TestConfig_setDefaults|TestStrayDataDirs' ./internal/config/
restore internal/config/load.go

echo "== M4 config: StrayDataDirs không báo gì"
mutate internal/config/load.go $'\troot := projectBoundary(workingDir)\n\tfound, err := fsext.LookupBounded' $'\treturn nil\n\troot := projectBoundary(workingDir)\n\tfound, err := fsext.LookupBounded'
gotest -run 'TestStrayDataDirs' ./internal/config/
restore internal/config/load.go

echo "== M5 backend: khoá workspace theo đường (cơ chế cũ)"
mutate internal/backend/backend.go $'\tif root, err := projects.NormalizeWorkspace(abs); err == nil {\n\t\treturn root, nil\n\t}' $'\tif root, err := filepath.EvalSymlinks(abs); err == nil {\n\t\treturn root, nil\n\t}\n\t_ = projects.NormalizeWorkspace'
gotest -run 'TestPathDedupe_SubdirSharesProjectWorkspace|TestV3Exposure_T1_' ./internal/backend/
cp -p internal/backend/backend.go "$S/bak/backend-M5.go"
restore internal/backend/backend.go

echo "== M6 backend: workspace giữ đường của client mở đầu (Path: args.Path)"
mutate internal/backend/backend.go $'\t\tPath:         key,' $'\t\tPath:         args.Path,'
gotest -run 'TestPathDedupe_SubdirSharesProjectWorkspace' ./internal/backend/
restore internal/backend/backend.go

echo "== M7 backend: -D tương đối tính theo gốc dự án (bỏ SmartJoin theo args.Path)"
mutate internal/backend/backend.go $'\t\tdataDir = filepathext.SmartJoin(args.Path, dataDir)\n' $'\t\t_ = filepathext.SmartJoin\n'
gotest -run 'TestPathDedupe_SubdirSharesProjectWorkspace' ./internal/backend/
restore internal/backend/backend.go

echo "== M8 Home: bỏ qua data dir của mục registry"
mutate internal/cmd/home.go 'cmp.Or(ws.DataDir, filepath.Join(ws.Path, ".crush"))' 'cmp.Or(filepath.Join(ws.Path, ".crush"), ws.DataDir)'
gotest -run 'TestHomeSessions' ./internal/cmd/
restore internal/cmd/home.go

git diff | sha256sum
cat "$S/bak/full.sha"
