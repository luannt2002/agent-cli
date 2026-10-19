package provider

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// steeringEnv là các biến lái endpoint, khoá và thư mục cấu hình của CLI con.
var steeringEnv = []string{
	"ANTHROPIC_BASE_URL", "ANTHROPIC_API_KEY", "OPENAI_BASE_URL", "OPENAI_API_KEY", "CLAUDE_CONFIG_DIR", "CODEX_HOME",
}

// CLI con được giao việc (claude/codex) nhận env GỐC của agent-cli: biến lái endpoint,
// khoá hay thư mục cấu hình của nó mà `.env` của repo (main nạp sau ảnh chụp) hoặc
// `env` của cấu hình dự án đặt thì không tới CLI con; giá trị người dùng export thì giữ.
// CLI giả chỉ in TÊN biến kèm nguồn, không in giá trị. Chạy ở tiến trình con vì ảnh
// chụp là toàn cục, chỉ chụp một lần.
func TestRunCommandIgnoresProjectDotenv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script CLI")
	}
	if os.Getenv("PROVIDER_ENV_CHILD") == "1" {
		execpolicy.Snapshot()
		require.NoError(t, godotenv.Load())
		// `env` của crush.json dự án được áp bằng os.Setenv, đè cả biến gốc.
		require.NoError(t, os.Setenv("OPENAI_API_KEY", "CANARY_CONFIGENV"))
		out, err := runCommand(context.Background(), ".", "./fake-cli")
		require.NoError(t, err)
		fmt.Print(string(out))
		return
	}
	dir := t.TempDir()
	dotenv := "ANTHROPIC_BASE_URL=http://CANARY_DOTENV\nANTHROPIC_API_KEY=CANARY_DOTENV\nOPENAI_BASE_URL=http://CANARY_DOTENV\n" +
		"OPENAI_API_KEY=CANARY_DOTENV\nCLAUDE_CONFIG_DIR=/CANARY_DOTENV\nCODEX_HOME=/CANARY_DOTENV\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(dotenv), 0o600))
	// CLI giả: in "NAME:<biến>:<nguồn>" cho biến mang giá trị canary, không in giá trị.
	cli := "#!/bin/sh\nenv | while IFS= read -r line; do\n  k=${line%%=*}; v=${line#*=}\n  case \"$v\" in\n" +
		"    *CANARY_DOTENV*) echo \"NAME:$k:dotenv\";;\n    *CANARY_CONFIGENV*) echo \"NAME:$k:configenv\";;\n" +
		"    *CANARY_ORIGINAL*) echo \"NAME:$k:original\";;\n  esac\ndone\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "fake-cli"), []byte(cli), 0o700))

	var env []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if !strings.HasPrefix(k, "AGENT_CLI_") && !slices.Contains(steeringEnv, k) {
			env = append(env, kv)
		}
	}
	// Giá trị người dùng export ở shell: phải tới CLI con, kể cả khi cấu hình dự án đè nó.
	env = append(env, "PROVIDER_ENV_CHILD=1", "OPENAI_API_KEY=CANARY_ORIGINAL")
	c := exec.Command(os.Args[0], "-test.run=^TestRunCommandIgnoresProjectDotenv$")
	c.Dir, c.Env = dir, env
	out, err := c.CombinedOutput()
	require.NoError(t, err, string(out)) // Tiến trình con chỉ in tên biến.

	var got []string
	for line := range strings.SplitSeq(string(out), "\n") {
		if name, ok := strings.CutPrefix(line, "NAME:"); ok {
			got = append(got, name)
		}
	}
	require.Equal(t, []string{"OPENAI_API_KEY:original"}, got,
		"only the user's own exported value reaches the CLI; nothing from the project .env or config env")
}
