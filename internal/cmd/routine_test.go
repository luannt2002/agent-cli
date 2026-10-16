package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/lock"
	"github.com/charmbracelet/crush/internal/loop"
	"github.com/charmbracelet/crush/internal/routine"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

func testDef(repo string) routine.Definition {
	return routine.Definition{
		Name: "nightly", Mode: routine.ModeMaintenance, Repo: repo,
		Goal: "do the maintenance", Verify: "test -f done.txt",
	}
}

type routineEnv struct {
	repo, data string
	opts       routineOpts
	runs       int
	sawDir     string
}

func newRoutineEnv(t *testing.T) *routineEnv {
	t.Helper()
	repo := gitInit(t)
	data := t.TempDir()
	e := &routineEnv{repo: repo, data: data}
	e.opts = routineOpts{Def: testDef(repo), Sum: strings.Repeat("a", 64), DataDir: data, Now: fixedClock()}
	return e
}

// fixedClock trả mỗi lần gọi một giây khác nhau (tên worktree/nhánh có dấu thời gian).
func fixedClock() func() time.Time {
	n := 0
	return func() time.Time {
		n++
		return time.Date(2026, 10, 16, 9, 0, n, 0, time.UTC)
	}
}

// agentThat dựng deps với agent giả chạy trong worktree.
func (e *routineEnv) deps(onTurn func(worktree string) error) routineDeps {
	return routineDeps{
		Stderr: &bytes.Buffer{},
		NewAgent: func(worktree string) (func(context.Context, string) (loop.RunInfo, error), func(), error) {
			e.sawDir = worktree
			return func(ctx context.Context, prompt string) (loop.RunInfo, error) {
				e.runs++
				var err error
				if onTurn != nil {
					err = onTurn(worktree)
				}
				return loop.RunInfo{SessionID: "s"}, err
			}, func() {}, nil
		},
	}
}

func TestExecuteRoutine_GreenInItsOwnWorktreeLeavesTheRepoUntouched(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	headBefore := gitIn(t, e.repo, "rev-parse", "HEAD")
	branchBefore := gitIn(t, e.repo, "branch", "--show-current")

	out, err := executeRoutine(t.Context(), e.opts, e.deps(func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "done.txt"), []byte("x"), 0o644)
	}))
	require.NoError(t, err)
	require.Equal(t, loop.StopGreen, out.Result.Reason)

	// Việc làm nằm trong worktree riêng trên nhánh routine/*, repo gốc nguyên vẹn.
	require.NotEqual(t, e.repo, e.sawDir)
	require.FileExists(t, filepath.Join(e.sawDir, "done.txt"), "worktree được giữ lại để người duyệt xem")
	require.NoFileExists(t, filepath.Join(e.repo, "done.txt"))
	require.Equal(t, headBefore, gitIn(t, e.repo, "rev-parse", "HEAD"))
	require.Equal(t, branchBefore, gitIn(t, e.repo, "branch", "--show-current"))
	require.Empty(t, gitIn(t, e.repo, "status", "--porcelain"), "cây làm việc của repo gốc không bị đụng")
	require.True(t, strings.HasPrefix(out.Report.Branch, "routine/nightly-"), out.Report.Branch)
	require.Contains(t, gitIn(t, e.repo, "branch", "--list", out.Report.Branch), out.Report.Branch)
	require.Equal(t, headBefore, out.Report.BaseSHA)

	// Báo cáo nằm NGOÀI worktree, chỉ chủ đọc.
	reportPath := filepath.Join(out.RunDir, "report.json")
	require.FileExists(t, reportPath)
	require.NotContains(t, out.RunDir, e.sawDir)
	info, err := os.Stat(reportPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	var rep routineReport
	raw, _ := os.ReadFile(reportPath)
	require.NoError(t, json.Unmarshal(raw, &rep))
	require.Equal(t, "GREEN", rep.Reason)
	require.Equal(t, e.opts.Sum, rep.DefinitionSHA256)
	require.Equal(t, e.sawDir, rep.Worktree)
	require.FileExists(t, out.JournalPath)
	require.Equal(t, out.JournalPath, rep.Journal)
}

