package cmd

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/lock"
	"github.com/charmbracelet/crush/internal/loop"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

// rawReport đọc report.json của một lượt dạng map: thấy đúng khoá nào có mặt.
func rawReport(t *testing.T, dir string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "report.json"))
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}

// Báo cáo RUNNING không được có "finished" (0001-01-01) hay "verify_exit_code" 0 (đọc
// như đã qua), và mang pid để `ps` kiểm lượt còn sống mà không thử khoá.
func TestExecuteRoutine_ReportHasNoFakeFinishedOrExitCode(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	var during map[string]any
	out, err := executeRoutine(t.Context(), e.opts, e.deps(func(wt string) error {
		dirs, _ := filepath.Glob(filepath.Join(e.data, "nightly", "runs", "*"))
		during = rawReport(t, dirs[len(dirs)-1])
		return os.WriteFile(filepath.Join(wt, "done.txt"), []byte("x"), 0o644)
	}))
	require.NoError(t, err)
	require.Equal(t, "RUNNING", during["reason"])
	require.NotContains(t, during, "finished", "a running run has not finished")
	require.NotContains(t, during, "verify_exit_code", "verify has not run yet")
	require.EqualValues(t, os.Getpid(), during["pid"])

	after := rawReport(t, out.RunDir)
	require.Equal(t, "GREEN", after["reason"])
	require.Contains(t, after, "finished")
	require.EqualValues(t, 0, after["verify_exit_code"], "verify ran and passed")
}

// Lượt agent lỗi thì verify không chạy sau nó: không có mã thoát nào để báo.
func TestExecuteRoutine_RunErrorHasNoVerifyExitCode(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	out, err := executeRoutine(t.Context(), e.opts, e.deps(func(string) error {
		return errors.New("connection refused")
	}))
	require.NoError(t, err)
	require.Equal(t, loop.StopRunError, out.Result.Reason)
	require.NotContains(t, rawReport(t, out.RunDir), "verify_exit_code")
}

// Lỗi của provider mang mã HTTP (401 khoá sai, 404 model sai, 429 hết hạn mức): báo cáo
// phải nêu nó, chuỗi lỗi của provider thường không có.
func TestExecuteRoutine_ProviderErrorNamesTheHTTPStatus(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	perr := &fantasy.ProviderError{Title: "not found", Message: "no such model", StatusCode: 404}
	out, err := executeRoutine(t.Context(), e.opts, e.deps(func(string) error {
		return fmt.Errorf("agent processing failed: %w", perr)
	}))
	require.NoError(t, err)
	require.Equal(t, loop.StopRunError, out.Result.Reason)
	require.Contains(t, out.Report.Detail, "HTTP 404")
	require.Contains(t, rawReport(t, out.RunDir)["detail"], "HTTP 404")
}

// Lượt phải nói nó chạy model nào (đã phân giải), ngay từ báo cáo RUNNING và trên stderr:
// 20/10 có 7 lượt chết vì hạn mức của model mặc định toàn cục trong khi người dùng tưởng
// chúng chạy model khác, và báo cáo không nói model nào đã chạy.
func TestExecuteRoutine_ReportsAndPrintsTheModelItRuns(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	var during map[string]any
	d := e.deps(func(string) error {
		dirs, _ := filepath.Glob(filepath.Join(e.data, "nightly", "runs", "*"))
		during = rawReport(t, dirs[len(dirs)-1])
		return nil
	})
	out, err := executeRoutine(t.Context(), e.opts, d)
	require.NoError(t, err)
	require.Equal(t, "fake/m1", during["model"], "a run killed mid-turn must still say which model it ran")
	require.Equal(t, "fake/m1", rawReport(t, out.RunDir)["model"])
	require.Contains(t, d.Stderr.(*bytes.Buffer).String(), "[routine] nightly: model fake/m1")
}

