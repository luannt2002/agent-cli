package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/lock"
	"github.com/stretchr/testify/require"
)

// lockProbe là đầu ra của một lượt dọn: mỗi lần lượt dọn in một dòng (giữa lúc đi qua các lượt),
// nó thử khoá của routine như một `routine run` vừa bắt đầu.
type lockProbe struct {
	path  string
	out   bytes.Buffer
	tries int
	err   error
}

func (p *lockProbe) Write(b []byte) (int, error) {
	p.tries++
	if release, err := lock.TryFile(p.path); err != nil {
		p.err = err
	} else {
		release()
	}
	return p.out.Write(b)
}

// Chạy thử (`routine gc --dry-run`) không ghi gì nên không lấy khoá routine: giữ khoá thì một lượt
// bắt đầu đúng lúc đó nhận lock_held, loại không được chạy lại (Q5), và mất im lặng.
func TestRoutineGC_DryRunTakesNoLock(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	dirty := e.run(t, func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "notes.txt"), []byte("x"), 0o644)
	})
	e.run(t, nil) // dirty không phải lượt mới nhất: dòng "keep" của nó in giữa lúc đi, ở mọi bản
	probe := &lockProbe{path: routineLockPath(e.data, "nightly")}
	require.NoError(t, routineGC(t.Context(), e.data, nil, gcOpts{DryRun: true}, probe))
	require.Contains(t, probe.out.String(), "keep "+dirty.Report.Worktree+": uncommitted changes")
	require.GreaterOrEqual(t, probe.tries, 2, "the probe must run while the dry run walks the runs")
	require.NoError(t, probe.err, "a run starting during a dry run must get the lock")
}

// `dew clean` không --apply là chạy thử: một routine đang giữ khoá (đang chạy) vẫn được liệt kê,
// không bị bỏ qua, và lượt dọn không đụng khoá của nó.
func TestClean_DryRunListsARoutineWhoseLockIsHeld(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	dirty := e.run(t, func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "notes.txt"), []byte("x"), 0o644)
	})
	e.run(t, nil)
	release, err := lock.TryFile(routineLockPath(e.data, "nightly"))
	require.NoError(t, err)
	defer release()
	tools := newFakeTools(t)
	var buf bytes.Buffer
	require.NoError(t, runClean(t.Context(), cleanOpts{DataDir: e.data, Home: tools.home, Run: tools.run}, &buf))
	out := buf.String()
	require.NotContains(t, out, "skip nightly")
	require.Contains(t, out, "keep "+dirty.Report.Worktree+": uncommitted changes")
}

// Chạy thử không giữ khoá: báo cáo RUNNING mà không chứng minh được tiến trình đã chết (không có
// pid) có thể là một lượt đang chạy: giữ (một dòng "skip"), không bao giờ "would remove", và không
// ghi lại báo cáo. Lượt dọn thật (giữ khoá) vẫn coi nó là lượt đã chết như trước.
func TestRoutineGC_DryRunNeverListsAMaybeRunningRunAsRemovable(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	maybe := e.run(t, nil)
	// gc_error: lượt dọn tự động của lượt sau không đụng nó (không thử lại lần dọn hỏng).
	editReport(t, maybe.RunDir, func(r *routineReport) { r.GCError = "parked by the test" })
	e.run(t, func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "notes.txt"), []byte("x"), 0o644)
	})
	editReport(t, maybe.RunDir, func(r *routineReport) {
		r.GCError, r.Reason, r.PID, r.Finished = "", "RUNNING", 0, time.Time{}
	})

	out := e.gc(t, gcOpts{DryRun: true})
	require.Contains(t, out, "skip "+maybe.RunDir+": ")
	require.NotContains(t, out, "would remove "+maybe.Report.Worktree)
	require.Equal(t, "RUNNING", rawReport(t, maybe.RunDir)["reason"])

	out = e.gc(t, gcOpts{})
	require.Contains(t, out, "removed "+maybe.Report.Worktree, "under the lock a RUNNING report is a dead run")
}

