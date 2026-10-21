package routine

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestHelperSleep is not a test: startFake re-executes the test binary into
// it, so a child with `routine run <name>` in its argv stays alive.
func TestHelperSleep(t *testing.T) {
	if os.Getenv("ROUTINE_RUNS_HELPER") == "1" {
		time.Sleep(time.Minute)
	}
}

// startFake starts a child whose argv contains `routine run <name>`.
func startFake(t *testing.T, name string) int {
	t.Helper()
	return startArgs(t, "routine", "run", name)
}

func startArgs(t *testing.T, args ...string) int {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("liveness is not supported on Windows")
	}
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestHelperSleep$"}, args...)...)
	cmd.Env = append(os.Environ(), "ROUTINE_RUNS_HELPER=1")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	return cmd.Process.Pid
}

// deadPID is the pid of a child that already exited and was reaped.
func deadPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	require.NoError(t, cmd.Run())
	return cmd.Process.Pid
}

// writeRun writes <dir>/<name>/runs/<stamp>/report.json from rep, or raw bytes.
func writeRun(t *testing.T, dir, name, stamp string, rep any) {
	t.Helper()
	d := filepath.Join(dir, name, "runs", stamp)
	require.NoError(t, os.MkdirAll(d, 0o755))
	raw, ok := rep.([]byte)
	if !ok {
		var err error
		raw, err = json.Marshal(rep)
		require.NoError(t, err)
	}
	require.NoError(t, os.WriteFile(filepath.Join(d, "report.json"), raw, 0o644))
}

var t0 = time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)

func ts(d time.Duration) string { return t0.Add(d).Format(time.RFC3339) }

func fixture(t *testing.T) (dir string, by map[string]RunSummary) {
	t.Helper()
	dir = t.TempDir()
	wt := t.TempDir()
	live := startFake(t, "live")
	// An older run must lose to the newest stamp.
	writeRun(t, dir, "live", "20261021-090000", map[string]any{"routine": "live", "started": ts(-3 * time.Hour), "reason": "GREEN"})
	writeRun(t, dir, "live", "20261021-100000", map[string]any{"routine": "live", "started": ts(0), "reason": "RUNNING", "pid": live, "worktree": "/wt/live"})
	writeRun(t, dir, "dead", "20261021-100000", map[string]any{"routine": "dead", "started": ts(0), "reason": "RUNNING", "pid": deadPID(t)})
	writeRun(t, dir, "old", "20261021-100000", map[string]any{"routine": "old", "started": ts(0), "finished": "0001-01-01T00:00:00Z", "reason": "RUNNING"})
	writeRun(t, dir, "green", "20261021-100000", map[string]any{"routine": "green", "started": ts(0), "finished": ts(95 * time.Second), "reason": "GREEN",
		"worktree": wt, "verify_status": "PASS", "verify_exit_code": 0, "attempts": 1, "changed_files": []string{"a.go", "b.go"}})
	writeRun(t, dir, "rerr", "20261021-100000", map[string]any{"routine": "rerr", "started": ts(0), "finished": ts(time.Minute), "reason": "RUN_ERROR", "verify_exit_code": 0})
	writeRun(t, dir, "rpass", "20261021-100000", map[string]any{"routine": "rpass", "started": ts(0), "finished": ts(time.Minute), "reason": "RUN_ERROR", "verify_status": "PASS"})
	writeRun(t, dir, "tpass", "20261021-100000", map[string]any{"routine": "tpass", "started": ts(0), "finished": ts(time.Minute), "reason": "TIMEOUT", "verify_status": "PASS"})
	writeRun(t, dir, "spass", "20261021-100000", map[string]any{"routine": "spass", "started": ts(0), "finished": ts(time.Minute), "reason": "STREAM_STALL", "verify_status": "PASS"})
	writeRun(t, dir, "epass", "20261021-100000", map[string]any{"routine": "epass", "started": ts(0), "finished": ts(time.Minute), "reason": "ERROR", "verify_status": "PASS"})
	writeRun(t, dir, "ncpass", "20261021-100000", map[string]any{"routine": "ncpass", "started": ts(0), "finished": ts(time.Minute), "reason": "NO_CHANGE", "verify_status": "PASS"})
	writeRun(t, dir, "bad", "20261021-110000", []byte(`{"routine": "bad", "sta`))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "notroutine", "misc"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stray.txt"), []byte("x"), 0o644))
	by = map[string]RunSummary{}
	for _, r := range LatestRuns(dir, t0.Add(10*time.Minute)) {
		by[r.Routine] = r
	}
	return dir, by
}