// Chế độ không người trực bật TRONG lúc agent chạy (cả khi chạy tay) và được trả lại sau.
func TestExecuteRoutine_UnattendedModeIsOnDuringTheRunAndOffAfter(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	for _, commit := range []bool{false, true} {
		e := newRoutineEnv(t)
		e.opts.Def.Commit = commit
		var during, duringCommit bool
		_, err := executeRoutine(t.Context(), e.opts, e.deps(func(wt string) error {
			during, duringCommit = execpolicy.Unattended(), execpolicy.UnattendedAllowsCommit()
			return nil
		}))
		require.NoError(t, err)
		require.True(t, during, "agent phải chạy dưới ràng chặn không người trực")
		require.Equal(t, commit, duringCommit, "git commit chỉ mở khi định nghĩa bật commit")
		require.False(t, execpolicy.Unattended(), "trả lại công tắc sau khi chạy")
		require.False(t, execpolicy.UnattendedAllowsCommit())
	}
}

func TestExecuteRoutine_RefusesWhileAnotherRunHoldsTheLock(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	lockPath := routineLockPath(e.data, "nightly")
	require.NoError(t, os.MkdirAll(filepath.Dir(lockPath), 0o700))
	release, err := lock.TryFile(lockPath)
	require.NoError(t, err)

	_, err = executeRoutine(t.Context(), e.opts, e.deps(nil))
	var ec *exitCodeError
	require.True(t, errors.As(err, &ec), "đang chạy rồi thì từ chối bằng mã riêng: %v", err)
	require.Equal(t, 75, ec.code)
	require.Zero(t, e.runs)
	require.Empty(t, gitIn(t, e.repo, "branch", "--list", "routine/*"), "bị từ chối thì không tạo nhánh hay worktree")

	release()
	_, err = executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err, "nhả khoá thì chạy được (không có khoá mồ côi)")
}

func TestExecuteRoutine_LockIsReleasedAfterARun(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	_, err := executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err)
	release, err := lock.TryFile(routineLockPath(e.data, "nightly"))
	require.NoError(t, err, "chạy xong phải nhả khoá")
	release()
}

func TestExecuteRoutine_ScheduledRunRefusesGoalModeAndChangedDefinition(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	var ue *usageError

	e := newRoutineEnv(t)
	e.opts.Def.Mode = routine.ModeGoal
	e.opts.Scheduled = true
	_, err := executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.True(t, errors.As(err, &ue), "việc có đích không được chạy theo lịch: %v", err)
	require.Zero(t, e.runs)

	e = newRoutineEnv(t)
	e.opts.Scheduled = true
	e.opts.ExpectSHA = strings.Repeat("b", 64)
	_, err = executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.True(t, errors.As(err, &ue), "định nghĩa đã đổi so với lúc cài lịch: %v", err)
	require.Contains(t, err.Error(), "changed")
	require.Zero(t, e.runs)
	require.Empty(t, gitIn(t, e.repo, "branch", "--list", "routine/*"))

	// Vế ngược: hash khớp thì chạy; goal-mode chạy tay (không --scheduled) vẫn được.
	e = newRoutineEnv(t)
	e.opts.Scheduled, e.opts.ExpectSHA = true, e.opts.Sum
	_, err = executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err)
	e = newRoutineEnv(t)
	e.opts.Def.Mode = routine.ModeGoal
	_, err = executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err)
}

func TestExecuteRoutine_NotAGitRepoFailsBeforeCreatingAnything(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.opts.Def.Repo = t.TempDir() // không phải repo git
	_, err := executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.Error(t, err)
	require.Contains(t, err.Error(), "git")
	require.Zero(t, e.runs)
	release, lerr := lock.TryFile(routineLockPath(e.data, "nightly"))
	require.NoError(t, lerr, "lỗi sớm vẫn phải nhả khoá")
	release()
}

func TestExecuteRoutine_NonGreenStillWritesTheReport(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	max := 0
	e.opts.Def.MaxRepairs = &max
	out, err := executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err)
	require.Equal(t, loop.StopMaxRepairs, out.Result.Reason)
	require.FileExists(t, filepath.Join(out.RunDir, "report.json"))
}

func TestExecuteRoutine_TwoRunsGetSeparateWorktreesAndBranches(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	a, err := executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err)
	b, err := executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err)
	require.NotEqual(t, a.Report.Branch, b.Report.Branch)
	require.NotEqual(t, a.Report.Worktree, b.Report.Worktree)
}

