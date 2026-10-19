package shell

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Server giữ nhiều workspace chung một bảng job: dừng job của một workspace
// không được dừng job của workspace khác, kể cả workspace lồng bên trong
// (hai cwd khác nhau dưới cùng một repo).
func TestKillOwnedLeavesOtherWorkspaceJobsRunning(t *testing.T) {
	t.Parallel()
	m := newBackgroundShellManager()
	parent := t.TempDir()
	child := filepath.Join(parent, "web")
	require.NoError(t, osMkdir(child))

	mine, err := m.StartFor(context.Background(), JobOwner{SessionID: "a", Root: parent}, parent, nil, "sleep 30", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = m.Kill(mine.ID) })
	theirs, err := m.StartFor(context.Background(), JobOwner{SessionID: "b", Root: child}, child, nil, "sleep 30", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = m.Kill(theirs.ID) })

	m.KillOwned(t.Context(), "") // Gốc rỗng không phải "mọi job" như Jobs("").
	require.False(t, mine.IsDone(), "an empty root must not stop any job")
	require.False(t, theirs.IsDone(), "an empty root must not stop any job")

	m.KillOwned(t.Context(), parent)

	require.True(t, mine.IsDone(), "job of the closed workspace must be stopped")
	require.Equal(t, JobKilled, findJob(t, m, parent, mine.ID).Status)
	require.False(t, theirs.IsDone(), "job of the other workspace must keep running")
	_, listed := m.Get(theirs.ID)
	require.True(t, listed, "job of the other workspace must stay in the job table")
}
