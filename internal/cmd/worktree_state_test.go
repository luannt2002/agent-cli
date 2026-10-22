package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/loop"
	"github.com/stretchr/testify/require"
)

// ignoreRepo là repo có .gitignore đã commit: out/ (không phải thư mục dựng lại được),
// node_modules/ (dựng lại được) và *.env.
func ignoreRepo(t *testing.T) string {
	t.Helper()
	dir := gitInit(t)
	writeFiles(t, dir, map[string]string{".gitignore": "out/\nnode_modules/\n*.env\n"})
	gitIn(t, dir, "add", ".gitignore")
	gitIn(t, dir, "commit", "-q", "-m", "ignore")
	return dir
}

// Bảng Q3 (docs/routine-invariants.md): mỗi mục của cây vào đúng một hạng; tập "việc" là mọi hạng
// trừ thư mục dựng lại được và dữ liệu của dew.
func TestClassifyWorktree_TruthTable(t *testing.T) {
	t.Parallel()
	repo := gitInit(t)
	writeFiles(t, repo, map[string]string{
		".gitignore": "*.env\nnode_modules/\nout/\n", "src/a.go": "package a\n", "build/keep.txt": "tracked\n",
		"hid.txt": "base\n", "skip.txt": "base\n", "sparse.txt": "base\n",
	})
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "base")
	base := gitIn(t, repo, "rev-parse", "HEAD")
	gitIn(t, repo, "checkout", "-q", "-b", "routine/x-1")
	writeFiles(t, repo, map[string]string{"committed.txt": "c\n"})
	gitIn(t, repo, "add", "committed.txt")
	gitIn(t, repo, "commit", "-q", "-m", "work")
	writeFiles(t, repo, map[string]string{
		"src/a.go":                "package a // changed\n",         // ` M`: tệp theo dõi bị đổi
		"notes.md":                "n\n",                            // `??`: chưa theo dõi
		".env":                    "S=1\n",                          // `!!` ngoài .crush, không dựng lại được
		"build/x.env":             "S=2\n",                          // `!!` trong build/ được theo dõi: việc
		"out/report.md":           "r\n",                            // `!! out/`: out không phải thư mục dựng lại được
		"web/node_modules/p/i.js": "x\n",                            // `!! web/node_modules/`: dựng lại được
		".crush/crush.db":         "db", ".crush/.gitignore": "*\n", // dữ liệu của dew
		"hid.txt":  "work\n", // assume-unchanged: bị giấu khỏi status
		"skip.txt": "work\n", // skip-worktree, có trên đĩa: bị giấu
	})
	gitIn(t, repo, "update-index", "--assume-unchanged", "hid.txt")
	gitIn(t, repo, "update-index", "--skip-worktree", "skip.txt", "sparse.txt")
	require.NoError(t, os.Remove(filepath.Join(repo, "sparse.txt"))) // skip-worktree, không có trên đĩa: không phải việc

	st, err := classifyWorktree(t.Context(), filepath.Join(repo, "src"), base, "routine/x-1")
	require.NoError(t, err)
	require.Equal(t, gitIn(t, repo, "rev-parse", "--show-toplevel"), st.Root, "paths are relative to the repo root, even from a subdir")
	st.Root = ""
	require.Equal(t, worktreeState{
		Changed:     []string{"src/a.go"},
		Untracked:   []string{"notes.md"},
		IgnoredWork: []string{".env", "build/x.env", "out/"},
		Regenerable: []string{"web/node_modules/"},
		DewData:     []string{".crush/.gitignore", ".crush/crush.db"}, // .crush/.gitignore "*" khớp từng tệp, không khớp thư mục
		Hidden:      []string{"hid.txt", "skip.txt"},
		Commits:     1,
		Committed:   []string{"committed.txt"},
	}, st)
	require.Equal(t, []string{".env", "build/x.env", "committed.txt", "hid.txt", "notes.md", "out/", "skip.txt", "src/a.go"}, st.workFiles())

	// Commit trên HEAD tách rời vẫn đếm; không có base thì không đếm (dấu vân tay băm HEAD).
	gitIn(t, repo, "checkout", "-q", "--detach")
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "detached")
	st, err = classifyWorktree(t.Context(), repo, base, "routine/x-1")
	require.NoError(t, err)
	require.Equal(t, 2, st.Commits)
	st, err = classifyWorktree(t.Context(), repo, "", "")
	require.NoError(t, err)
	require.Zero(t, st.Commits)
	require.Empty(t, st.Committed)

	_, err = classifyWorktree(t.Context(), t.TempDir(), "", "")
	require.Error(t, err, "not a git repo")
}

