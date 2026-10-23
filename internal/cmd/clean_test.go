package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	fang "charm.land/fang/v2"
	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/housekeep"
	"github.com/charmbracelet/crush/internal/lock"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// goCacheForTest dựng GOCACHE giả (README của Go) và trỏ biến GOCACHE vào đó.
func goCacheForTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README"),
		[]byte("This directory holds cached build artifacts from the Go build system.\n"), 0o644))
	t.Setenv("GOCACHE", dir)
	return dir
}

// fillGoCache thêm count mục 100 byte, cũ 2-6 ngày; n là số đầu (mục khác nhau mỗi lần gọi).
func fillGoCache(t *testing.T, dir string, n, count int) {
	t.Helper()
	for i := range count {
		sum := sha256.Sum256([]byte(strconv.Itoa(n + i)))
		name := hex.EncodeToString(sum[:]) + "-d"
		p := filepath.Join(dir, name[:2], name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(strings.Repeat("x", 100)), 0o644))
		when := time.Now().Add(-time.Duration(2+i) * 24 * time.Hour)
		require.NoError(t, os.Chtimes(p, when, when))
	}
}

func goCacheBytes(t *testing.T, dir string) int64 {
	t.Helper()
	c, err := housekeep.ScanGoCache(dir)
	require.NoError(t, err)
	return c.Bytes
}

// fakeTools giả các lệnh đọc mà `dew clean` chạy, ghi lại từng lệnh; mọi đường dẫn nằm trong
// thư mục tạm.
type fakeTools struct {
	home, pnpm, mod string
	calls           []string
}

func newFakeTools(t *testing.T) *fakeTools {
	t.Helper()
	f := &fakeTools{home: t.TempDir(), pnpm: t.TempDir(), mod: t.TempDir()}
	for _, p := range []string{filepath.Join(f.home, ".npm", "x"), filepath.Join(f.pnpm, "v3", "x"), filepath.Join(f.mod, "cache", "x")} {
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte("keep me"), 0o644))
	}
	return f
}

func (f *fakeTools) run(_ context.Context, name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, cmd)
	switch cmd {
	case "pnpm store path":
		return f.pnpm, nil
	case "go env GOMODCACHE":
		return f.mod, nil
	case "docker system df --format {{.Type}}\t{{.Size}}\t{{.Reclaimable}}":
		return "Images\t5GB\t2.5GB (50%)\nLocal Volumes\t1.2GB\t0B (0%)\nBuild Cache\t3GB\t3GB\n", nil
	}
	return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
}

// cleanSetup: một cache Go 500 B vượt trần 300 B, một worktree routine gỡ được và các cache
// ASK/NEVER giả.
func cleanSetup(t *testing.T) (*routineEnv, routineOutcome, string, *fakeTools) {
	t.Helper()
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	old := e.run(t, nil) // lượt mới nhất đã xong theo cùng luật (Q4): `routine gc` gỡ được
	cache := goCacheForTest(t)
	fillGoCache(t, cache, 0, 5)
	return e, old, cache, newFakeTools(t)
}

// Không --apply: in bảng SAFE/ASK/NEVER kèm lệnh, không xoá gì.
func TestClean_ShowsTheTableAndDeletesNothingWithoutApply(t *testing.T) {
	e, old, cache, tools := cleanSetup(t)
	var buf bytes.Buffer
	require.NoError(t, runClean(t.Context(), cleanOpts{GoCacheMax: 300, DataDir: e.data, Home: tools.home, Run: tools.run}, &buf))
	out := buf.String()
	for _, want := range []string{
		"SAFE   Go build cache", cache, "run: dew clean --apply",
		"SAFE   routine worktrees", "would remove " + old.Report.Worktree,
		"ASK    npm cache", "run: npm cache clean --force",
		"ASK    pnpm store", "ASK    Docker images", "run: docker image prune -a", "ASK    Docker build cache",
		"NEVER  Go module cache", "NEVER  Docker volumes",
		"Nothing was deleted",
	} {
		require.Contains(t, out, want)
	}
	require.Equal(t, int64(500), goCacheBytes(t, cache), "nothing trimmed")
	require.DirExists(t, old.Report.Worktree)
	require.FileExists(t, filepath.Join(tools.home, ".npm", "x"))

	buf.Reset()
	require.NoError(t, runClean(t.Context(), cleanOpts{JSON: true, GoCacheMax: 300, DataDir: e.data, Home: tools.home, Run: tools.run}, &buf))
	var rep cleanReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rep))
	require.Equal(t, "Go build cache", rep.Items[0].Name)
	require.Equal(t, int64(500), rep.Items[0].Bytes)
	require.Nil(t, rep.GoCacheTrim)
	require.DirExists(t, old.Report.Worktree)
}