// Model của lượt: không ghim thì model large hiện hành (mặc định lúc chạy); có ghim thì
// model mà lượt đã áp (override của tiến trình, lượt agent đầu áp như -m), chưa áp thì
// chính chuỗi ghim nguyên văn, kể cả khi nó trùng tên model mặc định (model ID có ở nhiều
// provider: chưa biết provider nào).
func TestRoutineModel(t *testing.T) {
	t.Parallel()
	sel := func(provider, model string) config.SelectedModel {
		return config.SelectedModel{Provider: provider, Model: model}
	}
	none := config.SelectedModel{}
	require.Equal(t, "zai/glm-5.3", routineModel(sel("zai", "glm-5.3"), none, ""), "no pin: the default at run time")
	require.Equal(t, "qwen3", routineModel(sel("zai", "glm-5.3"), none, "qwen3"), "pinned, not applied yet: never the default")
	require.Equal(t, "m1", routineModel(sel("fake", "m1"), none, "m1"), "pinned bare ID equal to the default's: still verbatim")
	require.Equal(t, "local/qwen3", routineModel(sel("local", "qwen3"), sel("local", "qwen3"), "qwen3"), "applied: the resolved provider/model")
	require.Empty(t, routineModel(none, none, ""))
}

// sandboxConfig cô lập cấu hình/dữ liệu dew trong thư mục tạm và trả một repo có crush.json
// với model large mặc định fake/<large>.
func sandboxConfig(t *testing.T, large string) (tmp, repo string) {
	t.Helper()
	tmp = t.TempDir()
	for k, v := range map[string]string{
		"HOME": tmp, "XDG_CONFIG_HOME": filepath.Join(tmp, "xc"), "XDG_DATA_HOME": filepath.Join(tmp, "xd"),
		"XDG_CACHE_HOME": filepath.Join(tmp, "xk"), "CRUSH_GLOBAL_CONFIG": filepath.Join(tmp, "config"),
		"CRUSH_GLOBAL_DATA": filepath.Join(tmp, "data"), "CRUSH_CACHE_DIR": filepath.Join(tmp, "cache"),
		"CRUSH_DISABLE_METRICS": "1", "CRUSH_DISABLE_PROVIDER_AUTO_UPDATE": "1",
	} {
		t.Setenv(k, v)
	}
	repo = filepath.Join(tmp, "repo")
	require.NoError(t, os.MkdirAll(repo, 0o755))
	writeRepoConfig(t, repo, large)
	return tmp, repo
}

func writeRepoConfig(t *testing.T, repo, large string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(repo, "crush.json"), []byte(`{"providers":{"fake":{"type":"openai-compat","base_url":"http://127.0.0.1:9/v1","api_key":"x","models":[{"id":"m1","name":"m1","context_window":100000,"default_max_tokens":1000},{"id":"m2","name":"m2","context_window":100000,"default_max_tokens":1000}]}},"models":{"large":{"provider":"fake","model":"`+large+`"},"small":{"provider":"fake","model":"m1"}}}`), 0o600))
}

// runRoutineNew chạy `dew routine new ...` qua rootCmd, trả stdout và stderr.
func runRoutineNew(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		if c, _, err := rootCmd.Find([]string{"routine", "new"}); err == nil {
			c.Flags().VisitAll(func(f *pflag.Flag) {
				_ = f.Value.Set(f.DefValue)
				f.Changed = false
			})
		}
	})
	rootCmd.SetArgs(append([]string{"routine", "new"}, args...))
	err := rootCmd.ExecuteContext(t.Context())
	return out.String(), errOut.String(), err
}

