package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// Lệnh `!` người dùng gõ (chế độ local) có thể là `dew run …`: biến tin cậy
// của nó phải theo ảnh chụp env gốc, không theo `.env` của repo mà main đã nạp.
// Biến thường của `.env` vẫn tới lệnh. Chỉ in TÊN biến mang giá trị canary. Chạy ở
// tiến trình con vì ảnh chụp là toàn cục, chỉ chụp một lần.
func TestShellCommandIgnoresProjectDotenvTrustedKeys(t *testing.T) {
	if os.Getenv("SHELL_ENV_DOTENV") == "1" {
		execpolicy.Snapshot()
		require.NoError(t, godotenv.Load())
		wd, err := os.Getwd()
		require.NoError(t, err)
		store, err := config.Load(wd, filepath.Join(wd, ".crush"), false)
		require.NoError(t, err)
		resp, err := NewAppWorkspace(nil, store).AgentRunShellCommand(t.Context(), "", "env", 0, nil, false)
		require.NoError(t, err)
		for line := range strings.SplitSeq(resp.Output, "\n") {
			if k, v, ok := strings.Cut(strings.TrimRight(line, "\r"), "="); ok && strings.Contains(v, "CANARY_DOTENV") {
				fmt.Println("NAME:" + k)
			}
		}
		return
	}
	dir := t.TempDir()
	dotenv := "AGENT_CLI_DELEGATION=CANARY_DOTENV\nAGENT_CLI_ALLOWED_MODELS=claude_code:CANARY_DOTENV-5\n" +
		"AGENT_CLI_ENV_FILE=/CANARY_DOTENV/env\nAGENT_CLI_CLAUDE_BIN=/CANARY_DOTENV/claude\n" +
		"AGENT_CLI_CODEX_SANDBOX=CANARY_DOTENV\nPLAIN_CANARY=CANARY_DOTENV\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(dotenv), 0o600))
	home := t.TempDir()
	var env []string
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); !strings.HasPrefix(k, "AGENT_CLI_") && !strings.HasPrefix(k, "CRUSH_") &&
			!strings.HasPrefix(k, "XDG_") && k != "HOME" && k != "PLAIN_CANARY" {
			env = append(env, kv)
		}
	}
	env = append(env, "SHELL_ENV_DOTENV=1", "HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"), "XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
		"XDG_CACHE_HOME="+filepath.Join(home, ".cache"), "CRUSH_DISABLE_PROVIDER_AUTO_UPDATE=1", "CRUSH_DISABLE_METRICS=1")
	c := exec.Command(os.Args[0], "-test.run=^TestShellCommandIgnoresProjectDotenvTrustedKeys$")
	c.Dir, c.Env = dir, env
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