func TestLoadRoutine(t *testing.T) {
	t.Parallel()
	defs := t.TempDir()
	repo := gitInit(t)
	good := `{"name":"nightly","mode":"maintenance","repo":"` + repo + `","goal":"g","verify":"true"}`
	require.NoError(t, os.WriteFile(filepath.Join(defs, "nightly.json"), []byte(good), 0o600))

	d, sum, err := loadRoutine(defs, "nightly")
	require.NoError(t, err)
	require.Equal(t, "nightly", d.Name)
	require.Len(t, sum, 64)

	d2, sum2, err := loadRoutine(defs, filepath.Join(defs, "nightly.json"))
	require.NoError(t, err)
	require.Equal(t, d.Name, d2.Name)
	require.Equal(t, sum, sum2)

	_, _, err = loadRoutine(defs, "missing")
	require.ErrorContains(t, err, "missing")

	require.NoError(t, os.WriteFile(filepath.Join(defs, "other.json"), []byte(good), 0o600))
	_, _, err = loadRoutine(defs, "other")
	require.ErrorContains(t, err, "name", "tên tệp phải khớp tên trong định nghĩa")

	bad := strings.Replace(good, `"verify":"true"`, `"verify":""`, 1)
	require.NoError(t, os.WriteFile(filepath.Join(defs, "bad.json"), []byte(strings.Replace(bad, "nightly", "bad", 1)), 0o600))
	_, _, err = loadRoutine(defs, "bad")
	require.ErrorContains(t, err, "verify")
}

func TestWriteDefinition(t *testing.T) {
	t.Parallel()
	defs := filepath.Join(t.TempDir(), "routines")
	d := testDef("/abs/repo")
	path, err := writeDefinition(defs, d)
	require.NoError(t, err)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	back, _, err := loadRoutine(defs, "nightly")
	require.NoError(t, err)
	require.Equal(t, d.Verify, back.Verify)

	_, err = writeDefinition(defs, d)
	require.Error(t, err, "không ghi đè định nghĩa có sẵn")

	bad := d
	bad.Name, bad.Verify = "x", ""
	_, err = writeDefinition(defs, bad)
	require.ErrorContains(t, err, "verify")
	require.NoFileExists(t, filepath.Join(defs, "x.json"))
}

func TestListRoutines(t *testing.T) {
	t.Parallel()
	defs, data := t.TempDir(), t.TempDir()
	repo := gitInit(t)
	for _, n := range []string{"alpha", "beta"} {
		body := `{"name":"` + n + `","mode":"maintenance","repo":"` + repo + `","goal":"g","verify":"true","schedule":{"cron":"0 9 * * *"}}`
		require.NoError(t, os.WriteFile(filepath.Join(defs, n+".json"), []byte(body), 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(defs, "broken.json"), []byte(`{`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(defs, "notes.txt"), []byte(`x`), 0o600))
	// beta đã chạy một lần
	runDir := filepath.Join(data, "beta", "runs", "20261016-090001")
	require.NoError(t, os.MkdirAll(runDir, 0o700))
	rep, _ := json.Marshal(routineReport{Routine: "beta", Reason: "STALLED", Finished: time.Date(2026, 10, 16, 9, 5, 0, 0, time.UTC)})
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "report.json"), rep, 0o600))

	got := listRoutines(defs, data)
	require.Len(t, got, 3, "alpha, beta và tệp hỏng (báo lỗi, không bỏ qua lặng lẽ); notes.txt không phải định nghĩa")
	byName := map[string]routineListEntry{}
	for _, g := range got {
		byName[g.Name] = g
	}
	require.Equal(t, "maintenance", byName["alpha"].Mode)
	require.True(t, byName["alpha"].Scheduled)
	require.Empty(t, byName["alpha"].LastReason, "chưa chạy lần nào")
	require.Equal(t, "STALLED", byName["beta"].LastReason)
	require.NotEmpty(t, byName["broken"].Error)
}

func TestExitCodeFor(t *testing.T) {
	t.Parallel()
	require.Equal(t, 2, exitCodeFor(&usageError{errors.New("x")}))
	require.Equal(t, 75, exitCodeFor(&exitCodeError{code: 75, err: errors.New("busy")}))
	require.Equal(t, 75, exitCodeFor(errors.Join(errors.New("wrap"), &exitCodeError{code: 75, err: errors.New("busy")})))
	require.Equal(t, 1, exitCodeFor(errors.New("boom")))
}

