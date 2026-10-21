package cmd

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/stretchr/testify/require"
)

// treeFiles là nội dung mọi tệp git thấy trong dir: tệp đã theo dõi còn trên đĩa cộng tệp chưa
// theo dõi không bị ignore.
func treeFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	m := map[string]string{}
	for _, p := range strings.Split(gitIn(t, dir, "ls-files", "-z", "-c", "-o", "--exclude-standard"), "\x00") {
		if p == "" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, p))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		require.NoError(t, err)
		m[p] = string(b)
	}
	return m
}

// applyToBase clone repo, checkout base, rồi `git apply` patch như người soát sẽ làm.
func applyToBase(t *testing.T, repo, base string, patch []byte) string {
	t.Helper()
	clone := t.TempDir()
	gitIn(t, clone, "clone", "-q", repo, ".")
	gitIn(t, clone, "checkout", "-q", "--detach", base)
	c := exec.Command("git", "apply")
	c.Dir, c.Stdin = clone, bytes.NewReader(patch)
	out, err := c.CombinedOutput()
	require.NoError(t, err, "git apply: %s\n%s", out, patch)
	return clone
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for p, body := range files {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, p), []byte(body), 0o644))
	}
}

// B3: một patch duy nhất so với base_sha gồm tệp sửa, tệp xoá, tệp nhị phân, tệp MỚI chưa theo
// dõi và commit của lượt; `git apply` được vào checkout sạch của base và ra đúng cây đó. Tệp bị
// gitignore chỉ vào khi --include-ignored (trừ .crush và thư mục dựng lại được). Xuất không
// đụng index của worktree và không ghi object vào repo.
func TestRoutineExport_PatchAppliesToACleanCheckoutOfBase(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	writeFiles(t, e.repo, map[string]string{"a.txt": "one\ntwo\n", "gone.txt": "bye\n", "bin.dat": "\x00\x01\x02", ".gitignore": "*.env\nnode_modules/\n"})
	gitIn(t, e.repo, "add", "-A")
	gitIn(t, e.repo, "commit", "-q", "-m", "files")
	e.opts.Def.Commit = true
	out := e.run(t, func(wt string) error {
		writeFiles(t, wt, map[string]string{"committed.txt": "c\n"})
		gitIn(t, wt, "add", "committed.txt")
		gitIn(t, wt, "commit", "-q", "-m", "agent commit")
		writeFiles(t, wt, map[string]string{
			"a.txt": "one\n2\n", "new/dir/b.txt": "fresh\n", "bin.dat": "\x00\xff\x02\x03",
			"secret.env": "TOKEN=1\n", "node_modules/x/i.js": "x\n",
		})
		if err := os.Remove(filepath.Join(wt, "gone.txt")); err != nil {
			return err
		}
		return writeTranscript(wt)
	})
	wt := out.Report.Worktree
	statusBefore := gitIn(t, wt, "status", "--porcelain")
	objectsBefore := gitIn(t, e.repo, "count-objects", "-v")

	res, err := exportRun(t.Context(), e.data, "nightly", exportOpts{})
	require.NoError(t, err)
	require.Equal(t, []string{"M\ta.txt", "M\tbin.dat", "A\tcommitted.txt", "D\tgone.txt", "A\tnew/dir/b.txt"}, res.Files)
	require.Equal(t, statusBefore, gitIn(t, wt, "status", "--porcelain"), "export must not touch the run's index")
	require.Equal(t, objectsBefore, gitIn(t, e.repo, "count-objects", "-v"), "export must not write objects into the repo")

	clone := applyToBase(t, e.repo, out.Report.BaseSHA, res.Patch)
	require.Equal(t, treeFiles(t, wt), treeFiles(t, clone), "the clean checkout plus the patch is the run's tree")
	require.NoFileExists(t, filepath.Join(clone, "secret.env"), "ignored files only with --include-ignored")

	res, err = exportRun(t.Context(), e.data, "nightly", exportOpts{Stamp: filepath.Base(out.RunDir), IncludeIgnored: true})
	require.NoError(t, err)
	require.Contains(t, res.Files, "A\tsecret.env")
	clone = applyToBase(t, e.repo, out.Report.BaseSHA, res.Patch)
	require.Equal(t, treeFiles(t, wt), treeFiles(t, clone))
	requireFile(t, filepath.Join(clone, "secret.env"), "TOKEN=1\n")
	require.NoDirExists(t, filepath.Join(clone, "node_modules"), "regenerable dirs are not the run's work")
	require.NoDirExists(t, filepath.Join(clone, ".crush"), "dew's own session data is not the run's work")
}

