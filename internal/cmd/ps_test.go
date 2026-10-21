package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/lock"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/stretchr/testify/require"
)

type fakePSSource struct {
	failInfo     map[string]bool
	failSessions map[string]bool
	workspaces   []proto.Workspace
	sessions     map[string][]proto.Session
	busy         map[string]bool
	waiting      map[string]bool
}

func (f fakePSSource) ListWorkspaces(context.Context) ([]proto.Workspace, error) {
	return f.workspaces, nil
}

func (f fakePSSource) ListSessions(_ context.Context, id string) ([]proto.Session, error) {
	if f.failSessions[id] {
		return nil, errors.New("boom sessions")
	}
	return f.sessions[id], nil
}

func (f fakePSSource) GetAgentSessionInfo(_ context.Context, _, sid string) (*proto.AgentSession, error) {
	if f.failInfo[sid] {
		return nil, errors.New("404")
	}
	return &proto.AgentSession{IsBusy: f.busy[sid], PendingPrompt: f.waiting[sid]}, nil
}

func newFakePSSource() fakePSSource {
	return fakePSSource{
		workspaces: []proto.Workspace{{ID: "w1", Path: "/p/one"}, {ID: "w2", Path: "/p/two"}},
		sessions: map[string][]proto.Session{
			"w1": {{ID: "a", Title: "alpha"}, {ID: "b", Title: "beta"}},
			"w2": {{ID: "c", Title: "gamma"}, {ID: "d", Title: "delta"}},
		},
		busy:    map[string]bool{"b": true, "d": true},
		waiting: map[string]bool{"d": true},
	}
}

func TestCollectPSRowsAcrossWorkspaces(t *testing.T) {
	t.Parallel()

	rows, err := collectPS(t.Context(), newFakePSSource(), 9)
	require.NoError(t, err)
	require.Equal(t, []psRow{
		{Path: "/p/one", SessionID: "a", Title: "alpha"},
		{Path: "/p/one", SessionID: "b", Title: "beta", Busy: true},
		{Path: "/p/two", SessionID: "c", Title: "gamma"},
		{Path: "/p/two", SessionID: "d", Title: "delta", Busy: true, Waiting: true},
	}, rows)

	limited, err := collectPS(t.Context(), newFakePSSource(), 1)
	require.NoError(t, err)
	require.Len(t, limited, 2)
}

func TestPSTableStatus(t *testing.T) {
	t.Parallel()

	rows, err := collectPS(t.Context(), newFakePSSource(), 9)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, writePS(&buf, rows, false))
	out := buf.String()
	require.Regexp(t, `PROJECT\s+SESSION\s+STATUS\s+TITLE`, out)
	require.Regexp(t, `/p/one\s+a\s+IDLE\s+alpha`, out)
	require.Regexp(t, `/p/one\s+b\s+BUSY\s+beta`, out)
	// A pending prompt wins over busy.
	require.Regexp(t, `/p/two\s+d\s+WAITING\s+delta`, out)
}

func TestPSJSONOutput(t *testing.T) {
	t.Parallel()

	rows, err := collectPS(t.Context(), newFakePSSource(), 9)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, writePS(&buf, rows, true))
	var got []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got, 4)
	require.Equal(t, "/p/one", got[0]["path"])
	require.Equal(t, "b", got[1]["session_id"])
	require.Equal(t, true, got[1]["busy"])
	require.Equal(t, true, got[3]["waiting"])
}

func TestPSNoServerDoesNotSpawn(t *testing.T) {
	t.Parallel()

	sock := filepath.Join(t.TempDir(), "dew-deadbeef.sock")
	var buf bytes.Buffer
	require.NoError(t, runPS(t.Context(), &buf, io.Discard, []string{"unix://" + sock}, false, 9, nil))
	require.Equal(t, "No dew server running (open projects with dew-tmux)\n", buf.String())
	// A spawned server would have created its socket.
	require.NoFileExists(t, sock)

	buf.Reset()
	require.NoError(t, runPS(t.Context(), &buf, io.Discard, []string{"unix://" + sock}, true, 9, nil))
	require.JSONEq(t, `[]`, buf.String())
}

