package cmd

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	exportCmd := &cobra.Command{
		Use:   "export NAME [--run STAMP] [-o FILE]",
		Short: "Write one patch of a run's work against its base commit, for review",
		Long: `Write the work of a routine run (the newest, or --run STAMP: the name of its run
directory) as ONE patch against the run's base commit: commits on its branch,
uncommitted changes and new untracked files, binary files included. Gitignored files
are left out unless --include-ignored (never .crush, nor regenerable dirs such as
node_modules, dist, build, target or .venv). The patch applies with "git apply" to a
clean checkout of the base commit. Prints every file (A added, M modified, D deleted)
and the patch's sha256; without -o the patch goes to stdout and that list to stderr.
The run's worktree, its index and the repository are not changed. A run whose
worktree is gone (removed by "routine gc") cannot be exported, nor one whose report
points anywhere but the run's own worktree, branch and base commit. When the
worktree's HEAD is no longer the run's branch (the agent switched or detached it),
export refuses unless --head.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var o exportOpts
			o.Stamp, _ = cmd.Flags().GetString("run")
			o.IncludeIgnored, _ = cmd.Flags().GetBool("include-ignored")
			o.Head, _ = cmd.Flags().GetBool("head")
			outPath, _ := cmd.Flags().GetString("output")
			// Ctrl+C/SIGTERM huỷ context: git con bị giết, buildPatch trả lỗi, thư mục tạm bị xoá.
			ctx, cancel := signal.NotifyContext(cmd.Context(), stopSignals()...)
			defer cancel()
			res, err := exportRun(ctx, routineDataDir(), args[0], o)
			if err != nil {
				return err
			}
			info, dest := cmd.OutOrStdout(), outPath
			if outPath == "" {
				if _, err := cmd.OutOrStdout().Write(res.Patch); err != nil {
					return err
				}
				info, dest = cmd.ErrOrStderr(), "-"
			} else if err := os.WriteFile(outPath, res.Patch, 0o600); err != nil {
				return err
			}
			for _, f := range res.Files {
				fmt.Fprintln(info, f)
			}
			fmt.Fprintf(info, "%d file(s) against base %s (run %s); apply to a clean checkout of it with: git apply %s\n", len(res.Files), res.Base, res.Stamp, dest)
			fmt.Fprintf(info, "%x  %s\n", sha256.Sum256(res.Patch), dest)
			return nil
		},
	}
	exportCmd.Flags().String("run", "", "Run to export: the name of its run directory (default: the newest run)")
	exportCmd.Flags().StringP("output", "o", "", "Write the patch to this file (mode 0600 when created) instead of stdout")
	exportCmd.Flags().Bool("include-ignored", false, "Also export gitignored files outside .crush and regenerable dirs")
	exportCmd.Flags().Bool("head", false, "Export the worktree as it is even when its HEAD left the run's branch")
	exportCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return &usageError{err} })
	routineCmd.AddCommand(exportCmd)
}

// commitSHA là một object ID đầy đủ (SHA-1 hoặc SHA-256), chữ thường.
var commitSHA = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`)

// exportOpts là tuỳ chọn của `routine export`.
type exportOpts struct {
	Stamp          string // tên thư mục lượt; rỗng = lượt mới nhất
	IncludeIgnored bool
	Head           bool
}

// exportResult là patch của một lượt và danh sách tệp của nó ("A\tđường/dẫn", theo thứ tự đường dẫn).
type exportResult struct {
	Patch []byte
	Files []string
	Base  string
	Stamp string
}

