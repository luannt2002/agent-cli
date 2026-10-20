package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/lock"
	"github.com/charmbracelet/crush/internal/loop"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// Trong lúc chạy, lượt mới nhất hiện RUNNING (báo cáo ghi ngay khi bắt đầu, khoá
// đang giữ), không phải kết quả của lượt trước.
func TestExecuteRoutine_RunningReportIsVisibleDuringTheRun(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	var during routineReport
	var ok bool
	_, err := executeRoutine(t.Context(), e.opts, e.deps(func(wt string) error {
		during, ok = lastReport(e.data, "nightly")
		return os.WriteFile(filepath.Join(wt, "done.txt"), []byte("x"), 0o644)
	}))
	require.NoError(t, err)
	require.True(t, ok, "the run must be visible while it runs")
	require.Equal(t, "RUNNING", during.Reason)

	after, ok := lastReport(e.data, "nightly")
	require.True(t, ok)
	require.Equal(t, "GREEN", after.Reason)
}

// Lượt bị kill (OOM, reboot, TimeoutStartSec) để lại báo cáo RUNNING mà không ai
// giữ khoá, hoặc không kịp ghi báo cáo nào. Nó phải hiện là ABANDONED, không được
// để lượt GREEN cũ hơn đứng làm "lần chạy cuối" (audit g3, HIGH).
func TestLastReport_AbandonedRunsAreShownNotHidden(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	out, err := executeRoutine(t.Context(), e.opts, e.deps(func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "done.txt"), []byte("x"), 0o644)
	}))
	require.NoError(t, err)
	require.Equal(t, loop.StopGreen, out.Result.Reason)

	runs := filepath.Join(e.data, "nightly", "runs")
	killed := filepath.Join(runs, "20261016-100000")
	require.NoError(t, os.MkdirAll(killed, 0o700))
	stale := out.Report
	stale.Reason, stale.Started = "RUNNING", stale.Started.Add(3600e9)
	raw, _ := json.Marshal(stale)
	require.NoError(t, os.WriteFile(filepath.Join(killed, "report.json"), raw, 0o600))

	rep, ok := lastReport(e.data, "nightly")
	require.True(t, ok)
	require.Equal(t, "ABANDONED", rep.Reason)
	require.Contains(t, rep.Detail, "stopped before finishing")

	// Lượt chết trước cả khi ghi được báo cáo: vẫn hiện, vẫn là ABANDONED.
	require.NoError(t, os.MkdirAll(filepath.Join(runs, "20261016-110000"), 0o700))
	rep, ok = lastReport(e.data, "nightly")
	require.True(t, ok)
	require.Equal(t, "ABANDONED", rep.Reason)
	require.Contains(t, rep.Detail, "no report")

	entries := listRoutines(t.TempDir(), e.data)
	require.Empty(t, entries, "no definitions in an empty dir")

	all := listRuns(e.data, "nightly")
	require.Len(t, all, 3)
	require.Equal(t, []string{"ABANDONED", "ABANDONED", "GREEN"}, []string{all[0].Reason, all[1].Reason, all[2].Reason})
}

// Lịch in ra mang theo đúng các biến định vị của launcher đang có trong env.
func TestScheduleEnvCarriesTheLauncherVariables(t *testing.T) {
	t.Setenv("CRUSH_GLOBAL_CONFIG", "/home/u/.config/dew")
	t.Setenv("CRUSH_GLOBAL_DATA", "")
	t.Setenv("CRUSH_CACHE_DIR", "/home/u/.cache/dew")
	env := scheduleEnv()
	require.Contains(t, env, "CRUSH_GLOBAL_CONFIG=/home/u/.config/dew")
	require.Contains(t, env, "CRUSH_CACHE_DIR=/home/u/.cache/dew")
	for _, kv := range env {
		require.NotContains(t, kv, "CRUSH_GLOBAL_DATA=", "empty variables are left out")
	}
}

// GREEN đạt được bằng cách sửa test hay tệp mà lệnh verify chạy là thành công
// giả (lượt soát độc lập g3): chỉ một câu trong prompt cấm việc đó. Báo cáo phải
// liệt kê tệp đã đổi và gắn cờ để người duyệt thấy.
func TestExecuteRoutine_ReportFlagsTestAndGateChanges(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.opts.Def.Verify = "sh check.sh"
	out, err := executeRoutine(t.Context(), e.opts, e.deps(func(wt string) error {
		if err := os.MkdirAll(filepath.Join(wt, "pkg"), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(wt, "pkg", "x_test.go"), []byte("package pkg\n"), 0o644); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(wt, "check.sh"), []byte("exit 0\n"), 0o644)
	}))
	require.NoError(t, err)
	require.Equal(t, loop.StopGreen, out.Result.Reason)
	require.Contains(t, out.Report.ChangedFiles, "pkg/x_test.go")
	require.Contains(t, out.Report.ChangedFiles, "check.sh")
	require.True(t, out.Report.TouchedTests)
	require.True(t, out.Report.TouchedGate)
}

// Đổi code thường thì không có cờ nào.
func TestExecuteRoutine_ReportNoFlagsForOrdinaryChanges(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	out, err := executeRoutine(t.Context(), e.opts, e.deps(func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "done.txt"), []byte("x"), 0o644)
	}))
	require.NoError(t, err)
	require.Equal(t, []string{"done.txt"}, out.Report.ChangedFiles)
	require.False(t, out.Report.TouchedTests)
	require.False(t, out.Report.TouchedGate)
}