func TestLatestRuns(t *testing.T) {
	_, by := fixture(t)
	require.Len(t, by, 11, "live dead old green rerr rpass tpass spass epass ncpass bad; notroutine and stray.txt are skipped")

	live := by["live"]
	require.True(t, live.Running)
	require.Equal(t, "RUNNING", live.Reason)
	require.Equal(t, 10*time.Minute, live.Duration)
	require.Equal(t, "20261021-100000", live.Stamp, "newest stamp wins")

	require.Equal(t, "ABANDONED", by["dead"].Reason, "pid present and provably dead")
	require.False(t, by["dead"].Running)
	require.False(t, by["dead"].Unknown)
	// No pid (old build): cannot tell dead from alive, so never "abandoned".
	require.Equal(t, "UNKNOWN", by["old"].Reason)
	require.True(t, by["old"].Unknown)
	require.False(t, by["old"].Running)
	require.Zero(t, by["old"].Duration)

	g := by["green"]
	require.Equal(t, "GREEN", g.Reason)
	require.Equal(t, "PASS", g.Verify)
	require.Equal(t, 95*time.Second, g.Duration)
	require.Equal(t, 2, g.Changed)
	require.True(t, g.WorktreeExists)
	require.False(t, by["live"].WorktreeExists, "/wt/live is not on disk")

	e := by["rerr"]
	require.Equal(t, "RUN_ERROR", e.Reason)
	require.Empty(t, e.Verify, "absent verify_status is never a pass, whatever the exit code")

	// Verify xanh chỉ là đạt khi run GREEN/NO_CHANGE. Run kết thúc khác mà verify có chạy và xanh (bị cắt, lỗi…)
	// thì nói đúng là verify xanh nhưng run không đạt, không giấu thành "chưa chạy".
	require.Equal(t, "PASS", by["ncpass"].Verify, "NO_CHANGE with a passing verify is a pass")
	for _, n := range []string{"rpass", "tpass", "spass", "epass"} {
		require.Equal(t, VerifyPassNotGreen, by[n].Verify, "%s: a passing verify on a run that is not green is reported as such", n)
	}

	require.True(t, by["bad"].Unreadable)
}

func TestLatestRunsOrder(t *testing.T) {
	dir, _ := fixture(t)
	got := LatestRuns(dir, t0.Add(time.Hour))
	require.Equal(t, "live", got[0].Routine, "running first")
	require.Equal(t, "bad", got[1].Routine, "unreadable report is ordered by its run stamp (newest here), not last")
}

func TestLatestRunsMissingDir(t *testing.T) {
	require.Empty(t, LatestRuns(filepath.Join(t.TempDir(), "nope"), t0))
}

func TestRunAlive(t *testing.T) {
	pid := startFake(t, "x")
	require.True(t, RunAlive(pid, "x"))
	if _, err := os.Stat("/proc/self"); err == nil { // Without /proc (macOS) only Signal(0) is checked.
		require.False(t, RunAlive(pid, "other"), "pid alive but argv has another routine name")
		require.False(t, RunAlive(os.Getpid(), "x"), "pid alive but is not a routine run")
		require.False(t, RunAlive(startArgs(t, "run", "x"), "x"), "argv without `routine`")
		require.False(t, RunAlive(startArgs(t, "routine", "x"), "x"), "argv without `run`")
	}
	require.True(t, RunAlive(startArgs(t, "routine", "run", "/d/x.json"), "x"), "definition path form")
	require.False(t, RunAlive(0, "x"))
	require.False(t, RunAlive(deadPID(t), "x"))
}

// Model và Repo đọc từ report.json (provider/model đã phân giải, repo của lượt); vắng thì rỗng.
func TestLatestRunsModelAndRepo(t *testing.T) {
	dir := t.TempDir()
	writeRun(t, dir, "with", "20261021-100000", map[string]any{"started": ts(0), "finished": ts(time.Minute), "reason": "GREEN", "model": "zai/glm-5.3", "repo": "/r/dew"})
	writeRun(t, dir, "without", "20261021-100000", map[string]any{"started": ts(0), "finished": ts(time.Minute), "reason": "GREEN"})
	by := map[string]RunSummary{}
	for _, r := range LatestRuns(dir, t0) {
		by[r.Routine] = r
	}
	require.Equal(t, "zai/glm-5.3", by["with"].Model)
	require.Equal(t, "/r/dew", by["with"].Repo)
	require.Empty(t, by["without"].Model)
	require.Empty(t, by["without"].Repo)
}

func TestLatestRunsUnreadableNotHiddenByOrder(t *testing.T) {
	dir := t.TempDir()
	for i := range 7 {
		writeRun(t, dir, fmt.Sprintf("ok%d", i), "20261021-090000", map[string]any{"started": ts(-time.Hour), "finished": ts(-time.Minute), "reason": "GREEN"})
	}
	writeRun(t, dir, "bad", "20261021-120000", []byte("{"))
	got := LatestRuns(dir, t0)
	require.Equal(t, "bad", got[0].Routine, "newest stamp first, so a row cap cannot hide it")
}