// Chạy thử thấy lượt đang chạy thật (pid còn sống, đúng `routine run <tên>`): giữ, nói vì sao.
func TestRoutineGC_DryRunSkipsALiveRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs sh, and signal 0 is not supported on Windows")
	}
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	live := e.run(t, nil)
	c := exec.Command("sh", "-c", "sleep 30; true", "routine", "run", "nightly")
	require.NoError(t, c.Start())
	t.Cleanup(func() { _ = c.Process.Kill(); _ = c.Wait() })
	editReport(t, live.RunDir, func(r *routineReport) { r.Reason, r.PID, r.Finished = "RUNNING", c.Process.Pid, time.Time{} })

	out := e.gc(t, gcOpts{DryRun: true})
	require.Contains(t, out, "skip "+live.RunDir+": its run is still alive")
	require.NotContains(t, out, "would remove "+live.Report.Worktree)
}

// Q4: lượt mới nhất đã xong theo cùng luật với các lượt khác (routine `goal` chạy một lần: giữ "mới
// nhất" vô điều kiện thì không bao giờ gỡ được gì). Còn việc thì vẫn giữ, và mỗi worktree giữ có một
// dòng lý do cộng số đếm ở dòng tổng.
func TestRoutineGC_TheNewestFinishedRunFollowsTheSameRules(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	dirty := e.run(t, func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "notes.txt"), []byte("x"), 0o644)
	})
	newest := e.run(t, writeTranscript)

	out := e.gc(t, gcOpts{})
	require.Contains(t, out, "removed "+newest.Report.Worktree)
	require.NoDirExists(t, newest.Report.Worktree)
	require.Empty(t, gitIn(t, e.repo, "branch", "--list", newest.Report.Branch))
	requireFile(t, filepath.Join(newest.RunDir, "crush.db"), "transcript")
	require.Contains(t, out, "keep "+dirty.Report.Worktree+": uncommitted changes")
	require.Contains(t, out, "removed 1 worktree(s)")
	require.Contains(t, out, "kept 1;")

	// Lượt dọn tự động đầu lượt cũng vậy: lượt mới nhất sạch của lần trước bị gỡ, và stderr nói số
	// worktree đã giữ.
	prev := e.run(t, nil)
	zero := 0
	o := e.opts
	o.Def.MaxRepairs = &zero
	d := e.deps(nil)
	_, err := executeRoutine(t.Context(), o, d)
	require.NoError(t, err)
	require.NoDirExists(t, prev.Report.Worktree)
	stderr := d.Stderr.(*bytes.Buffer).String()
	require.Contains(t, stderr, "removed 1 old worktree(s)")
	require.Contains(t, stderr, "kept 1 old worktree(s)")
}

// adminEntry là thư mục quản trị (<common dir>/worktrees/<id>) có gitdir trỏ vào wt/.git, "" khi không có.
func adminEntry(t *testing.T, repo, wt string) string {
	t.Helper()
	dirs, _ := filepath.Glob(filepath.Join(repo, ".git", "worktrees", "*"))
	for _, d := range dirs {
		raw, err := os.ReadFile(filepath.Join(d, "gitdir"))
		if err == nil && filepath.Clean(strings.TrimSpace(string(raw))) == filepath.Join(wt, ".git") {
			return d
		}
	}
	return ""
}

