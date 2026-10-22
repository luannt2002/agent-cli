package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/loop"
	"github.com/stretchr/testify/require"
)

func gitInit(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@t"},
		{"config", "user.name", "t"},
		{"commit", "-q", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	return dir
}

func TestShellVerify_Classification(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	verify := shellVerify(dir, 2*time.Second)

	for _, tc := range []struct {
		name, cmd string
		status    loop.Status
		exit      int
		contains  string
	}{
		{"pass", "true", loop.StatusPass, 0, ""},
		{"fail keeps exit code and output", "echo boom; exit 3", loop.StatusFail, 3, "boom"},
		{"stderr is captured", "echo oops >&2; exit 1", loop.StatusFail, 1, "oops"},
		{"command not found is UNMEASURED", "no_such_command_xyz", loop.StatusUnmeasured, 127, ""},
		{"syntax error is UNMEASURED", "if [", loop.StatusUnmeasured, 0, ""},
		{"empty command is UNMEASURED", "   ", loop.StatusUnmeasured, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := verify(t.Context(), tc.cmd)
			require.NoError(t, err)
			require.Equal(t, tc.status, res.Status)
			if tc.status != loop.StatusUnmeasured {
				require.Equal(t, tc.exit, res.ExitCode)
			}
			require.Contains(t, res.Output, tc.contains)
		})
	}
}

func TestShellVerify_TimeoutIsUnmeasuredNotFail(t *testing.T) {
	t.Parallel()
	verify := shellVerify(t.TempDir(), 150*time.Millisecond)
	start := time.Now()
	res, err := verify(t.Context(), "sleep 5")
	require.NoError(t, err)
	require.Equal(t, loop.StatusUnmeasured, res.Status, "quá hạn là CHƯA ĐO: không biết đỏ hay xanh")
	require.Contains(t, res.Output, "timed out")
	require.Less(t, time.Since(start), 3*time.Second)
}

func TestShellVerify_RunsInTheWorkspaceDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "marker"), []byte("x"), 0o644))
	res, err := shellVerify(dir, 2*time.Second)(t.Context(), "test -f marker")
	require.NoError(t, err)
	require.Equal(t, loop.StatusPass, res.Status)
}

func TestGitFingerprint(t *testing.T) {
	t.Parallel()
	dir := gitInit(t)
	fp := gitFingerprint(dir)

	a, err := fp(t.Context())
	require.NoError(t, err)
	b, err := fp(t.Context())
	require.NoError(t, err)
	require.Equal(t, a, b, "cây không đổi thì dấu vân tay không đổi")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "new.txt"), []byte("1"), 0o644))
	c, _ := fp(t.Context())
	require.NotEqual(t, a, c, "tệp mới chưa theo dõi phải làm đổi dấu vân tay")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "new.txt"), []byte("2"), 0o644))
	d, _ := fp(t.Context())
	require.NotEqual(t, c, d, "sửa nội dung tệp chưa theo dõi phải làm đổi dấu vân tay")

	// Tệp đã được commit rồi mới sửa: chỉ `git diff HEAD` mới thấy (không còn trong
	// danh sách chưa theo dõi), nên đây là ca phân biệt vế "tệp đã theo dõi".
	for _, args := range [][]string{{"add", "new.txt"}, {"commit", "-q", "-m", "add new"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	e, _ := fp(t.Context())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "new.txt"), []byte("4"), 0o644))
	f, _ := fp(t.Context())
	require.NotEqual(t, e, f, "sửa tệp đã theo dõi phải làm đổi dấu vân tay")
}

func TestGitFingerprint_NotAGitRepoAssumesChanged(t *testing.T) {
	t.Parallel()
	fp := gitFingerprint(t.TempDir())
	a, _ := fp(t.Context())
	b, _ := fp(t.Context())
	require.NotEqual(t, a, b, "không phải repo git thì không biết cây có đổi không: coi như đã đổi, để không dừng oan vì STALLED")
}