// exportRun tìm lượt (o.Stamp rỗng: lượt mới nhất) rồi dựng patch của worktree của nó (buildPatch).
func exportRun(ctx context.Context, dataDir, name string, o exportOpts) (exportResult, error) {
	stamp := o.Stamp
	if filepath.Base(name) != name || strings.HasPrefix(name, ".") {
		return exportResult{}, &usageError{fmt.Errorf("%q is not a routine name", name)}
	}
	runs := filepath.Join(dataDir, name, "runs")
	if stamp == "" {
		dirs, _ := filepath.Glob(filepath.Join(runs, "*"))
		if len(dirs) == 0 {
			return exportResult{}, fmt.Errorf("routine %q has no runs in %s", name, dataDir)
		}
		stamp = filepath.Base(slices.Max(dirs))
	} else if filepath.Base(stamp) != stamp || strings.HasPrefix(stamp, ".") {
		return exportResult{}, &usageError{fmt.Errorf("--run %q is not the name of a run directory", stamp)}
	}
	rep, err := readReport(filepath.Join(runs, stamp))
	if err != nil {
		return exportResult{}, fmt.Errorf("run %s of %q: %w", stamp, name, err)
	}
	res := exportResult{Base: rep.BaseSHA, Stamp: stamp}
	// report.json sửa được từ bên ngoài: chỉ tin worktree/nhánh đúng chỗ lượt này đặt (không
	// symlink sang chỗ khác) và base_sha là SHA thật (dạng cờ như --output=<tệp> sẽ thành cờ của git).
	fi, statErr := os.Lstat(rep.Worktree)
	switch {
	case rep.Worktree != "" && !runPathsMatch(dataDir, name, stamp, rep),
		statErr == nil && !fi.IsDir():
		return res, fmt.Errorf("run %s of %q: %s; not exported", stamp, name, errRunPaths)
	case rep.Worktree == "" || statErr != nil || !rep.WorktreeRemoved.IsZero():
		msg := fmt.Sprintf("the worktree of run %s of %q is gone (%s): nothing to export", stamp, name,
			cmp.Or(rep.Worktree, "never created: "+rep.Detail))
		if _, err := os.Stat(rep.SessionDB); rep.SessionDB != "" && err == nil {
			msg += "; its session database (the transcript) is kept at " + rep.SessionDB
		}
		return res, errors.New(msg)
	case !commitSHA.MatchString(rep.BaseSHA):
		return res, fmt.Errorf("run %s of %q: the report's base_sha %q is not a commit SHA; not exported", stamp, name, rep.BaseSHA)
	}
	// Agent chuyển nhánh hay tách HEAD: cây làm việc giờ dựa trên commit khác nhánh của lượt.
	if head, _ := runGit(ctx, rep.Worktree, "symbolic-ref", "-q", "HEAD"); !o.Head && head != "refs/heads/"+rep.Branch {
		return res, fmt.Errorf("run %s of %q: the worktree's HEAD is %s, not the run's branch %s (switched or detached during the run); pass --head to export the worktree as it is",
			stamp, name, cmp.Or(head, "detached"), rep.Branch)
	}
	res.Patch, res.Files, err = buildPatch(ctx, rep.Worktree, rep.BaseSHA, o.IncludeIgnored)
	if err == nil && len(res.Files) == 0 {
		err = fmt.Errorf("run %s of %q has no changes against its base %s: nothing to export", stamp, name, shortSHA(rep.BaseSHA))
	}
	return res, err
}

