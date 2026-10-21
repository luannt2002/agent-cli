package shell

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// GOFLAGS mang -modcache làm mọi `go clean` sau đó xoá module cache (`go env -w` còn ghi vĩnh
// viễn); `docker rm -v` / `compose rm -v` xoá volume vô danh của container. Không chặn nhầm:
// -modcacherw, GOPROXY, rm không -v, `--rm -v` của docker run.
func TestUnattendedDeniesModcacheViaGOFLAGSAndContainerVolumes(t *testing.T) {
	t.Parallel()
	guard := UnattendedGuard(func() bool { return true }, func() bool { return false }, nil)
	for _, args := range [][]string{
		{"env", "GOFLAGS=-modcache", "go", "clean"},
		{"env", "GOFLAGS=-modcache=true", "go", "clean"},
		{"env", "GOFLAGS=-mod=mod -modcache", "make", "clean"},
		{"sh", "-c", "GOFLAGS=-modcache go clean"},
		{"bash", "-c", "export GOFLAGS='-mod=mod --modcache'; go clean"},
		{"go", "env", "-w", "GOFLAGS=-modcache"},
		{"go", "env", "-w", "GOFLAGS=-mod=mod -modcache"},
		{"docker", "rm", "-v", "c1"},
		{"docker", "rm", "--volumes", "c1"},
		{"docker", "container", "rm", "-fv", "c1"},
		{"docker", "compose", "rm", "-v"},
		{"docker-compose", "rm", "-s", "-v"},
	} {
		require.Truef(t, guard(args), "%q phải bị chặn", args)
	}
	for _, args := range [][]string{
		{"go", "clean", "-cache"},
		{"env", "GOFLAGS=-modcacherw", "go", "build", "./..."},
		{"env", "GOFLAGS=-mod=mod", "go", "clean", "-testcache"},
		{"go", "env", "-w", "GOPROXY=https://proxy.golang.org"},
		{"go", "env", "-w", "GOFLAGS=-mod=mod"},
		{"docker", "compose", "down"},
		{"docker", "rm", "-f", "c1"},
		{"docker", "run", "--rm", "-v", "data:/d", "img", "rm", "-rf", "build"},
	} {
		require.Falsef(t, guard(args), "%q không được chặn", args)
	}
}

// Chạy thật qua trình thông dịch: GOFLAGS gán đầu lệnh hay export từ trước nằm trong env chứ
// không trong argv. `go` ở đây là bản giả ghi một tệp dấu: không người trực thì không có dấu;
// có người trực thì có (chứng minh lệnh thật sự chạy tới `go`, test không mù).
func TestUnattendedDeniesGOFLAGSModcacheEndToEnd(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("fake go is a shell script")
	}
	for name, command := range map[string]string{
		"assign prefix":    `GOFLAGS=-modcache go clean`,
		"export then go":   `export GOFLAGS=-modcache; go clean`,
		"env launcher":     `env GOFLAGS=-modcache go clean`,
		"nested sh -c":     `sh -c 'GOFLAGS=-modcache go clean'`,
		"persistent -w":    `go env -w GOFLAGS=-modcache`,
		"export then make": `export GOFLAGS="-mod=mod -modcache"; sh -c 'go clean'`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			bin := t.TempDir()
			mark := filepath.Join(bin, "ran")
			require.NoError(t, os.WriteFile(filepath.Join(bin, "go"), []byte("#!/bin/sh\ntouch "+mark+"\n"), 0o755))
			env := []string{"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"), "HOME=" + t.TempDir()}
			run := func(unattended bool) error {
				guard := UnattendedGuard(func() bool { return unattended }, func() bool { return false }, nil)
				return Run(context.Background(), RunOptions{Command: command, Cwd: t.TempDir(), Env: env, BlockFuncs: []BlockFunc{guard}})
			}
			err := run(true)
			require.Error(t, err)
			require.NoFileExists(t, mark, "blocked before reaching go")
			require.NoError(t, run(false))
			require.FileExists(t, mark, "attended, the same command reaches go")
		})
	}
}
