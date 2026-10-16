package loop

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFileJournal_AppendReadAndReopen(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "sub", "run.jsonl")

	j, err := OpenFileJournal(path)
	require.NoError(t, err)
	require.NoError(t, j.Append(Event{Time: time.Unix(1, 0), Kind: "start", Note: "n"}))
	require.NoError(t, j.Append(Event{Time: time.Unix(2, 0), Kind: "verify", Attempt: 1, Status: "FAIL", ExitCode: 1, Output: "boom"}))
	require.NoError(t, j.Close())

	// Mở lại và nối tiếp (tiến trình khởi động lại giữa chừng).
	j, err = OpenFileJournal(path)
	require.NoError(t, err)
	require.NoError(t, j.Append(Event{Time: time.Unix(3, 0), Kind: "stop", Reason: "GREEN"}))
	require.NoError(t, j.Close())

	events, err := ReadJournal(path)
	require.NoError(t, err)
	require.Len(t, events, 3)
	require.Equal(t, "start", events[0].Kind)
	require.Equal(t, "boom", events[1].Output)
	require.Equal(t, 1, events[1].ExitCode)
	require.Equal(t, "GREEN", events[2].Reason)

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "journal chứa output lệnh: chỉ chủ đọc")
}

func TestReadJournal_SkipsTruncatedLastLine(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "run.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"kind":"start"}`+"\n"+`{"kind":"verify","sta`), 0o600))

	events, err := ReadJournal(path)
	require.NoError(t, err)
	require.Len(t, events, 1, "dòng bị cắt do kill giữa chừng không làm hỏng các dòng trước")
	require.Equal(t, "start", events[0].Kind)
}
