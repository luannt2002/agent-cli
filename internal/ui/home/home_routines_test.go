package home

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/charmbracelet/crush/internal/routine"
)

// TestHelperSleep is not a test: startRoutineProc re-executes the test binary
// into it, so a child with `routine run <name>` in its argv stays alive.
func TestHelperSleep(t *testing.T) {
	if os.Getenv("HOME_ROUTINES_HELPER") == "1" {
		time.Sleep(time.Minute)
	}
}

func startRoutineProc(t *testing.T, name string) int {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("liveness is not supported on Windows")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperSleep$", "routine", "run", name)
	cmd.Env = append(os.Environ(), "HOME_ROUTINES_HELPER=1")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	return cmd.Process.Pid
}

func deadPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	require.NoError(t, cmd.Run())
	return cmd.Process.Pid
}

func writeReport(t *testing.T, dir, name string, rep any) {
	t.Helper()
	d := filepath.Join(dir, name, "runs", "20261021-100000")
	if _, ok := rep.([]byte); ok {
		// Unreadable: ordered by its stamp, which is newer than the others' start.
		d = filepath.Join(dir, name, "runs", time.Now().UTC().Format("20060102-150405"))
	}
	require.NoError(t, os.MkdirAll(d, 0o755))
	raw, ok := rep.([]byte)
	if !ok {
		var err error
		raw, err = json.Marshal(rep)
		require.NoError(t, err)
	}
	require.NoError(t, os.WriteFile(filepath.Join(d, "report.json"), raw, 0o644))
}

// routinesFixture is a routines dir with one routine of each kind and the
// worktree of "green" on disk. "recent" started a minute ago.
func routinesFixture(t *testing.T) (dir, worktree string) {
	t.Helper()
	dir, worktree = t.TempDir(), t.TempDir()
	started := time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339)
	done := time.Now().Add(-10*time.Minute + 95*time.Second).UTC().Format(time.RFC3339)
	writeReport(t, dir, "live", map[string]any{"started": started, "reason": "RUNNING", "pid": startRoutineProc(t, "live")})
	writeReport(t, dir, "dead", map[string]any{"started": started, "reason": "RUNNING", "pid": deadPID(t), "worktree": filepath.Join(dir, "gone")})
	writeReport(t, dir, "green", map[string]any{"started": started, "finished": done, "reason": "GREEN", "verify_status": "PASS",
		"attempts": 1, "changed_files": []string{"a.go", "b.go"}, "worktree": worktree})
	writeReport(t, dir, "rerr", map[string]any{"started": started, "finished": done, "reason": "RUN_ERROR", "verify_exit_code": 0})
	writeReport(t, dir, "bad", []byte(`{"started": `))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "notroutine", "x"), 0o755))
	return dir, worktree
}

func routineModel(t *testing.T, dir string) (*Model, *fakeSource) {
	t.Helper()
	src := &fakeSource{workspaces: tenWorkspaces()[:2]}
	m := newTestModel(src)
	m.routinesDir = dir
	drive(m, m.Init())
	return m, src
}

// routineLines returns the rendered lines under the "Routines" header.
func routineLines(m *Model) []string {
	var out []string
	in := false
	for _, l := range strings.Split(screenText(m), "\n") {
		switch {
		case strings.Contains(l, "Routines"):
			in = true
		case in && strings.TrimSpace(l) != "" && !strings.Contains(l, "→"):
			out = append(out, strings.TrimSpace(l))
		case in:
			return out
		}
	}
	return out
}

