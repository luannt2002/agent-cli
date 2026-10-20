package shell

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// gitInitWithFile tạo repo git thật ở dir, commit sẵn precious.txt.
func gitInitWithFile(t *testing.T, dir string) string {
	t.Helper()
	precious := filepath.Join(dir, "precious.txt")
	require.NoError(t, os.WriteFile(precious, []byte("x\n"), 0o644))
	for _, args := range [][]string{
		{"init", "-q"},
		{"add", "precious.txt"},
		{"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	return precious
}

func runGuarded(t *testing.T, cwd, command string, unattended bool, reasons *[]string) error {
	t.Helper()
	guard := UnattendedGuard(func() bool { return unattended }, func() bool { return false },
		func(reason string, _ []string) { *reasons = append(*reasons, reason) })
	return Run(context.Background(), RunOptions{
		Command:    command,
		Cwd:        cwd,
		Env:        os.Environ(),
		BlockFuncs: []BlockFunc{guard},
	})
}

// SEC-1: chặn -C/--git-dir/--work-tree là chưa đủ, git cũng bị trỏ sang repo khác
// qua BIẾN MÔI TRƯỜNG (gán đầu lệnh, env, sh -c), và GIT_CONFIG_* làm việc của -c.
// Mỗi ca chạy thật: không người trực thì tệp ngoài workspace còn nguyên; cùng lệnh
// khi có người trực thì tệp mất (chứng minh lệnh thật sự phá được, test không mù).
func TestUnattendedGitDeniesRedirectingEnvEndToEnd(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	cases := map[string]string{
		"assign prefix":     `GIT_DIR=$O/.git GIT_WORK_TREE=$O git rm -q precious.txt`,
		"env launcher":      `env GIT_DIR=$O/.git GIT_WORK_TREE=$O git rm -q precious.txt`,
		"nested sh -c":      `sh -c 'GIT_DIR=$O/.git GIT_WORK_TREE=$O git rm -q precious.txt'`,
		"export then git":   `export GIT_DIR=$O/.git GIT_WORK_TREE=$O; git rm -q precious.txt`,
		"env into sh -c":    `GIT_DIR=$O/.git GIT_WORK_TREE=$O sh -c 'git rm -q precious.txt'`,
		"export in sh -c":   `sh -c 'export GIT_DIR=$O/.git GIT_WORK_TREE=$O; git rm -q precious.txt'`,
		"config injection":  `GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=core.fsmonitor GIT_CONFIG_VALUE_0="rm -f $O/precious.txt" git status`,
		"config parameters": `GIT_CONFIG_PARAMETERS="'core.fsmonitor'='rm -f $O/precious.txt'" git status`,
		// Lượt soát 2: các dạng mà phân tích phép gán bỏ sót.
		"quoted export in sh -c":   `sh -c "export 'GIT_DIR=$O/.git' 'GIT_WORK_TREE=$O'; git rm -q precious.txt"`,
		"function prefix in sh -c": `sh -c 'f(){ git rm -q precious.txt; }; GIT_DIR=$O/.git GIT_WORK_TREE=$O f'`,
		"dynamic name in sh -c":    `sh -c 'V=GIT_DIR; W=GIT_WORK_TREE; export "$V=$O/.git" "$W=$O"; git rm -q precious.txt'`,
		"env -S":                   `env -S 'GIT_DIR=$O/.git GIT_WORK_TREE=$O git rm -q precious.txt'`,
		"find -exec":               `GIT_DIR=$O/.git GIT_WORK_TREE=$O find . -maxdepth 0 -exec git rm -q precious.txt ';'`,
		"eval of env value":        `X='GIT_DIR=$O/.git GIT_WORK_TREE=$O git rm -q precious.txt' sh -c 'eval "$X"'`,
		"env with dynamic words":   `sh -c 'a=GIT_DIR=$O/.git; b=GIT_WORK_TREE=$O; env $a $b git rm -q precious.txt'`,
	}
	for name, tmpl := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ws := t.TempDir()
			gitInitWithFile(t, ws)
			outside := t.TempDir()
			precious := gitInitWithFile(t, outside)
			command := strings.ReplaceAll(tmpl, "$O", outside)

			var reasons []string
			err := runGuarded(t, ws, command, true, &reasons)
			require.Error(t, err)
			require.Contains(t, err.Error(), "not allowed for security reasons")
			require.FileExists(t, precious, "unattended: file outside the workspace must survive")
			require.NotEmpty(t, reasons)
			require.Contains(t, reasons[0], "GIT_")

			reasons = nil
			_ = runGuarded(t, ws, command, false, &reasons)
			require.Empty(t, reasons)
			require.NoFileExists(t, precious, "attended control: the same command must really delete")
		})
	}
}