func TestPSSocketDirsFindsSockets(t *testing.T) {
	rt := t.TempDir()
	t.Setenv("DEW_RUNTIME_DIR", rt)
	t.Setenv("XDG_RUNTIME_DIR", rt) // Duplicate dir must not duplicate hosts.
	t.Setenv("HOME", t.TempDir())
	want := filepath.Join(rt, "dew-12345678.sock")
	require.NoError(t, os.WriteFile(want, nil, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(rt, "other.sock"), nil, 0o600))

	require.Equal(t, []string{"unix://" + want}, discoverPSHosts())
}

func TestCollectPSSkipsFailingSession(t *testing.T) {
	t.Parallel()

	src := newFakePSSource()
	src.failInfo = map[string]bool{"a": true}
	rows, err := collectPS(t.Context(), src, 9)
	require.ErrorContains(t, err, "session a")
	require.Len(t, rows, 3)
	require.Equal(t, "b", rows[0].SessionID)
}

func TestCollectPSKeepsRowsWhenListSessionsFails(t *testing.T) {
	t.Parallel()

	src := newFakePSSource()
	src.failSessions = map[string]bool{"w1": true}
	rows, err := collectPS(t.Context(), src, 9)
	require.ErrorContains(t, err, "boom sessions")
	require.Len(t, rows, 2)
	require.Equal(t, "/p/two", rows[0].Path)
}

func TestPSTitleSanitized(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	require.NoError(t, writePS(&buf, []psRow{{Path: "/p", SessionID: "x", Title: "a\tb\nc"}}, false))
	require.Contains(t, buf.String(), "a b c")
	require.Len(t, bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")), 2)
}

func TestQueryPSHostTimeout(t *testing.T) {
	t.Parallel()

	sock := filepath.Join(t.TempDir(), "hung.sock")
	ln, err := net.Listen("unix", sock)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer conn.Close() // Accept and never answer.
		}
	}()

	start := time.Now()
	_, alive, err := queryPSHost(t.Context(), "unix://"+sock, 500*time.Millisecond, 9)
	require.Error(t, err)
	require.False(t, alive)
	require.Less(t, time.Since(start), 4*time.Second)
}

func TestPSLiveServerFailureIsNotNoServer(t *testing.T) {
	t.Parallel()

	sock := filepath.Join(t.TempDir(), "dew-live.sock")
	ln, err := net.Listen("unix", sock)
	require.NoError(t, err)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/health" {
			http.Error(w, "boom", http.StatusInternalServerError)
		}
	})}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	var out, errOut bytes.Buffer
	host := "unix://" + sock
	require.NoError(t, runPS(t.Context(), &out, &errOut, []string{host}, false, 9, nil))
	require.NotContains(t, out.String(), "No dew server running")
	require.Contains(t, out.String(), "PROJECT")
	require.Contains(t, errOut.String(), "Skipping "+host+": ")
}

func TestPSAllIncludesOldestBusySession(t *testing.T) {
	t.Parallel()

	src := fakePSSource{
		workspaces: []proto.Workspace{{ID: "w", Path: "/p"}},
		sessions:   map[string][]proto.Session{"w": nil},
		busy:       map[string]bool{"s11": true}, // Oldest, listed last.
	}
	for i := range 12 {
		src.sessions["w"] = append(src.sessions["w"], proto.Session{ID: fmt.Sprintf("s%d", i)})
	}

	rows, err := collectPS(t.Context(), src, psSessionsPerWorkspace)
	require.NoError(t, err)
	require.Len(t, rows, 9)
	for _, r := range rows {
		require.NotEqual(t, "s11", r.SessionID)
	}

	rows, err = collectPS(t.Context(), src, math.MaxInt)
	require.NoError(t, err)
	require.Len(t, rows, 12)
	require.Equal(t, psRow{Path: "/p", SessionID: "s11", Busy: true}, rows[11])
}

func TestPSAllFlagRegistered(t *testing.T) {
	t.Parallel()

	require.NotNil(t, psCmd.Flags().Lookup("all"))
}