// Lỗi dùng thật (audit 22/10): agent chỉ ghi tệp bị gitignore (out/report.md) thì dấu vân tay cũ
// không đổi, loop báo NO_CHANGE/STALLED oan. Tệp bị ignore ngoài .crush/ và ngoài thư mục dựng lại
// được là việc, tính theo TÊN (kể cả tên tệp mới trong thư mục bị ignore trọn), không theo nội dung
// (Q3: verify ghi lại artefact bị ignore mỗi lần); node_modules/ và .crush/ thì không tính.
func TestGitFingerprint_IgnoredWorkCountsRegenerableAndDewDataDoNot(t *testing.T) {
	t.Parallel()
	dir := ignoreRepo(t)
	fp := gitFingerprint(dir)
	next := func() string {
		t.Helper()
		s, err := fp(t.Context())
		require.NoError(t, err)
		return s
	}
	a := next()
	writeFiles(t, dir, map[string]string{"node_modules/p/i.js": "x\n", "web/node_modules/q.js": "y\n"})
	require.Equal(t, a, next(), "regenerable dirs are not work")
	writeFiles(t, dir, map[string]string{".crush/crush.db": "db", ".crush/.gitignore": "*\n"})
	require.Equal(t, a, next(), "dew's own data is not work")

	writeFiles(t, dir, map[string]string{"out/report.md": "1\n"})
	b := next()
	require.NotEqual(t, a, b, "a new ignored file outside regenerable dirs is work")
	writeFiles(t, dir, map[string]string{"out/report.md": "2\n"})
	require.Equal(t, b, next(), "rewriting an ignored file (a verify artefact) is not a change")
	writeFiles(t, dir, map[string]string{"out/second.md": "x\n"})
	c := next()
	require.NotEqual(t, b, c, "a new file inside an already ignored dir is work")
	writeFiles(t, dir, map[string]string{"x.env": "S=1\n"})
	d := next()
	require.NotEqual(t, c, d, "an ignored file at the root is work")
	writeFiles(t, dir, map[string]string{"x.env": "S=2\n"})
	require.Equal(t, d, next(), "rewriting an ignored root file is not a change")
	writeFiles(t, dir, map[string]string{"notes.md": "1\n"})
	e := next()
	writeFiles(t, dir, map[string]string{"notes.md": "2\n"})
	require.NotEqual(t, e, next(), "untracked (not ignored) files still count by content")
}

// Q3: verify ghi lại artefact bị ignore (junit.xml, cover.out) với nội dung mới mỗi lần chạy. Băm nội
// dung thì cây "đổi" sau mỗi verify: STALLED không bao giờ bắt được, và agent chỉ chạy lại test (ghi
// lại artefact) thành GREEN thay vì NO_CHANGE.
func TestExecuteLoop_VerifyArtefactsDoNotHideStalledOrNoChange(t *testing.T) {
	t.Parallel()
	const artefacts = "mkdir -p out; date +%s%N > out/junit.xml; date +%s%N > cover.env"
	for _, tc := range []struct {
		name, verify string
		rerunsTests  bool // agent chạy lại test: ghi lại artefact, không sửa gì
		want         loop.Stop
	}{
		{"failing gate, agent does nothing", artefacts + "; echo 'FAIL TestX'; exit 1", false, loop.StopStalled},
		{"green gate, agent only reruns the tests", artefacts, true, loop.StopNoChange},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := ignoreRepo(t)
			runs := 0
			agent := func(ctx context.Context, _ string) (loop.RunInfo, error) {
				runs++
				if tc.rerunsTests {
					res, err := shellVerify(dir, 5*time.Second)(ctx, artefacts)
					require.NoError(t, err)
					require.Equal(t, loop.StatusPass, res.Status, res.Output)
				}
				return loop.RunInfo{SessionID: "s"}, nil
			}
			out, err := executeLoop(t.Context(), newLoopParams(t, dir, tc.verify), loopDeps{Agent: agent, Stderr: &bytes.Buffer{}})
			require.NoError(t, err)
			require.Equal(t, tc.want, out.Result.Reason, out.Result.Detail)
		})
	}
}