// --apply chỉ làm hạng SAFE: trim cache Go (về 80% trần, cũ nhất trước) và gỡ worktree như
// `routine gc`; cache ASK/NEVER còn nguyên và chỉ có lệnh đọc được chạy.
func TestClean_ApplyOnlyDeletesTheSafeRows(t *testing.T) {
	e, old, cache, tools := cleanSetup(t)
	var buf bytes.Buffer
	require.NoError(t, runClean(t.Context(), cleanOpts{Apply: true, GoCacheMax: 300, DataDir: e.data, Home: tools.home, Run: tools.run}, &buf))
	out := buf.String()
	require.Contains(t, out, "removed "+old.Report.Worktree)
	require.Contains(t, out, "Go build cache: removed 3 old entries")
	require.Equal(t, int64(200), goCacheBytes(t, cache), "trimmed to 80% of the cap: 240 B")
	require.NoDirExists(t, old.Report.Worktree)
	for _, p := range []string{filepath.Join(tools.home, ".npm", "x"), filepath.Join(tools.pnpm, "v3", "x"), filepath.Join(tools.mod, "cache", "x")} {
		require.FileExists(t, p)
	}
	for _, c := range tools.calls {
		require.Contains(t, []string{
			"pnpm store path", "pip cache dir", "go env GOMODCACHE",
			"docker system df --format {{.Type}}\t{{.Size}}\t{{.Reclaimable}}",
		}, c, "only read-only commands run")
	}

	// DEW_GOCACHE_MAX=0: không trần, không trim.
	fillGoCache(t, cache, 100, 5)
	buf.Reset()
	require.NoError(t, runClean(t.Context(), cleanOpts{Apply: true, DataDir: e.data, Home: tools.home, Run: tools.run}, &buf))
	require.Equal(t, int64(700), goCacheBytes(t, cache))
}

// Worktree có tệp của user khác (build trong Docker chạy root): `dew clean` nêu nó ở hạng
// NEVER, nói tệp của ai và cách xử lý.
func TestClean_NamesWorktreesHoldingFilesOfAnotherUser(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no file owners on Windows")
	}
	e, _, _, tools := cleanSetup(t)
	uid := currentUID
	currentUID = func() int { return os.Getuid() + 1 }
	t.Cleanup(func() { currentUID = uid })
	var buf bytes.Buffer
	_ = runClean(t.Context(), cleanOpts{DataDir: e.data, Home: tools.home, Run: tools.run}, &buf)
	out := buf.String()
	require.Contains(t, out, "NEVER  undeletable files")
	require.Contains(t, out, `sudo chown -R "$(id -u):$(id -g)"`)
	require.Contains(t, out, `docker run --user "$(id -u):$(id -g)"`)
}

