package backend

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// Lệnh `!` ở chế độ server chạy với env của server cộng env client gửi lên
// (ws.Env, client chụp SAU khi nạp `.env` của repo): biến tin cậy của lệnh phải
// theo ảnh chụp env gốc của server, để `!dew run …` không nhận giao việc hay
// allowlist từ `.env`. Biến thường vẫn tới lệnh. Chỉ in TÊN biến mang giá trị
// canary. Chạy ở tiến trình con vì ảnh chụp là toàn cục, chỉ chụp một lần.
func TestRunShellCommandIgnoresProjectDotenvTrustedKeys(t *testing.T) {
	if os.Getenv("BACKEND_SHELL_DOTENV") == "1" {
		execpolicy.Snapshot()
		require.NoError(t, godotenv.Load())
		b, _ := newTestBackend(t)
		wd, err := os.Getwd()
		require.NoError(t, err)
		ws := &Workspace{
			ID:           uuid.New().String(),
			Path:         wd,
			resolvedPath: wd,
			Env:          os.Environ(), // như root.go gửi trong request tạo workspace
			clients:      make(map[string]*clientState),
			shutdownFn:   func() {},
		}
		ws.ctx, ws.cancel = context.WithCancel(b.ctx)
		InsertWorkspaceForTest(b, ws)
		resp, err := b.RunShellCommand(t.Context(), ws.ID, proto.ShellCommandRequest{Command: "env"})
		require.NoError(t, err)
		for line := range strings.SplitSeq(resp.Output, "\n") {
			if k, v, ok := strings.Cut(strings.TrimRight(line, "\r"), "="); ok && strings.Contains(v, "CANARY_DOTENV") {
				fmt.Println("NAME:" + k)
			}
		}
		return
	}
	dir := t.TempDir()
	dotenv := "DEW_DELEGATION=CANARY_DOTENV\nDEW_ALLOWED_MODELS=claude_code:CANARY_DOTENV-5\n" +
		"DEW_ENV_FILE=/CANARY_DOTENV/env\nDEW_CLAUDE_BIN=/CANARY_DOTENV/claude\n" +
		"DEW_CODEX_SANDBOX=CANARY_DOTENV\nPLAIN_CANARY=CANARY_DOTENV\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(dotenv), 0o600))
	var env []string
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); !strings.HasPrefix(k, "DEW_") && k != "PLAIN_CANARY" {
			env = append(env, kv)
		}
	}
	c := exec.Command(os.Args[0], "-test.run=^TestRunShellCommandIgnoresProjectDotenvTrustedKeys$")
	c.Dir, c.Env = dir, append(env, "BACKEND_SHELL_DOTENV=1")
	out, err := c.CombinedOutput()
	require.NoError(t, err, string(out)) // Tiến trình con chỉ in tên biến.

	var got []string
	for line := range strings.SplitSeq(string(out), "\n") {
		if name, ok := strings.CutPrefix(line, "NAME:"); ok {
			got = append(got, name)
		}
	}
	require.Equal(t, []string{"PLAIN_CANARY"}, got, "trusted keys from a project .env must not reach a ! command")
}