// Q4: worktree bị gỡ từ bên ngoài (thư mục mất), nhánh của lượt còn (đo 22/10: 95 lượt GREEN ở dạng
// này, không có worktree_removed). Nhánh không có commit ngoài base thì xoá, sau khi gỡ đúng mục quản
// trị của worktree đó (git không xoá nhánh mà một worktree đã đăng ký còn dùng); ghi vào báo cáo để
// lần dọn sau không làm lại.
func TestRoutineGC_DeletesTheBranchOfAWorktreeRemovedOutsideDew(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	gone := e.run(t, func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "done.txt"), []byte("x"), 0o644)
	})
	require.Equal(t, "GREEN", gone.Report.Reason)
	entry := adminEntry(t, e.repo, gone.Report.Worktree)
	require.NotEmpty(t, entry)
	require.NoError(t, os.RemoveAll(gone.Report.Worktree))

	out := e.gc(t, gcOpts{DryRun: true})
	require.Contains(t, out, "would delete branch "+gone.Report.Branch)
	require.NotEmpty(t, gitIn(t, e.repo, "branch", "--list", gone.Report.Branch), "dry run deletes nothing")
	require.DirExists(t, entry)

	out = e.gc(t, gcOpts{})
	require.Contains(t, out, "deleted branch "+gone.Report.Branch)
	require.Empty(t, gitIn(t, e.repo, "branch", "--list", gone.Report.Branch))
	require.NoDirExists(t, entry)
	rep, err := readReport(gone.RunDir)
	require.NoError(t, err)
	require.False(t, rep.WorktreeGone.IsZero(), "recorded, so later cleanups do not retry it")
	require.True(t, rep.WorktreeRemoved.IsZero(), "dew did not remove that worktree")
	require.Equal(t, "GREEN", rep.Reason)
	require.NotContains(t, e.gc(t, gcOpts{}), gone.Report.Branch, "handled once")
}