func TestPSHungServerIsNotedNotNoServer(t *testing.T) {
	t.Parallel()

	sock := filepath.Join(t.TempDir(), "dew-hung.sock")
	ln, err := net.Listen("unix", sock)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer conn.Close() // Accept and never answer.
		}
	}()

	// Shorter than psServerTimeout is not injectable via runPS, so use the
	// real timeout; the test takes about 3s.
	var out, errOut bytes.Buffer
	host := "unix://" + sock
	require.NoError(t, runPS(t.Context(), &out, &errOut, []string{host}, false, 9, nil))
	require.NotContains(t, out.String(), "No dew server running")
	require.Contains(t, errOut.String(), "Skipping "+host+": not responding (")
}

func TestPSDeadServersStaySilent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// A stale socket file: bound then closed without unlinking.
	stale := filepath.Join(dir, "dew-stale.sock")
	ln, err := net.Listen("unix", stale)
	require.NoError(t, err)
	ln.(*net.UnixListener).SetUnlinkOnClose(false)
	require.NoError(t, ln.Close())

	var out, errOut bytes.Buffer
	hosts := []string{"unix://" + stale, "unix://" + filepath.Join(dir, "missing.sock")}
	require.NoError(t, runPS(t.Context(), &out, &errOut, hosts, false, 9, nil))
	require.Empty(t, errOut.String())
	require.Contains(t, out.String(), "No dew server running")
}

// Sessions whose status cannot be read are counted on stderr ("not counted
// = not seen"); stdout keeps the JSON of the rows that were read.
func TestPSCountsSkippedSessionsOnStderr(t *testing.T) {
	t.Parallel()

	sock := filepath.Join(t.TempDir(), "dew-skip.sock")
	ln, err := net.Listen("unix", sock)
	require.NoError(t, err)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
		case "/v1/workspaces":
			_ = json.NewEncoder(w).Encode([]proto.Workspace{{ID: "w", Path: "/p"}})
		case "/v1/workspaces/w/sessions":
			_ = json.NewEncoder(w).Encode([]proto.Session{{ID: "a"}, {ID: "b"}, {ID: "c"}})
		case "/v1/workspaces/w/agent/sessions/c":
			_ = json.NewEncoder(w).Encode(proto.AgentSession{IsBusy: true})
		default:
			http.Error(w, "gone", http.StatusNotFound)
		}
	})}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	var out, errOut bytes.Buffer
	host := "unix://" + sock
	require.NoError(t, runPS(t.Context(), &out, &errOut, []string{host}, true, 9, nil))
	var rows []psRow
	require.NoError(t, json.Unmarshal(out.Bytes(), &rows))
	require.Equal(t, []psRow{{Path: "/p", SessionID: "c", Busy: true}}, rows)
	require.Equal(t, 1, strings.Count(errOut.String(), "\n"), errOut.String())
	require.Contains(t, errOut.String(), "Skipping "+host+": 2 sessions skipped (session a: ")
}

// putRoutineRun ghi báo cáo của một lượt routine (body là JSON) vào data/name/runs/stamp.
func putRoutineRun(t *testing.T, data, name, stamp, body string) {
	t.Helper()
	dir := filepath.Join(data, name, "runs", stamp)
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "report.json"), []byte(body), 0o600))
}