func TestRoutinesSection(t *testing.T) {
	dir, _ := routinesFixture(t)
	m, _ := routineModel(t, dir)
	got := routineLines(m)
	t.Log("\n" + strings.Join(got, "\n"))
	require.Len(t, got, 5, "live dead green rerr bad; notroutine skipped")
	require.Equal(t, "bad  report unreadable", got[1], "unreadable is ordered by its stamp (newest here)")
	// The clock is real: 10m00s to 10m05s depending on how slow the test is.
	require.Regexp(t, `^live  running  10m0[0-5]s  verify not run$`, got[0], "running first; no files/attempts columns while running")

	text := strings.Join(got, "\n")
	require.Contains(t, text, "green  green  1m35s  verify PASS  2 files  1 attempts")
	require.Contains(t, text, "rerr  run error  1m35s  verify not run", "RUN_ERROR never looks like a pass")
	require.NotContains(t, text, "rerr  run error  1m35s  verify PASS")
	require.Contains(t, text, "dead  abandoned  verify not run\n", "RUNNING with a dead pid is not running")
	require.NotContains(t, text, "dead  running")
	require.NotContains(t, text, "notroutine")
}

func TestRoutinesColoredByStatus(t *testing.T) {
	dir, _ := routinesFixture(t)
	m, _ := routineModel(t, dir)
	fg := func(name string) string {
		i := slices.IndexFunc(m.rows, func(r row) bool { return r.kind == rowRoutine && r.run.Routine == name })
		require.GreaterOrEqual(t, i, 0, name)
		return fmt.Sprint(m.rowSegments(m.rows[i])[1].style.GetForeground()) // The status segment.
	}
	ok, bad, run, dead := fg("green"), fg("rerr"), fg("live"), fg("dead")
	require.NotEqual(t, ok, bad, "ok and error differ")
	require.NotEqual(t, run, bad)
	require.NotEqual(t, run, ok)
	require.Equal(t, bad, dead, "abandoned is error-coloured")
}

func TestRoutinesDoNotBreakOrStealCursor(t *testing.T) {
	dir, _ := routinesFixture(t)
	m, _ := routineModel(t, dir)
	r, ok := m.selected()
	require.True(t, ok)
	require.NotEqual(t, rowRoutine, r.kind, "default selection stays on a workspace")
	require.Contains(t, screenText(m), "/w/a", "workspaces still listed")
}

func TestNoRoutinesDirNoSection(t *testing.T) {
	m, _ := routineModel(t, filepath.Join(t.TempDir(), "missing"))
	require.NotContains(t, screenText(m), "Routines")
	m2 := newTestModel(&fakeSource{workspaces: tenWorkspaces()[:2]})
	require.NotContains(t, screenText(m2), "Routines", "no routinesDir: Home reads nothing")
}

func TestRoutinesCapped(t *testing.T) {
	dir := t.TempDir()
	for i := range RoutineLimit + 3 {
		writeReport(t, dir, fmt.Sprintf("r%d", i), map[string]any{"started": time.Now().Add(-time.Duration(i) * time.Hour).UTC().Format(time.RFC3339), "reason": "GREEN"})
	}
	m, _ := routineModel(t, dir)
	require.Len(t, routineLines(m), RoutineLimit)
}

func selectRoutine(t *testing.T, m *Model, name string) {
	selectRow(t, m, func(r row) bool { return r.kind == rowRoutine && r.run.Routine == name })
}

func TestEnterOnRoutine(t *testing.T) {
	dir, worktree := routinesFixture(t)

	m, src := routineModel(t, dir)
	selectRoutine(t, m, "live")
	press(m, keyEnter)
	require.Equal(t, screenWorkspaces, m.screen)
	require.Contains(t, m.note, "still running")

	selectRoutine(t, m, "rerr") // no worktree in the report
	press(m, keyEnter)
	require.Equal(t, screenWorkspaces, m.screen)
	require.Contains(t, m.note, "no worktree")

	selectRoutine(t, m, "dead") // worktree gone from disk
	press(m, keyEnter)
	require.Equal(t, screenWorkspaces, m.screen)
	require.Contains(t, m.note, "no longer exists")

	selectRoutine(t, m, "green")
	press(m, keyEnter)
	require.Equal(t, screenSessions, m.screen)
	require.Equal(t, worktree, m.ws.Path)
	require.Equal(t, []string{worktree}, src.listed)
}

