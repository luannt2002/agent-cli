#!/usr/bin/env bash
# Kiểm răng vòng 2: mỗi mutant hoàn đúng một sửa mới, chạy test của nó, rồi chép lại tệp gốc từ bản sao.
set -u
W=/home/luannt/application/opensource/agent-cli-p2c
S=/tmp/claude-1000/-home-luannt-application/1ba1a44b-c74c-4a9d-b148-f58af7166a87/scratchpad/p2c
B=$S/bak2
cd "$W"
mkdir -p "$B"
FILES="internal/config/load.go internal/cmd/root.go internal/cmd/home.go internal/projects/projects.go internal/backend/backend.go internal/ui/home/view.go"
for f in $FILES; do mkdir -p "$B/$(dirname $f)"; cp -p "$f" "$B/$f"; done
before=$(git diff | sha256sum)
gotest() { flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off CGO_ENABLED=0 GOEXPERIMENT=greenteagc go test -p 2 -count=1 "$@" 2>&1 | grep -E "^(--- FAIL|ok|FAIL)|Error:|expected|actual|Messages|should" | head -8; }
mutate() { python3 -I - "$1" "$2" "$3" <<'PY'
import sys
p, old, new = sys.argv[1:]
s = open(p).read()
assert s.count(old) == 1, ("no unique match", p, old)
open(p, "w").write(s.replace(old, new))
PY
}
restore() { cp -p "$B/$1" "$W/$1"; }

echo "== R1 local: workspace ở thư mục mở (config.Init(cwd))"
mutate internal/cmd/root.go 'store, err := config.InitProject(cwd, dataDir, debug)' 'store, err := config.Init(cwd, dataDir, debug)'
gotest -run 'TestLocalWorkspaceFromSubdirIsTheProject' ./internal/cmd/
restore internal/cmd/root.go

echo "== R2 data_directory của cấu hình ghép theo thư mục mở"
mutate internal/config/load.go 'filepathext.SmartJoin(ProjectRoot(workingDir), cmp.Or(' 'filepathext.SmartJoin(workingDir, cmp.Or('
gotest -run 'TestDataDirFromSubdirectory' ./internal/config/
restore internal/config/load.go

echo "== R3 InitProject: -D tương đối ghép theo gốc (bỏ SmartJoin theo dir)"
mutate internal/config/load.go $'\t\tdataDir = filepathext.SmartJoin(dir, dataDir)\n' $'\t\t_ = dir\n'
gotest -run 'TestPathDedupe_SubdirSharesProjectWorkspace' ./internal/backend/
restore internal/config/load.go

echo "== R4 NormalizeWorkspace không dùng ProjectRoot (trả đường vật lý)"
mutate internal/projects/projects.go $'\treturn config.ProjectRoot(abs), nil\n' $'\treturn abs, nil\n'
gotest -run 'TestSubdirSharesEntryWithRepoRoot|TestRegisterKeepsKnownDataDirs' ./internal/projects/
restore internal/projects/projects.go

echo "== R5 hỏi git không có hạn"
mutate internal/config/load.go 'context.WithTimeout(context.Background(), worktreeRootTimeout)' 'context.WithCancel(context.Background())'
gotest -run 'TestProjectRootGitTimeout' ./internal/config/
restore internal/config/load.go

echo "== R6 ProjectRoot ngoài git trả đường theo chữ (bỏ EvalSymlinks)"
mutate internal/config/load.go $'\tif resolved, err := filepath.EvalSymlinks(abs); err == nil {\n\t\treturn resolved\n\t}\n\treturn abs\n}\n\n// InitProject' $'\treturn abs\n}\n\n// InitProject'
gotest -run 'TestProjectRoot$' ./internal/config/
restore internal/config/load.go

echo "== R7 LegacyDataDirs bỏ qua sổ dự án (known)"
mutate internal/config/load.go 'for _, d := range append(found, known...) {' 'for _, d := range found {'
gotest -run 'TestLegacyDataDirs' ./internal/config/
gotest -run 'TestClientServerWarnsAboutDataDirs' ./internal/cmd/
restore internal/config/load.go

echo "== R8 Register xoá locations (như vòng 1)"
mutate internal/projects/projects.go $'\t\t\tLocations:    locs,\n' ''
gotest -run 'TestRegisterKeepsKnownDataDirs' ./internal/projects/
restore internal/projects/projects.go

echo "== R9a bỏ lời gọi cảnh báo data dir cũ ở chế độ local"
mutate internal/cmd/root.go $'\twarnLegacyDataDirs(cmd, cwd, cfg.Options.DataDirectory)\n' ''
gotest -run 'TestLocalWorkspaceFromSubdirIsTheProject' ./internal/cmd/
restore internal/cmd/root.go

echo "== R9b bỏ lời gọi cảnh báo data dir cũ qua server"
mutate internal/cmd/root.go $'\twarnLegacyDataDirs(cmd, cwd, ws.DataDir)\n' ''
gotest -run 'TestClientServerWarnsAboutDataDirs' ./internal/cmd/
restore internal/cmd/root.go

echo "== R10a bỏ lời gọi cảnh báo -D lệch"
mutate internal/cmd/root.go $'\tif msg := dataDirMismatchWarning(requested, ws.DataDir, ws.DataDirFromFlag); msg != "" {\n\t\twarnUser(cmd, msg)\n\t}\n' $'\t_ = requested\n'
gotest -run 'TestClientServerWarnsAboutDataDirs|TestDataDirMismatchWarning' ./internal/cmd/
restore internal/cmd/root.go

echo "== R10b server không báo data dir từ -D"
mutate internal/backend/backend.go $'\tws.dataDirFromFlag = args.DataDir != ""\n' ''
gotest -run 'TestPathDedupe_SubdirSharesProjectWorkspace' ./internal/backend/
restore internal/backend/backend.go

echo "== R11 Home tin data_dir của sổ (mục cũ sub/.crush)"
mutate internal/cmd/home.go 'if dataDir == "" || filepath.Base(dataDir) == ".crush" {' 'if dataDir == "" {'
gotest -run 'TestHomeSourceAdd' ./internal/cmd/
restore internal/cmd/home.go

echo "== R12 Home không báo .crush cũ"
mutate internal/ui/home/view.go $'\tif m.screen != screenSessions || len(m.ws.Legacy) == 0 {' $'\tif true {'
gotest -run 'TestLegacyDataDirNote' ./internal/ui/home/
restore internal/ui/home/view.go

echo "== khôi phục"
echo "before: $before"
echo "after:  $(git diff | sha256sum)"