// Cùng lỗi trên loop thật: cổng xanh sẵn, agent chỉ ghi out/report.md (bị ignore) thì là GREEN, không
// NO_CHANGE; chỉ ghi vào node_modules/ thì vẫn NO_CHANGE.
func TestExecuteLoop_IgnoredWorkIsNotNoChange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		file string
		want loop.Stop
	}{
		{"out/report.md", loop.StopGreen},
		{"node_modules/p/i.js", loop.StopNoChange},
	} {
		dir := ignoreRepo(t)
		agent := func(context.Context, string) (loop.RunInfo, error) {
			writeFiles(t, dir, map[string]string{tc.file: "done\n"})
			return loop.RunInfo{SessionID: "s"}, nil
		}
		out, err := executeLoop(t.Context(), newLoopParams(t, dir, "true"), loopDeps{Agent: agent, Stderr: &bytes.Buffer{}})
		require.NoError(t, err)
		require.Equal(t, tc.want, out.Result.Reason, tc.file)
	}
}

// Cổng đỏ mãi, agent mỗi lượt tạo một tệp bị ignore mới trong out/: cây CÓ đổi, nên hết lượt sửa
// (MAX_REPAIRS) chứ không STALLED; chỉ ghi vào node_modules/ thì STALLED.
func TestExecuteLoop_IgnoredWorkIsNotStalled(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		file string
		want loop.Stop
	}{
		{"out/report-%d.md", loop.StopMaxRepairs},
		{"node_modules/p/i-%d.js", loop.StopStalled},
	} {
		dir := ignoreRepo(t)
		runs := 0
		agent := func(context.Context, string) (loop.RunInfo, error) {
			runs++
			writeFiles(t, dir, map[string]string{fmt.Sprintf(tc.file, runs): "x\n"})
			return loop.RunInfo{SessionID: "s"}, nil
		}
		out, err := executeLoop(t.Context(), newLoopParams(t, dir, "echo 'FAIL TestX'; exit 1"), loopDeps{Agent: agent, Stderr: &bytes.Buffer{}})
		require.NoError(t, err)
		require.Equal(t, tc.want, out.Result.Reason, tc.file)
	}
}

// changed_files là tập "việc" của Q3: có cả tệp bị ignore ngoài thư mục dựng lại được (thư mục bị
// ignore trọn hiện một lần, như dòng keep của gc), không có node_modules/ hay .crush/.
func TestExecuteRoutine_ChangedFilesIncludeIgnoredWork(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	writeFiles(t, e.repo, map[string]string{".gitignore": "out/\nnode_modules/\n*.env\n"})
	gitIn(t, e.repo, "add", ".gitignore")
	gitIn(t, e.repo, "commit", "-q", "-m", "ignore")
	out := e.run(t, func(wt string) error {
		writeFiles(t, wt, map[string]string{"out/report.md": "r\n", "x.env": "S=1\n", "node_modules/p/i.js": "x\n", "done.txt": "d\n"})
		return writeTranscript(wt)
	})
	require.Equal(t, []string{"done.txt", "out/", "x.env"}, out.Report.ChangedFiles)
}

