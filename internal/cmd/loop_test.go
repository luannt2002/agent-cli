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
	"testing"
	"time"

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
		Goal: "do the thing", Verify: verify, Name: "t", MaxRepairs: 2,
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

func TestExecuteLoop_AgentTimeoutIsRunError(t *testing.T) {
	t.Parallel()
	dir := gitInit(t)
	p := newLoopParams(t, dir, "exit 1")
	p.RunTimeout = 100 * time.Millisecond
	agent := func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		<-ctx.Done()
		return loop.RunInfo{}, ctx.Err()
	}
	out, err := executeLoop(t.Context(), p, loopDeps{Agent: agent, Stderr: &bytes.Buffer{}})
	require.NoError(t, err)
	require.Equal(t, loop.StopRunError, out.Result.Reason, "quá hạn một lượt agent là lỗi chạy, không phải bị huỷ")
	require.Contains(t, out.Result.Detail, "timed out")
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
	} {
		err := validateLoopParams(p)
		require.Error(t, err, name)
		var ue *usageError
		require.True(t, errors.As(err, &ue), "%s: thiếu đề bài hoặc lệnh 'xong' là lỗi dùng lệnh (exit 2)", name)
	}
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