// `routine run` chạy agent in-process, không có server: ps thấy nó qua báo cáo RUNNING.
// Báo cáo có pid thì chỉ hiện khi tiến trình còn sống và đúng là `routine run <tên>`
// (PID tái dùng sau khi khởi động lại không thành routine ma), KHÔNG thử khoá (thử khoá
// có thể làm một lượt vừa khởi động thoát 75). Báo cáo cũ không pid thì theo listRuns.
// Giới hạn theo hệ điều hành: Windows không có tín hiệu 0 (không bao giờ hiện hàng có
// pid), macOS không có /proc (không nhận ra PID tái dùng).
func TestRoutinePSShowsLiveRunsWithoutProbingTheLock(t *testing.T) {
	data := t.TempDir()
	dead := exec.Command(os.Args[0], "-test.run=^$")
	require.NoError(t, dead.Run())
	running := func(wt string, pid int) string {
		return fmt.Sprintf(`{"reason":"RUNNING","worktree":%q,"started":"2026-10-21T09:00:00Z","pid":%d}`, wt, pid)
	}
	putRoutineRun(t, data, "dead", "20261021-090000", running("/wt/dead", dead.ProcessState.Pid()))
	putRoutineRun(t, data, "done", "20261021-090000", `{"reason":"GREEN","worktree":"/wt/done"}`)
	putRoutineRun(t, data, "old", "20261021-090000", `{"reason":"RUNNING","worktree":"/wt/old","started":"2026-10-21T09:00:00Z"}`)
	putRoutineRun(t, data, "oldgone", "20261021-090000", `{"reason":"RUNNING","worktree":"/wt/oldgone"}`)
	release, err := lock.TryFile(routineLockPath(data, "old"))
	require.NoError(t, err)
	t.Cleanup(release)
	want, unprobed := []string{"old"}, []string{"dead", "done"}

	livePID := 0
	if runtime.GOOS != "windows" {
		live := exec.Command("sh", "-c", "sleep 30; true", "routine", "run", "alive")
		require.NoError(t, live.Start())
		t.Cleanup(func() { _ = live.Process.Kill(); _ = live.Wait() })
		livePID = live.Process.Pid
		putRoutineRun(t, data, "alive", "20261020-090000", `{"reason":"GREEN"}`)
		putRoutineRun(t, data, "alive", "20261021-090000", running("/wt/alive", livePID))
		putRoutineRun(t, data, "stale", "20261021-090000", running("/wt/stale", livePID))
		putRoutineRun(t, data, "stale", "20261021-100000", `{"reason":"GREEN"}`)
		want, unprobed = []string{"alive", "old"}, append(unprobed, "alive", "stale")
		if _, err := os.Stat("/proc/self/cmdline"); err == nil {
			// Tiến trình sống nhưng không phải `routine run <tên này>`.
			putRoutineRun(t, data, "reused", "20261021-090000", running("/wt/reused", os.Getpid()))
			putRoutineRun(t, data, "other", "20261021-090000", running("/wt/other", livePID))
			unprobed = append(unprobed, "reused", "other")
		}
	}

	rows := routinePS(data, time.Date(2026, 10, 21, 9, 5, 0, 0, time.UTC))
	var got []string
	for _, r := range rows {
		got = append(got, r.Routine)
	}
	require.Equal(t, want, got, "%+v", rows)
	if livePID != 0 {
		require.Equal(t, "/wt/alive", rows[0].Path)
		require.Equal(t, livePID, rows[0].PID)
		require.True(t, rows[0].Busy, "a running routine is busy for scripts that check busy")
		require.Equal(t, "ROUTINE", rows[0].status())
		require.Contains(t, rows[0].Title, "5m0s")
	}
	for _, n := range unprobed {
		require.NoFileExists(t, routineLockPath(data, n), "probing creates the lock file")
	}
}

// Không server nào chạy mà có routine đang chạy: in bảng có routine, không phải "No dew
// server running"; JSON chỉ thêm trường (path/busy giữ nghĩa cho dew-tmux).
func TestPSShowsRoutinesWithoutAServer(t *testing.T) {
	t.Parallel()

	sock := filepath.Join(t.TempDir(), "dew-none.sock")
	rows := []psRow{{Path: "/wt/n", Routine: "nightly", PID: 42, Busy: true, Title: "nightly, running 5m0s"}}
	var buf bytes.Buffer
	require.NoError(t, runPS(t.Context(), &buf, io.Discard, []string{"unix://" + sock}, false, 9, rows))
	require.NotContains(t, buf.String(), "No dew server running")
	require.Regexp(t, `/wt/n\s+-\s+ROUTINE\s+nightly, running 5m0s`, buf.String())

	buf.Reset()
	require.NoError(t, runPS(t.Context(), &buf, io.Discard, nil, true, 9, rows))
	var got []map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got, 1)
	require.Equal(t, "/wt/n", got[0]["path"])
	require.Equal(t, true, got[0]["busy"])
	require.Equal(t, "nightly", got[0]["routine"])
	require.EqualValues(t, 42, got[0]["pid"])
}
