package cmd

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/charmbracelet/crush/internal/proto"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Miễn so chính sách chỉ dành cho lệnh không gửi prompt. Duyệt TOÀN cây lệnh:
// tập lệnh được miễn phải đúng bằng danh sách này, nên lệnh con mới thêm dưới
// jobs/login/logout (hay annotation gắn nhầm lên lệnh chạy model) làm test đỏ.
func TestRunsNoAgentExactCommandSet(t *testing.T) {
	t.Parallel()
	var got []string
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		if runsNoAgent(c) {
			got = append(got, c.CommandPath())
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(rootCmd)
	require.ElementsMatch(t, []string{
		"agent-cli jobs", "agent-cli jobs output", "agent-cli jobs kill", "agent-cli login", "agent-cli logout",
	}, got)
	for _, c := range []*cobra.Command{rootCmd, runCmd, sendCmd, loopCmd} {
		require.False(t, runsNoAgent(c), c.CommandPath())
	}
}

// connectToServer gửi đúng cờ của lệnh lên dây: lệnh chạy model gửi
// no_agent_runs=false (bị so), lệnh không prompt (kể cả lệnh con) gửi true.
func TestConnectToServerSendsNoAgentRuns(t *testing.T) {
	isolateServerCache(t)
	var mu sync.Mutex
	var bodies []proto.Workspace
	dataDir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/workspaces" {
			var req proto.Workspace
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			mu.Lock()
			bodies = append(bodies, req)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "w1", "path": req.Path, "data_dir": dataDir,
				"config": map[string]any{"options": map[string]any{"data_directory": dataDir, "disable_metrics": true}},
			})
		}
	}))
	t.Cleanup(srv.Close)
	old := clientHost
	clientHost = "tcp://" + srv.Listener.Addr().String()
	t.Cleanup(func() { clientHost = old })
	t.Chdir(t.TempDir())

	model := &cobra.Command{Use: "run"}
	noPrompt := &cobra.Command{Use: "jobs", Annotations: map[string]string{noAgentRunsAnnotation: "true"}}
	sub := &cobra.Command{Use: "kill"}
	noPrompt.AddCommand(sub)
	for _, tc := range []struct {
		cmd  *cobra.Command
		want bool
	}{{model, false}, {noPrompt, true}, {sub, true}} {
		tc.cmd.SetContext(t.Context())
		_, _, cleanup, err := connectToServer(tc.cmd)
		require.NoError(t, err)
		cleanup()
		mu.Lock()
		last := bodies[len(bodies)-1]
		mu.Unlock()
		require.Equal(t, tc.want, last.NoAgentRuns, tc.cmd.CommandPath())
		// Lệnh chạy model mang chính sách tin cậy để server so.
		require.Equal(t, !tc.want, last.TrustedEnv != nil, tc.cmd.CommandPath())
	}
}

// isolateServerCache trỏ thư mục cache toàn cục (log server-<host> mà ensureServer
// dựng, khoá spawn) vào thư mục tạm của test thay vì ~/.cache thật, và trả lại
// logger mặc định khi test xong: crushlog.Setup chỉ chạy một lần mỗi tiến trình, test
// sau mà còn ghi vào tệp đó sẽ dựng lại thư mục tạm đã xoá. Không parallel: đặt env.
func isolateServerCache(t *testing.T) {
	t.Helper()
	t.Setenv("CRUSH_CACHE_DIR", t.TempDir())
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
}