func newLoopParams(t *testing.T, dir, verify string) loopParams {
	t.Helper()
	return loopParams{
		Goal: "do the thing", Verify: verify, Name: "t", MaxRepairs: 2, StreamRetries: 2,
		Cwd: dir, JournalDir: t.TempDir(), VerifyTimeout: 5 * time.Second,
	}
}

func TestExecuteLoop_GreenAfterOneRepair(t *testing.T) {
	t.Parallel()
	dir := gitInit(t)
	p := newLoopParams(t, dir, "test -f done.txt")
	var prompts []string
	agent := func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		prompts = append(prompts, prompt)
		name := "wip.txt"
		if len(prompts) == 2 {
			name = "done.txt"
		}
		return loop.RunInfo{SessionID: "s"}, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644)
	}
	var stderr bytes.Buffer
	out, err := executeLoop(t.Context(), p, loopDeps{Agent: agent, Stderr: &stderr})
	require.NoError(t, err)
	require.Equal(t, loop.StopGreen, out.Result.Reason)
	require.Equal(t, 2, out.Result.Attempts)
	require.Contains(t, prompts[1], "exit code 1", "lượt sửa nhận kết quả verify thật")

	events, err := loop.ReadJournal(out.JournalPath)
	require.NoError(t, err)
	require.Equal(t, "stop", events[len(events)-1].Kind)
	require.Equal(t, "GREEN", events[len(events)-1].Reason)
	require.Contains(t, stderr.String(), "GREEN")
}

func TestExecuteLoop_StalledWhenAgentChangesNothing(t *testing.T) {
	t.Parallel()
	dir := gitInit(t)
	p := newLoopParams(t, dir, "echo 'FAIL TestX 12ms'; exit 1")
	runs := 0
	agent := func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		runs++
		return loop.RunInfo{SessionID: "s"}, nil
	}
	out, err := executeLoop(t.Context(), p, loopDeps{Agent: agent, Stderr: &bytes.Buffer{}})
	require.NoError(t, err)
	require.Equal(t, loop.StopStalled, out.Result.Reason)
	require.Equal(t, 2, runs)
}

func TestExecuteLoop_UnrunnableVerifyNeverStartsTheAgent(t *testing.T) {
	t.Parallel()
	dir := gitInit(t)
	p := newLoopParams(t, dir, "no_such_command_xyz")
	runs := 0
	agent := func(ctx context.Context, prompt string) (loop.RunInfo, error) { runs++; return loop.RunInfo{}, nil }
	out, err := executeLoop(t.Context(), p, loopDeps{Agent: agent, Stderr: &bytes.Buffer{}})
	require.NoError(t, err)
	require.Equal(t, loop.StopUnmeasured, out.Result.Reason)
	require.Zero(t, runs, "không đo được cổng thì không đốt lượt agent")
}

// F2: quá --run-timeout là TIMEOUT (không phải bị huỷ, không phải RUN_ERROR), và verify vẫn
// chạy một lần trên cây agent bỏ lại; xanh cũng không thành GREEN, thoát 1. Không lượt sửa nào
// (MaxRepairs 0): verify FAIL sau lúc cắt không còn lượt để chạy tiếp (Q6), nên đúng một lượt.
func TestExecuteLoop_AgentTurnTimeoutIsTimeoutAndVerifiesOnce(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		writes bool
		status loop.Status
		exit   int
		note   string
	}{
		"work left behind passes verify": {true, loop.StatusPass, 0, "verify PASS after the agent was cut off — review before trusting"},
		"nothing done":                   {false, loop.StatusFail, 1, "verify FAIL (exit 1) after the agent was cut off"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := gitInit(t)
			p := newLoopParams(t, dir, "test -f done.txt")
			p.RunTimeout = 100 * time.Millisecond
			p.MaxRepairs = 0
			agent := func(ctx context.Context, prompt string) (loop.RunInfo, error) {
				if tc.writes {
					_ = os.WriteFile(filepath.Join(dir, "done.txt"), []byte("x"), 0o644)
				}
				<-ctx.Done()
				return loop.RunInfo{}, ctx.Err()
			}
			out, err := executeLoop(t.Context(), p, loopDeps{Agent: agent, Stderr: &bytes.Buffer{}})
			require.NoError(t, err)
			require.Equal(t, loop.StopTimeout, out.Result.Reason, out.Result.Detail)
			require.Contains(t, out.Result.Detail, "agent turn timed out after 100ms (per-turn run timeout)")
			require.Contains(t, out.Result.Detail, tc.note)
			require.Equal(t, tc.status, out.Result.Last.Status)
			require.Equal(t, tc.exit, out.Result.Last.ExitCode)
			require.Equal(t, 1, out.Result.Attempts)
			require.Error(t, loopExitErr(out.Result.Reason), "TIMEOUT thoát 1")
			var b bytes.Buffer
			require.NoError(t, printLoopJSON(&b, out))
			require.Contains(t, b.String(), fmt.Sprintf(`"verify_exit_code":%d`, tc.exit), "verify đã chạy sau lượt bị cắt")
		})
	}
}