// F4: `routine new` không --model ĐÓNG BĂNG model large mặc định hiện tại vào định nghĩa
// (20/10: 7 lượt chết vì hạn mức của model mặc định mà người dùng tưởng là model khác), nói
// model nào và cách đổi; đổi mặc định sau đó không đổi routine. Không tạo tệp trong repo.
func TestRoutineNew_FreezesTheDefaultModel(t *testing.T) {
	tmp, repo := sandboxConfig(t, "m1")
	stdout, stderr, err := runRoutineNew(t, "nightly", "--mode", "maintenance", "--repo", repo, "--goal", "g", "--verify", "true")
	require.NoError(t, err)
	def := filepath.Join(tmp, "config", "routines", "nightly.json")
	d, _, err := loadRoutine(filepath.Dir(def), "nightly")
	require.NoError(t, err)
	require.Equal(t, "fake/m1", d.Model, "the default at creation time is written into the definition")
	require.Contains(t, stdout, "fake/m1")
	require.Contains(t, stdout, "froze")
	require.Contains(t, stdout, def, "says where to change it")
	require.NotContains(t, stderr, "warning")
	require.NoDirExists(t, filepath.Join(repo, ".crush"), "resolving the default must not create files in the repo")

	writeRepoConfig(t, repo, "m2") // mặc định đổi sau khi tạo
	now, err := defaultLargeModel(repo)
	require.NoError(t, err)
	require.Equal(t, "fake/m2", now)
	d, _, err = loadRoutine(filepath.Dir(def), "nightly")
	require.NoError(t, err)
	require.Equal(t, "fake/m1", d.Model, "the routine keeps the model it was created with")
}

// --model có sẵn thì giữ nguyên chuỗi đó, không đóng băng gì.
func TestRoutineNew_KeepsAnExplicitModel(t *testing.T) {
	tmp, repo := sandboxConfig(t, "m1")
	stdout, stderr, err := runRoutineNew(t, "pinned", "--mode", "goal", "--repo", repo, "--goal", "g", "--verify", "true", "--model", "m2")
	require.NoError(t, err)
	d, _, err := loadRoutine(filepath.Join(tmp, "config", "routines"), "pinned")
	require.NoError(t, err)
	require.Equal(t, "m2", d.Model)
	require.NotContains(t, stdout+stderr, "froze")
	require.NotContains(t, stdout+stderr, "warning")
}

// Không phân giải được model mặc định: để trống như trước và cảnh báo (nêu lỗi, cách ghim).
func TestRoutineNew_WarnsWhenTheDefaultCannotBeResolved(t *testing.T) {
	tmp, repo := sandboxConfig(t, "m1")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "crush.json"), []byte(`{`), 0o600))
	stdout, stderr, err := runRoutineNew(t, "broken", "--mode", "goal", "--repo", repo, "--goal", "g", "--verify", "true")
	require.NoError(t, err)
	d, _, err := loadRoutine(filepath.Join(tmp, "config", "routines"), "broken")
	require.NoError(t, err)
	require.Empty(t, d.Model)
	require.NotContains(t, stdout, "froze")
	require.Contains(t, stderr, "warning: no --model")
	require.Contains(t, stderr, "now unknown")
	require.Contains(t, stderr, "--model provider/model")
}

// Verify không chạy được ngay từ baseline (exit 127): chưa có lượt agent nào, nên không
// có mã thoát "sau lượt agent" để báo.
func TestExecuteRoutine_UnmeasuredBaselineHasNoVerifyExitCode(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.opts.Def.Verify = "exit 127"
	out, err := executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err)
	require.Equal(t, loop.StopUnmeasured, out.Result.Reason)
	require.Zero(t, e.runs)
	require.NotContains(t, rawReport(t, out.RunDir), "verify_exit_code")
}

// Mã HTTP chỉ gắn vào RUN_ERROR: lượt bị huỷ giữa chừng thì lỗi provider là hậu quả
// của việc huỷ, không phải lý do dừng.
func TestExecuteRoutine_HTTPStatusOnlyForRunErrors(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	out, err := executeRoutine(ctx, e.opts, e.deps(func(string) error {
		cancel()
		return fmt.Errorf("agent processing failed: %w", &fantasy.ProviderError{Message: "aborted", StatusCode: 499})
	}))
	require.NoError(t, err)
	require.Equal(t, loop.StopCancelled, out.Result.Reason)
	require.NotContains(t, out.Report.Detail, "HTTP")
}

