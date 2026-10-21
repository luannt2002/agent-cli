package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/lock"
	"github.com/stretchr/testify/require"
)

// run chạy một lượt dừng ngay sau lượt agent đầu (verify đỏ, không lượt sửa).
func (e *routineEnv) run(t *testing.T, onTurn func(wt string) error) routineOutcome {
	t.Helper()
	zero := 0
	o := e.opts
	o.Def.MaxRepairs = &zero
	out, err := executeRoutine(t.Context(), o, e.deps(onTurn))
	require.NoError(t, err)
	return out
}

// gc chạy `routine gc` cho routine của e và trả output (lỗi thì test hỏng).
func (e *routineEnv) gc(t *testing.T, o gcOpts) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, routineGC(t.Context(), e.data, nil, o, &buf))
	return buf.String()
}

// editReport sửa report.json của một lượt trên đĩa.
func editReport(t *testing.T, dir string, edit func(*routineReport)) {
	t.Helper()
	rep, err := readReport(dir)
	require.NoError(t, err)
	edit(&rep)
	require.NoError(t, writeReport(dir, rep))
}

// beforeFakeClock là trước mọi mốc của fixedClock: tệp phiên ghi lúc này là của chính lượt.
var beforeFakeClock = time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)

// writeTranscript giả dữ liệu dew của lượt: crush.db (+ -wal, -shm) và logs/ trong .crush
// mà .crush/.gitignore che (như dew tạo), nên cây vẫn "sạch".
func writeTranscript(wt string) error {
	dir := filepath.Join(wt, ".crush")
	if err := os.MkdirAll(filepath.Join(dir, "logs"), 0o755); err != nil {
		return err
	}
	for name, body := range map[string]string{".gitignore": "*\n", "crush.db": "transcript", "crush.db-wal": "wal", "crush.db-shm": "shm", "logs/crush.log": "log"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			return err
		}
		if err := os.Chtimes(p, beforeFakeClock, beforeFakeClock); err != nil {
			return err
		}
	}
	return nil
}

func requireFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, string(got))
}

// Mỗi lượt gỡ worktree+nhánh của lượt cũ đã xong mà cây sạch và không có commit; giữ
// worktree bẩn, worktree có commit (commit=false thì việc của agent chỉ còn ở đó) và
// worktree của lượt mới nhất. Dữ liệu dew (crush.db, -wal, -shm, logs) chuyển ra cạnh
// report.json trước khi gỡ.
func TestExecuteRoutine_RemovesOnlyOldCleanWorktrees(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	clean := e.run(t, writeTranscript)
	dirty := e.run(t, func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "notes.txt"), []byte("x"), 0o644)
	})
	committed := e.run(t, func(wt string) error {
		if err := os.WriteFile(filepath.Join(wt, "c.txt"), []byte("x"), 0o644); err != nil {
			return err
		}
		gitIn(t, wt, "add", "c.txt")
		gitIn(t, wt, "commit", "-q", "-m", "work")
		return nil
	})
	previous := e.run(t, nil) // sạch, nhưng là lượt mới nhất lúc lượt sau khởi động
	newest := e.run(t, nil)

	require.NoDirExists(t, clean.Report.Worktree)
	require.Empty(t, gitIn(t, e.repo, "branch", "--list", clean.Report.Branch))
	for _, kept := range []routineOutcome{dirty, committed, previous, newest} {
		require.DirExists(t, kept.Report.Worktree)
		require.NotEmpty(t, gitIn(t, e.repo, "branch", "--list", kept.Report.Branch))
	}

	// Báo cáo, journal và dữ liệu dew còn; báo cáo nói worktree đã gỡ và chúng ở đâu.
	rep, err := readReport(clean.RunDir)
	require.NoError(t, err)
	require.FileExists(t, rep.Journal)
	require.False(t, rep.WorktreeRemoved.IsZero())
	require.Equal(t, filepath.Join(clean.RunDir, "crush.db"), rep.SessionDB)
	require.Equal(t, filepath.Join(clean.RunDir, "logs"), rep.SessionLogs)
	for name, want := range map[string]string{"crush.db": "transcript", "crush.db-wal": "wal", "crush.db-shm": "shm", "logs/crush.log": "log"} {
		requireFile(t, filepath.Join(clean.RunDir, name), want)
	}
}