// Q6 (docs/routine-invariants.md): lượt bị cắt vì --run-timeout mà verify sau lúc cắt FAIL, còn lượt
// sửa thì chạy tiếp một lượt sửa; hết lượt sửa thì TIMEOUT. Mặc định 2 lượt sửa: 3 lượt bị cắt,
// mỗi lượt một lần verify (cộng baseline).
func TestExecuteLoop_CutTurnWithFailingVerifyContinuesWhileRepairsRemain(t *testing.T) {
	t.Parallel()
	dir := gitInit(t)
	count := filepath.Join(t.TempDir(), "verifies")
	p := newLoopParams(t, dir, "echo x >> '"+count+"'; exit 1")
	p.RunTimeout = 100 * time.Millisecond
	runs := 0
	agent := func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		runs++
		<-ctx.Done()
		return loop.RunInfo{SessionID: "s"}, ctx.Err()
	}
	out, err := executeLoop(t.Context(), p, loopDeps{Agent: agent, Stderr: &bytes.Buffer{}})
	require.NoError(t, err)
	require.Equal(t, loop.StopTimeout, out.Result.Reason, out.Result.Detail)
	require.Equal(t, 3, runs)
	require.Equal(t, 3, out.Result.Attempts)
	verifies, err := os.ReadFile(count)
	require.NoError(t, err)
	require.Equal(t, 4, strings.Count(string(verifies), "x"), "baseline plus one verify after each cut")
}

type nopJournal struct{}

func (nopJournal) Append(loop.Event) error { return nil }

// Sự kiện "continue" của engine (chạy tiếp sau lượt bị cắt) có một dòng tiến trình, như verify.
func TestProgressJournal_PrintsContinue(t *testing.T) {
	t.Parallel()
	var b bytes.Buffer
	j := progressJournal{inner: nopJournal{}, w: &b}
	require.NoError(t, j.Append(loop.Event{Kind: "continue", Attempt: 2, Note: "turn 1 was cut off by the run timeout; verify FAIL (exit 1): repair turn 2"}))
	require.Equal(t, "[loop] attempt 2: turn 1 was cut off by the run timeout; verify FAIL (exit 1): repair turn 2\n", b.String())
}

// busyAgent giả một lượt agent in-process mà coordinator còn chạy linger sau khi ctx của lượt hết
// hạn (dọn tool, ghi DB), như RunNonInteractive trả về ngay lúc cắt. live đếm lượt đang chạy.
type busyAgent struct {
	linger  time.Duration
	live    atomic.Int32
	overlap atomic.Bool
}

func (a *busyAgent) run(ctx context.Context, _ string) (loop.RunInfo, error) {
	if a.live.Add(1) > 1 {
		a.overlap.Store(true)
	}
	go func() {
		<-ctx.Done()
		if a.linger >= 0 {
			time.Sleep(a.linger)
			a.live.Add(-1)
		}
	}()
	<-ctx.Done()
	return loop.RunInfo{SessionID: "s"}, ctx.Err()
}