// Công tắc phải đi tới tiến trình CON qua env (agent chạy `crush loop`, hay một CLI khác,
// từ bash): atomic trong tiến trình không đủ. Env được trả lại sau lượt chạy.
func TestExecuteRoutine_ExportsTheSwitchToChildProcessesAndRestoresIt(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	for _, commit := range []bool{false, true} {
		t.Setenv(execpolicy.UnattendedEnv, "")
		t.Setenv(execpolicy.UnattendedCommitEnv, "")
		e := newRoutineEnv(t)
		e.opts.Def.Commit = commit
		var envOn, envCommit string
		_, err := executeRoutine(t.Context(), e.opts, e.deps(func(string) error {
			envOn, envCommit = os.Getenv(execpolicy.UnattendedEnv), os.Getenv(execpolicy.UnattendedCommitEnv)
			return nil
		}))
		require.NoError(t, err)
		require.Equal(t, "1", envOn, "tiến trình con phải thấy chế độ không người trực")
		if commit {
			require.Equal(t, "1", envCommit)
		} else {
			require.NotEqual(t, "1", envCommit, "không mở commit khi định nghĩa không bật")
		}
		require.Empty(t, os.Getenv(execpolicy.UnattendedEnv), "trả lại env sau khi chạy")
		require.Empty(t, os.Getenv(execpolicy.UnattendedCommitEnv))
	}
}

// Dựng agent lỗi (vd chưa cấu hình provider) thì chưa có việc nào được làm: không để lại
// worktree/nhánh mồ côi (một timer hằng ngày sẽ dồn dần), nhưng vẫn có báo cáo ERROR để
// người dùng thấy routine hỏng thay vì im lặng.
func TestExecuteRoutine_AgentSetupFailureLeavesNoOrphanButWritesAReport(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	deps := e.deps(nil)
	deps.NewAgent = func(string) (func(context.Context, string) (loop.RunInfo, error), func(), error) {
		return nil, nil, errors.New("no providers configured")
	}
	out, err := executeRoutine(t.Context(), e.opts, deps)
	require.ErrorContains(t, err, "no providers configured")
	require.Empty(t, gitIn(t, e.repo, "branch", "--list", "routine/*"), "nhánh tạm phải được dọn")
	require.Len(t, strings.Split(gitIn(t, e.repo, "worktree", "list"), "\n"), 1, "chỉ còn worktree chính")
	require.False(t, execpolicy.Unattended(), "trả lại công tắc")

	raw, rerr := os.ReadFile(filepath.Join(out.RunDir, "report.json"))
	require.NoError(t, rerr)
	var rep routineReport
	require.NoError(t, json.Unmarshal(raw, &rep))
	require.Equal(t, "ERROR", rep.Reason)
	require.Contains(t, rep.Detail, "no providers configured")

	// Khoá đã nhả: chạy lại được.
	release, lerr := lock.TryFile(routineLockPath(e.data, "nightly"))
	require.NoError(t, lerr)
	release()
}

// `routine run` làm việc trong worktree của nó: --cwd sẽ đưa agent thẳng vào repo của người dùng.
func TestRejectCwdFlag(t *testing.T) {
	t.Parallel()
	c := &cobra.Command{Use: "x"}
	c.Flags().String("cwd", "", "")
	require.NoError(t, rejectCwdFlag(c))
	require.NoError(t, c.Flags().Set("cwd", "/tmp"))
	err := rejectCwdFlag(c)
	require.Error(t, err)
	var ue *usageError
	require.True(t, errors.As(err, &ue))
	require.Contains(t, err.Error(), "worktree")
}

// Driver chạy `git diff HEAD` trong repo mà agent vừa sửa cấu hình: cấu hình đó không
// được chạy mã (diff.external, core.fsmonitor).
func TestGitFingerprint_DoesNotRunRepoConfiguredPrograms(t *testing.T) {
	t.Parallel()
	dir := gitInit(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("1\n"), 0o644))
	gitIn(t, dir, "add", "a.txt")
	gitIn(t, dir, "commit", "-q", "-m", "a")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("2\n"), 0o644))

	marker := filepath.Join(t.TempDir(), "ran")
	script := filepath.Join(t.TempDir(), "evil.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\ntouch '"+marker+"'\nexit 0\n"), 0o755))
	gitIn(t, dir, "config", "diff.external", script)
	gitIn(t, dir, "config", "core.fsmonitor", script)

	fp, err := gitFingerprint(dir)(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, fp)
	require.NoFileExists(t, marker, "dấu vân tay không được chạy chương trình do cấu hình repo chỉ định")
}