// `routine gc`: chạy thử chỉ in (kèm số byte), chạy thật thì gỡ; routine đang giữ khoá
// thì bỏ qua cả routine (lượt đang chạy của nó có thể chính là thứ đang bị dọn).
func TestRoutineGC_DryRunRemoveAndSkipWhileLocked(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	dirty := e.run(t, func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "notes.txt"), []byte("x"), 0o644)
	})
	old := e.run(t, nil)
	newest := e.run(t, nil)
	require.DirExists(t, old.Report.Worktree, "the newest run at the time keeps its worktree")

	release, err := lock.TryFile(routineLockPath(e.data, "nightly"))
	require.NoError(t, err)
	out := e.gc(t, gcOpts{})
	require.Contains(t, out, "skip nightly")
	require.DirExists(t, old.Report.Worktree)
	release()

	out = e.gc(t, gcOpts{DryRun: true})
	require.Contains(t, out, "would remove "+old.Report.Worktree)
	require.Contains(t, out, "keep "+dirty.Report.Worktree+": uncommitted changes")
	require.Regexp(t, `would remove 1 worktree\(s\), \d+ B; kept 1;`, out)
	require.DirExists(t, old.Report.Worktree, "dry run removes nothing")

	var buf bytes.Buffer
	require.NoError(t, routineGC(t.Context(), e.data, []string{"nightly"}, gcOpts{}, &buf))
	require.Contains(t, buf.String(), "removed 1 worktree(s)")
	require.NoDirExists(t, old.Report.Worktree)
	require.DirExists(t, newest.Report.Worktree)
	require.DirExists(t, dirty.Report.Worktree)
	require.FileExists(t, filepath.Join(old.RunDir, "report.json"))

	require.Error(t, routineGC(t.Context(), e.data, []string{"../x"}, gcOpts{}, &buf))
}

// --dry-run không đổi gì: không ghi lại báo cáo RUNNING, không chuyển tệp phiên.
func TestRoutineGC_DryRunChangesNothing(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	old := e.run(t, writeTranscript)
	e.run(t, nil)
	editReport(t, old.RunDir, func(r *routineReport) { r.Reason, r.PID = "RUNNING", 999999 })

	out := e.gc(t, gcOpts{DryRun: true})
	require.Contains(t, out, "would remove "+old.Report.Worktree)
	require.Equal(t, "RUNNING", rawReport(t, old.RunDir)["reason"], "dry run must not rewrite reports")
	require.FileExists(t, filepath.Join(old.Report.Worktree, ".crush", "crush.db"))
	require.NoFileExists(t, filepath.Join(old.RunDir, "crush.db"))
}

// status.showUntrackedFiles=no làm `git status --porcelain` im lặng trước tệp mới, và
// `git worktree remove` (không --force) khi đó xoá luôn tệp đó: phải ép hiện đủ.
func TestRoutineGC_KeepsUntrackedFilesTheStatusConfigHides(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	old := e.run(t, func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "new.txt"), []byte("work"), 0o644)
	})
	e.run(t, nil)
	gitIn(t, e.repo, "config", "status.showUntrackedFiles", "no")

	out := e.gc(t, gcOpts{IncludeIgnored: true})
	require.Contains(t, out, "keep "+old.Report.Worktree+": uncommitted changes")
	requireFile(t, filepath.Join(old.Report.Worktree, "new.txt"), "work")
}

// Lớp thứ hai: chính `git worktree remove` cũng phải thấy tệp mới dù cấu hình giấu nó.
func TestRemoveRunWorktree_GitAlsoSeesUntrackedFilesTheConfigHides(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	old := e.run(t, func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "new.txt"), []byte("work"), 0o644)
	})
	gitIn(t, e.repo, "config", "status.showUntrackedFiles", "no")

	rep := old.Report
	require.Error(t, removeRunWorktree(t.Context(), old.RunDir, &rep))
	requireFile(t, filepath.Join(old.Report.Worktree, "new.txt"), "work")
}