// Đầu mỗi lượt: cache Go vượt trần thì trim về 80% trần, một dòng stderr; tối đa mỗi giờ
// một lần (dấu thời gian trong data dir), và chỉ khi giữ được khoá trim chung.
func TestExecuteRoutine_TrimsAnOversizedGoCacheAtMostOncePerHour(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	cache := goCacheForTest(t)
	fillGoCache(t, cache, 0, 5)
	clock := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	e.opts.Now = func() time.Time { clock = clock.Add(time.Second); return clock }
	e.opts.GoCacheMax = 300
	zero := 0
	e.opts.Def.MaxRepairs = &zero
	run := func() string {
		d := e.deps(nil)
		_, err := executeRoutine(t.Context(), e.opts, d)
		require.NoError(t, err)
		return d.Stderr.(*bytes.Buffer).String()
	}

	stderr := run()
	require.Equal(t, int64(200), goCacheBytes(t, cache))
	require.Contains(t, stderr, "[routine] Go build cache 500 B was over DEW_GOCACHE_MAX")
	require.Contains(t, stderr, "removed 3 old entries")

	fillGoCache(t, cache, 100, 5)
	stderr = run()
	require.Equal(t, int64(700), goCacheBytes(t, cache), "trimmed less than an hour ago")
	require.NotContains(t, stderr, "Go build cache")

	clock = clock.Add(time.Hour)
	release, err := lock.TryFile(filepath.Join(e.data, ".gocache-trim.lock"))
	require.NoError(t, err)
	run()
	require.Equal(t, int64(700), goCacheBytes(t, cache), "another run holds the trim lock")
	release()

	run()
	require.Equal(t, int64(200), goCacheBytes(t, cache), "an hour later, with the lock free")

	e.opts.GoCacheMax = 0
	fillGoCache(t, cache, 200, 5)
	clock = clock.Add(2 * time.Hour)
	run()
	require.Equal(t, int64(700), goCacheBytes(t, cache), "0 = off")
}

// Sau bước trim, đĩa chứa data dir còn ít hơn trần thì từ chối chạy: thoát 75, lời báo nêu
// `dew clean` và tên biến, không tạo nhánh, không chạy agent. 0 = không kiểm.
func TestExecuteRoutine_RefusesToRunWhenTheDiskIsAlmostFull(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	cache := goCacheForTest(t)
	fillGoCache(t, cache, 0, 5)
	e.opts.GoCacheMax = 300
	e.opts.MinFree = 10_000_000_000
	d := e.deps(nil)
	var asked string
	d.FreeBytes = func(p string) (uint64, error) { asked = p; return 5_000_000_000, nil }
	_, err := executeRoutine(t.Context(), e.opts, d)
	var ec *exitCodeError
	require.True(t, errors.As(err, &ec), "a full disk must exit 75: %v", err)
	require.Equal(t, 75, ec.code)
	require.Equal(t, "low_disk", ec.refused)
	require.ErrorContains(t, err, "dew clean")
	require.ErrorContains(t, err, "DEW_MIN_FREE_GB=10")
	require.Equal(t, e.data, asked)
	require.Zero(t, e.runs)
	require.Empty(t, gitIn(t, e.repo, "branch", "--list", "routine/*"))
	require.Equal(t, int64(200), goCacheBytes(t, cache), "the trim runs before the check")

	d.FreeBytes = func(string) (uint64, error) { return 11_000_000_000, nil }
	_, err = executeRoutine(t.Context(), e.opts, d)
	require.NoError(t, err)
	e.opts.MinFree = 0
	d.FreeBytes = func(string) (uint64, error) { return 0, nil }
	_, err = executeRoutine(t.Context(), e.opts, d)
	require.NoError(t, err, "0 = no check")
}

