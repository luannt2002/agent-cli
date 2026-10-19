package app

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/lsp"
	"github.com/charmbracelet/crush/internal/shell"
	"github.com/stretchr/testify/require"
)

// newLiveTestApp dựng App tối thiểu đủ cho Shutdown và ghi nó là đang sống
// với gốc workspace root, như New làm.
func newLiveTestApp(t *testing.T, root string) *App {
	t.Helper()
	a := &App{LSPManager: lsp.NewManager(config.NewTestStore(&config.Config{}))}
	a.trackLive(root)
	t.Cleanup(func() { a.releaseLive() })
	return a
}

// Hai workspace chung một server (hai cwd dưới cùng một repo): đóng một cái
// chỉ dừng job nền của nó và không đóng MCP client chung; App cuối cùng dừng
// mọi job (kể cả job không thuộc workspace nào) và đóng MCP.
func TestShutdownStopsOnlyItsOwnWorkspace(t *testing.T) {
	// Không parallel: bảng job, liveApps và closeMCP là chung cả tiến trình.
	var mcpCloses atomic.Int32
	prev := closeMCP
	closeMCP = func(context.Context) error { mcpCloses.Add(1); return nil }
	t.Cleanup(func() { closeMCP = prev })

	parent := t.TempDir()
	child := filepath.Join(parent, "web")
	require.NoError(t, os.Mkdir(child, 0o755))
	first, second := newLiveTestApp(t, parent), newLiveTestApp(t, child)

	jobs := shell.GetBackgroundShellManager()
	track := func(bg *shell.BackgroundShell, err error) *shell.BackgroundShell {
		t.Helper()
		require.NoError(t, err)
		t.Cleanup(func() { _ = jobs.Kill(bg.ID) })
		return bg
	}
	firstJob := track(jobs.StartFor(context.Background(), shell.JobOwner{Root: parent}, parent, nil, "sleep 30", ""))
	secondJob := track(jobs.StartFor(context.Background(), shell.JobOwner{Root: child}, child, nil, "sleep 30", ""))
	orphan := track(jobs.Start(context.Background(), t.TempDir(), nil, "sleep 30", ""))

	first.Shutdown()
	require.True(t, firstJob.IsDone(), "closing a workspace stops its own jobs")
	require.False(t, secondJob.IsDone(), "closing a workspace must not stop another workspace's jobs")
	require.False(t, orphan.IsDone(), "only the last app stops jobs that belong to no workspace")
	require.Zero(t, mcpCloses.Load(), "MCP clients are process-wide: the second-to-last shutdown must not close them")

	second.Shutdown()
	require.True(t, secondJob.IsDone(), "closing the last workspace stops its jobs")
	require.True(t, orphan.IsDone(), "the last app stops every job")
	require.EqualValues(t, 1, mcpCloses.Load(), "the last shutdown closes MCP clients once")
}

// Nhiều App đóng cùng lúc: đúng một cái thấy mình là App cuối, nên MCP chỉ
// bị đóng một lần và không đóng khi còn App sống; Shutdown lần hai không tính.
func TestReleaseLiveExactlyOneLast(t *testing.T) {
	apps := make([]*App, 16)
	for i := range apps {
		apps[i] = &App{}
		apps[i].trackLive(t.TempDir())
	}
	var lasts atomic.Int32
	var wg sync.WaitGroup
	for _, a := range apps {
		wg.Go(func() {
			if _, last := a.releaseLive(); last {
				lasts.Add(1)
			}
		})
	}
	wg.Wait()
	require.EqualValues(t, 1, lasts.Load())

	_, last := apps[0].releaseLive()
	require.False(t, last, "a second shutdown of the same app must not count as the last one")
}
