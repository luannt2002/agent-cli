package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/crush/internal/housekeep"
	"github.com/charmbracelet/crush/internal/lock"
	"github.com/charmbracelet/crush/internal/routine"
	"github.com/spf13/cobra"
)

func init() {
	rmCmd := &cobra.Command{
		Use:   "rm NAME [--discard]",
		Short: "Remove a routine you are done with: its runs' worktrees and branches, then its definition",
		Long: `Remove a routine: the worktree and branch of every run, then its definition file.
Reports, journals and the session data moved next to them stay in its runs directory.

Nothing is removed while the routine is running. A run whose worktree or branch still
holds work (uncommitted or untracked files, gitignored files outside .crush, files hidden
from git status, commits not in its base, or a session opened after it finished) stops
the whole removal and is listed: export it first with "dew routine export NAME --run
STAMP", or pass --discard to delete that work too (the list is printed either way). A
report whose worktree or branch is not the run's own, a worktree holding files you
cannot delete, or a worktree you locked with "git worktree lock" stops it even with
--discard. The session data (.crush/crush.db, logs) is moved next to the report before a
worktree is removed, as "routine gc" does. The .git/worktrees entry of a worktree that
is already gone is removed only while its HEAD is still the run's branch.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			discard, _ := cmd.Flags().GetBool("discard")
			return removeRoutine(cmd.Context(), routineDefsDir(), routineDataDir(), args[0], discard, cmd.OutOrStdout())
		},
	}
	rmCmd.Flags().Bool("discard", false, "Also remove runs whose worktree or branch holds work (that work is lost)")
	rmCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return &usageError{err} })
	routineCmd.AddCommand(rmCmd)
}

// rmRun là một lượt mà `routine rm` sẽ dọn: worktree còn trên đĩa (wt) hay chỉ còn nhánh.
type rmRun struct {
	dir   string
	rep   routineReport
	wt    bool
	entry worktreeEntry // mục quản trị của worktree (khi worktree đã mất: gỡ theo dropGoneWorktreeEntry)
}

// rmIssue là một lượt chặn `routine rm`: what (worktree hay nhánh) và vì sao.
type rmIssue struct {
	what, why, stamp string
	wt               bool
}

// removeRoutine là `routine rm` (Q4): gỡ worktree và nhánh của mọi lượt rồi tệp định nghĩa; báo cáo,
// journal và dữ liệu phiên đã chuyển ra được giữ. Giữ khoá của routine suốt lúc gỡ (lượt mới không
// bắt đầu giữa chừng). Xét HẾT các lượt trước khi xoá gì: lượt còn sống, lượt còn việc (trừ khi
// discard), đường dẫn không phải của lượt hay tệp không xoá được thì không xoá gì cả (fail closed).
func removeRoutine(ctx context.Context, defsDir, dataDir, name string, discard bool, w io.Writer) error {
	if name == "" || filepath.Base(name) != name || strings.HasPrefix(name, ".") {
		return &usageError{fmt.Errorf("%q is not a routine name", name)}
	}
	defPath := filepath.Join(defsDir, name+".json")
	routineDir := filepath.Join(dataDir, name)
	_, defErr := os.Lstat(defPath)
	if _, err := os.Stat(routineDir); err != nil {
		if errors.Is(defErr, fs.ErrNotExist) {
			return fmt.Errorf("no routine %q (no %s and no %s)", name, defPath, routineDir)
		}
	} else {
		release, err := lock.TryFile(routineLockPath(dataDir, name))
		if errors.Is(err, lock.ErrContended) {
			return fmt.Errorf("routine %q is running (its lock is held): nothing removed", name)
		}
		if err != nil {
			return err
		}
		defer release()
	}

	runs, blocked, work, err := planRoutineRm(ctx, dataDir, name)
	if err != nil {
		return err
	}
	for _, is := range blocked {
		fmt.Fprintf(w, "cannot remove %s: %s (run %s)\n", is.what, is.why, is.stamp)
	}
	if len(blocked) > 0 {
		return fmt.Errorf("routine %q: %d run(s) cannot be removed, even with --discard; nothing removed", name, len(blocked))
	}
	// Cả khi --discard: in đúng việc sẽ mất trước khi gỡ.
	for _, is := range work {
		fmt.Fprintf(w, "holds work: %s: %s (run %s)\n", is.what, is.why, is.stamp)
		if is.wt {
			fmt.Fprintf(w, "  export it first: dew routine export %s --run %s\n", name, is.stamp)
		} else {
			fmt.Fprintf(w, "  its worktree is gone; the commits are only on that branch: merge or cherry-pick them first\n")
		}
	}
	switch {
	case len(work) > 0 && !discard:
		fmt.Fprintf(w, "or pass --discard to delete that work too\n")
		return fmt.Errorf("routine %q: %d run(s) hold work; nothing removed", name, len(work))
	case len(work) > 0:
		fmt.Fprintf(w, "discarding the work above (--discard)\n")
	}

	var errs []error
	removed, branches := 0, 0
	for _, r := range runs {
		rep := r.rep
		var err error
		switch {
		case r.wt:
			err = removeRunWorktree(ctx, r.dir, &rep, discard)
			if !rep.WorktreeRemoved.IsZero() {
				removed++
				fmt.Fprintf(w, "removed %s (branch %s)\n", rep.Worktree, rep.Branch)
			}
		default:
			if rep.Worktree != "" {
				err = dropGoneWorktreeEntry(rep.Worktree, r.entry)
			}
			if err == nil && discard {
				err = discardRunBranch(ctx, rep)
			} else if err == nil {
				err = deleteRunBranch(ctx, rep)
			}
			if err == nil {
				branches++
				fmt.Fprintf(w, "deleted branch %s%s\n", rep.Branch, entryStaysNote(r.entry))
				if rep.Worktree != "" && rep.WorktreeRemoved.IsZero() {
					rep.WorktreeGone = time.Now()
				}
			}
		}
		rep.GCError = ""
		if err != nil {
			rep.GCError = err.Error()
			errs = append(errs, fmt.Errorf("%s: %w", r.dir, err))
		}
		errs = append(errs, writeReport(r.dir, rep))
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("routine %q: its definition is kept; fix this and run rm again: %w", name, err)
	}
	if err := os.Remove(defPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	fmt.Fprintf(w, "removed routine %s: %d worktree(s), %d branch(es) of runs whose worktree was gone, definition %s; reports, journals and session data stay in %s\n",
		name, removed, branches, defPath, filepath.Join(routineDir, "runs"))
	return nil
}

// planRoutineRm xét từng lượt của routine (người gọi giữ khoá): runs là những lượt có worktree hay
// nhánh để dọn; blocked là lượt không bao giờ gỡ (đường dẫn không phải của lượt, tệp không xoá được);
// work là lượt còn việc (Q3, hay commit trên nhánh), chỉ gỡ khi --discard. Lượt còn sống thì lỗi.
func planRoutineRm(ctx context.Context, dataDir, name string) (runs []rmRun, blocked, work []rmIssue, err error) {
	dirs, _ := filepath.Glob(filepath.Join(dataDir, name, "runs", "*"))
	sort.Strings(dirs)
	for _, dir := range dirs {
		rep, err := readReport(dir)
		if err != nil {
			continue // chết trước khi ghi báo cáo, tức trước khi có worktree
		}
		if rep.Reason == "RUNNING" {
			// Giữ khoá: RUNNING là lượt đã chết, trừ khi pid của nó vẫn là `routine run` này (NFS).
			if routine.RunAlive(rep.PID, name) {
				return nil, nil, nil, fmt.Errorf("routine %q is running (pid %d, %s): nothing removed", name, rep.PID, dir)
			}
			rep = abandonRun(dir, rep)
		}
		stamp := filepath.Base(dir)
		wt := rep.Worktree != "" && rep.WorktreeRemoved.IsZero() && rep.WorktreeGone.IsZero()
		if _, err := os.Lstat(rep.Worktree); wt && errors.Is(err, fs.ErrNotExist) {
			wt = false
		}
		exists, n, berr := runBranchCommits(ctx, rep)
		if !wt && !exists && berr == nil {
			continue // không còn gì để gỡ
		}
		is := rmIssue{what: rep.Branch, stamp: stamp, wt: wt}
		if wt {
			is.what = rep.Worktree
		}
		// Lượt hỏng trước khi có worktree (ERROR) không ghi worktree nhưng có thể còn nhánh.
		if !runPathsMatch(dataDir, name, stamp, rep) && (rep.Worktree != "" || rep.Branch != "routine/"+name+"-"+stamp) {
			is.why = errRunPaths
			blocked = append(blocked, is)
			continue
		}
		var entry worktreeEntry
		if rep.Worktree != "" {
			if entry, err = findWorktreeEntry(ctx, rep.Repo, rep.Worktree, rep.Branch); err != nil {
				is.why = "cannot read its worktree entry: " + err.Error()
				blocked = append(blocked, is)
				continue
			}
		}
		// Người dùng khoá worktree (`git worktree lock`) là cố ý giữ, kể cả khi --discard.
		if entry.Locked {
			is.why = fmt.Sprintf("its worktree entry %s is locked (worktree locked); git worktree unlock it first", entry.Admin)
			blocked = append(blocked, is)
			continue
		}
		switch {
		case wt:
			if hint := housekeep.FindStuck(rep.Worktree, currentUID()).Hint(rep.Worktree); hint != "" {
				is.why = hint
				blocked = append(blocked, is)
				continue
			}
			is.why, _ = keepReason(ctx, dataDir, name, dir, rep, gcOpts{})
		case berr != nil:
			is.why = "cannot count its commits: " + berr.Error()
		case n != "0":
			is.why = fmt.Sprintf("%s commit(s) not in base %s", n, shortSHA(rep.BaseSHA))
		}
		if is.why != "" {
			work = append(work, is)
		}
		runs = append(runs, rmRun{dir: dir, rep: rep, wt: wt, entry: entry})
	}
	return runs, blocked, work, nil
}
