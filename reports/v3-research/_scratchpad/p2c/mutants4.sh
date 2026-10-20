#!/usr/bin/env bash
# Kiểm răng vòng 4 (loop): mỗi mutant hoàn một phần, chạy test, chép lại tệp gốc từ bản sao.
set -u
W=/home/luannt/application/opensource/agent-cli-p2c
S=/tmp/claude-1000/-home-luannt-application/1ba1a44b-c74c-4a9d-b148-f58af7166a87/scratchpad/p2c
B=$S/bak4
cd "$W"
for f in internal/cmd/loop.go internal/loop/loop.go; do mkdir -p "$B/$(dirname $f)"; cp -p "$f" "$B/$f"; done
before=$(git diff | sha256sum)
gotest() { flock /tmp/claude-1000/agent-go.lock env GOMAXPROCS=2 GOFLAGS=-mod=readonly GOPROXY=off CGO_ENABLED=0 GOEXPERIMENT=greenteagc go test -p 2 -count=1 "$@" 2>&1 | grep -E "^(--- FAIL|ok|FAIL)|Error:|Messages|does not contain|Should not" | head -8; }
mutate() { python3 -I - "$1" "$2" "$3" <<'PY'
import sys
p, old, new = sys.argv[1:]
s = open(p).read()
assert s.count(old) == 1, ("no unique match", p, old)
open(p, "w").write(s.replace(old, new))
PY
}
restore() { cp -p "$B/$1" "$W/$1"; }

echo "== V1 cmd không đặt VerifyDir (prompt không nêu thư mục verify)"
mutate internal/cmd/loop.go $'\t\tverifyDir = filepath.ToSlash(rel) + "/"\n' $'\t\t_ = rel\n'
gotest -run 'TestExecuteLoop_VerifyDirIsNamedInThePrompt' ./internal/cmd/
restore internal/cmd/loop.go

echo "== V2 dấu vân tay tính ở thư mục mở (như cũ)"
mutate internal/cmd/loop.go 'Fingerprint: gitFingerprint(root),' 'Fingerprint: gitFingerprint(p.Cwd),'
gotest -run 'TestExecuteLoop_FingerprintCoversTheWholeRepo' ./internal/cmd/
restore internal/cmd/loop.go

echo "== V3 prompt đổi cả khi VerifyDir rỗng"
mutate internal/loop/loop.go $'\tin := ""\n\tif s.VerifyDir != "" {\n\t\tin = " in `" + s.VerifyDir + "`"\n\t}' $'\tin := " in `" + s.VerifyDir + "`"'
gotest -run 'TestPromptsNameTheVerifyDirOnlyWhenItDiffers' ./internal/loop/
restore internal/loop/loop.go

echo "== V4 lượt sửa không nêu thư mục verify"
mutate internal/loop/loop.go $'\t\tin = " (run in `" + s.VerifyDir + "`)"\n' ''
gotest -run 'TestPromptsNameTheVerifyDirOnlyWhenItDiffers' ./internal/loop/
gotest -run 'TestExecuteLoop_VerifyDirIsNamedInThePrompt' ./internal/cmd/
restore internal/loop/loop.go

echo "before: $before"
echo "after:  $(git diff | sha256sum)"