func TestRoutineUnknownLivenessIsNotOpened(t *testing.T) {
	dir, wt := t.TempDir(), t.TempDir()
	// An old build's RUNNING report: no pid, so alive and dead cannot be told apart.
	writeReport(t, dir, "old", map[string]any{"started": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), "reason": "RUNNING", "worktree": wt})
	m, src := routineModel(t, dir)
	t.Log("\n" + strings.Join(routineLines(m), "\n"))
	require.Equal(t, []string{"old  unknown  verify not run"}, routineLines(m))
	selectRoutine(t, m, "old")
	press(m, keyEnter)
	require.Equal(t, screenWorkspaces, m.screen, "worktree exists, but the run may be alive")
	require.Contains(t, m.note, "cannot tell")
	require.Empty(t, src.listed)
}

func TestRoutinesComeAfterTheActions(t *testing.T) {
	dir, _ := routinesFixture(t)
	m, _ := routineModel(t, dir)
	// 15 lines leave room for 9 rows: routines placed first would push the actions off screen.
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 15})
	var kinds []rowKind
	for _, r := range m.rows {
		kinds = append(kinds, r.kind)
	}
	first := slices.Index(kinds, rowRoutine)
	require.Greater(t, first, slices.Index(kinds, rowShowAll), "routines after Show all")
	for _, want := range []string{"Open current folder", "Show all"} {
		require.Contains(t, screenText(m), want)
	}
}

func TestBackFromRoutineReturnsToItsRow(t *testing.T) {
	dir, _ := routinesFixture(t)
	m, _ := routineModel(t, dir)
	selectRoutine(t, m, "green")
	press(m, keyEnter)
	require.Equal(t, screenSessions, m.screen)
	press(m, keyEsc)
	r, ok := m.selected()
	require.True(t, ok)
	require.Equal(t, "green", r.run.Routine)
}

func TestCursorFollowsRoutineAcrossRefresh(t *testing.T) {
	mk := func(names ...string) []routine.RunSummary {
		var l []routine.RunSummary
		for _, n := range names {
			l = append(l, routine.RunSummary{Routine: n, Reason: "GREEN"})
		}
		return l
	}
	m, _ := routineModel(t, t.TempDir())
	m.Update(liveMsg{live: map[string]Live{}, routines: mk("a", "b", "c")})
	selectRoutine(t, m, "b")
	cur := func() string {
		r, _ := m.selected()
		return r.run.Routine
	}
	m.Update(liveMsg{live: map[string]Live{}, routines: mk("c", "b", "a")})
	require.Equal(t, "b", cur(), "reordered: the cursor stays on routine b")

	m.Update(liveMsg{live: map[string]Live{}, routines: mk("c", "a")}) // b is gone
	gone := cur()
	require.Contains(t, []string{"c", "a"}, gone, "falls to a neighbouring routine")
	m.Update(liveMsg{live: map[string]Live{}, routines: mk("a", "c")})
	require.Equal(t, gone, cur(), "and stays there: a routine row is an item worth following")
}

// Verify xanh trên run không đạt (vd bị cắt ở TIMEOUT) hiện là cảnh báo cần soát, không phải "chưa chạy" hay đạt.
func TestRoutinePassOnARunThatIsNotGreen(t *testing.T) {
	dir := t.TempDir()
	started := time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339)
	done := time.Now().Add(-9 * time.Minute).UTC().Format(time.RFC3339)
	writeReport(t, dir, "cut", map[string]any{"started": started, "finished": done, "reason": "TIMEOUT", "verify_status": "PASS", "attempts": 1})
	m, _ := routineModel(t, dir)
	text := strings.Join(routineLines(m), "\n")
	require.Contains(t, text, "cut  timeout  1m00s  verify PASS (run not green — review)")
	require.NotContains(t, text, "verify not run")
}