// Q4, các ca phải GIỮ nhánh của worktree đã mất: nhánh có commit ngoài base; mục quản trị bị khoá
// (mục vẫn nguyên); báo cáo trỏ nhánh không phải của lượt; base_sha không phải SHA. Mục quản trị của
// worktree khác (trỏ đường khác) không bao giờ bị gỡ.
func TestRoutineGC_KeepsTheBranchOfAGoneWorktreeWhenItMayHoldWork(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	commit := func(wt string) error {
		writeFiles(t, wt, map[string]string{"c.txt": "x\n"})
		gitIn(t, wt, "add", "c.txt")
		gitIn(t, wt, "commit", "-q", "-m", "work")
		return nil
	}

	t.Run("commits beyond base", func(t *testing.T) {
		e := newRoutineEnv(t)
		r := e.run(t, commit)
		require.NoError(t, os.RemoveAll(r.Report.Worktree))
		out := e.gc(t, gcOpts{})
		require.Contains(t, out, "keep branch "+r.Report.Branch)
		require.Contains(t, out, "1 commit(s) not in base")
		require.NotEmpty(t, gitIn(t, e.repo, "branch", "--list", r.Report.Branch))
		require.NotEmpty(t, adminEntry(t, e.repo, r.Report.Worktree), "the entry is only removed to delete the branch")
	})

	t.Run("locked admin entry", func(t *testing.T) {
		e := newRoutineEnv(t)
		r := e.run(t, nil)
		gitIn(t, e.repo, "worktree", "lock", "--reason", "on a removable disk", r.Report.Worktree)
		entry := adminEntry(t, e.repo, r.Report.Worktree)
		require.NoError(t, os.RemoveAll(r.Report.Worktree))
		// Chạy thử ra đúng quyết định của lượt thật: giữ, không "would delete".
		out := e.gc(t, gcOpts{DryRun: true})
		require.Contains(t, out, "keep branch "+r.Report.Branch)
		require.Contains(t, out, "(worktree locked)")
		require.NotContains(t, out, "would delete branch")
		// Worktree bị khoá là người dùng cố ý giữ: một dòng keep, đếm vào số giữ, không lỗi (rc 0).
		var buf bytes.Buffer
		require.NoError(t, routineGC(t.Context(), e.data, nil, gcOpts{}, &buf))
		require.Contains(t, buf.String(), "keep branch "+r.Report.Branch)
		require.Contains(t, buf.String(), "(worktree locked)")
		require.Contains(t, buf.String(), "kept 1;")
		require.NotEmpty(t, gitIn(t, e.repo, "branch", "--list", r.Report.Branch))
		raw, err := os.ReadFile(filepath.Join(entry, "locked"))
		require.NoError(t, err, "the locked entry is untouched")
		require.Equal(t, "on a removable disk", strings.TrimSpace(string(raw)))
		tools := newFakeTools(t)
		require.NoError(t, runClean(t.Context(), cleanOpts{Apply: true, DataDir: e.data, Home: tools.home, Run: tools.run}, &buf), "dew clean --apply is not an error either")

		// Worktree còn trên đĩa mà bị khoá: cũng giữ như vậy.
		e = newRoutineEnv(t)
		r = e.run(t, nil)
		gitIn(t, e.repo, "worktree", "lock", r.Report.Worktree)
		buf.Reset()
		require.NoError(t, routineGC(t.Context(), e.data, nil, gcOpts{}, &buf))
		require.Contains(t, buf.String(), "keep "+r.Report.Worktree+": ")
		require.Contains(t, buf.String(), "(worktree locked)")
		require.DirExists(t, r.Report.Worktree)
	})

	t.Run("commits on the detached HEAD of the gone worktree", func(t *testing.T) {
		e := newRoutineEnv(t)
		sha := ""
		r := e.run(t, func(wt string) error { // người duyệt checkout --detach rồi commit trong worktree
			gitIn(t, wt, "checkout", "-q", "--detach")
			writeFiles(t, wt, map[string]string{"d.txt": "x\n"})
			gitIn(t, wt, "add", "d.txt")
			gitIn(t, wt, "commit", "-q", "-m", "detached work")
			sha = gitIn(t, wt, "rev-parse", "HEAD")
			return nil
		})
		entry := adminEntry(t, e.repo, r.Report.Worktree)
		require.NoError(t, os.RemoveAll(r.Report.Worktree))
		var buf bytes.Buffer
		require.NoError(t, routineGC(t.Context(), e.data, nil, gcOpts{}, &buf))
		require.DirExists(t, entry, "its HEAD is the only thing that still reaches the detached commit")
		require.Contains(t, gitIn(t, e.repo, "rev-list", "--all"), sha, "the detached commit stays reachable")
		require.Empty(t, gitIn(t, e.repo, "branch", "--list", r.Report.Branch), "the run's branch has no commits beyond base: deleted (git does not block it, the entry is not on it)")
	})

	t.Run("another worktree's entry", func(t *testing.T) {
		e := newRoutineEnv(t)
		r := e.run(t, nil)
		other := filepath.Join(t.TempDir(), "other")
		gitIn(t, e.repo, "worktree", "add", "-q", "-b", "other", other, "HEAD")
		otherEntry := adminEntry(t, e.repo, other)
		require.NoError(t, os.RemoveAll(other)) // cũng đã mất, nhưng không phải của lượt
		require.NoError(t, os.RemoveAll(adminEntry(t, e.repo, r.Report.Worktree)))
		require.NoError(t, os.RemoveAll(r.Report.Worktree))
		out := e.gc(t, gcOpts{})
		require.Contains(t, out, "deleted branch "+r.Report.Branch)
		require.DirExists(t, otherEntry, "never prune the whole repo")
		require.NotEmpty(t, gitIn(t, e.repo, "branch", "--list", "other"))
	})

	// Một worktree khác (đã mất) đứng trên chính nhánh của lượt, vd người dùng `worktree add -f` để duyệt:
	// mục của nó không bao giờ bị gỡ thay cho mục của lượt; git vẫn chặn xoá nhánh, nên nhánh còn.
	t.Run("another gone worktree on the run's branch", func(t *testing.T) {
		e := newRoutineEnv(t)
		r := e.run(t, nil)
		other := filepath.Join(t.TempDir(), "review")
		gitIn(t, e.repo, "worktree", "add", "-q", "-f", other, r.Report.Branch)
		otherEntry := adminEntry(t, e.repo, other)
		require.NoError(t, os.RemoveAll(other))
		require.NoError(t, os.RemoveAll(adminEntry(t, e.repo, r.Report.Worktree)))
		require.NoError(t, os.RemoveAll(r.Report.Worktree))
		var buf bytes.Buffer
		require.Error(t, routineGC(t.Context(), e.data, nil, gcOpts{}, &buf), "git refuses: the branch is used by the other worktree")
		require.Contains(t, buf.String(), "keep branch "+r.Report.Branch)
		require.DirExists(t, otherEntry)
		require.NotEmpty(t, gitIn(t, e.repo, "branch", "--list", r.Report.Branch))
	})

	t.Run("not the run's branch", func(t *testing.T) {
		e := newRoutineEnv(t)
		r := e.run(t, nil)
		gitIn(t, e.repo, "branch", "keepme")
		require.NoError(t, os.RemoveAll(r.Report.Worktree))
		editReport(t, r.RunDir, func(rep *routineReport) { rep.Branch = "keepme" })
		out := e.gc(t, gcOpts{})
		require.Contains(t, out, "keep branch keepme")
		require.NotEmpty(t, gitIn(t, e.repo, "branch", "--list", "keepme"))
	})

	t.Run("base_sha is not a SHA", func(t *testing.T) {
		e := newRoutineEnv(t)
		r := e.run(t, nil)
		require.NoError(t, os.RemoveAll(r.Report.Worktree))
		editReport(t, r.RunDir, func(rep *routineReport) { rep.BaseSHA = "HEAD" })
		out := e.gc(t, gcOpts{})
		require.Contains(t, out, "keep branch "+r.Report.Branch)
		require.Contains(t, out, "base_sha is not a commit SHA")
		require.NotEmpty(t, gitIn(t, e.repo, "branch", "--list", r.Report.Branch))
	})
}