func (a *busyAgent) busy() bool { return a.live.Load() > 0 }

// Lượt bị cắt (hay lỗi) chỉ trả về khi coordinator đã dừng hẳn: lượt sau (chạy tiếp sau khi cắt,
// chạy lại sau lỗi tạm thời) không bao giờ chạy chồng lên nó. Lượt xong bình thường không chờ.
func TestSettleTurn_WaitsUntilTheCutTurnStops(t *testing.T) {
	t.Parallel()
	a := &busyAgent{linger: 300 * time.Millisecond}
	turn := settleTurn(t.Context(), a.run, a.busy, 5*time.Second)
	for range 2 {
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		start := time.Now()
		_, err := turn(ctx, "p")
		cancel()
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.GreaterOrEqual(t, time.Since(start), 300*time.Millisecond, "returned before the cut turn stopped")
	}
	require.False(t, a.overlap.Load(), "a turn started while the previous one was still running")

	done := settleTurn(t.Context(), func(context.Context, string) (loop.RunInfo, error) { return loop.RunInfo{}, nil },
		func() bool { return true }, 5*time.Second)
	start := time.Now()
	_, err := done(t.Context(), "p")
	require.NoError(t, err)
	require.Less(t, time.Since(start), time.Second, "a turn that ran to its end does not wait")
}

// Lượt bị cắt không dừng trong hạn chờ: dừng cả loop (RUN_ERROR, không verify, không lượt nào nữa)
// thay vì chạy tiếp chồng lên nó. Ctrl+C (ctx ngoài) cắt ngang lúc chờ.
func TestSettleTurn_StopsTheLoopWhenTheTurnDoesNotStop(t *testing.T) {
	t.Parallel()
	dir := gitInit(t)
	a := &busyAgent{linger: -1} // không bao giờ dừng
	runs := 0
	turn := settleTurn(t.Context(), a.run, a.busy, 200*time.Millisecond)
	p := newLoopParams(t, dir, "exit 1")
	p.RunTimeout = 100 * time.Millisecond
	out, err := executeLoop(t.Context(), p, loopDeps{Agent: func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		runs++
		return turn(ctx, prompt)
	}, Stderr: &bytes.Buffer{}})
	require.NoError(t, err)
	require.Equal(t, loop.StopRunError, out.Result.Reason, out.Result.Detail)
	require.Contains(t, out.Result.Detail, "did not stop")
	require.Equal(t, 1, runs)

	outer, cancel := context.WithCancel(t.Context())
	turn = settleTurn(outer, a.run, a.busy, time.Minute)
	ctx, stop := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer stop()
	time.AfterFunc(200*time.Millisecond, cancel)
	start := time.Now()
	_, _ = turn(ctx, "p")
	require.Less(t, time.Since(start), 5*time.Second, "Ctrl+C ends the wait")
}

// F4: hết hạn mức (429 có lời "Usage limit") qua bộ phân loại thật: QUOTA ngay, không chờ
// để chạy lại; 429 không có lời hạn mức thì vẫn là lỗi tạm thời.
func TestExecuteLoop_QuotaStopsAtOnceButRateLimitStillRetries(t *testing.T) {
	t.Parallel()
	dir := gitInit(t)
	p := newLoopParams(t, dir, "exit 1")
	runs := 0
	quota := &fantasy.ProviderError{Title: "too many requests", StatusCode: 429, Message: "Usage limit reached for 5 hour. Your limit will reset at 12:00"}
	agent := func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		runs++
		return loop.RunInfo{SessionID: "s"}, fmt.Errorf("agent processing failed: %w", &fantasy.RetryError{Errors: []error{quota}})
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second) // chờ chạy lại 30s = sai
	defer cancel()
	out, err := executeLoop(ctx, p, loopDeps{Agent: agent, Stderr: &bytes.Buffer{}})
	require.NoError(t, err)
	require.Equal(t, loop.StopQuota, out.Result.Reason, out.Result.Detail)
	require.Equal(t, 1, runs)
	require.Contains(t, out.Result.Detail, "Usage limit reached for 5 hour")
	require.Error(t, loopExitErr(out.Result.Reason))

	rate := errors.New("agent run failed: too many requests: Rate limit reached for requests per min")
	ctx2, cancel2 := context.WithCancel(t.Context())
	defer cancel2()
	stderr := &cancelOnWrite{want: "temporary rate_limit", cancel: cancel2}
	out, err = executeLoop(ctx2, p, loopDeps{Agent: func(context.Context, string) (loop.RunInfo, error) { return loop.RunInfo{}, rate }, Stderr: stderr})
	require.NoError(t, err)
	require.Equal(t, loop.StopCancelled, out.Result.Reason, "429 không có lời hạn mức: chờ để chạy lại")
	require.Contains(t, out.Result.Detail, "waiting to retry")
}

