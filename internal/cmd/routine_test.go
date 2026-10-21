package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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
	t.Setenv("GOCACHE", t.TempDir()) // không test nào trim được cache Go thật của người chạy test
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
		NewAgent: func(worktree string) (func(context.Context, string) (loop.RunInfo, error), func(), func() string, error) {
			e.sawDir = worktree
			return func(ctx context.Context, prompt string) (loop.RunInfo, error) {
				e.runs++
				var err error
				if onTurn != nil {
					err = onTurn(worktree)
				}
				return loop.RunInfo{SessionID: "s"}, err
			}, func() {}, func() string { return "fake/m1" }, nil
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

// Giới hạn chung cho mọi routine: hết chỗ thì thoát 75 như khoá bận, nêu tên biến;
// không tạo nhánh hay worktree. Slots=0 là không giới hạn.
func TestExecuteRoutine_RefusesWhenEveryRoutineSlotIsTaken(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	slot := filepath.Join(e.data, "slots", "0.lock")
	require.NoError(t, os.MkdirAll(filepath.Dir(slot), 0o700))
	release, err := lock.TryFile(slot)
	require.NoError(t, err)

	e.opts.Slots = 1
	_, err = executeRoutine(t.Context(), e.opts, e.deps(nil))
	var ec *exitCodeError
	require.True(t, errors.As(err, &ec), "no free slot must exit 75: %v", err)
	require.Equal(t, 75, ec.code)
	require.ErrorContains(t, err, "DEW_ROUTINE_SLOTS")
	require.Zero(t, e.runs)
	require.Empty(t, gitIn(t, e.repo, "branch", "--list", "routine/*"))

	e.opts.Slots = 2 // chỗ thứ hai còn trống
	_, err = executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err)
	e.opts.Slots = 0
	_, err = executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err, "0 = no limit")

	release()
	e.opts.Slots = 1
	_, err = executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err)
	again, err := lock.TryFile(slot)
	require.NoError(t, err, "the run gives its slot back")
	again()
}

// Lượt theo lịch chạy binary trực tiếp (không qua crushrc nạp tệp env): giới hạn đặt
// trong tệp env vẫn phải có hiệu lực; biến export thắng tệp như mọi biến DEW_*.
func TestRoutineSlots_ExportedEnvThenEnvFile(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), "env")
	t.Setenv(execpolicy.EnvFileEnv, envFile)
	t.Setenv("DEW_ROUTINE_SLOTS", "")
	n, err := routineSlots()
	require.NoError(t, err)
	require.Equal(t, 2, n, "default (21/10: >=4 at once 0/14 GREEN, <=1.5 18/20)")

	require.NoError(t, os.WriteFile(envFile, []byte("DEW_API_KEY=x\nexport DEW_ROUTINE_SLOTS=5 # slow machine\n"), 0o600))
	n, err = routineSlots()
	require.NoError(t, err)
	require.Equal(t, 5, n)

	t.Setenv("DEW_ROUTINE_SLOTS", "0")
	n, err = routineSlots()
	require.NoError(t, err)
	require.Equal(t, 0, n)

	for _, bad := range []string{"-1", "two"} {
		t.Setenv("DEW_ROUTINE_SLOTS", bad)
		_, err = routineSlots()
		require.ErrorContains(t, err, "DEW_ROUTINE_SLOTS", bad)
	}
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
	deps.NewAgent = func(string) (func(context.Context, string) (loop.RunInfo, error), func(), func() string, error) {
		return nil, nil, nil, errors.New("no providers configured")
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

// F6: nhiều lượt khởi động CÙNG LÚC không bao giờ vượt trần. Mỗi chỗ là một flock riêng: mỗi
// lần mở tệp là một fd riêng nên goroutine tranh nhau như tiến trình. Lặp 100 lần, 8 lượt mỗi
// lần, cùng xuất phát; mỗi lượt giữ chỗ một chút để các lượt khác thấy nó.
func TestTakeRoutineSlot_RacingStartsNeverExceedTheCap(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const slots, racers = 2, 8
	var live, peak atomic.Int32
	for range 100 {
		start := make(chan struct{})
		var wg sync.WaitGroup
		var won atomic.Int32
		for range racers {
			wg.Go(func() {
				<-start
				release, err := takeRoutineSlot(dir, slots)
				if err != nil {
					if ec, ok := errors.AsType[*exitCodeError](err); !ok || ec.code != exitCodeBusy {
						t.Errorf("no free slot must be exit 75, got %v", err)
					}
					return
				}
				won.Add(1)
				n := live.Add(1)
				for p := peak.Load(); n > p && !peak.CompareAndSwap(p, n); p = peak.Load() {
				}
				time.Sleep(time.Millisecond)
				live.Add(-1)
				release()
			})
		}
		close(start)
		wg.Wait()
		require.Positive(t, won.Load(), "some start must get a slot")
	}
	require.LessOrEqual(t, peak.Load(), int32(slots), "more runs held a slot at once than the cap")
	require.Equal(t, int32(slots), peak.Load(), "the cap itself must be reachable")
}

// DEW_ROUTINE_MIN_MEM_MB đọc như DEW_ROUTINE_SLOTS: biến export, rồi tệp env; mặc định 0 (tắt).
func TestRoutineMinMemMB_ExportedEnvThenEnvFile(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), "env")
	t.Setenv(execpolicy.EnvFileEnv, envFile)
	t.Setenv("DEW_ROUTINE_MIN_MEM_MB", "")
	n, err := routineMinMemMB()
	require.NoError(t, err)
	require.Equal(t, 0, n, "default: no check")

	require.NoError(t, os.WriteFile(envFile, []byte("export DEW_ROUTINE_MIN_MEM_MB=3000\n"), 0o600))
	n, err = routineMinMemMB()
	require.NoError(t, err)
	require.Equal(t, 3000, n)

	t.Setenv("DEW_ROUTINE_MIN_MEM_MB", "0")
	n, err = routineMinMemMB()
	require.NoError(t, err)
	require.Equal(t, 0, n)

	for _, bad := range []string{"-1", "3GB", "x"} {
		t.Setenv("DEW_ROUTINE_MIN_MEM_MB", bad)
		_, err = routineMinMemMB()
		require.ErrorContains(t, err, "DEW_ROUTINE_MIN_MEM_MB", bad)
	}
}