// Tệp skip-worktree không có trên đĩa (sparse-checkout) không phải việc: patch không được xoá nó.
// Tệp stage rồi xoá (`AD`, không có ở HEAD lẫn trên đĩa) không làm hỏng export.
func TestRoutineExport_SparseFileIsNotExportedAsDeleted(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	writeFiles(t, e.repo, map[string]string{"sparse.txt": "base\n"})
	gitIn(t, e.repo, "add", "sparse.txt")
	gitIn(t, e.repo, "commit", "-q", "-m", "sparse")
	e.run(t, func(wt string) error {
		gitIn(t, wt, "update-index", "--skip-worktree", "sparse.txt")
		writeFiles(t, wt, map[string]string{"new.txt": "x\n", "staged.txt": "s\n"})
		gitIn(t, wt, "add", "staged.txt")
		if err := os.Remove(filepath.Join(wt, "staged.txt")); err != nil {
			return err
		}
		return os.Remove(filepath.Join(wt, "sparse.txt"))
	})
	res, err := exportRun(t.Context(), e.data, "nightly", exportOpts{})
	require.NoError(t, err)
	require.Equal(t, []string{"A\tnew.txt"}, res.Files)
}

// Soát độc lập #1 (mất dữ liệu): report.json có base_sha rỗng hay không phải SHA (sửa tay, agent ghi,
// hỏng) mà worktree có commit trên HEAD tách rời: phải GIỮ. Bản cũ giữ vì `rev-list ... ^` hỏng; bộ
// phân loại không được đếm 0 commit rồi gỡ (commit thành unreachable).
func TestRoutineGC_KeepsWhenTheReportBaseIsNotACommit(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	for _, bad := range []string{"", "HEAD"} {
		e := newRoutineEnv(t)
		old := e.run(t, func(wt string) error {
			gitIn(t, wt, "checkout", "-q", "--detach")
			writeFiles(t, wt, map[string]string{"c.txt": "x\n"})
			gitIn(t, wt, "add", "c.txt")
			gitIn(t, wt, "commit", "-q", "-m", "work")
			return nil
		})
		e.run(t, nil)
		editReport(t, old.RunDir, func(r *routineReport) { r.BaseSHA = bad })
		out := e.gc(t, gcOpts{IncludeIgnored: true})
		require.Contains(t, out, "keep "+old.Report.Worktree+": the report's base_sha is not a commit SHA", "base_sha=%q", bad)
		require.DirExists(t, old.Report.Worktree)
	}
	// Không người gọi nào khác được hỏng mở: có nhánh mà không có base là lỗi.
	_, err := classifyWorktree(t.Context(), gitInit(t), "", "routine/x-1")
	require.Error(t, err)
}

// Soát lại R1: agent đổi tên (hay xoá) nhánh của lượt thì rev-list trên refs/heads/<nhánh> hỏng. Bộ
// dọn vẫn giữ khi lỗi, nhưng changed_files không được rỗng: v0.3.1 vẫn liệt kê tệp, và cờ "GREEN mà
// đã sửa test" dựa vào nó.
func TestExecuteRoutine_ChangedFilesSurviveARenamedBranch(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	out := e.run(t, func(wt string) error {
		writeFiles(t, wt, map[string]string{"pkg/x_test.go": "package pkg\n"})
		gitIn(t, wt, "add", "pkg/x_test.go")
		gitIn(t, wt, "commit", "-q", "-m", "test")
		writeFiles(t, wt, map[string]string{"untracked.txt": "u\n"})
		gitIn(t, wt, "branch", "-m", "feat/x")
		return nil
	})
	require.Equal(t, []string{"pkg/x_test.go", "untracked.txt"}, out.Report.ChangedFiles)
	require.True(t, out.Report.TouchedTests)
	_, err := classifyWorktree(t.Context(), out.Report.Worktree, out.Report.BaseSHA, out.Report.Branch)
	require.Error(t, err, "the classifier itself still fails closed (GC keeps)")
}

