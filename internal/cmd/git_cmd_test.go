package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/stretchr/testify/require"
)

// Mọi lệnh git mà dew tự chạy đi qua gitCmd (không hook, không fsmonitor, không khoá tuỳ chọn): chuỗi
// "git" trong mã không phải test của internal/cmd chỉ được nằm trong hàm gitCmd. Bắt cả
// exec.Command("git") lẫn name := "git".
func TestGitIsOnlyRunThroughGitCmd(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	fset := token.NewFileSet()
	checked := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		require.NoError(t, err)
		checked++
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "gitCmd" {
				continue
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING && (lit.Value == `"git"` || lit.Value == "`git`") {
					t.Errorf("%s: git runs outside gitCmd (the repo's hooks and fsmonitor would run)", fset.Position(lit.Pos()))
				}
				return true
			})
		}
	}
	require.Greater(t, checked, 10, "the scan must read the package's files (an empty scan is not green)")
}

// gitCmd mang đủ cờ cố định, trước mọi cờ của người gọi.
func TestGitCmdFlags(t *testing.T) {
	t.Parallel()
	c := gitCmd(t.Context(), "-C", "/x", "status")
	require.Equal(t, []string{"git", "--no-optional-locks", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null", "-C", "/x", "status"}, c.Args)
}

// Hook do agent cài vào .git/hooks của repo chính (theo A: nó chạy ở lượt sau, khi dew gọi `worktree
// add`) không chạy ở lệnh git nào dew tự gọi trên repo đó: `worktree add -b` lúc bắt đầu lượt
// (post-checkout, reference-transaction) và `branch -D` lúc dọn (reference-transaction).
func TestRoutineGitDoesNotRunRepoHooks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the hooks are sh scripts")
	}
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	marker := filepath.Join(t.TempDir(), "ran")
	hooks := filepath.Join(e.repo, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooks, 0o755))
	for _, h := range []string{"post-checkout", "reference-transaction"} {
		require.NoError(t, os.WriteFile(filepath.Join(hooks, h), []byte("#!/bin/sh\necho "+h+" >> '"+marker+"'\n"), 0o755))
	}
	gitIn(t, e.repo, "branch", "control") // đối chứng: với git thường hook CÓ chạy
	require.FileExists(t, marker)
	gitIn(t, e.repo, "branch", "-D", "control")
	require.NoError(t, os.Remove(marker))

	old := e.run(t, nil)
	if raw, err := os.ReadFile(marker); err == nil {
		t.Fatalf("starting a run (git worktree add) ran repo hooks:\n%s", raw)
	}
	e.run(t, nil)
	e.gc(t, gcOpts{})
	require.NoDirExists(t, old.Report.Worktree)
	require.Empty(t, gitIn(t, e.repo, "branch", "--list", old.Report.Branch), "the cleanup must have deleted the branch")
	if raw, err := os.ReadFile(marker); err == nil {
		t.Fatalf("the cleanup (git worktree remove, branch -D) ran repo hooks:\n%s", raw)
	}
}