// Worktree đã bị gỡ: từ chối, nói rõ, và chỉ chỗ crush.db còn giữ transcript. Lượt không có
// thay đổi nào cũng từ chối (patch rỗng không `git apply` được). Tên lượt phải là tên thư mục.
func TestRoutineExport_RefusesWithoutAWorktreeOrChanges(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	out := e.run(t, writeTranscript)
	_, err := exportRun(t.Context(), e.data, "nightly", exportOpts{})
	require.ErrorContains(t, err, "no changes")

	rep := out.Report
	require.NoError(t, removeRunWorktree(t.Context(), out.RunDir, &rep))
	require.NoError(t, writeReport(out.RunDir, rep))
	_, err = exportRun(t.Context(), e.data, "nightly", exportOpts{Stamp: filepath.Base(out.RunDir)})
	require.ErrorContains(t, err, "is gone")
	require.ErrorContains(t, err, filepath.Join(out.RunDir, "crush.db"))

	for _, bad := range []string{"../nightly", "20991231-000000"} {
		_, err = exportRun(t.Context(), e.data, "nightly", exportOpts{Stamp: bad})
		require.Error(t, err, bad)
	}
	_, err = exportRun(t.Context(), e.data, "nope", exportOpts{})
	require.ErrorContains(t, err, "no runs")
}

// report.json sửa được từ bên ngoài (lớp chặn không phải sandbox): export chỉ tin worktree và
// nhánh đúng chỗ lượt này đặt chúng, và base_sha là một SHA thật. base_sha dạng cờ
// (`--output=<tệp>`) không bao giờ tới được git (git 2.43 nhận --output và ghi đè tệp đó);
// worktree trỏ sang repo khác (checkout của người dùng, có .env) hay symlink sang đó bị từ chối.
func TestRoutineExport_RefusesATamperedReport(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	out := e.run(t, func(wt string) error {
		writeFiles(t, wt, map[string]string{"new.txt": "x\n"})
		return nil
	})
	writeFiles(t, e.repo, map[string]string{"user-change.txt": "mine\n", ".env": "SECRET=1\n"})
	victim := filepath.Join(t.TempDir(), "bashrc")
	stamp := filepath.Base(out.RunDir)
	export := func() error {
		_, err := exportRun(t.Context(), e.data, "nightly", exportOpts{IncludeIgnored: true})
		return err
	}
	tamper := func(edit func(*routineReport)) {
		t.Helper()
		require.NoError(t, writeReport(out.RunDir, out.Report))
		editReport(t, out.RunDir, edit)
	}

	for _, bad := range []string{"--output=" + victim, "HEAD", strings.Repeat("A", 40), out.Report.BaseSHA[:39]} {
		tamper(func(r *routineReport) { r.BaseSHA = bad })
		require.ErrorContains(t, export(), "base_sha", bad)
		require.NoFileExists(t, victim)
	}
	tamper(func(r *routineReport) { r.Worktree = e.repo })
	require.ErrorContains(t, export(), "not where this run puts them")
	tamper(func(r *routineReport) { r.Branch = "main" })
	require.ErrorContains(t, export(), "not where this run puts them")

	tamper(func(*routineReport) {})
	wt := filepath.Join(e.data, "nightly", "wt", stamp)
	require.NoError(t, os.Rename(wt, wt+".real"))
	require.NoError(t, os.Symlink(e.repo, wt))
	require.ErrorContains(t, export(), "not where this run puts them", "a symlink to another checkout")
	require.NoError(t, os.Remove(wt))
	require.NoError(t, os.Rename(wt+".real", wt))
	require.NoError(t, export(), "the untampered report exports")
}

// Lớp phòng thủ thứ hai: kể cả khi base dạng cờ lọt qua kiểm tra, --end-of-options đứng trước
// revision nên git coi nó là revision (lỗi), không phải cờ.
func TestBuildPatch_FlagLookingBaseIsNeverAnOption(t *testing.T) {
	t.Parallel()
	repo := gitInit(t)
	writeFiles(t, repo, map[string]string{"a.txt": "x\n"})
	victim := filepath.Join(t.TempDir(), "out")
	_, _, err := buildPatch(t.Context(), repo, "--output="+victim, false)
	require.Error(t, err)
	require.NoFileExists(t, victim)
}