// Soát độc lập #2: tệp theo dõi `foo` bị thay bằng thư mục foo/ (foo/bar, foo/.env bị ignore). Mục
// ` D foo` giờ là một thư mục: `add -f foo` kéo cả foo/.env vào patch khi không có --include-ignored.
func TestRoutineExport_FileReplacedByADirectoryKeepsIgnoredChildrenOut(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	writeFiles(t, e.repo, map[string]string{"foo": "file\n", ".gitignore": "*.env\n"})
	gitIn(t, e.repo, "add", "-A")
	gitIn(t, e.repo, "commit", "-q", "-m", "foo")
	e.run(t, func(wt string) error {
		if err := os.Remove(filepath.Join(wt, "foo")); err != nil {
			return err
		}
		writeFiles(t, wt, map[string]string{"foo/bar": "b\n", "foo/.env": "S=1\n"})
		return nil
	})
	res, err := exportRun(t.Context(), e.data, "nightly", exportOpts{})
	require.NoError(t, err)
	require.Equal(t, []string{"D\tfoo", "A\tfoo/bar"}, res.Files)
	require.NotContains(t, string(res.Patch), "S=1")
	res, err = exportRun(t.Context(), e.data, "nightly", exportOpts{IncludeIgnored: true})
	require.NoError(t, err)
	require.Equal(t, []string{"D\tfoo", "A\tfoo/.env", "A\tfoo/bar"}, res.Files)
}

// Soát độc lập #3: agent commit cả .crush/crush.db (`git add -f`): export vẫn không bao giờ xuất .crush/.
func TestRoutineExport_NeverExportsCommittedDewData(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.run(t, func(wt string) error {
		if err := writeTranscript(wt); err != nil {
			return err
		}
		writeFiles(t, wt, map[string]string{"new.txt": "x\n"})
		gitIn(t, wt, "add", "-f", ".crush/crush.db", "new.txt")
		gitIn(t, wt, "commit", "-q", "-m", "with session db")
		return nil
	})
	for _, ign := range []bool{false, true} {
		res, err := exportRun(t.Context(), e.data, "nightly", exportOpts{IncludeIgnored: ign})
		require.NoError(t, err)
		require.Equal(t, []string{"A\tnew.txt"}, res.Files, "include-ignored=%v", ign)
		require.NotContains(t, string(res.Patch), ".crush")
	}
}

// Soát lại M1: GIT_LITERAL_PATHSPECS=1 trong env của người dùng tắt pathspec ':(exclude).crush' của
// diff: .crush trong commit của agent lại vào patch. (t.Setenv: không chạy song song.)
func TestRoutineExport_ExcludesDewDataWithLiteralPathspecsInTheEnv(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.run(t, func(wt string) error {
		if err := writeTranscript(wt); err != nil {
			return err
		}
		writeFiles(t, wt, map[string]string{"new.txt": "x\n"})
		gitIn(t, wt, "add", "-f", ".crush/crush.db", "new.txt")
		gitIn(t, wt, "commit", "-q", "-m", "with session db")
		return nil
	})
	t.Setenv("GIT_LITERAL_PATHSPECS", "1")
	res, err := exportRun(t.Context(), e.data, "nightly", exportOpts{})
	require.NoError(t, err)
	require.Equal(t, []string{"A\tnew.txt"}, res.Files)
	require.NotContains(t, string(res.Patch), ".crush")
}

// Soát độc lập #4: hook do repo cấu hình (core.hooksPath; post-index-change chạy mỗi lần index được
// ghi: read-tree, add của export) không chạy khi phân loại, băm dấu vân tay hay export.
func TestClassifierFingerprintAndExportDoNotRunRepoHooks(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	out := e.run(t, func(wt string) error {
		writeFiles(t, wt, map[string]string{"new.txt": "x\n"})
		return nil
	})
	marker := filepath.Join(t.TempDir(), "ran")
	hooks := t.TempDir()
	for _, h := range []string{"post-index-change", "post-checkout", "pre-commit", "reference-transaction", "pre-auto-gc"} {
		require.NoError(t, os.WriteFile(filepath.Join(hooks, h), []byte("#!/bin/sh\necho "+h+" >> '"+marker+"'\n"), 0o755))
	}
	gitIn(t, e.repo, "config", "core.hooksPath", hooks)
	gitIn(t, out.Report.Worktree, "add", "new.txt") // đối chứng: với git thường hook này CÓ chạy
	requireFile(t, marker, "post-index-change\n")
	gitIn(t, out.Report.Worktree, "reset", "-q")
	require.NoError(t, os.Remove(marker))

	_, err := classifyWorktree(t.Context(), out.Report.Worktree, out.Report.BaseSHA, out.Report.Branch)
	require.NoError(t, err)
	_, err = gitFingerprint(out.Report.Worktree)(t.Context())
	require.NoError(t, err)
	_, err = exportRun(t.Context(), e.data, "nightly", exportOpts{IncludeIgnored: true})
	require.NoError(t, err)
	require.NoFileExists(t, marker)
}