// Ít RAM trống hơn DEW_ROUTINE_MIN_MEM_MB thì không chạy (75, nêu tên biến và số đo), không
// tạo nhánh hay worktree. Bằng đúng ngưỡng là đủ. Không đo được (không phải Linux) thì chạy,
// cảnh báo đúng một dòng. 0 thì không đo.
func TestExecuteRoutine_RefusesWhenAvailableMemoryIsBelowTheMinimum(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.opts.MinMemMB = 3000
	d := e.deps(nil)
	d.MemAvailableKB = func() (uint64, error) { return 1000 * 1024, nil }
	_, err := executeRoutine(t.Context(), e.opts, d)
	ec, ok := errors.AsType[*exitCodeError](err)
	require.True(t, ok, "low memory must exit 75: %v", err)
	require.Equal(t, exitCodeBusy, ec.code)
	require.ErrorContains(t, err, "1000 MB")
	require.ErrorContains(t, err, "DEW_ROUTINE_MIN_MEM_MB=3000")
	require.Zero(t, e.runs)
	require.Empty(t, gitIn(t, e.repo, "branch", "--list", "routine/*"))

	d.MemAvailableKB = func() (uint64, error) { return 3000 * 1024, nil }
	_, err = executeRoutine(t.Context(), e.opts, d)
	require.NoError(t, err, "exactly the minimum is enough")

	var stderr bytes.Buffer
	d.Stderr = &stderr
	d.MemAvailableKB = func() (uint64, error) { return 0, errors.New("available memory is only measured on Linux") }
	_, err = executeRoutine(t.Context(), e.opts, d)
	require.NoError(t, err, "cannot measure: run anyway")
	var warn []string
	for _, l := range strings.Split(stderr.String(), "\n") {
		if strings.Contains(l, "DEW_ROUTINE_MIN_MEM_MB") {
			warn = append(warn, l)
		}
	}
	require.Len(t, warn, 1, stderr.String())
	require.Contains(t, warn[0], "warning")
	require.Contains(t, warn[0], "only measured on Linux")

	e.opts.MinMemMB = 0
	d.MemAvailableKB = func() (uint64, error) { t.Error("0 = no check: memory must not be read"); return 0, nil }
	_, err = executeRoutine(t.Context(), e.opts, d)
	require.NoError(t, err)
}

func TestParseMemAvailable(t *testing.T) {
	t.Parallel()
	kb, err := parseMemAvailable([]byte("MemTotal:       16289700 kB\nMemFree:          630000 kB\nMemAvailable:    7345680 kB\nBuffers: 1 kB\n"))
	require.NoError(t, err)
	require.EqualValues(t, 7345680, kb)
	_, err = parseMemAvailable([]byte("MemTotal: 1 kB\n"))
	require.Error(t, err, "a kernel without MemAvailable is not 0 MB available")
	_, err = parseMemAvailable([]byte("MemAvailable: lots kB\n"))
	require.Error(t, err)
}