// Tệp bị gitignore ngoài .crush (.env, notes/, báo cáo...) là việc agent để lại mà
// changed_files không thấy: lượt tự động giữ worktree; `routine gc` chỉ gỡ khi có
// --include-ignored, và --dry-run liệt kê chúng.
func TestRoutineGC_KeepsIgnoredFilesUnlessIncluded(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	require.NoError(t, os.WriteFile(filepath.Join(e.repo, ".gitignore"), []byte("notes/\n"), 0o644))
	gitIn(t, e.repo, "add", ".gitignore")
	gitIn(t, e.repo, "commit", "-q", "-m", "ignore notes")
	old := e.run(t, func(wt string) error {
		if err := writeTranscript(wt); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(wt, "notes"), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(wt, "notes", "n.md"), []byte("findings"), 0o644)
	})
	e.run(t, nil)
	e.run(t, nil) // lượt tự động: old không phải mới nhất nhưng có tệp bị gitignore
	require.DirExists(t, old.Report.Worktree)

	out := e.gc(t, gcOpts{DryRun: true})
	require.Contains(t, out, "keep "+old.Report.Worktree+": 1 ignored path(s) outside .crush (8 B): notes/;")
	out = e.gc(t, gcOpts{})
	require.Contains(t, out, "keep "+old.Report.Worktree)
	requireFile(t, filepath.Join(old.Report.Worktree, "notes", "n.md"), "findings")

	out = e.gc(t, gcOpts{IncludeIgnored: true})
	require.Contains(t, out, "removed "+old.Report.Worktree)
	require.NoDirExists(t, old.Report.Worktree)
	requireFile(t, filepath.Join(old.RunDir, "crush.db"), "transcript")
}

// Commit trên HEAD tách rời không nằm trên nhánh nào: gỡ worktree là mất nó.
func TestRoutineGC_KeepsCommitsOnADetachedHead(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	old := e.run(t, func(wt string) error {
		gitIn(t, wt, "checkout", "-q", "--detach")
		if err := os.WriteFile(filepath.Join(wt, "c.txt"), []byte("x"), 0o644); err != nil {
			return err
		}
		gitIn(t, wt, "add", "c.txt")
		gitIn(t, wt, "commit", "-q", "-m", "work")
		return nil
	})
	e.run(t, nil)

	out := e.gc(t, gcOpts{IncludeIgnored: true})
	require.Contains(t, out, "keep "+old.Report.Worktree+": 1 commit(s) not in base")
	require.DirExists(t, old.Report.Worktree)
}

// crush.db ghi SAU lúc lượt xong là có người mở worktree đó bằng dew: giữ nó. Vài giây sau
// finished là lần đóng DB của chính lượt (lượt của bản cũ đóng DB sau finished): vẫn gỡ được.
func TestRoutineGC_KeepsAWorktreeWhoseSessionChangedAfterTheRun(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	old := e.run(t, writeTranscript)
	e.run(t, nil)
	wal := filepath.Join(old.Report.Worktree, ".crush", "crush.db-wal")
	closing := old.Report.Finished.Add(30 * time.Second)
	require.NoError(t, os.Chtimes(wal, closing, closing))
	require.Contains(t, e.gc(t, gcOpts{DryRun: true}), "would remove "+old.Report.Worktree)

	later := old.Report.Finished.Add(time.Hour)
	require.NoError(t, os.Chtimes(wal, later, later))
	out := e.gc(t, gcOpts{IncludeIgnored: true})
	require.Contains(t, out, "keep "+old.Report.Worktree+": its session db changed after the run finished")
	require.DirExists(t, old.Report.Worktree)
	require.NoFileExists(t, filepath.Join(old.RunDir, "crush.db"))
}