// Script có shebang chạy thẳng bằng os/exec (không qua blockHandler): biến git gán
// đầu lệnh hay export vẫn tới script (soát lượt 3, F5). Đối chứng có người trực: tệp mất.
func TestUnattendedGitEnvDoesNotReachShebangScripts(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	for name, tmpl := range map[string]string{
		"prefix": `GIT_DIR=$O/.git GIT_WORK_TREE=$O ./rm.sh`,
		"export": `export GIT_DIR=$O/.git GIT_WORK_TREE=$O; ./rm.sh`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ws := t.TempDir()
			gitInitWithFile(t, ws)
			require.NoError(t, os.WriteFile(filepath.Join(ws, "rm.sh"), []byte("#!/bin/sh\ngit rm -q precious.txt\n"), 0o755))
			outside := t.TempDir()
			precious := gitInitWithFile(t, outside)
			command := strings.ReplaceAll(tmpl, "$O", outside)

			var reasons []string
			require.Error(t, runGuarded(t, ws, command, true, &reasons))
			require.FileExists(t, precious, "unattended: the script must not run against another repo")
			reasons = nil
			_ = runGuarded(t, ws, command, false, &reasons)
			require.NoFileExists(t, precious, "attended control: the same command really deletes")
		})
	}
}

// Biến không liên quan git và lệnh không đụng biến git thì vẫn chạy như cũ. (Lệnh
// KHÔNG phải git mà mang biến git thì bị chặn: biến đi theo tiến trình con, vd
// make hay find -exec chạy git; xem TestUnattendedGitEnvGuardUnit.)
func TestUnattendedGitAllowsUnrelatedEnv(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	ws := t.TempDir()
	gitInitWithFile(t, ws)
	for _, command := range []string{
		`env FOO=1 git status`,
		`FOO=1 git status`,
		`sh -c 'FOO=1 git status'`,
		`MY_GIT_DIR=x git status`,
	} {
		var reasons []string
		require.NoErrorf(t, runGuarded(t, ws, command, true, &reasons), "%s", command)
		require.Emptyf(t, reasons, "%s", command)
	}
}

// Lớp guard thuần (không chạy shell): lệnh nào có chữ nhắc tới biến git (kể cả
// lệnh không phải git, vì biến đi theo tiến trình con) đều bị chặn khi không người
// trực; biến khác tên (MY_GIT_DIR, GIT_SSH_COMMAND) thì không.
func TestUnattendedGitEnvGuardUnit(t *testing.T) {
	t.Parallel()
	g := UnattendedGuard(func() bool { return true }, func() bool { return false }, nil)
	for _, args := range [][]string{
		{"env", "GIT_DIR=/o/.git", "git", "status"},
		{"env", "GIT_WORK_TREE=/o", "git", "add", "."},
		{"env", "GIT_COMMON_DIR=/o", "git", "status"},
		{"env", "GIT_INDEX_FILE=/o/idx", "git", "add", "."},
		{"env", "GIT_OBJECT_DIRECTORY=/o", "git", "status"},
		{"env", "GIT_NAMESPACE=x", "git", "status"},
		{"env", "GIT_CONFIG_PARAMETERS='a.b'='c'", "git", "status"},
		{"env", "GIT_CONFIG_COUNT=1", "git", "status"},
		{"env", "GIT_CONFIG_KEY_3=a.b", "git", "status"},
		{"env", "GIT_CONFIG_VALUE_3=c", "git", "status"},
		{"env", "GIT_CONFIG_GLOBAL=/o/cfg", "git", "status"},
		{"env", "GIT_CONFIG_SYSTEM=/o/cfg", "git", "status"},
		{"env", "GIT_EXEC_PATH=/o", "git", "status"},
		{"nice", "env", "GIT_DIR=/o", "timeout", "5", "git", "status"},
		{"env", "GIT_DIR=/o/.git", "bash", "-c", "git status"},
		{"bash", "-c", "GIT_DIR=x; git status"},
		{"sh", "-c", "declare -x GIT_DIR=x; git status"},
		{"sh", "-c", "g=git; GIT_DIR=x $g rm f"},
		// Chính sách chặt hơn (lượt soát 2): lệnh không phải git cũng bị chặn.
		{"env", "GIT_DIR=/o", "ls"},
		{"sh", "-c", "GIT_DIR=x make"},
		{"env", "-S", "GIT_DIR=/o git status"},
		{"find", ".", "-exec", "env", "GIT_DIR=/o", "git", "rm", "f", ";"},
	} {
		require.Truef(t, g(args), "%q must be blocked", args)
	}
	for _, args := range [][]string{
		{"env", "FOO=1", "git", "status"},
		{"env", "GIT_SSH_COMMAND_X=1", "git", "status"},
		{"env", "MY_GIT_DIR=/o", "git", "status"},
		{"git", "rev-parse", "--git-dir"},
	} {
		require.Falsef(t, g(args), "%q must pass", args)
	}
	off := UnattendedGuard(func() bool { return false }, func() bool { return false }, nil)
	require.False(t, off([]string{"env", "GIT_DIR=/o", "git", "rm", "f"}), "attended mode unchanged")
}

// Biến git có sẵn trong môi trường của chính dew (vd chạy trong git hook)
// là của người khởi chạy, không phải do lệnh đặt: không chặn. Lệnh đổi giá trị thì
// vẫn chặn. Không Parallel vì t.Setenv.
func TestUnattendedGitAllowsInheritedEnv(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	ws := t.TempDir()
	gitInitWithFile(t, ws)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(ws, ".git", "index"))
	var reasons []string
	require.NoError(t, runGuarded(t, ws, `git status`, true, &reasons))
	require.Empty(t, reasons)
	require.Error(t, runGuarded(t, ws, `GIT_INDEX_FILE=`+filepath.Join(t.TempDir(), "idx")+` git status`, true, &reasons))
	require.NotEmpty(t, reasons)
}
