package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/loop"
	"github.com/charmbracelet/crush/internal/routine"
	"github.com/stretchr/testify/require"
)

// Cổng xanh từ trước, agent không đổi gì: NO_CHANGE, và `loop` thoát khác 0.
func TestExecuteLoop_NoChangeWhenTheGateAlreadyPassesAndNothingChanged(t *testing.T) {
	t.Parallel()
	dir := gitInit(t)
	p := newLoopParams(t, dir, "true")
	agent := func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		return loop.RunInfo{SessionID: "s"}, nil
	}
	out, err := executeLoop(t.Context(), p, loopDeps{Agent: agent, Stderr: &bytes.Buffer{}})
	require.NoError(t, err)
	require.Equal(t, loop.StopNoChange, out.Result.Reason)
	require.Error(t, loopExitErr(out.Result.Reason))
}

func TestLoopExitErr_OnlyGreenExitsZero(t *testing.T) {
	t.Parallel()
	require.NoError(t, loopExitErr(loop.StopGreen))
	for _, r := range []loop.Stop{loop.StopNoChange, loop.StopMaxRepairs, loop.StopStalled, loop.StopUnmeasured} {
		require.ErrorContains(t, loopExitErr(r), string(r))
	}
}

// NO_CHANGE là kết quả hợp lệ của routine bảo trì (hôm nay không có việc), nhưng
// với goal thì việc chưa được chứng minh.
func TestRoutineExitErr_NoChangeIsZeroOnlyForMaintenance(t *testing.T) {
	t.Parallel()
	require.NoError(t, routineExitErr(routine.ModeMaintenance, loop.StopNoChange))
	require.ErrorContains(t, routineExitErr(routine.ModeGoal, loop.StopNoChange), "NO_CHANGE")
	require.NoError(t, routineExitErr(routine.ModeMaintenance, loop.StopGreen))
	require.NoError(t, routineExitErr(routine.ModeGoal, loop.StopGreen))
	require.Error(t, routineExitErr(routine.ModeMaintenance, loop.StopMaxRepairs))
	require.Error(t, routineExitErr(routine.ModeMaintenance, loop.StopStalled))
}

// Báo cáo ghi NO_CHANGE, và list/show (cùng đọc listRuns) hiện đúng lý do đó.
func TestExecuteRoutine_NoChangeIsReportedAndListed(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.opts.Def.Verify = "true"
	out, err := executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err)
	require.Equal(t, loop.StopNoChange, out.Result.Reason)
	require.Equal(t, "NO_CHANGE", out.Report.Reason)
	require.NoError(t, routineExitErr(e.opts.Def.Mode, out.Result.Reason))

	runs := listRuns(e.data, "nightly")
	require.Len(t, runs, 1)
	require.Equal(t, "NO_CHANGE", runs[0].Reason)

	defs := t.TempDir()
	body := `{"name":"nightly","mode":"maintenance","repo":"` + e.repo + `","goal":"g","verify":"true"}`
	require.NoError(t, os.WriteFile(filepath.Join(defs, "nightly.json"), []byte(body), 0o600))
	entries := listRoutines(defs, e.data)
	require.Len(t, entries, 1)
	require.Equal(t, "NO_CHANGE", entries[0].LastReason)
}

// Cùng tình huống ở goal mode: báo cáo NO_CHANGE, `routine run` thoát 1.
func TestExecuteRoutine_NoChangeInGoalModeExitsNonZero(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.opts.Def.Verify = "true"
	e.opts.Def.Mode = routine.ModeGoal
	out, err := executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.NoError(t, err)
	require.Equal(t, "NO_CHANGE", out.Report.Reason)
	require.Error(t, routineExitErr(e.opts.Def.Mode, out.Result.Reason))
}

// Allowlist trong env của shell không đi vào lịch nữa: in cảnh báo bảo người dùng
// đặt nó vào tệp env. Không đặt thì im lặng.
func TestWarnAllowlistNotScheduled(t *testing.T) {
	t.Setenv("AGENT_CLI_ALLOWED_MODELS", "openai:gpt-5")
	var b bytes.Buffer
	warnAllowlistNotScheduled(&b)
	require.Contains(t, b.String(), "AGENT_CLI_ALLOWED_MODELS")
	require.Contains(t, b.String(), "env file")

	t.Setenv("AGENT_CLI_ALLOWED_MODELS", "")
	b.Reset()
	warnAllowlistNotScheduled(&b)
	require.Empty(t, b.String())
}