// Mỗi tệp phiên chuyển riêng; một tệp không chuyển được thì KHÔNG gỡ worktree (phần còn
// lại chỉ ở đó), session_db chỉ ghi khi crush.db đã chuyển, lỗi ghi vào báo cáo và lượt
// sau không cảnh báo lại; `routine gc` thử lại được.
func TestRoutineGC_KeepsTheWorktreeWhenASessionFileCannotMove(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	old := e.run(t, writeTranscript)
	e.run(t, nil)
	blocker := filepath.Join(old.RunDir, "crush.db-wal")
	require.NoError(t, os.WriteFile(blocker, []byte("someone else's"), 0o600))

	var buf bytes.Buffer
	require.Error(t, routineGC(t.Context(), e.data, nil, gcOpts{}, &buf))
	require.Contains(t, buf.String(), "keep "+old.Report.Worktree)
	require.DirExists(t, old.Report.Worktree)
	requireFile(t, filepath.Join(old.Report.Worktree, ".crush", "crush.db-wal"), "wal")
	requireFile(t, blocker, "someone else's")
	requireFile(t, filepath.Join(old.RunDir, "crush.db"), "transcript")
	requireFile(t, filepath.Join(old.RunDir, "crush.db-shm"), "shm")
	rep, err := readReport(old.RunDir)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(old.RunDir, "crush.db"), rep.SessionDB)
	require.NotEmpty(t, rep.GCError)
	require.True(t, rep.WorktreeRemoved.IsZero())

	// Lượt sau không thử lại, không cảnh báo lại.
	zero := 0
	o := e.opts
	o.Def.MaxRepairs = &zero
	d := e.deps(nil)
	_, err = executeRoutine(t.Context(), o, d)
	require.NoError(t, err)
	require.NotContains(t, d.Stderr.(*bytes.Buffer).String(), "cleaning old runs")

	require.NoError(t, os.Remove(blocker))
	out := e.gc(t, gcOpts{})
	require.Contains(t, out, "removed "+old.Report.Worktree)
	requireFile(t, filepath.Join(old.RunDir, "crush.db-wal"), "wal")
	rep, err = readReport(old.RunDir)
	require.NoError(t, err)
	require.Empty(t, rep.GCError)
}

// report.json không phải nguồn tin cậy (mã của agent sửa được nó): chỉ gỡ đúng worktree
// và nhánh mà lượt đó tạo ra.
func TestRoutineGC_DoesNotTrustTheReportPaths(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	old := e.run(t, nil)
	newest := e.run(t, nil)
	editReport(t, old.RunDir, func(r *routineReport) { r.Worktree, r.Branch = newest.Report.Worktree, newest.Report.Branch })
	out := e.gc(t, gcOpts{IncludeIgnored: true})
	require.Contains(t, out, "keep "+newest.Report.Worktree+": the report's worktree or branch is not where this run puts them")
	require.DirExists(t, newest.Report.Worktree)

	e = newRoutineEnv(t)
	old = e.run(t, nil)
	e.run(t, nil)
	gitIn(t, e.repo, "branch", "keepme")
	editReport(t, old.RunDir, func(r *routineReport) { r.Branch = "keepme" })
	e.gc(t, gcOpts{IncludeIgnored: true})
	require.NotEmpty(t, gitIn(t, e.repo, "branch", "--list", "keepme"), "a branch that is not the run's must survive")
	require.DirExists(t, old.Report.Worktree)
}

// flock không có tác dụng trên NFS/sshfs: báo cáo RUNNING mà pid còn sống và đúng là
// `routine run <tên>` thì không ghi lại thành ABANDONED, không dọn.
func TestExecuteRoutine_LeavesARunThatIsStillAlive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs sh, and signal 0 is not supported on Windows (only the lock protects a live run there)")
	}
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	start := func(args ...string) int {
		c := exec.Command("sh", append([]string{"-c", "sleep 30; true"}, args...)...)
		require.NoError(t, c.Start())
		t.Cleanup(func() { _ = c.Process.Kill(); _ = c.Wait() })
		return c.Process.Pid
	}
	put := func(stamp string, pid int) string {
		dir := filepath.Join(e.data, "nightly", "runs", stamp)
		require.NoError(t, os.MkdirAll(dir, 0o700))
		raw, _ := json.Marshal(routineReport{Routine: "nightly", Reason: "RUNNING", PID: pid})
		require.NoError(t, os.WriteFile(filepath.Join(dir, "report.json"), raw, 0o600))
		return dir
	}
	alive := put("20261015-080000", start("routine", "run", "nightly"))
	// Tiến trình sống nhưng là routine khác: chỉ phân biệt được khi đọc được dòng lệnh (/proc).
	other := ""
	if _, err := os.Stat("/proc/self/cmdline"); err == nil {
		other = put("20261015-090000", start("routine", "run", "other"))
	}

	e.run(t, nil)
	require.Equal(t, "RUNNING", rawReport(t, alive)["reason"])
	if other != "" {
		require.Equal(t, "ABANDONED", rawReport(t, other)["reason"])
	}
}