// Soát độc lập #5: `routine run --json` hỏng hạ tầng trước khi có báo cáo (repo không còn là repo git...)
// vẫn in MỘT object {"routine","error"}; mã thoát không đổi (1). Lỗi dùng sai (2) không in gì.
func TestPrintRoutineRunJSON_InfrastructureErrorWithoutAReport(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	e := newRoutineEnv(t)
	e.opts.Def.Repo = t.TempDir()
	out, runErr := executeRoutine(t.Context(), e.opts, e.deps(nil))
	require.Error(t, runErr)
	require.Equal(t, 1, exitCodeFor(runErr))
	var b bytes.Buffer
	require.NoError(t, printRoutineRunJSON(&b, "nightly", out, runErr))
	var got map[string]string
	require.NoError(t, json.Unmarshal(b.Bytes(), &got), b.String())
	require.Equal(t, map[string]string{"routine": "nightly", "error": runErr.Error()}, got)
}

// Soát độc lập #6: thư mục chưa theo dõi (-uall chỉ hiện thư mục cho repo lồng) băm theo tên như bản
// cũ, không đi vào đọc từng tệp (node_modules của repo lồng...).
func TestGitFingerprint_UntrackedNestedRepoIsHashedByName(t *testing.T) {
	t.Parallel()
	dir := gitInit(t)
	nested := filepath.Join(dir, "nested")
	require.NoError(t, os.Mkdir(nested, 0o755))
	gitIn(t, nested, "init", "-q")
	writeFiles(t, nested, map[string]string{"a.txt": "1\n", "node_modules/p/i.js": "x\n"})
	st, err := classifyWorktree(t.Context(), dir, "", "")
	require.NoError(t, err)
	require.Equal(t, []string{"nested/"}, st.Untracked)
	fp := gitFingerprint(dir)
	a, err := fp(t.Context())
	require.NoError(t, err)
	writeFiles(t, nested, map[string]string{"a.txt": "2\n", "node_modules/p/i.js": "y\n"})
	b, err := fp(t.Context())
	require.NoError(t, err)
	require.Equal(t, a, b, "files inside an untracked nested repo are not read")
}

// Soát độc lập #7: nhánh của lượt có commit nhưng HEAD đã tách về base: tệp của commit đó vẫn vào
// changed_files (Commits > 0 mà Committed rỗng là thiếu).
func TestClassifyWorktree_BranchCommitsWhenHEADIsBackAtBase(t *testing.T) {
	t.Parallel()
	repo := gitInit(t)
	base := gitIn(t, repo, "rev-parse", "HEAD")
	gitIn(t, repo, "checkout", "-q", "-b", "routine/x-1")
	writeFiles(t, repo, map[string]string{"b.txt": "b\n"})
	gitIn(t, repo, "add", "b.txt")
	gitIn(t, repo, "commit", "-q", "-m", "on branch")
	gitIn(t, repo, "checkout", "-q", "--detach", base)
	st, err := classifyWorktree(t.Context(), repo, base, "routine/x-1")
	require.NoError(t, err)
	require.Equal(t, 1, st.Commits)
	require.Equal(t, []string{"b.txt"}, st.Committed)
	require.Equal(t, []string{"b.txt"}, changedFiles(t.Context(), repo, base, "routine/x-1"))
}