func TestValidateLoopParams(t *testing.T) {
	t.Parallel()
	ok := loopParams{Goal: "g", Verify: "true", MaxRepairs: 2}
	require.NoError(t, validateLoopParams(ok))

	for name, p := range map[string]loopParams{
		"no goal":          {Verify: "true", MaxRepairs: 2},
		"no verify":        {Goal: "g", MaxRepairs: 2},
		"blank verify":     {Goal: "g", Verify: "  ", MaxRepairs: 2},
		"negative repairs": {Goal: "g", Verify: "true", MaxRepairs: -1},
		"negative retries": {Goal: "g", Verify: "true", StreamRetries: -1},
		"too many retries": {Goal: "g", Verify: "true", StreamRetries: 6},
	} {
		err := validateLoopParams(p)
		require.Error(t, err, name)
		var ue *usageError
		require.True(t, errors.As(err, &ue), "%s: thiếu đề bài hoặc lệnh 'xong' là lỗi dùng lệnh (exit 2)", name)
	}
	for _, n := range []int{0, 5} {
		require.NoError(t, validateLoopParams(loopParams{Goal: "g", Verify: "true", StreamRetries: n}), "--stream-retries %d", n)
	}
	require.ErrorContains(t, validateLoopParams(loopParams{Goal: "g", Verify: "true", StreamRetries: 6}), "--stream-retries")
	err := validateLoopParams(loopParams{Goal: "g", MaxRepairs: 2})
	require.True(t, strings.Contains(err.Error(), "--verify"), "phải nói rõ thiếu cờ nào: %v", err)
}

// loop mở từ thư mục con: agent làm việc ở gốc repo, verify vẫn chạy ở thư mục
// mở. Prompt phải nói verify chạy ở đâu (tương đối theo gốc); mở ở gốc thì
// prompt như cũ.
func TestExecuteLoop_VerifyDirIsNamedInThePrompt(t *testing.T) {
	t.Parallel()
	root := gitInit(t)
	sub := filepath.Join(root, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))
	for cwd, want := range map[string]string{
		sub:  "Definition of done: running `test -f done.txt` in `sub/` exits with code 0.",
		root: "Definition of done: running `test -f done.txt` exits with code 0.",
	} {
		var prompts []string
		agent := func(ctx context.Context, prompt string) (loop.RunInfo, error) {
			prompts = append(prompts, prompt)
			return loop.RunInfo{SessionID: "s"}, os.WriteFile(filepath.Join(root, "wip.txt"), []byte(prompt), 0o644)
		}
		_, err := executeLoop(t.Context(), newLoopParams(t, cwd, "test -f done.txt"), loopDeps{Agent: agent, Stderr: &bytes.Buffer{}})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(prompts), 2)
		require.Contains(t, prompts[0], want)
		if cwd == sub {
			require.Contains(t, prompts[1], "Verify command: `test -f done.txt` (run in `sub/`)")
		} else {
			require.Contains(t, prompts[1], "Verify command: `test -f done.txt`\nResult:")
		}
	}
}