// Huỷ giữa lượt (SIGTERM, hết giờ của timer): việc agent đã làm vẫn phải vào báo cáo.
func TestExecuteRoutine_CancelledRunStillListsChangedFiles(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	out, err := executeRoutine(ctx, e.opts, e.deps(func(wt string) error {
		cancel()
		return os.WriteFile(filepath.Join(wt, "half.txt"), []byte("x"), 0o644)
	}))
	require.NoError(t, err)
	require.Equal(t, loop.StopCancelled, out.Result.Reason)
	require.Contains(t, out.Report.ChangedFiles, "half.txt")
}

// Giữ được khoá của routine nghĩa là không lượt nào khác của nó còn sống: báo cáo nào
// còn RUNNING là của lượt đã chết, ghi lại thành ABANDONED ngay trên đĩa với lúc kết
// thúc là lần cuối lượt đó ghi gì (journal, không có thì chính báo cáo).
func TestExecuteRoutine_RewritesDeadRunningReportsAsAbandoned(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	runs := filepath.Join(e.data, "nightly", "runs")
	withJournal := filepath.Join(runs, "20261015-080000")
	noJournal := filepath.Join(runs, "20261015-090000")
	for _, d := range []string{withJournal, noJournal} {
		require.NoError(t, os.MkdirAll(d, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(d, "report.json"),
			[]byte(`{"routine":"nightly","reason":"RUNNING","started":"2026-10-15T08:00:00Z","pid":999999}`), 0o600))
	}
	journal := filepath.Join(withJournal, "nightly-20261015-080000.jsonl")
	require.NoError(t, os.WriteFile(journal, []byte("{}\n"), 0o600))
	lastWrite := time.Date(2026, 10, 15, 8, 5, 0, 0, time.UTC)
	reportWrite := time.Date(2026, 10, 15, 9, 0, 2, 0, time.UTC)
	require.NoError(t, os.Chtimes(journal, lastWrite, lastWrite))
	require.NoError(t, os.Chtimes(filepath.Join(withJournal, "report.json"), lastWrite.Add(-time.Hour), lastWrite.Add(-time.Hour)))
	require.NoError(t, os.Chtimes(filepath.Join(noJournal, "report.json"), reportWrite, reportWrite))

	_, err := executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err)

	for dir, want := range map[string]time.Time{withJournal: lastWrite, noJournal: reportWrite} {
		var rep routineReport
		raw, err := os.ReadFile(filepath.Join(dir, "report.json"))
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &rep))
		require.Equal(t, "ABANDONED", rep.Reason, dir)
		require.True(t, want.Equal(rep.Finished), "%s: finished %s, want %s", dir, rep.Finished, want)
	}
	require.Equal(t, journal, rawReport(t, withJournal)["journal"])
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