// buildPatch dựng patch của worktree wt so với base: commit từ base tới HEAD cộng các mục "việc"
// của bộ phân loại (classifyWorktree; mục bị ignore chỉ khi includeIgnored, .crush/ không bao
// giờ). Làm trên index và object store TẠM (GIT_INDEX_FILE, GIT_OBJECT_DIRECTORY + alternates):
// index của lượt, có thể đang chạy, và object store của repo không đổi.
func buildPatch(ctx context.Context, wt, base string, includeIgnored bool) ([]byte, []string, error) {
	st, err := classifyWorktree(ctx, wt, "", "")
	if err != nil {
		return nil, nil, err
	}
	objects, err := runGit(ctx, wt, "rev-parse", "--path-format=absolute", "--git-path", "objects")
	if err != nil {
		return nil, nil, err
	}
	tmp, err := os.MkdirTemp("", "dew-export-")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(tmp)
	if err := os.Mkdir(filepath.Join(tmp, "objects"), 0o700); err != nil {
		return nil, nil, err
	}
	env := append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(tmp, "index"),
		"GIT_OBJECT_DIRECTORY="+filepath.Join(tmp, "objects"), "GIT_ALTERNATE_OBJECT_DIRECTORIES="+objects)
	// Không chạy fsmonitor hay hook do repo cấu hình (post-index-change chạy mỗi lần read-tree/add ghi
	// index). Clean filter do repo khai vẫn chạy khi add (xem classifyWorktree).
	git := func(stdin string, args ...string) ([]byte, error) {
		c := exec.CommandContext(ctx, "git", append([]string{"-C", wt, "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
		var stderr bytes.Buffer
		c.Env, c.Stdin, c.Stderr = env, strings.NewReader(stdin), &stderr
		out, err := c.Output()
		if err != nil {
			return nil, fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(stderr.String()))
		}
		return out, nil
	}
	// Index tạm bắt đầu từ HEAD (tệp đã theo dõi mà khớp .gitignore không thành "đã xoá", tệp
	// sparse không có trên đĩa không thành "đã xoá"), rồi nhận đúng các mục "việc": sửa, xoá, tệp
	// mới, tệp bị giấu khỏi status (index tạm không mang cờ assume-unchanged). .crush loại tường
	// minh, kể cả tệp repo theo dõi trong đó.
	if _, err := git("", "read-tree", "HEAD"); err != nil {
		return nil, nil, err
	}
	// Mục còn trên đĩa thì add, mục đã mất thì bỏ khỏi index tạm (--ignore-unmatch: stage rồi xoá
	// thì HEAD cũng không có nó). -f cho tệp (tệp stage cưỡng bức trong index của lượt có thể khớp
	// .gitignore) và cho mục bị ignore được chọn; KHÔNG cho thư mục khác (tệp `foo` thành thư mục
	// foo/): -f trên thư mục kéo theo con bị ignore (foo/.env).
	var force, plain, gone []string
	sortOut := func(paths []string, ignored bool) {
		for _, p := range paths {
			switch fi, err := os.Lstat(filepath.Join(st.Root, p)); {
			case p == ".crush" || strings.HasPrefix(p, ".crush/"):
			case err != nil:
				gone = append(gone, p)
			case fi.IsDir() && !ignored:
				plain = append(plain, p)
			default:
				force = append(force, p)
			}
		}
	}
	sortOut(slices.Concat(st.Changed, st.Untracked, st.Hidden), false)
	if includeIgnored {
		sortOut(st.IgnoredWork, true)
	}
	for _, step := range []struct {
		paths []string
		args  []string
	}{{force, []string{"add", "-f"}}, {plain, []string{"add"}}, {gone, []string{"rm", "--cached", "-r", "-q", "--ignore-unmatch"}}} {
		if len(step.paths) == 0 {
			continue
		}
		// --literal-pathspecs: tên tệp có * hay : là tên, không phải mẫu.
		args := append(append([]string{"--literal-pathspecs"}, step.args...), "--pathspec-from-file=-", "--pathspec-file-nul")
		if _, err := git(strings.Join(step.paths, "\x00"), args...); err != nil {
			return nil, nil, err
		}
	}
	// Cấu hình của repo không được đổi dạng patch (màu, tiền tố, đổi tên, submodule dạng log,
	// chương trình diff ngoài). --end-of-options: base không bao giờ được đọc như một cờ. Pathspec loại
	// .crush cả trong commit của lượt (agent `git add -f .crush/crush.db` rồi commit);
	// --no-literal-pathspecs: GIT_LITERAL_PATHSPECS=1 trong env của người dùng không được tắt nó.
	diff := []string{"--no-literal-pathspecs", "diff", "--cached", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-relative", "--no-color", "--submodule=short"}
	paths := []string{"--", ".", ":(exclude).crush"}
	names, err := git("", slices.Concat(diff, []string{"--name-status", "-z", "--end-of-options", base}, paths)...)
	if err != nil {
		return nil, nil, err
	}
	var files []string
	fields := strings.Split(strings.TrimSuffix(string(names), "\x00"), "\x00")
	for i := 0; i+1 < len(fields); i += 2 {
		files = append(files, fields[i]+"\t"+fields[i+1])
	}
	if len(files) == 0 {
		return nil, nil, nil
	}
	patch, err := git("", slices.Concat(diff, []string{"--binary", "--full-index", "--src-prefix=a/", "--dst-prefix=b/", "--end-of-options", base}, paths)...)
	return patch, files, err
}
