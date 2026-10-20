#!/usr/bin/env bash
# Kiểm răng vòng 3 (mục 1, 4, 5): mỗi mutant hoàn một sửa, chạy test, chép lại tệp gốc từ bản sao.
set -u
W=/home/luannt/application/opensource/agent-cli-p2c
S=/tmp/claude-1000/-home-luannt-application/1ba1a44b-c74c-4a9d-b148-f58af7166a87/scratchpad/p2c
B=$S/bak3
cd "$W"
FILES="internal/cmd/root.go internal/config/load.go internal/backend/backend.go internal/workspace/client_workspace.go"
for f in $FILES; do mkdir -p "$B/$(dirname $f)"; cp -p "$f" "$B/$f"; done
before=$(git diff | sha256sum)
gotest() { flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off CGO_ENABLED=0 GOEXPERIMENT=greenteagc go test -p 2 -count=1 "$@" 2>&1 | grep -E "^(--- FAIL|ok|FAIL)|Error:|expected|actual|Messages" | head -8; }
mutate() { python3 -I - "$1" "$2" "$3" <<'PY'
import sys
p, old, new = sys.argv[1:]
s = open(p).read()
assert s.count(old) == 1, ("no unique match", p, old)
open(p, "w").write(s.replace(old, new))
PY
}
restore() { cp -p "$B/$1" "$W/$1"; }

echo "== U1a local: không chuyển tiến trình sang working dir"
mutate internal/cmd/root.go $'\tenterWorkspaceDir(store.WorkingDir())\n' ''
gotest -run 'TestLocalWorkspaceFromSubdirIsTheProject' ./internal/cmd/
restore internal/cmd/root.go

echo "== U1b server: không chuyển tiến trình sang ws.Path"
mutate internal/cmd/root.go $'\tenterWorkspaceDir(ws.Path)\n' ''
gotest -run 'TestClientServerWarnsAboutDataDirs' ./internal/cmd/
restore internal/cmd/root.go

echo "== U4a nhớ cả lần git quá hạn"
mutate internal/config/load.go $'\tif !ok {\n\t\t// Git quá hạn' $'\tif false && !ok {\n\t\t// Git quá hạn'
gotest -run 'TestProjectRootGitTimeout' ./internal/config/
restore internal/config/load.go

echo "== U4b backend tính lại gốc thay vì dùng khoá"
mutate internal/backend/backend.go 'config.InitProject(key, args.Path, args.DataDir, args.Debug)' 'config.InitProject(config.ProjectRoot(args.Path), args.Path, args.DataDir, args.Debug)'
gotest -run 'TestCreateWorkspace_GitTimeoutKeepsKeyAndConfigTogether' ./internal/backend/
restore internal/backend/backend.go

echo "== U5 recreateArgs luôn gửi DataDir"
mutate internal/workspace/client_workspace.go $'\tif ws.DataDirFromFlag {\n\t\tdataDir = ws.DataDir\n\t}' $'\tdataDir = ws.DataDir'
gotest -run 'TestRecreateArgsSendsDataDirOnlyFromFlag' ./internal/workspace/
restore internal/workspace/client_workspace.go

echo "== khôi phục"
echo "before: $before"
echo "after:  $(git diff | sha256sum)"
