package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/charmbracelet/crush/internal/projects"
)

func TestDataDirMismatchWarning(t *testing.T) {
	t.Parallel()

	require.Empty(t, dataDirMismatchWarning("", "/r/.crush", false), "no -D on either side")
	require.Empty(t, dataDirMismatchWarning("/r/sub/../.crush", "/r/.crush", true), "same directory")
	msg := dataDirMismatchWarning("/r/sub/x", "/r/.crush", false)
	require.Contains(t, msg, "--data-dir /r/sub/x was ignored")
	require.Contains(t, msg, "already uses /r/.crush")
	msg = dataDirMismatchWarning("", "/r/sub/x", true)
	require.Contains(t, msg, "uses the data directory /r/sub/x, chosen with --data-dir by an earlier client")
}

// dataDirCmd là lệnh tối thiểu mà connectToServer/setupLocalWorkspace đọc cờ,
// với stderr ghi vào buffer.
func dataDirCmd(t *testing.T, cwd, dataDir string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	cmd.Flags().String("cwd", "", "")
	cmd.Flags().String("data-dir", "", "")
	if cwd != "" {
		require.NoError(t, cmd.Flags().Set("cwd", cwd))
	}
	if dataDir != "" {
		require.NoError(t, cmd.Flags().Set("data-dir", dataDir))
	}
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	return cmd, &stderr
}

// Mở từ GỐC qua server: sổ dự án biết .crush cũ của thư mục con (còn
// crush.db) thì vẫn cảnh báo kèm lệnh mở lại; -D khác data dir mà workspace
// trên server đã chọn thì cảnh báo -D bị bỏ qua. Không parallel: đặt env, cwd
// và biến --host toàn cục.
func TestClientServerWarnsAboutDataDirs(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmp, "data"))
	t.Setenv("CRUSH_GLOBAL_CONFIG", filepath.Join(tmp, "config"))
	t.Setenv("CRUSH_CACHE_DIR", filepath.Join(tmp, "cache"))
	t.Setenv("CRUSH_DISABLE_METRICS", "1")
	root, err := filepath.EvalSymlinks(tmp)
	require.NoError(t, err)
	root = filepath.Join(root, "repo")
	legacy := filepath.Join(root, "sub", ".crush")
	require.NoError(t, os.MkdirAll(legacy, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "crush.db"), nil, 0o600))
	require.NoError(t, projects.Register(root, legacy)) // Như bản cũ ghi khi mở từ sub.
	dataDir := filepath.Join(root, ".crush")

	var fromFlag atomic.Bool // Workspace trên server dựng bằng -D của client đầu.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/workspaces" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "w1", "path": root, "data_dir": dataDir, "data_dir_from_flag": fromFlag.Load(),
				"config": map[string]any{"options": map[string]any{"data_directory": dataDir}, "providers": map[string]any{}},
			})
		}
	}))
	t.Cleanup(srv.Close)
	old := clientHost
	clientHost = "tcp://" + srv.Listener.Addr().String()
	t.Cleanup(func() { clientHost = old })
	t.Chdir(tmp)

	cmd, stderr := dataDirCmd(t, root, "")
	_, cleanup, err := setupClientServerWorkspace(cmd)
	require.NoError(t, err)
	cleanup()
	require.Contains(t, stderr.String(), "warning: "+legacy+" is not used")
	require.Contains(t, stderr.String(), "CRUSH_CLIENT_SERVER=0 agent-cli -c "+filepath.Dir(legacy)+" -D "+legacy)
	require.NotContains(t, stderr.String(), "--data-dir")

	cmd, stderr = dataDirCmd(t, filepath.Dir(legacy), "x")
	_, cleanup, err = setupClientServerWorkspace(cmd)
	require.NoError(t, err)
	cleanup()
	require.Contains(t, stderr.String(), "--data-dir "+filepath.Join(filepath.Dir(legacy), "x")+" was ignored")

	// Mở từ sub KHÔNG có -c (cd sub && agent-cli): tiến trình phải sang gốc,
	// nơi workspace trên server làm việc.
	t.Chdir(filepath.Dir(legacy))
	cmd, _ = dataDirCmd(t, "", "")
	_, cleanup, err = setupClientServerWorkspace(cmd)
	require.NoError(t, err)
	cleanup()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, root, wd, "process cwd is the workspace's working dir")

	require.NoError(t, os.Remove(filepath.Join(legacy, "crush.db")))
	cmd, stderr = dataDirCmd(t, root, "")
	_, cleanup, err = setupClientServerWorkspace(cmd)
	require.NoError(t, err)
	cleanup()
	require.Empty(t, stderr.String(), "no crush.db left in the old data dir: nothing to warn about")

	// Client không -D vào workspace mà client đầu dựng bằng -D: session của nó
	// không vào data dir của dự án, phải được báo.
	fromFlag.Store(true)
	cmd, stderr = dataDirCmd(t, root, "")
	_, cleanup, err = setupClientServerWorkspace(cmd)
	require.NoError(t, err)
	cleanup()
	require.Contains(t, stderr.String(), "chosen with --data-dir by an earlier client")
}

// Chế độ local giống server: mở từ thư mục con thì workspace (working dir) là
// gốc repo và DB ở gốc; .crush cũ của thư mục con được cảnh báo. Không
// parallel: đặt env và cwd.
func TestLocalWorkspaceFromSubdirIsTheProject(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	tmp := t.TempDir()
	for k, v := range map[string]string{
		"HOME": tmp, "XDG_CONFIG_HOME": filepath.Join(tmp, "xc"), "XDG_DATA_HOME": filepath.Join(tmp, "xd"),
		"XDG_CACHE_HOME": filepath.Join(tmp, "xk"), "CRUSH_GLOBAL_CONFIG": filepath.Join(tmp, "config"),
		"CRUSH_GLOBAL_DATA": filepath.Join(tmp, "data"), "CRUSH_CACHE_DIR": filepath.Join(tmp, "cache"),
		"CRUSH_DISABLE_METRICS": "1", "CRUSH_DISABLE_PROVIDER_AUTO_UPDATE": "1",
	} {
		t.Setenv(k, v)
	}
	root, err := filepath.EvalSymlinks(tmp)
	require.NoError(t, err)
	root = filepath.Join(root, "repo")
	sub := filepath.Join(root, "sub")
	legacy := filepath.Join(sub, ".crush")
	require.NoError(t, os.MkdirAll(legacy, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "crush.db"), nil, 0o600))
	out, err := exec.Command("git", "init", "-q", root).CombinedOutput()
	require.NoError(t, err, string(out))
	t.Chdir(tmp)

	cmd, stderr := dataDirCmd(t, sub, "")
	ws, cleanup, err := setupLocalWorkspace(cmd)
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.Equal(t, root, ws.WorkingDir())
	// Tiến trình cũng ở gốc: @-mention, đính kèm và completion của TUI đọc cwd
	// tiến trình; ở sub thì đường tương đối theo sub mà tool ghép theo gốc.
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.Equal(t, root, wd, "process cwd is the workspace's working dir")
	require.Equal(t, filepath.Join(root, ".crush"), ws.Config().Options.DataDirectory)
	require.FileExists(t, filepath.Join(root, ".crush", "crush.db"))
	require.Contains(t, stderr.String(), "warning: "+legacy+" is not used")
}