// A1: `routine list` nói đúng lượt mới nhất của từng routine. RUNNING chỉ khi pid còn sống và
// đúng là `routine run <tên>`; báo cáo RUNNING của tiến trình đã chết là ABANDONED, không có
// pid thì UNKNOWN (không suy "đang chạy" từ report.json). Verify chỉ PASS khi GREEN/NO_CHANGE;
// xanh mà run không đạt là PASS (not green); không có verify_status là "not run"; báo cáo hỏng
// là UNREADABLE, không phải đạt. Thư mục dữ liệu không có định nghĩa thì không liệt kê.
func TestListRoutines_StateModelVerifyOfTheNewestRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs sh and signal 0")
	}
	defs, data := t.TempDir(), t.TempDir()
	repo := gitInit(t)
	live := exec.Command("sh", "-c", "sleep 30; true", "routine", "run", "alive")
	require.NoError(t, live.Start())
	t.Cleanup(func() { _ = live.Process.Kill(); _ = live.Wait() })
	dead := exec.Command(os.Args[0], "-test.run=^$")
	require.NoError(t, dead.Run())
	started := time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339)
	running := func(pid int) string {
		return fmt.Sprintf(`{"reason":"RUNNING","started":%q,"pid":%d,"model":"p/live","attempts":1,"changed_files":["a"]}`, started, pid)
	}
	finished := `"started":"2026-10-21T09:00:00Z","finished":"2026-10-21T09:01:30Z","model":"p/m","attempts":2,"changed_files":["a","b"]`
	for name, body := range map[string]string{
		"alive":  running(live.Process.Pid),
		"ghost":  running(dead.ProcessState.Pid()),
		"nopid":  `{"reason":"RUNNING","started":"2026-10-21T09:00:00Z"}`,
		"green":  `{"reason":"GREEN","verify_status":"PASS",` + finished + `}`,
		"cut":    `{"reason":"TIMEOUT","verify_status":"PASS",` + finished + `}`,
		"failed": `{"reason":"RUN_ERROR",` + finished + `}`,
		"red":    `{"reason":"MAX_REPAIRS","verify_status":"FAIL",` + finished + `}`,
		"broken": `{"reason":"GRE`,
		"never":  "",
	} {
		def := `{"name":"` + name + `","mode":"goal","repo":"` + repo + `","goal":"g","verify":"true"}`
		require.NoError(t, os.WriteFile(filepath.Join(defs, name+".json"), []byte(def), 0o600))
		if body != "" {
			putRoutineRun(t, data, name, "20261021-090000", body)
		}
	}
	putRoutineRun(t, data, "nodef", "20261021-090000", `{"reason":"GREEN","started":"2026-10-21T09:00:00Z"}`)

	got := map[string]routineListEntry{}
	for _, e := range listRoutines(defs, data) {
		got[e.Name] = e
	}
	require.Len(t, got, 9, "nodef has runs but no definition")
	type want struct{ state, verify, model string }
	for name, w := range map[string]want{
		"alive": {"RUNNING", "NOT_RUN", "p/live"}, "ghost": {"ABANDONED", "NOT_RUN", "p/live"},
		"nopid": {"UNKNOWN", "NOT_RUN", ""}, "green": {"GREEN", "PASS", "p/m"},
		"cut": {"TIMEOUT", "PASS_NOT_GREEN", "p/m"}, "failed": {"RUN_ERROR", "NOT_RUN", "p/m"},
		"red": {"MAX_REPAIRS", "FAIL", "p/m"}, "broken": {"UNREADABLE", "", ""}, "never": {"", "", ""},
	} {
		e := got[name]
		require.Equal(t, w, want{e.State, e.Verify, e.Model}, name)
	}
	require.Equal(t, live.Process.Pid, got["alive"].PID)
	require.GreaterOrEqual(t, got["alive"].DurationSeconds, int64(300), "running: from started until now")
	require.Zero(t, got["ghost"].PID, "pid only for a live run")
	require.Equal(t, int64(90), got["green"].DurationSeconds)
	require.Equal(t, 2, got["green"].ChangedCount)
	require.Equal(t, 2, got["green"].Attempts)
	require.Equal(t, "GREEN", got["green"].LastReason, "the old field stays")

	var buf bytes.Buffer
	printRoutineList(&buf, listRoutines(defs, data))
	out := buf.String()
	line := func(name string) string {
		for _, l := range strings.Split(out, "\n") {
			if strings.HasPrefix(l, name+" ") {
				return l
			}
		}
		t.Fatalf("no line for %s in:\n%s", name, out)
		return ""
	}
	require.Regexp(t, fmt.Sprintf(`last: RUNNING \(pid %d\).*model p/live.*running \d+m\d+s  verify not run  changed 1  attempts 1$`, live.Process.Pid), line("alive"))
	require.Regexp(t, `last: ABANDONED .*verify not run`, line("ghost"))
	require.Regexp(t, `last: UNKNOWN .*model unknown`, line("nopid"))
	require.Regexp(t, `last: GREEN \d{4}-\d\d-\d\d \d\d:\d\d  model p/m  took 1m30s  verify PASS  changed 2  attempts 2$`, line("green"))
	require.Regexp(t, `last: TIMEOUT .*verify PASS \(not green\)`, line("cut"))
	require.Regexp(t, `last: RUN_ERROR .*verify not run`, line("failed"))
	require.Regexp(t, `last: MAX_REPAIRS .*verify FAIL`, line("red"))
	require.Regexp(t, `last: UNREADABLE`, line("broken"))
	require.NotContains(t, line("broken"), "PASS")
	require.Regexp(t, `last: never run`, line("never"))
	require.NotContains(t, out, "nodef")
}

