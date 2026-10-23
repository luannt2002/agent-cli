package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/lock"
	"github.com/stretchr/testify/require"
)

// rmEnv là một routine có định nghĩa trên đĩa (routine rm gỡ cả tệp đó).
func rmEnv(t *testing.T) (*routineEnv, string) {
	t.Helper()
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	defs := t.TempDir()
	_, err := writeDefinition(defs, e.opts.Def)
	require.NoError(t, err)
	return e, defs
}

func (e *routineEnv) rm(t *testing.T, defs string, discard bool) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	err := removeRoutine(t.Context(), defs, e.data, "nightly", discard, &buf)
	return buf.String(), err
}

// routineBranches là các nhánh routine/* còn trong repo.
func routineBranches(t *testing.T, repo string) string {
	t.Helper()
	return gitIn(t, repo, "branch", "--list", "routine/*")
}

// `routine rm`: gỡ worktree và nhánh của mọi lượt (cả lượt có worktree đã bị gỡ từ bên ngoài),
// rồi định nghĩa; báo cáo, journal và crush.db đã chuyển ra còn nguyên.
func TestRoutineRm_RemovesWorktreesBranchesAndTheDefinition(t *testing.T) {
	e, defs := rmEnv(t)
	gone := e.run(t, nil)
	require.NoError(t, os.RemoveAll(gone.Report.Worktree))
	last := e.run(t, writeTranscript)
	require.DirExists(t, last.Report.Worktree)

	out, err := e.rm(t, defs, false)
	require.NoError(t, err, out)
	require.NoDirExists(t, last.Report.Worktree)
	require.Empty(t, routineBranches(t, e.repo))
	require.NoFileExists(t, filepath.Join(defs, "nightly.json"))
	require.Len(t, splitLines(gitIn(t, e.repo, "worktree", "list")), 1, "only the main worktree is left")
	for _, r := range []routineOutcome{gone, last} {
		require.FileExists(t, filepath.Join(r.RunDir, "report.json"))
		require.FileExists(t, r.JournalPath)
	}
	requireFile(t, filepath.Join(last.RunDir, "crush.db"), "transcript")
	require.Contains(t, out, "removed routine nightly")
}

// Lượt còn việc (tệp chưa commit, commit ngoài base, kể cả trên nhánh của worktree đã mất): từ chối,
// không gỡ gì (cả lượt sạch lẫn định nghĩa), liệt kê từng lượt và gợi ý export; --discard thì gỡ.
func TestRoutineRm_RefusesRunsWithWorkUnlessDiscard(t *testing.T) {
	e, defs := rmEnv(t)
	committed := e.run(t, func(wt string) error {
		writeFiles(t, wt, map[string]string{"c.txt": "x\n"})
		gitIn(t, wt, "add", "c.txt")
		gitIn(t, wt, "commit", "-q", "-m", "work")
		return nil
	})
	require.NoError(t, os.RemoveAll(committed.Report.Worktree))
	dirty := e.run(t, func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "notes.txt"), []byte("x"), 0o644)
	})
	clean := e.run(t, nil)

	out, err := e.rm(t, defs, false)
	require.Error(t, err)
	require.Contains(t, out, dirty.Report.Worktree+": uncommitted changes")
	require.Contains(t, out, committed.Report.Branch+": 1 commit(s) not in base")
	require.Contains(t, out, "dew routine export nightly --run "+filepath.Base(dirty.RunDir))
	require.Contains(t, out, "--discard")
	require.FileExists(t, filepath.Join(defs, "nightly.json"))
	require.DirExists(t, clean.Report.Worktree, "nothing is removed when one run holds work")
	requireFile(t, filepath.Join(dirty.Report.Worktree, "notes.txt"), "x")
	require.NotEmpty(t, gitIn(t, e.repo, "branch", "--list", committed.Report.Branch))

	out, err = e.rm(t, defs, true)
	require.NoError(t, err, out)
	// --discard vẫn in đúng danh sách việc sẽ mất, kèm gợi ý export, trước khi gỡ.
	require.Contains(t, out, "holds work: "+dirty.Report.Worktree+": uncommitted changes")
	require.Contains(t, out, committed.Report.Branch+": 1 commit(s) not in base")
	require.Contains(t, out, "dew routine export nightly --run "+filepath.Base(dirty.RunDir))
	require.Contains(t, out, "discarding")
	require.NoDirExists(t, dirty.Report.Worktree)
	require.NoDirExists(t, clean.Report.Worktree)
	require.Empty(t, routineBranches(t, e.repo))
	require.NoFileExists(t, filepath.Join(defs, "nightly.json"))
}