// Đo 22/10 trên dữ liệu thật: 11 worktree sạch bị giữ vì "2 ignored path(s) outside .crush (0 B):
// backend/node_modules, frontend/node_modules". Cả 110 node_modules trong worktree là SYMLINK do bộ điều
// phối ngoài link vào; git liệt kê `!! backend/node_modules` (không có "/"), nên không được nhận là thư
// mục dựng lại được. Symlink bị ignore mang tên trong regenerableDirs là dựng lại được; `git worktree
// remove` (không --force) chỉ gỡ link, đích còn nguyên. Symlink bị ignore mang tên khác (`.env`) vẫn là việc.
func TestRoutineGC_SymlinkedRegenerableDirIsNotWork(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs symlinks")
	}
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	require.NoError(t, os.WriteFile(filepath.Join(e.repo, ".gitignore"), []byte("node_modules\n.env\n"), 0o644))
	gitIn(t, e.repo, "add", ".gitignore")
	gitIn(t, e.repo, "commit", "-q", "-m", "ignore")
	deps := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(deps, "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(deps, "pkg", "index.js"), []byte("dep"), 0o644))
	secret := filepath.Join(t.TempDir(), "secret.env")
	require.NoError(t, os.WriteFile(secret, []byte("SECRET=1"), 0o600))

	env := e.run(t, func(wt string) error { return os.Symlink(secret, filepath.Join(wt, ".env")) })
	linked := e.run(t, func(wt string) error {
		if err := os.MkdirAll(filepath.Join(wt, "backend"), 0o755); err != nil {
			return err
		}
		return os.Symlink(deps, filepath.Join(wt, "backend", "node_modules"))
	})
	st, err := classifyWorktree(t.Context(), linked.Report.Worktree, linked.Report.BaseSHA, linked.Report.Branch)
	require.NoError(t, err)
	require.Equal(t, []string{"backend/node_modules"}, st.Regenerable)
	require.Empty(t, st.IgnoredWork)

	out := e.gc(t, gcOpts{})
	require.Contains(t, out, "removed "+linked.Report.Worktree)
	require.NoDirExists(t, linked.Report.Worktree)
	requireFile(t, filepath.Join(deps, "pkg", "index.js"), "dep") // chỉ link bị gỡ
	require.Contains(t, out, "keep "+env.Report.Worktree+": 1 ignored path(s) outside .crush")
	require.Contains(t, out, ".env")
	requireFile(t, secret, "SECRET=1")
}