// HEAD của worktree không còn là nhánh của lượt (agent đã chuyển nhánh hay tách HEAD): cây làm
// việc giờ so với một commit khác, nên từ chối và nói rõ; --head xuất cây đúng như nó đang là.
func TestRoutineExport_RefusesWhenHEADLeftTheRunsBranch(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	out := e.run(t, func(wt string) error {
		gitIn(t, wt, "checkout", "-q", "--detach")
		writeFiles(t, wt, map[string]string{"new.txt": "x\n"})
		return nil
	})
	_, err := exportRun(t.Context(), e.data, "nightly", exportOpts{})
	require.ErrorContains(t, err, out.Report.Branch)
	require.ErrorContains(t, err, "--head")
	res, err := exportRun(t.Context(), e.data, "nightly", exportOpts{Head: true})
	require.NoError(t, err)
	require.Equal(t, []string{"A\tnew.txt"}, res.Files)
}

// Dữ liệu phiên của dew (.crush/) không bao giờ vào patch, kể cả khi agent xoá .crush/.gitignore
// (khi đó crush.db là tệp chưa theo dõi, không bị ignore).
func TestRoutineExport_NeverExportsDewSessionData(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.run(t, func(wt string) error {
		if err := writeTranscript(wt); err != nil {
			return err
		}
		writeFiles(t, wt, map[string]string{"new.txt": "x\n"})
		return os.Remove(filepath.Join(wt, ".crush", ".gitignore"))
	})
	for _, ign := range []bool{false, true} {
		res, err := exportRun(t.Context(), e.data, "nightly", exportOpts{IncludeIgnored: ign})
		require.NoError(t, err)
		require.Equal(t, []string{"A\tnew.txt"}, res.Files, "include-ignored=%v", ign)
		require.NotContains(t, string(res.Patch), ".crush")
	}
}

// Cấu hình diff của repo (không tiền tố, màu, tiền tố gợi nhớ, submodule dạng log) không được
// đổi dạng patch: nó vẫn `git apply` được, kể cả khi gitlink đổi.
func TestRoutineExport_RepoDiffConfigDoesNotBreakThePatch(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	writeFiles(t, e.repo, map[string]string{"a.txt": "one\n"})
	gitIn(t, e.repo, "add", "a.txt")
	gitIn(t, e.repo, "update-index", "--add", "--cacheinfo", "160000,"+strings.Repeat("1", 40)+",sub")
	gitIn(t, e.repo, "commit", "-q", "-m", "files")
	for k, v := range map[string]string{"diff.noprefix": "true", "color.diff": "always", "diff.mnemonicPrefix": "true", "diff.submodule": "log"} {
		gitIn(t, e.repo, "config", k, v)
	}
	e.opts.Def.Commit = true
	out := e.run(t, func(wt string) error {
		gitIn(t, wt, "update-index", "--cacheinfo", "160000,"+strings.Repeat("2", 40)+",sub")
		gitIn(t, wt, "commit", "-q", "-m", "bump sub")
		writeFiles(t, wt, map[string]string{"a.txt": "two\n"})
		return nil
	})
	res, err := exportRun(t.Context(), e.data, "nightly", exportOpts{})
	require.NoError(t, err)
	require.Equal(t, []string{"M\ta.txt", "M\tsub"}, res.Files)
	require.Contains(t, string(res.Patch), "+Subproject commit "+strings.Repeat("2", 40))
	clone := applyToBase(t, e.repo, out.Report.BaseSHA, res.Patch)
	requireFile(t, filepath.Join(clone, "a.txt"), "two\n")
}

// Thư mục tạm của export (index, object) luôn bị xoá, kể cả khi bị huỷ giữa chừng (Ctrl+C huỷ
// context, git con bị giết, hàm trả lỗi).
func TestRoutineExport_RemovesItsTempDirEvenWhenCancelled(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.run(t, func(wt string) error {
		writeFiles(t, wt, map[string]string{"new.txt": "x\n"})
		return nil
	})
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	_, err := exportRun(t.Context(), e.data, "nightly", exportOpts{})
	require.NoError(t, err)
	left, _ := os.ReadDir(tmp)
	require.Empty(t, left, "a finished export leaves nothing in TMPDIR")

	ctx, cancel := context.WithCancel(t.Context())
	go func() { // huỷ ngay khi thư mục tạm xuất hiện
		for ctx.Err() == nil {
			if m, _ := filepath.Glob(filepath.Join(tmp, "dew-export-*")); len(m) > 0 {
				cancel()
			}
		}
	}()
	_, _ = exportRun(ctx, e.data, "nightly", exportOpts{})
	cancel()
	left, _ = os.ReadDir(tmp)
	require.Empty(t, left, "a cancelled export leaves nothing in TMPDIR")
}