// Hai biến đọc như DEW_ROUTINE_SLOTS: biến export, rồi tệp env, rồi mặc định; số GB nguyên.
func TestHousekeepEnv_ExportedEnvThenEnvFileThenDefault(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), "env")
	t.Setenv(execpolicy.EnvFileEnv, envFile)
	t.Setenv(goCacheMaxEnv, "")
	t.Setenv(minFreeEnv, "")
	n, err := envGB(goCacheMaxEnv, 20)
	require.NoError(t, err)
	require.Equal(t, int64(20_000_000_000), n)
	n, err = envGB(minFreeEnv, 10)
	require.NoError(t, err)
	require.Equal(t, int64(10_000_000_000), n)

	require.NoError(t, os.WriteFile(envFile, []byte("DEW_GOCACHE_MAX=50\nexport DEW_MIN_FREE_GB=0 # off\n"), 0o600))
	n, err = envGB(goCacheMaxEnv, 20)
	require.NoError(t, err)
	require.Equal(t, int64(50_000_000_000), n)
	n, err = envGB(minFreeEnv, 10)
	require.NoError(t, err)
	require.Zero(t, n)

	t.Setenv(goCacheMaxEnv, "7")
	n, err = envGB(goCacheMaxEnv, 20)
	require.NoError(t, err)
	require.Equal(t, int64(7_000_000_000), n)

	for _, bad := range []string{"-1", "20GB", "1.5", "99999999999"} {
		t.Setenv(minFreeEnv, bad)
		_, err = envGB(minFreeEnv, 10)
		require.ErrorContains(t, err, minFreeEnv, bad)
	}
}

// Tệp của user khác (build trong Docker chạy root) làm `git worktree remove` hỏng GIỮA chừng:
// git đã xoá .git và bỏ đăng ký worktree, không lượt dọn nào gỡ tiếp được. Nên đó là một lý do
// giữ: `--dry-run` nói "keep" kèm tệp của ai và cách xử lý; trả quyền rồi `routine gc` gỡ được.
func TestRoutineGC_KeepsAWorktreeHoldingFilesOfAnotherUser(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no file owners on Windows")
	}
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	old := e.run(t, nil)
	uid := currentUID
	currentUID = func() int { return os.Getuid() + 1 } // mọi tệp trông như của người khác
	t.Cleanup(func() { currentUID = uid })

	out := e.gc(t, gcOpts{DryRun: true})
	for _, want := range []string{
		"keep " + old.Report.Worktree + ": you cannot delete", "owned by",
		`sudo chown -R "$(id -u):$(id -g)" ` + old.Report.Worktree, `docker run --user "$(id -u):$(id -g)"`,
	} {
		require.Contains(t, out, want)
	}
	require.Contains(t, e.gc(t, gcOpts{}), "keep "+old.Report.Worktree+": you cannot delete")
	require.FileExists(t, filepath.Join(old.Report.Worktree, ".git"), "git never started deleting")
	require.Contains(t, gitIn(t, e.repo, "worktree", "list"), old.Report.Worktree)

	currentUID = uid
	require.Contains(t, e.gc(t, gcOpts{}), "removed "+old.Report.Worktree)
}

// Thư mục chỉ đọc của chính mình (module cache Go mà một lệnh build để trong worktree là 0555)
// làm `git worktree remove` hỏng giữa chừng y như vậy: giữ worktree, gợi ý chmod -R u+w.
func TestRoutineGC_KeepsAWorktreeWithAReadOnlyDirectory(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions and a non-root user")
	}
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	require.NoError(t, os.WriteFile(filepath.Join(e.repo, ".gitignore"), []byte("build/\n"), 0o644))
	gitIn(t, e.repo, "add", ".gitignore")
	gitIn(t, e.repo, "commit", "-q", "-m", "ignore")
	var mod string
	old := e.run(t, func(wt string) error {
		mod = filepath.Join(wt, "build", "gopath", "pkg", "mod", "x@v1")
		if err := os.MkdirAll(mod, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(mod, "go.mod"), []byte("module x"), 0o444); err != nil {
			return err
		}
		return os.Chmod(mod, 0o555)
	})
	t.Cleanup(func() { _ = os.Chmod(mod, 0o755) })
	e.run(t, nil)

	out := e.gc(t, gcOpts{DryRun: true})
	require.Contains(t, out, "keep "+old.Report.Worktree+": you cannot delete 1 read-only dir(s), e.g. "+mod)
	require.Contains(t, out, "chmod -R u+w "+old.Report.Worktree)
	require.Contains(t, e.gc(t, gcOpts{}), "keep "+old.Report.Worktree)
	require.FileExists(t, filepath.Join(old.Report.Worktree, ".git"), "git never started deleting")
	require.Contains(t, gitIn(t, e.repo, "worktree", "list"), old.Report.Worktree)

	require.NoError(t, os.Chmod(mod, 0o755))
	require.Contains(t, e.gc(t, gcOpts{}), "removed "+old.Report.Worktree)
}