// Báo cáo RUNNING mang model ngay từ lần ghi đầu (chuỗi ghim, trước cả khi dựng worktree và
// agent: `git worktree add` của repo lớn mất vài giây), và được cập nhật thành provider/model
// khi lượt agent đầu đã áp model ghim: ps/list/Home luôn thấy model của lượt đang chạy.
func TestExecuteRoutine_RunningReportCarriesTheModelFromTheStart(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.opts.Def.Model = "m1"
	one := 1
	e.opts.Def.MaxRepairs = &one
	running := func() any {
		dirs, _ := filepath.Glob(filepath.Join(e.data, "nightly", "runs", "*"))
		return rawReport(t, dirs[len(dirs)-1])["model"]
	}
	var atStart, inSecondTurn any
	applied := false
	turns := 0
	d := e.deps(nil)
	d.NewAgent = func(string) (func(context.Context, string) (loop.RunInfo, error), func(), func() string, error) {
		atStart = running()
		return func(context.Context, string) (loop.RunInfo, error) {
				turns++
				if turns == 2 {
					inSecondTurn = running()
				}
				applied = true // lượt đầu áp model ghim như -m
				return loop.RunInfo{SessionID: "s"}, nil
			}, func() {}, func() string {
				if applied {
					return "fake/m1"
				}
				return "m1"
			}, nil
	}
	out, err := executeRoutine(t.Context(), e.opts, d)
	require.NoError(t, err)
	require.Equal(t, 2, turns)
	require.Equal(t, "m1", atStart, "the pinned model, before the worktree and agent exist")
	require.Equal(t, "fake/m1", inSecondTurn, "provider/model once the first turn applied it")
	require.Equal(t, "fake/m1", rawReport(t, out.RunDir)["model"])
}

// TOCTOU: lượt kia ghi báo cáo cuối (GREEN) rồi thoát ĐÚNG giữa lúc đọc báo cáo RUNNING và lúc
// kiểm pid. Bản RUNNING cũ không được ghi đè lên báo cáo thật (mất verify_status, changed_files).
func TestExecuteRoutine_GhostCloserDoesNotOverwriteAReportWrittenMeanwhile(t *testing.T) {
	if _, err := os.Stat("/proc/self/cmdline"); runtime.GOOS != "linux" || err != nil {
		t.Skip("proving a pid dead needs Linux /proc")
	}
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	dir := filepath.Join(e.data, "other", "runs", "20261015-080000")
	putRoutineRun(t, e.data, "other", "20261015-080000", `{"routine":"other","reason":"RUNNING","started":"2026-10-15T08:00:00Z","pid":424242}`)
	final := `{"routine":"other","reason":"GREEN","verify_status":"PASS","changed_files":["a.go"],"started":"2026-10-15T08:00:00Z","finished":"2026-10-15T08:09:00Z","pid":424242}`
	d := e.deps(nil)
	d.RunAlive = func(pid int, name string) bool {
		if name == "other" { // lượt kia vừa xong và thoát trong lúc ta kiểm
			require.NoError(t, os.WriteFile(filepath.Join(dir, "report.json"), []byte(final), 0o600))
			return false
		}
		return routine.RunAlive(pid, name)
	}
	_, err := executeRoutine(t.Context(), e.opts, d)
	require.NoError(t, err)
	got := rawReport(t, dir)
	require.Equal(t, "GREEN", got["reason"])
	require.Equal(t, "PASS", got["verify_status"])
	require.Equal(t, []any{"a.go"}, got["changed_files"])
}