// F2: lượt bị cắt vì hết run_timeout sau khi đã sửa 2 tệp: báo cáo TIMEOUT, vẫn đo verify
// một lần (trạng thái và mã thoát thật), liệt kê 2 tệp; verify xanh cũng không thành GREEN.
func TestExecuteRoutine_TurnTimeoutKeepsAndReportsTheWork(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.opts.Def.RunTimeout = "300ms"
	out, err := executeRoutine(t.Context(), e.opts, routineDeps{
		Stderr: &bytes.Buffer{},
		NewAgent: func(wt string) (func(context.Context, string) (loop.RunInfo, error), func(), func() string, error) {
			return func(ctx context.Context, _ string) (loop.RunInfo, error) {
				_ = os.WriteFile(filepath.Join(wt, "done.txt"), []byte("x"), 0o644)
				_ = os.WriteFile(filepath.Join(wt, "notes.md"), []byte("x"), 0o644)
				<-ctx.Done()
				return loop.RunInfo{SessionID: "s"}, ctx.Err()
			}, func() {}, func() string { return "fake/m1" }, nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, loop.StopTimeout, out.Result.Reason, out.Report.Detail)
	raw := rawReport(t, out.RunDir)
	require.Equal(t, "TIMEOUT", raw["reason"])
	require.Equal(t, "PASS", raw["verify_status"])
	require.EqualValues(t, 0, raw["verify_exit_code"], "verify ran after the cut")
	require.Equal(t, []any{"done.txt", "notes.md"}, raw["changed_files"])
	require.Contains(t, raw["detail"], "per-turn run timeout")
	require.Contains(t, raw["detail"], "review before trusting")
	require.Contains(t, raw, "finished")
	require.Error(t, routineExitErr(e.opts.Def.Mode, out.Result.Reason), "TIMEOUT thoát 1")
}

// F2: bị cắt khi chưa sửa gì: verify vẫn chạy, mã thoát là mã thật (khác 0), không phải 0 giả.
func TestExecuteRoutine_TurnTimeoutWithoutChangesHasTheRealVerifyExit(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.opts.Def.RunTimeout = "200ms"
	out, err := executeRoutine(t.Context(), e.opts, routineDeps{
		Stderr: &bytes.Buffer{},
		NewAgent: func(string) (func(context.Context, string) (loop.RunInfo, error), func(), func() string, error) {
			return func(ctx context.Context, _ string) (loop.RunInfo, error) {
				<-ctx.Done()
				return loop.RunInfo{}, ctx.Err()
			}, func() {}, func() string { return "fake/m1" }, nil
		},
	})
	require.NoError(t, err)
	raw := rawReport(t, out.RunDir)
	require.Equal(t, "TIMEOUT", raw["reason"])
	require.Equal(t, "FAIL", raw["verify_status"])
	require.EqualValues(t, 1, raw["verify_exit_code"])
	require.NotContains(t, raw, "changed_files")
}

// F1+F2: stream_retries=0 và stream im ("no data"): STREAM_STALL ngay sau lượt đầu (không chờ
// chạy lại), verify vẫn đo một lần.
func TestExecuteRoutine_StreamStallWithoutRetries(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	zero := 0
	e.opts.Def.StreamRetries = &zero
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second) // một lần chờ chạy lại là 30s
	defer cancel()
	out, err := executeRoutine(ctx, e.opts, e.deps(func(string) error {
		return errors.New("agent run failed: LLM stream received no data for 2m0s")
	}))
	require.NoError(t, err)
	require.Equal(t, loop.StopStreamStall, out.Result.Reason, out.Report.Detail)
	require.Equal(t, 1, e.runs)
	raw := rawReport(t, out.RunDir)
	require.Equal(t, "STREAM_STALL", raw["reason"])
	require.Equal(t, "FAIL", raw["verify_status"])
	require.EqualValues(t, 1, raw["verify_exit_code"])
}

// F4: hết hạn mức: QUOTA ngay, không chạy lại, chi tiết có lời provider và mã HTTP (một lần).
func TestExecuteRoutine_QuotaStopsAtOnce(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	perr := &fantasy.ProviderError{Title: "too many requests", StatusCode: 429, Message: "Usage limit reached for 5 hour"}
	out, err := executeRoutine(ctx, e.opts, e.deps(func(string) error {
		return fmt.Errorf("agent processing failed: %w", perr)
	}))
	require.NoError(t, err)
	require.Equal(t, loop.StopQuota, out.Result.Reason, out.Report.Detail)
	require.Equal(t, 1, e.runs)
	require.Contains(t, out.Report.Detail, "Usage limit reached for 5 hour")
	require.Equal(t, 1, strings.Count(out.Report.Detail, "HTTP 429"), out.Report.Detail)
	require.Equal(t, "QUOTA", rawReport(t, out.RunDir)["reason"])
	require.Error(t, routineExitErr(e.opts.Def.Mode, out.Result.Reason))
}

// 5xx hết lần chạy lại: RUN_ERROR, mã HTTP nêu đúng một lần (engine đã nêu, routine không lặp).
func TestExecuteRoutine_ServerErrorAfterRetriesNamesTheStatusOnce(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	zero := 0
	e.opts.Def.StreamRetries = &zero
	perr := &fantasy.ProviderError{Title: "service unavailable", StatusCode: 503, Message: "busy"}
	out, err := executeRoutine(t.Context(), e.opts, e.deps(func(string) error {
		return fmt.Errorf("agent processing failed: %w", perr)
	}))
	require.NoError(t, err)
	require.Equal(t, loop.StopRunError, out.Result.Reason)
	require.Equal(t, 1, strings.Count(out.Report.Detail, "HTTP 503"), out.Report.Detail)
	require.Equal(t, "FAIL", out.Report.VerifyStatus, "verify chạy sau khi lượt bị cắt")
}

// F3: huỷ giữa lượt (SIGTERM/SIGINT đi qua signal.NotifyContext thành ctx bị huỷ): báo cáo
// đóng thành CANCELLED có finished, và khoá của routine được nhả.
func TestExecuteRoutine_CancelledRunIsClosedAndReleasesTheLock(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	out, err := executeRoutine(ctx, e.opts, e.deps(func(string) error {
		cancel()
		return context.Canceled
	}))
	require.NoError(t, err)
	raw := rawReport(t, out.RunDir)
	require.Equal(t, "CANCELLED", raw["reason"])
	require.NotEmpty(t, raw["finished"])
	release, err := lock.TryFile(routineLockPath(e.data, "nightly"))
	require.NoError(t, err, "a cancelled run releases its lock")
	release()
}

// Verify sau khi bị cắt không đo được (quá verify_timeout): verify_status UNMEASURED và KHÔNG
// có verify_exit_code (0 đọc như đã qua).
func TestExecuteRoutine_UnmeasuredVerifyAfterACutHasNoExitCode(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.opts.Def.RunTimeout, e.opts.Def.VerifyTimeout = "200ms", "500ms"
	e.opts.Def.Verify = "if [ -f done.txt ]; then sleep 5; fi; exit 1" // baseline FAIL nhanh, sau khi cắt thì treo
	out, err := executeRoutine(t.Context(), e.opts, routineDeps{
		Stderr: &bytes.Buffer{},
		NewAgent: func(wt string) (func(context.Context, string) (loop.RunInfo, error), func(), func() string, error) {
			return func(ctx context.Context, _ string) (loop.RunInfo, error) {
				_ = os.WriteFile(filepath.Join(wt, "done.txt"), []byte("x"), 0o644)
				<-ctx.Done()
				return loop.RunInfo{}, ctx.Err()
			}, func() {}, func() string { return "fake/m1" }, nil
		},
	})
	require.NoError(t, err)
	raw := rawReport(t, out.RunDir)
	require.Equal(t, "TIMEOUT", raw["reason"], raw["detail"])
	require.Equal(t, "UNMEASURED", raw["verify_status"])
	require.NotContains(t, raw, "verify_exit_code", "a verify that timed out has no exit code")
	var b bytes.Buffer
	require.NoError(t, printLoopJSON(&b, loopOutcome{Result: out.Result}))
	require.NotContains(t, b.String(), "verify_exit_code")
}

// Soát độc lập 21/10: model mà `routine new` đóng băng trùng model large mặc định thì lượt chạy
// phải y như mặc định. Áp nó như -m sẽ thay cả SelectedModel (mất max_tokens, temperature,
// reasoning_effort...) và đổi model nhỏ sang model nhỏ mặc định của provider. Provider "fake"
// không có trong catwalk nên model nhỏ dự phòng là chính model large: không che được việc đổi.
func TestNewLocalLoopAgent_FrozenDefaultRunsExactlyLikeTheDefault(t *testing.T) {
	var (
		mu    sync.Mutex
		calls []map[string]any
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		calls = append(calls, body)
		mu.Unlock()
		if body["stream"] != true {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"1","object":"chat.completion","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"Title"},"finish_reason":"stop"}]}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, d := range []string{
			`{"id":"1","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":"ok"},"finish_reason":null}]}`,
			`{"id":"1","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		} {
			fmt.Fprintf(w, "data: %s\n\n", d)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	_, repo := sandboxConfig(t, "m1")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "crush.json"), []byte(`{"providers":{"fake":{"type":"openai-compat","base_url":"`+srv.URL+`/v1","api_key":"x","models":[`+
		`{"id":"m1","name":"m1","context_window":100000,"default_max_tokens":1000,"can_reason":true,"reasoning_levels":["low","high"]},`+
		`{"id":"m2","name":"m2","context_window":100000,"default_max_tokens":1000}]}},`+
		`"models":{"large":{"provider":"fake","model":"m1","max_tokens":777,"temperature":0.3,"reasoning_effort":"high"},"small":{"provider":"fake","model":"m2"}},`+
		`"options":{"disable_provider_auto_update":true,"disable_metrics":true}}`), 0o600))
	t.Chdir(repo) // setupLocalWorkspace chdir vào workspace; t.Chdir trả lại cwd sau test
	c := &cobra.Command{}
	c.Flags().Bool("debug", false, "")
	c.Flags().Bool("yolo", true, "")
	c.Flags().StringSlice("channels", nil, "")
	c.Flags().String("data-dir", "", "")
	c.Flags().String("cwd", "", "")
	c.SetContext(t.Context())

	agent, cleanup, model, err := newLocalLoopAgent(c, loopParams{Model: "fake/m1"}, io.Discard)
	require.NoError(t, err)
	defer cleanup()
	_, err = agent(t.Context(), "LARGEPROMPT do the thing")
	require.NoError(t, err)
	require.Equal(t, "fake/m1", model())

	var large map[string]any
	smallSeen := false
	defer func() { // tóm tắt từng request khi hỏng (prompt hệ thống quá dài để in nguyên)
		if t.Failed() {
			mu.Lock()
			defer mu.Unlock()
			for _, b := range calls {
				raw, _ := json.Marshal(b["messages"])
				t.Logf("model=%v max_tokens=%v temperature=%v reasoning_effort=%v stream=%v main=%v",
					b["model"], cmp.Or(b["max_tokens"], b["max_completion_tokens"]), b["temperature"], b["reasoning_effort"], b["stream"], strings.Contains(string(raw), "LARGEPROMPT"))
			}
		}
	}()
	require.Eventually(t, func() bool { // tiêu đề session chạy bằng model nhỏ, có thể sau lượt
		mu.Lock()
		defer mu.Unlock()
		for _, b := range calls {
			raw, _ := json.Marshal(b["messages"])
			if strings.Contains(string(raw), "LARGEPROMPT") && !strings.Contains(string(raw), "Generate a concise title") {
				large = b
			}
			smallSeen = smallSeen || b["model"] == "m2"
		}
		return large != nil && smallSeen
	}, 10*time.Second, 50*time.Millisecond, "need the main request and a request with the configured small model m2")
	if t.Failed() {
		return
	}
	require.Equal(t, "m1", large["model"])
	require.EqualValues(t, 777, cmp.Or(large["max_tokens"], large["max_completion_tokens"]), "the user's max_tokens survives")
	require.EqualValues(t, 0.3, large["temperature"], "the user's temperature survives")
	require.Equal(t, "high", large["reasoning_effort"], "the user's reasoning_effort survives")
}