// Routine đang chạy (khoá bị giữ, hoặc báo cáo RUNNING mà tiến trình còn sống) thì luôn từ chối,
// kể cả --discard.
func TestRoutineRm_RefusesARunningRoutine(t *testing.T) {
	e, defs := rmEnv(t)
	r := e.run(t, nil)
	release, err := lock.TryFile(routineLockPath(e.data, "nightly"))
	require.NoError(t, err)
	_, err = e.rm(t, defs, true)
	require.ErrorContains(t, err, "running")
	release()
	require.DirExists(t, r.Report.Worktree)
	require.FileExists(t, filepath.Join(defs, "nightly.json"))

	if runtime.GOOS == "windows" {
		return // signal 0 không có trên Windows: chỉ khoá bảo vệ lượt đang chạy
	}
	c := exec.Command("sh", "-c", "sleep 30; true", "routine", "run", "nightly")
	require.NoError(t, c.Start())
	t.Cleanup(func() { _ = c.Process.Kill(); _ = c.Wait() })
	editReport(t, r.RunDir, func(rep *routineReport) { rep.Reason, rep.PID, rep.Finished = "RUNNING", c.Process.Pid, time.Time{} })
	_, err = e.rm(t, defs, true)
	require.ErrorContains(t, err, "running")
	require.DirExists(t, r.Report.Worktree)
	require.FileExists(t, filepath.Join(defs, "nightly.json"))
}

// Báo cáo trỏ worktree hay nhánh không phải của lượt (report.json sửa được): không bao giờ gỡ,
// kể cả --discard; không gỡ gì cả.
func TestRoutineRm_NeverRemovesWhatIsNotTheRunsOwn(t *testing.T) {
	e, defs := rmEnv(t)
	r := e.run(t, nil)
	gitIn(t, e.repo, "branch", "keepme")
	editReport(t, r.RunDir, func(rep *routineReport) { rep.Branch = "keepme" })
	out, err := e.rm(t, defs, true)
	require.Error(t, err)
	require.Contains(t, out, errRunPaths)
	require.NotEmpty(t, gitIn(t, e.repo, "branch", "--list", "keepme"))
	require.DirExists(t, r.Report.Worktree)
	require.FileExists(t, filepath.Join(defs, "nightly.json"))
}

// Worktree đã mất mà HEAD của mục quản trị là commit tách rời (người duyệt checkout --detach rồi
// commit): mục đó là thứ duy nhất còn giữ commit, rm (kể cả --discard) không gỡ nó; nhánh của lượt
// (không commit ngoài base) vẫn xoá được.
func TestRoutineRm_KeepsTheEntryHoldingADetachedCommit(t *testing.T) {
	for _, discard := range []bool{false, true} {
		e, defs := rmEnv(t)
		sha := ""
		r := e.run(t, func(wt string) error {
			gitIn(t, wt, "checkout", "-q", "--detach")
			writeFiles(t, wt, map[string]string{"d.txt": "x\n"})
			gitIn(t, wt, "add", "d.txt")
			gitIn(t, wt, "commit", "-q", "-m", "detached work")
			sha = gitIn(t, wt, "rev-parse", "HEAD")
			return nil
		})
		entry := adminEntry(t, e.repo, r.Report.Worktree)
		require.NoError(t, os.RemoveAll(r.Report.Worktree))
		out, err := e.rm(t, defs, discard)
		require.NoError(t, err, out)
		require.DirExists(t, entry, "discard=%v", discard)
		require.Contains(t, out, "its worktree entry "+entry+" stays: its HEAD is not the run's branch (it may hold commits)", "not silent (discard=%v)", discard)
		require.Contains(t, gitIn(t, e.repo, "rev-list", "--all"), sha, "discard=%v", discard)
		require.Empty(t, routineBranches(t, e.repo))
		require.NoFileExists(t, filepath.Join(defs, "nightly.json"))
	}
}

// Worktree bị khoá (`git worktree lock`) là người dùng cố ý giữ: rm từ chối, không gỡ gì.
func TestRoutineRm_RefusesALockedWorktree(t *testing.T) {
	for _, gone := range []bool{false, true} {
		e, defs := rmEnv(t)
		r := e.run(t, nil)
		gitIn(t, e.repo, "worktree", "lock", r.Report.Worktree)
		if gone {
			require.NoError(t, os.RemoveAll(r.Report.Worktree))
		}
		out, err := e.rm(t, defs, true)
		require.Error(t, err, "gone=%v", gone)
		require.Contains(t, out, "(worktree locked)")
		require.NotEmpty(t, gitIn(t, e.repo, "branch", "--list", r.Report.Branch))
		require.FileExists(t, filepath.Join(defs, "nightly.json"))
	}
}

func splitLines(s string) []string {
	var out []string
	for _, l := range bytes.Split([]byte(s), []byte("\n")) {
		if len(l) > 0 {
			out = append(out, string(l))
		}
	}
	return out
}