// Đĩa thiếu mà cache Go còn dưới trần: lượt routine sẽ bị từ chối (75) dù --apply không trim gì,
// nên `dew clean` in chỗ trống và gợi ý `go clean -cache` (hạng ASK) bên cạnh các lệnh ASK khác.
func TestClean_SuggestsGoCleanCacheWhenTheDiskIsLow(t *testing.T) {
	e, _, cache, tools := cleanSetup(t)
	o := cleanOpts{
		GoCacheMax: 1000, MinFree: 10_000_000_000, DataDir: e.data, Home: tools.home, Run: tools.run,
		FreeBytes: func(string) (uint64, error) { return 5_000_000_000, nil },
	}
	var buf bytes.Buffer
	require.NoError(t, runClean(t.Context(), o, &buf))
	out := buf.String()
	require.Contains(t, out, "ASK    Go build cache")
	require.Contains(t, out, "run: go clean -cache")
	require.Contains(t, out, "free space on the disk holding "+e.data+": 5.0 GB, below DEW_MIN_FREE_GB=10")
	require.Equal(t, int64(500), goCacheBytes(t, cache))

	buf.Reset()
	o.JSON = true
	require.NoError(t, runClean(t.Context(), o, &buf))
	var rep cleanReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rep))
	require.Equal(t, int64(5_000_000_000), rep.FreeBytes)

	buf.Reset()
	o.JSON = false
	o.FreeBytes = func(string) (uint64, error) { return 11_000_000_000, nil }
	require.NoError(t, runClean(t.Context(), o, &buf))
	require.Contains(t, buf.String(), "free space on the disk holding "+e.data+": 11 GB\n")
	require.NotContains(t, buf.String(), "go clean -cache", "enough space: no extra suggestion")
}

// Môi trường test của routine trỏ GOCACHE vào thư mục tạm: test quên đặt nó cũng không bao giờ
// trim cache thật của người chạy test.
func TestNewRoutineEnvNeverPointsAtTheRealGoCache(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	newRoutineEnv(t)
	gc := os.Getenv("GOCACHE")
	require.NotEmpty(t, gc)
	require.True(t, strings.HasPrefix(gc, os.TempDir()), gc)
}

// fang viết hoa từ đầu của lỗi (cases.Title): lỗi bắt đầu bằng tên biến in ra thành
// `Dew_routine_slots="Two"`. Lỗi của cả ba biến phải giữ nguyên tên và giá trị.
func TestEnvErrorsKeepTheVariableNameThroughFang(t *testing.T) {
	t.Setenv(execpolicy.EnvFileEnv, filepath.Join(t.TempDir(), "env"))
	for key, read := range map[string]func() error{
		routineSlotsEnv: func() error { _, err := routineSlots(); return err },
		goCacheMaxEnv:   func() error { _, err := envGB(goCacheMaxEnv, 20); return err },
		minFreeEnv:      func() error { _, err := envGB(minFreeEnv, 10); return err },
	} {
		t.Setenv(key, "two")
		var buf bytes.Buffer
		c := &cobra.Command{Use: "x", RunE: func(*cobra.Command, []string) error { return read() }}
		c.SetOut(io.Discard)
		c.SetErr(&buf)
		c.SetArgs([]string{})
		require.Error(t, fang.Execute(t.Context(), c, fang.WithoutVersion(), fang.WithoutCompletions(), fang.WithoutManpage()))
		require.Contains(t, buf.String(), key+`="two"`)
		t.Setenv(key, "")
	}
}
