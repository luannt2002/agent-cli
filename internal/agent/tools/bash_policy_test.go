package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"mvdan.cc/sh/v3/syntax"

	"github.com/charmbracelet/crush/internal/shell"
	"github.com/stretchr/testify/require"
)

// Lệnh bash gọi thẳng CLI agent khác bị chặn ngay trong shell (cả khi lồng
// trong chuỗi lệnh); lệnh thường vẫn chạy.
func TestBashBlocksExternalAgentCLIs(t *testing.T) {
	t.Parallel()
	sh := shell.NewShell(&shell.Options{WorkingDir: t.TempDir(), BlockFuncs: blockFuncs()})
	for _, cmd := range []string{
		"claude -p hi",
		"true && codex exec x",
		"env FOO=1 gemini -p x",
		"npx -y @anthropic-ai/claude-code -p x",
	} {
		_, _, err := sh.Exec(t.Context(), cmd)
		require.Errorf(t, err, "%q", cmd)
		require.Containsf(t, err.Error(), "not allowed", "%q", cmd)
	}
	out, _, err := sh.Exec(t.Context(), "echo claude")
	require.NoError(t, err)
	require.Equal(t, "claude\n", out)
}

// Script có shebang gọi qua đường dẫn (vd ~/.local/bin/claude là script bash)
// được chạy thẳng bằng os/exec: vẫn phải qua bộ chặn trước khi chạy.
func TestBashBlocksPathPrefixedAgentScripts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	marker := filepath.Join(dir, "ran")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "bin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bin", "claude"), []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0o755))
	sh := shell.NewShell(&shell.Options{WorkingDir: dir, BlockFuncs: blockFuncs()})
	for _, cmd := range []string{"./bin/claude -p hi", filepath.Join(dir, "bin", "claude") + " -p hi"} {
		_, _, err := sh.Exec(t.Context(), cmd)
		require.Errorf(t, err, "%q", cmd)
		require.Containsf(t, err.Error(), "not allowed", "%q", cmd)
	}
	require.NoFileExists(t, marker)
}

// CLI giả chỉ ghi sentinel: chặn trước interpreter thì không có tác dụng phụ.
func TestBashBlocksInterpreterAgentLaunch(t *testing.T) {
	t.Parallel()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node không có trên máy test")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "codex")
	marker := filepath.Join(dir, "ran")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\nprintf ran > '"+marker+"'\n"), 0o755))
	sh := shell.NewShell(&shell.Options{WorkingDir: dir, BlockFuncs: blockFuncs()})
	code := "require('child_process').execFileSync(" + strconv.Quote(bin) + ", ['exec', 'fixture'])"
	quoted, err := syntax.Quote(code, syntax.LangBash)
	require.NoError(t, err)
	_, _, err = sh.Exec(t.Context(), node+" -e "+quoted)
	require.ErrorContains(t, err, "not allowed")
	require.NoFileExists(t, marker)
	out, _, err := sh.Exec(t.Context(), node+` -e 'console.log("codex")'`)
	require.NoError(t, err)
	require.Equal(t, "codex\n", out)
}

// Bộ chặn tự giết tiến trình điều khiển phải được NỐI vào shell mà tool bash
// dùng (blockFuncs). Xoá dòng nối ở bash.go thì test này đỏ. `kill` trần là
// builtin chưa cài đặt của trình thông dịch (không giết được ai, không qua
// BlockFunc) nên test dùng các đường giết thật: /bin/kill, env/xargs, và shell
// con có builtin kill thật. Dùng `kill -0` (tín hiệu 0) để nếu guard hỏng thì
// lệnh cũng không làm hại gì.
func TestBashBlocksKillingOwnProcessTree(t *testing.T) {
	t.Parallel()
	pid := os.Getpid()
	sh := shell.NewShell(&shell.Options{WorkingDir: t.TempDir(), BlockFuncs: blockFuncs()})
	for _, cmd := range []string{
		fmt.Sprintf("/bin/kill -0 %d", pid),
		fmt.Sprintf("env kill -0 %d", pid),
		fmt.Sprintf("sh -c 'kill -0 %d'", pid),
		fmt.Sprintf("bash -c 'echo x; kill -0 %d'", pid),
		"xargs kill -0 < /dev/null",
	} {
		_, _, err := sh.Exec(t.Context(), cmd)
		require.Errorf(t, err, "%q", cmd)
		require.Containsf(t, err.Error(), "not allowed", "%q", cmd)
	}
	// Lệnh thường vẫn chạy.
	out, _, err := sh.Exec(t.Context(), "echo ok")
	require.NoError(t, err)
	require.Equal(t, "ok\n", out)
}
