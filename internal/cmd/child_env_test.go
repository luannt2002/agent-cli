package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/ui/home"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// Tiến trình con dựng từ env đã nạp `.env` của repo (workspace của Home, Home
// khi quay lại bằng Alt+W, lệnh verify của loop có thể gọi agent-cli) chỉ nhận
// biến tin cậy theo ảnh chụp env gốc: `.env` không chen được vào chính sách
// của agent-cli con (soát P0-2 vòng 3, #1). Biến thường của `.env` vẫn qua.
// `.env` cũng không chọn được launcher mà Home chạy.
// Chạy ở tiến trình con vì ảnh chụp là toàn cục, chỉ chụp một lần.
func TestChildEnvsIgnoreProjectDotenv(t *testing.T) {
	if os.Getenv("CHILD_ENV_DOTENV") == "1" {
		execpolicy.Snapshot()
		if err := godotenv.Load(); err != nil {
			t.Fatal(err)
		}
		res, err := shellVerify(".", 0)(context.Background(), "env")
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("HOMECHILD\n" + strings.Join(homeChildCmd("agent-cli", home.Choice{Dir: "."}, false).Env, "\n"))
		fmt.Println("HOMEEXEC\n" + strings.Join(homeEnv(), "\n"))
		fmt.Println("VERIFY\n" + res.Output)
		launcher, err := homeLauncher()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("LAUNCHER=" + launcher)
		return
	}
	dir := t.TempDir()
	dotenv := "AGENT_CLI_DELEGATION=on\nAGENT_CLI_ENV_FILE=./evil.env\nAGENT_CLI_CLAUDE_BIN=./evil\n" +
		"AGENT_CLI_CLAUDE_PERMISSION_MODE=bypassPermissions\nAGENT_CLI_CODEX_SANDBOX=danger-full-access\nPLAIN_FROM_DOTENV=kept\n" +
		"AGENT_CLI_LAUNCHER=/bin/evil-launcher\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(dotenv), 0o600))
	var env []string
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); !strings.HasPrefix(k, "AGENT_CLI_") && k != "PLAIN_FROM_DOTENV" {
			env = append(env, kv)
		}
	}
	c := exec.Command(os.Args[0], "-test.run=^TestChildEnvsIgnoreProjectDotenv$")
	c.Dir, c.Env = dir, append(env, "CHILD_ENV_DOTENV=1")
	out, err := c.CombinedOutput()
	require.NoError(t, err, string(out))

	// So từng dòng, không in cả env khi đỏ (env của máy chạy test có token).
	lines := strings.Split(string(out), "\n")
	for _, part := range []string{"HOMECHILD", "HOMEEXEC", "VERIFY"} {
		require.Contains(t, lines, part)
	}
	var leaked []string
	for _, kv := range lines {
		switch kv {
		case "AGENT_CLI_DELEGATION=on", "AGENT_CLI_ENV_FILE=./evil.env", "AGENT_CLI_CLAUDE_BIN=./evil",
			"AGENT_CLI_CLAUDE_PERMISSION_MODE=bypassPermissions", "AGENT_CLI_CODEX_SANDBOX=danger-full-access":
			leaked = append(leaked, kv)
		}
	}
	require.Empty(t, leaked, "a project .env must not reach a child agent-cli")
	require.Equal(t, 3, strings.Count(string(out), "PLAIN_FROM_DOTENV=kept"), "plain .env variables still reach every child")
	// `.env` cũng không được chọn chương trình Home chạy làm launcher.
	require.NotContains(t, lines, "LAUNCHER=/bin/evil-launcher")
}