// Hai tiến trình ghi báo cáo cùng thư mục cùng lúc (closeGhostRuns của routine A và gc của
// routine B): không lần ghi nào lỗi, báo cáo cuối là JSON trọn vẹn, không để lại tệp tạm.
func TestWriteReport_ConcurrentWritersNeverCollide(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for i := range 200 {
		var wg sync.WaitGroup
		for w := range 4 {
			wg.Go(func() {
				if err := writeReport(dir, routineReport{Routine: "x", Reason: fmt.Sprintf("R%d-%d", i, w)}); err != nil {
					t.Errorf("writer %d: %v", w, err)
				}
			})
		}
		wg.Wait()
	}
	_, err := readReport(dir)
	require.NoError(t, err)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "only report.json is left")
}

// F3: `routine run`, khi đã giữ khoá của nó, đóng báo cáo RUNNING MỚI NHẤT của routine KHÁC
// mà pid có mặt và chắc chắn đã chết (hoặc đã là tiến trình khác: PID tái dùng). pid còn sống
// đúng là lượt đó, báo cáo không pid, lượt cũ hơn của routine khác và lượt đã xong thì để
// nguyên (chúng thuộc `routine gc` của routine đó).
func TestExecuteRoutine_ClosesGhostRunsOfOtherRoutines(t *testing.T) {
	if _, err := os.Stat("/proc/self/cmdline"); runtime.GOOS != "linux" || err != nil {
		t.Skip("proving a pid dead needs Linux /proc")
	}
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	live := exec.Command("sh", "-c", "sleep 30; true", "routine", "run", "alive")
	require.NoError(t, live.Start())
	t.Cleanup(func() { _ = live.Process.Kill(); _ = live.Wait() })
	dead := exec.Command(os.Args[0], "-test.run=^$")
	require.NoError(t, dead.Run())
	running := func(pid int) string {
		return fmt.Sprintf(`{"routine":"x","reason":"RUNNING","started":"2026-10-15T08:00:00Z","pid":%d}`, pid)
	}
	dir := func(name, stamp string) string { return filepath.Join(e.data, name, "runs", stamp) }
	putRoutineRun(t, e.data, "ghost", "20261015-070000", running(dead.ProcessState.Pid()))
	putRoutineRun(t, e.data, "ghost", "20261015-080000", running(dead.ProcessState.Pid()))
	putRoutineRun(t, e.data, "alive", "20261015-080000", running(live.Process.Pid))
	putRoutineRun(t, e.data, "reused", "20261015-080000", running(live.Process.Pid))
	putRoutineRun(t, e.data, "nopid", "20261015-080000", `{"routine":"nopid","reason":"RUNNING","started":"2026-10-15T08:00:00Z"}`)
	putRoutineRun(t, e.data, "done", "20261015-080000", `{"routine":"done","reason":"GREEN","pid":1}`)
	doneBefore, err := os.ReadFile(filepath.Join(dir("done", "20261015-080000"), "report.json"))
	require.NoError(t, err)

	var stderr bytes.Buffer
	d := e.deps(nil)
	d.Stderr = &stderr
	_, err = executeRoutine(t.Context(), e.opts, d)
	require.NoError(t, err)

	newest := rawReport(t, dir("ghost", "20261015-080000"))
	require.Equal(t, "ABANDONED", newest["reason"])
	require.NotEmpty(t, newest["finished"])
	require.Contains(t, stderr.String(), "ghost")
	require.Equal(t, "ABANDONED", rawReport(t, dir("reused", "20261015-080000"))["reason"], "pid alive but another routine: this run is gone")
	require.Equal(t, "RUNNING", rawReport(t, dir("ghost", "20261015-070000"))["reason"], "only the newest run of another routine")
	require.Equal(t, "RUNNING", rawReport(t, dir("alive", "20261015-080000"))["reason"], "a live run is never closed")
	require.Equal(t, "RUNNING", rawReport(t, dir("nopid", "20261015-080000"))["reason"], "no pid: liveness cannot be proven")
	doneAfter, err := os.ReadFile(filepath.Join(dir("done", "20261015-080000"), "report.json"))
	require.NoError(t, err)
	require.Equal(t, string(doneBefore), string(doneAfter))
}