// `--ignored=matching` (cùng -uall) liệt kê một thư mục bị ignore MỘT lần; chế độ mặc
// định (traditional) cùng -uall liệt kê từng tệp bên trong (node_modules: hàng chục nghìn).
func TestGitStatusIgnoredMatchingListsAnIgnoredDirOnce(t *testing.T) {
	t.Parallel()
	repo := gitInit(t)
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("node_modules/\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "web", "node_modules", "a"), 0o755))
	for _, f := range []string{"web/node_modules/a/x.js", "web/node_modules/y.js"} {
		require.NoError(t, os.WriteFile(filepath.Join(repo, f), []byte("x"), 0o644))
	}
	traditional := gitIn(t, repo, "status", "--porcelain", "--untracked-files=all", "--ignored")
	matching := gitIn(t, repo, "status", "--porcelain", "--untracked-files=all", "--ignored=matching")
	t.Logf("%s\n--ignored (traditional) -uall:\n%s\n--ignored=matching -uall:\n%s", gitIn(t, repo, "version"), traditional, matching)
	require.Contains(t, traditional, "!! web/node_modules/a/x.js")
	require.Equal(t, "?? .gitignore\n!! web/node_modules/", matching)
}

// Thư mục dựng lại được (node_modules, dist, .venv...) bị gitignore không phải việc của
// agent: đo 21/10, mọi worktree routine trên máy thật đều có. Tệp bị gitignore khác
// (ghi chú) vẫn giữ worktree; bỏ nó đi thì lượt tự động gỡ, và dry-run nói bỏ bao nhiêu.
func TestRoutineGC_RegenerableIgnoredDirsAreNotWork(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	require.NoError(t, os.WriteFile(filepath.Join(e.repo, ".gitignore"), []byte("node_modules/\nnotes/\n"), 0o644))
	gitIn(t, e.repo, "add", ".gitignore")
	gitIn(t, e.repo, "commit", "-q", "-m", "ignore")
	old := e.run(t, func(wt string) error {
		for f, body := range map[string]string{"web/node_modules/pkg/index.js": "x", "notes/n.md": "findings"} {
			if err := os.MkdirAll(filepath.Dir(filepath.Join(wt, f)), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(wt, f), []byte(body), 0o644); err != nil {
				return err
			}
		}
		return nil
	})
	e.run(t, nil)
	e.run(t, nil) // tự động: old không phải mới nhất, nhưng còn ghi chú
	require.DirExists(t, old.Report.Worktree)

	out := e.gc(t, gcOpts{DryRun: true})
	require.Contains(t, out, "keep "+old.Report.Worktree+": 1 ignored path(s) outside .crush (8 B): notes/;")
	require.NoError(t, os.RemoveAll(filepath.Join(old.Report.Worktree, "notes")))
	out = e.gc(t, gcOpts{DryRun: true})
	require.Contains(t, out, "would remove "+old.Report.Worktree)
	require.Contains(t, out, "drops 1 regenerable ignored path(s)")

	e.run(t, nil)
	require.NoDirExists(t, old.Report.Worktree, "only regenerable ignored dirs left: the automatic cleanup removes it")
}

// Chỉ thư mục bị ignore TRỌN (hiện một lần, `!! web/node_modules/`) mới là dựng lại được.
// Mục sâu hơn trong một thư mục mang tên đó (`!! build/x.env`) chứng tỏ thư mục ấy không bị
// ignore trọn (repo theo dõi build/): đó là việc của agent, giữ worktree.
func TestRoutineGC_IgnoredFileInATrackedBuildDirIsWork(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	require.NoError(t, os.MkdirAll(filepath.Join(e.repo, "build"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(e.repo, "build", "keep.txt"), []byte("tracked"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(e.repo, ".gitignore"), []byte("*.env\n"), 0o644))
	gitIn(t, e.repo, "add", ".gitignore", "build/keep.txt")
	gitIn(t, e.repo, "commit", "-q", "-m", "tracked build dir")
	old := e.run(t, func(wt string) error {
		return os.WriteFile(filepath.Join(wt, "build", "x.env"), []byte("SECRET=1"), 0o644)
	})
	e.run(t, nil)
	e.run(t, nil) // lượt tự động: phải giữ old

	requireFile(t, filepath.Join(old.Report.Worktree, "build", "x.env"), "SECRET=1")
	require.Contains(t, e.gc(t, gcOpts{DryRun: true}), "keep "+old.Report.Worktree+": 1 ignored path(s) outside .crush (8 B): build/x.env;")
}

func TestRegenerable(t *testing.T) {
	t.Parallel()
	for p, want := range map[string]bool{
		"node_modules/": true, "web/node_modules/": true, "svc/.venv/": true,
		"build/x.env": false, "node_modules/x/": false, "dist": false, "notes/": false,
	} {
		require.Equal(t, want, regenerable(p), p)
	}
}

// Lượt tự động không đo dung lượng (đi cả cây node_modules mỗi lượt); `routine gc` đo.
func TestGCRoutine_OnlyManualCleanupMeasuresSizes(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.run(t, nil)
	e.run(t, nil)
	st, err := gcRoutine(t.Context(), e.data, "nightly", gcOpts{DryRun: true, Manual: true}, io.Discard)
	require.NoError(t, err)
	require.Equal(t, 1, st.Removed)
	require.Positive(t, st.Freed)
	st, err = gcRoutine(t.Context(), e.data, "nightly", gcOpts{}, io.Discard)
	require.NoError(t, err)
	require.Equal(t, 1, st.Removed)
	require.Zero(t, st.Freed)
}

// Gỡ được worktree mà `branch -D` hỏng (nhánh đang được checkout ở checkout chính để
// duyệt): lần dọn sau, kể cả tự động, thử lại sau khi đếm lại commit của nhánh; nhánh có
// commit ngoài base thì giữ.
func TestRoutineGC_RetriesABranchItCouldNotDelete(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	for _, commitOnBranch := range []bool{false, true} {
		e := newRoutineEnv(t)
		old := e.run(t, nil)
		e.run(t, nil)
		gitIn(t, e.repo, "checkout", "-q", "--ignore-other-worktrees", old.Report.Branch)
		var buf bytes.Buffer
		require.Error(t, routineGC(t.Context(), e.data, nil, gcOpts{}, &buf))
		require.NoDirExists(t, old.Report.Worktree)
		rep, err := readReport(old.RunDir)
		require.NoError(t, err)
		require.NotEmpty(t, rep.GCError)
		if commitOnBranch {
			require.NoError(t, os.WriteFile(filepath.Join(e.repo, "review.txt"), []byte("x"), 0o644))
			gitIn(t, e.repo, "add", "review.txt")
			gitIn(t, e.repo, "commit", "-q", "-m", "review note")
		}
		gitIn(t, e.repo, "checkout", "-q", "-")

		e.run(t, nil)
		if commitOnBranch {
			require.NotEmpty(t, gitIn(t, e.repo, "branch", "--list", old.Report.Branch), "a commit on the branch must not be deleted")
			continue
		}
		require.Empty(t, gitIn(t, e.repo, "branch", "--list", old.Report.Branch))
		rep, err = readReport(old.RunDir)
		require.NoError(t, err)
		require.Empty(t, rep.GCError)
	}
}

// assume-unchanged / skip-worktree giấu sửa đổi khỏi git status, và `worktree remove` xoá
// luôn sửa đổi đó. skip-worktree mà tệp không có trên đĩa (như sparse-checkout) không chặn.
func TestRoutineGC_KeepsChangesHiddenFromStatus(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	for _, f := range []string{"a.txt", "b.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(e.repo, f), []byte("base"), 0o644))
	}
	gitIn(t, e.repo, "add", "a.txt", "b.txt")
	gitIn(t, e.repo, "commit", "-q", "-m", "files")
	hide := func(flag, file string, remove bool) func(string) error {
		return func(wt string) error {
			gitIn(t, wt, "update-index", flag, file)
			if remove {
				return os.Remove(filepath.Join(wt, file))
			}
			return os.WriteFile(filepath.Join(wt, file), []byte("work"), 0o644)
		}
	}
	assumed := e.run(t, hide("--assume-unchanged", "a.txt", false))
	skipped := e.run(t, hide("--skip-worktree", "a.txt", false))
	sparse := e.run(t, hide("--skip-worktree", "b.txt", true))
	e.run(t, nil)

	out := e.gc(t, gcOpts{IncludeIgnored: true})
	require.Contains(t, out, "keep "+assumed.Report.Worktree+": a.txt is hidden from git status")
	require.Contains(t, out, "keep "+skipped.Report.Worktree+": a.txt is hidden from git status")
	requireFile(t, filepath.Join(assumed.Report.Worktree, "a.txt"), "work")
	requireFile(t, filepath.Join(skipped.Report.Worktree, "a.txt"), "work")
	require.Contains(t, out, "removed "+sparse.Report.Worktree)
}
