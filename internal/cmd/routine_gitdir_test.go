package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/stretchr/testify/require"
)

// Agent đổi được .git dùng chung từ worktree của nó (config, hook, tag) mà không hiện trong
// changed_files hay export: báo cáo liệt kê ở git_dir_changed, stderr cảnh báo một dòng và dòng tổng
// nhắc tới. Không đổi gì thì không có trường đó.
func TestExecuteRoutine_ReportsChangesToTheSharedGitDir(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	zero := 0
	o := e.opts
	o.Def.MaxRepairs = &zero

	quiet := e.run(t, func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "done.txt"), []byte("x"), 0o644)
	})
	require.Empty(t, quiet.Report.GitDirChanged)
	require.NotContains(t, rawReport(t, quiet.RunDir), "git_dir_changed")

	d := e.deps(func(wt string) error {
		gitIn(t, wt, "config", "dew.probe", "1") // config của repo: dùng chung với mọi worktree
		gitIn(t, wt, "tag", "agent-tag")
		return os.WriteFile(filepath.Join(e.repo, ".git", "hooks", "post-merge"), []byte("#!/bin/sh\n"), 0o755)
	})
	out, err := executeRoutine(t.Context(), o, d)
	require.NoError(t, err)
	want := []string{"config", "hooks/post-merge", "refs/tags/agent-tag"}
	require.Equal(t, want, out.Report.GitDirChanged)
	rep, err := readReport(out.RunDir)
	require.NoError(t, err)
	require.Equal(t, want, rep.GitDirChanged)
	stderr := d.Stderr.(*bytes.Buffer).String()
	require.Contains(t, stderr, "warning: the run changed the repository's shared .git")
	require.Contains(t, stderr, "config, hooks/post-merge, refs/tags/agent-tag")
}

// Không ồn, không hở: refs/remotes/* (IDE tự fetch; lớp chặn đã cấm fetch) và info/refs (gc --auto,
// update-server-info ghi lại) không tính; nhánh của chính lượt không tính; nhánh routine của lượt khác
// được tạo hay xoá (lượt khác bắt đầu, gc của nó) không tính, nhưng bị VIẾT LẠI thì có; hooks là symlink
// thì chụp các tệp nó trỏ tới.
func TestExecuteRoutine_GitDirChangedIgnoresNoiseButSeesOtherRunsBranches(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a symlinked hooks dir")
	}
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	zero := 0
	o := e.opts
	o.Def.MaxRepairs = &zero
	pending, cleaned := "routine/other-20261015-080000", "routine/other-20261015-070000"
	gitIn(t, e.repo, "branch", pending)
	gitIn(t, e.repo, "branch", cleaned)
	shared := t.TempDir()
	hooks := filepath.Join(e.repo, ".git", "hooks")
	require.NoError(t, os.RemoveAll(hooks))
	require.NoError(t, os.Symlink(shared, hooks))

	d := e.deps(func(wt string) error {
		gitIn(t, wt, "update-ref", "refs/remotes/origin/main", "HEAD") // như IDE tự fetch
		gitIn(t, wt, "update-server-info")                             // ghi info/refs
		writeFiles(t, wt, map[string]string{"c.txt": "x\n"})
		gitIn(t, wt, "add", "c.txt")
		gitIn(t, wt, "commit", "-q", "-m", "own work")            // nhánh của chính lượt
		gitIn(t, wt, "branch", "routine/other-20261016-100000")   // lượt khác bắt đầu
		gitIn(t, wt, "branch", "-D", cleaned)                     // gc của lượt khác
		gitIn(t, wt, "update-ref", "refs/heads/"+pending, "HEAD") // viết lại nhánh của lượt khác
		return os.WriteFile(filepath.Join(shared, "pre-commit"), []byte("#!/bin/sh\n"), 0o755)
	})
	out, err := executeRoutine(t.Context(), o, d)
	require.NoError(t, err)
	require.Equal(t, []string{"hooks/pre-commit", "refs/heads/" + pending}, out.Report.GitDirChanged)
}

// Agent đổi được `.git/hooks` thành symlink trỏ vào $HOME (lớp chặn không cấm): mọi lượt sau trên repo
// đó sẽ băm cả $HOME trong lúc giữ khoá và slot. Có trần: quá gitDirWalkMax mục thì cây đó chỉ còn MỘT
// dấu "hooks/" (đổi từ bình thường sang quá lớn vẫn thấy được), không có từng tệp.
func TestGitDirSnapshot_CapsAHugeSymlinkedHooksTree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a symlinked hooks dir")
	}
	t.Parallel()
	repo := gitInit(t)
	big := t.TempDir()
	for i := range gitDirWalkMax + 50 {
		require.NoError(t, os.WriteFile(filepath.Join(big, fmt.Sprintf("f%05d", i)), []byte("x"), 0o644))
	}
	hooks := filepath.Join(repo, ".git", "hooks")
	require.NoError(t, os.RemoveAll(hooks))
	require.NoError(t, os.Symlink(big, hooks))

	snap, err := gitDirSnapshot(t.Context(), repo, "routine/x-1")
	require.NoError(t, err)
	var files []string
	for k := range snap {
		if strings.HasPrefix(k, "hooks/") && k != "hooks/" {
			files = append(files, k)
		}
	}
	require.Empty(t, files, "over the cap: no per-file entries")
	require.Contains(t, snap["hooks/"], "too large")
	require.Contains(t, snap, "config", "the rest of the snapshot is still taken")
}

// canceledErrCtx: Err() đã huỷ nhưng Done() là nil, nên lệnh git (exec chỉ nhìn Done) vẫn chạy; chỉ một
// lần kiểm ctx.Err() trong lúc đi cây mới dừng được.
type canceledErrCtx struct{ context.Context }

func (canceledErrCtx) Err() error { return context.Canceled }

// Đi cây hooks/info kiểm ctx giữa chừng: lượt bị huỷ không phải chờ hết một cây lớn.
func TestGitDirSnapshot_StopsTheWalkWhenTheContextIsDone(t *testing.T) {
	t.Parallel()
	repo := gitInit(t)
	_, err := gitDirSnapshot(canceledErrCtx{context.Background()}, repo, "routine/x-1")
	require.ErrorIs(t, err, context.Canceled)
}
