package log

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Panic log từng được tạo ở THƯ MỤC HIỆN TẠI (thường là repo đang làm) với quyền
// mặc định: dễ bị commit nhầm stack trace. Nay ghi vào thư mục log, quyền 0600; và
// cleanup (khôi phục terminal) luôn chạy kể cả khi không tạo được tệp.
func TestRecoverPanicWritesToTheLogDirNotTheWorkingDir(t *testing.T) {
	logs, cwd := t.TempDir(), t.TempDir()
	t.Chdir(cwd)
	prev, _ := panicDir.Load().(string)
	panicDir.Store(logs)
	t.Cleanup(func() { panicDir.Store(prev) })

	cleaned := false
	func() {
		defer RecoverPanic("probe", func() { cleaned = true })
		panic("boom")
	}()

	require.True(t, cleaned)
	entries, err := os.ReadDir(cwd)
	require.NoError(t, err)
	require.Empty(t, entries, "nothing is written into the working directory")
	matches, err := filepath.Glob(filepath.Join(logs, "crush-panic-probe-*.log"))
	require.NoError(t, err)
	require.Len(t, matches, 1)
	info, err := os.Stat(matches[0])
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestRecoverPanicRunsCleanupWhenTheFileCannotBeCreated(t *testing.T) {
	prev, _ := panicDir.Load().(string)
	panicDir.Store(filepath.Join(t.TempDir(), "missing", "dir"))
	t.Cleanup(func() { panicDir.Store(prev) })

	cleaned := false
	func() {
		defer RecoverPanic("probe", func() { cleaned = true })
		panic("boom")
	}()
	require.True(t, cleaned)
}