// Dấu vân tay tính trên cả repo: loop mở từ thư mục con mà agent (làm việc ở
// gốc) tạo tệp ở gốc thì cây ĐÃ đổi, không được dừng STALLED oan.
func TestExecuteLoop_FingerprintCoversTheWholeRepo(t *testing.T) {
	t.Parallel()
	root := gitInit(t)
	sub := filepath.Join(root, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))
	p := newLoopParams(t, sub, "echo 'FAIL TestX 12ms'; exit 1")
	runs := 0
	agent := func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		runs++
		return loop.RunInfo{SessionID: "s"}, os.WriteFile(filepath.Join(root, fmt.Sprintf("f%d.txt", runs)), []byte("x"), 0o644)
	}
	out, err := executeLoop(t.Context(), p, loopDeps{Agent: agent, Stderr: &bytes.Buffer{}})
	require.NoError(t, err)
	require.NotEqual(t, loop.StopStalled, out.Result.Reason, "the agent changed the repo outside sub/")
	require.Equal(t, 3, runs)
}

// cancelOnWrite huỷ ctx khi tiến trình in ra dòng chứa want.
type cancelOnWrite struct {
	buf    bytes.Buffer
	want   string
	cancel context.CancelFunc
}

func (w *cancelOnWrite) Write(p []byte) (int, error) {
	n, err := w.buf.Write(p)
	if strings.Contains(w.buf.String(), w.want) {
		w.cancel()
	}
	return n, err
}

// executeLoop nối bộ phân loại lỗi tạm thời thật: lỗi 503 qua server làm loop
// chờ để chạy lại (ở đây huỷ trong lúc chờ), không dừng RUN_ERROR ngay.
func TestExecuteLoop_TransientAgentErrorWaitsToRetry(t *testing.T) {
	t.Parallel()
	dir := gitInit(t)
	p := newLoopParams(t, dir, "exit 1")
	agent := func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		return loop.RunInfo{SessionID: "s"}, errors.New("agent run failed: retry error: service unavailable: upstream busy")
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	stderr := &cancelOnWrite{want: "temporary server_error", cancel: cancel}
	out, err := executeLoop(ctx, p, loopDeps{Agent: agent, Stderr: stderr})
	require.NoError(t, err)
	require.Equal(t, loop.StopCancelled, out.Result.Reason, stderr.buf.String())
	require.Contains(t, out.Result.Detail, "waiting to retry")

	events, err := loop.ReadJournal(out.JournalPath)
	require.NoError(t, err)
	var classified []loop.Event
	for _, e := range events {
		if e.ErrorClass != "" {
			classified = append(classified, e)
		}
	}
	require.Len(t, classified, 1)
	require.Equal(t, "server_error", classified[0].ErrorClass)
	require.Equal(t, 503, classified[0].HTTPStatus)
}

// verify_exit_code chỉ có khi verify thật sự cho ra mã thoát; không có thì đừng in 0 (giống xanh).
func TestPrintLoopJSON_VerifyExitCodeOnlyWhenVerifyRan(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		last loop.VerifyResult
		want string
	}{
		"agent failed before any verify": {loop.VerifyResult{}, ""},
		"verify failed":                  {loop.VerifyResult{Status: loop.StatusFail, ExitCode: 1}, `"verify_exit_code":1`},
		"verify passed":                  {loop.VerifyResult{Status: loop.StatusPass}, `"verify_exit_code":0`},
		"verify timed out (no exit)":     {loop.VerifyResult{Status: loop.StatusUnmeasured}, ""},
		"verify command not found":       {loop.VerifyResult{Status: loop.StatusUnmeasured, ExitCode: 127}, `"verify_exit_code":127`},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var b bytes.Buffer
			require.NoError(t, printLoopJSON(&b, loopOutcome{Result: loop.Result{Reason: loop.StopRunError, Attempts: 1, Last: tc.last}}))
			if tc.want == "" {
				require.NotContains(t, b.String(), "verify_exit_code")
			} else {
				require.Contains(t, b.String(), tc.want)
			}
		})
	}
}