// Tạo worktree hỏng (đích đã có, index.lock, đĩa đầy) phải để lại báo cáo ERROR
// mang lý do thật, không thành ABANDONED "chết trước khi ghi báo cáo" (soát LC-1).
func TestExecuteRoutine_WorktreeFailureWritesAnErrorReport(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	// fixedClock: lần gọi đầu là 09:00:01, đúng dấu thời gian của lượt này.
	target := filepath.Join(e.data, "nightly", "wt", "20261016-090001")
	require.NoError(t, os.MkdirAll(target, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(target, "occupied"), []byte("x"), 0o600))

	_, err := executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.ErrorContains(t, err, "create the worktree")
	require.Zero(t, e.runs)

	rep, ok := lastReport(e.data, "nightly")
	require.True(t, ok)
	require.Equal(t, "ERROR", rep.Reason)
	require.Contains(t, rep.Detail, "create the worktree")
	require.Contains(t, rep.Detail, "already exists", "the git message must be kept")
}

// Lượt mới nhất chưa kịp ghi báo cáo nhưng đang giữ khoá là đang chạy, không phải
// đã chết (soát LC-2).
func TestListRuns_NewestRunWithoutReportIsRunningWhileLocked(t *testing.T) {
	data := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(data, "nightly", "runs", "20261016-120000"), 0o700))
	release, err := lock.TryFile(routineLockPath(data, "nightly"))
	require.NoError(t, err)
	runs := listRuns(data, "nightly")
	require.Len(t, runs, 1)
	require.Equal(t, "RUNNING", runs[0].Reason)

	release()
	runs = listRuns(data, "nightly")
	require.Equal(t, "ABANDONED", runs[0].Reason)
}

// Biến định vị và ranh giới provider trong lịch in ra phải lấy từ env GỐC, không
// từ `.env` của thư mục hiện tại (main nạp nó sau ảnh chụp; soát SEC-5). Chạy ở
// tiến trình con vì ảnh chụp là toàn cục, chỉ chụp một lần.
func TestScheduleEnv_IgnoresProjectDotenv(t *testing.T) {
	if os.Getenv("SCHEDULE_ENV_CHILD") == "1" {
		execpolicy.Snapshot()
		if err := godotenv.Load(); err != nil {
			t.Fatal(err)
		}
		for _, kv := range scheduleEnv() {
			fmt.Println(kv)
		}
		return
	}
	dir := t.TempDir()
	dotenv := "CRUSH_GLOBAL_CONFIG=/evil/config\nCRUSH_CACHE_DIR=/evil/cache\nDEW_ENV_FILE=/evil/env\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(dotenv), 0o600))

	var env []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if !strings.HasPrefix(k, "CRUSH_") && !strings.HasPrefix(k, "DEW_") {
			env = append(env, kv)
		}
	}
	env = append(env, "SCHEDULE_ENV_CHILD=1",
		"CRUSH_GLOBAL_DATA=/home/u/.local/share/dew",
		"DEW_ALLOWED_MODELS=openai:gpt-5,anthropic:claude-x")
	c := exec.Command(os.Args[0], "-test.run=^TestScheduleEnv_IgnoresProjectDotenv$")
	c.Dir, c.Env = dir, env
	out, err := c.CombinedOutput()
	require.NoError(t, err, string(out))

	got := string(out)
	require.NotContains(t, got, "/evil", "a project .env must not reach the schedule")
	require.Contains(t, got, "CRUSH_GLOBAL_DATA=/home/u/.local/share/dew\n")
	require.NotContains(t, got, "DEW_ALLOWED_MODELS", "the allowlist belongs in the env file, not the schedule")
}

// Allowlist (dạng nào cũng vậy, kể cả cách nhau bằng dấu cách) không vào lịch:
// biến export thắng tệp env, nên chép vào lịch sẽ làm việc siết lại trong tệp env
// bị lượt theo lịch lặng lẽ bỏ qua.
func TestScheduleEnv_DropsASpaceSeparatedAllowlist(t *testing.T) {
	t.Setenv("DEW_ALLOWED_MODELS", " zai:glm-5.3  openai:gpt-5,x:y ")
	for _, kv := range scheduleEnv() {
		require.NotContains(t, kv, "DEW_ALLOWED_MODELS")
	}
}

// Báo cáo mới nhất đã xong (GREEN/ERROR) thì không thử khoá: mỗi lần thử, một lượt
// chạy theo lịch khởi động đúng lúc đó thấy khoá bận và thoát 75 (soát LC2-5).
func TestListRuns_DoesNotProbeTheLockForAFinishedRun(t *testing.T) {
	data := t.TempDir()
	dir := filepath.Join(data, "nightly", "runs", "20261016-120000")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "report.json"), []byte(`{"routine":"nightly","reason":"GREEN"}`), 0o600))
	runs := listRuns(data, "nightly")
	require.Equal(t, "GREEN", runs[0].Reason)
	require.NoFileExists(t, routineLockPath(data, "nightly"), "probing creates the lock file")
}

func TestScheduleEnv_CarriesTheProviderBoundary(t *testing.T) {
	t.Setenv("DEW_ENV_FILE", "/home/u/.config/dew/env")
	t.Setenv("DEW_ALLOWED_MODELS", "openai:gpt-5")
	env := scheduleEnv()
	require.Contains(t, env, "DEW_ENV_FILE=/home/u/.config/dew/env")
	for _, kv := range env {
		require.NotContains(t, kv, "DEW_ALLOWED_MODELS", "the env file owns the allowlist for scheduled runs")
	}
}
