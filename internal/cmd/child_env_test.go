package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/ui/home"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// Tiến trình con dựng từ env đã nạp `.env` của repo không nhận biến tin cậy từ `.env`
// (soát P0-2 vòng 3, #1). Workspace của Home, Home khi quay lại bằng Alt+W và server
// của dự án nhận env GỐC: không biến nào của `.env` (kể cả biến thường như khoá API,
// launcher), vì chúng tự nạp lại `.env` của cùng cwd và env gốc của chúng là thứ CLI
// con giao việc nhận (soát P0-2b vòng 2, #1). Verify của loop chỉ đặt lại biến tin cậy.
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
		fmt.Println("HOMECHILD\n" + strings.Join(homeChildCmd("dew", home.Choice{Dir: "."}, false).Env, "\n"))
		fmt.Println("HOMEEXEC\n" + strings.Join(homeEnv(), "\n"))
		fmt.Println("SERVER\n" + strings.Join(serverEnv(), "\n"))
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
	run := func(extra ...string) []string {
		c := exec.Command(os.Args[0], "-test.run=^TestChildEnvsIgnoreProjectDotenv$")
		c.Dir, c.Env = dir, append(append(slices.Clone(env), "CHILD_ENV_DOTENV=1"), extra...)
		out, err := c.CombinedOutput()
		require.NoError(t, err, withoutEnvLines(string(out)))
		return strings.Split(string(out), "\n")
	}

	// So từng dòng, không in cả env khi đỏ (env của máy chạy test có token).
	fromDotenv := []string{"AGENT_CLI_DELEGATION=on", "AGENT_CLI_ENV_FILE=./evil.env", "AGENT_CLI_CLAUDE_BIN=./evil",
		"AGENT_CLI_CLAUDE_PERMISSION_MODE=bypassPermissions", "AGENT_CLI_CODEX_SANDBOX=danger-full-access",
		"PLAIN_FROM_DOTENV=kept", "AGENT_CLI_LAUNCHER=/bin/evil-launcher"}
	lines := run()
	for _, part := range []string{"HOMECHILD", "HOMEEXEC", "SERVER"} {
		var leaked []string
		for _, kv := range envSection(lines, part) {
			if slices.Contains(fromDotenv, kv) {
				leaked = append(leaked, kv)
			}
		}
		require.Empty(t, leaked, "%s: no variable of a project .env may reach it", part)
	}
	// Không dùng Contains trên cả env: khi đỏ, testify in nguyên slice (env của máy có token).
	verify := envSection(lines, "VERIFY")
	var leaked []string
	for _, kv := range verify {
		if slices.Contains(fromDotenv[:5], kv) {
			leaked = append(leaked, kv)
		}
	}
	require.Empty(t, leaked, "VERIFY: trusted keys of a project .env must not reach it")
	require.True(t, slices.Contains(verify, "PLAIN_FROM_DOTENV=kept"), "VERIFY: plain .env variables still reach it")
	// `.env` cũng không được chọn chương trình Home chạy làm launcher.
	require.False(t, slices.Contains(lines, "LAUNCHER=/bin/evil-launcher"))
	// Launcher gốc của người dùng thì giữ, đúng một lần trong env của Home.
	launcher := filepath.Join(dir, "launcher")
	require.NoError(t, os.WriteFile(launcher, []byte("#!/bin/sh\n"), 0o700))
	lines = run(launcherEnv + "=" + launcher)
	for _, part := range []string{"HOMECHILD", "HOMEEXEC"} {
		var got []string
		for _, kv := range envSection(lines, part) {
			if strings.HasPrefix(kv, launcherEnv+"=") {
				got = append(got, kv)
			}
		}
		require.Equal(t, []string{launcherEnv + "=" + launcher}, got, part)
	}
	require.True(t, slices.Contains(lines, "LAUNCHER="+launcher))
}

// envSection trả các dòng env in sau tiêu đề name, tới tiêu đề kế tiếp.
func envSection(lines []string, name string) []string {
	start := slices.Index(lines, name) + 1
	end := start
	for end < len(lines) && !slices.Contains([]string{"HOMECHILD", "HOMEEXEC", "SERVER", "VERIFY"}, lines[end]) &&
		!strings.HasPrefix(lines[end], "LAUNCHER=") {
		end++
	}
	return lines[start:end]
}

// withoutEnvLines bỏ các dòng dạng KEY=value khỏi output trước khi in lúc đỏ: env của máy
// chạy test có token.
func withoutEnvLines(out string) string {
	var keep []string
	for _, line := range strings.Split(out, "\n") {
		if k, _, ok := strings.Cut(line, "="); !ok || k == "" || strings.ContainsAny(k, " \t") {
			keep = append(keep, line)
		}
	}
	return strings.Join(keep, "\n")
}
