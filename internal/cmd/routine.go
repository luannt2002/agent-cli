package cmd

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/event"
	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/housekeep"
	"github.com/charmbracelet/crush/internal/lock"
	"github.com/charmbracelet/crush/internal/loop"
	"github.com/charmbracelet/crush/internal/routine"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

// exitCodeBusy là EX_TEMPFAIL: có một lượt chạy khác của routine đang giữ khoá.
const exitCodeBusy = 75

// exitCodeError mang một mã thoát riêng (ngoài 1 và 2).
type exitCodeError struct {
	code int
	err  error
	// refused là loại từ chối chạy (Q5 của docs/routine-invariants.md), cho `routine run --json`;
	// rỗng khi lỗi không phải một lần từ chối.
	refused string
}

// Loại từ chối chạy (thoát 75). Chỉ lock_held là chạy lại sau cũng vô ích (sẽ chạy trùng).
const (
	refusedLockHeld  = "lock_held"
	refusedNoSlot    = "no_slot"
	refusedLowMemory = "low_memory"
	refusedLowDisk   = "low_disk"
)

func (e *exitCodeError) Error() string { return e.err.Error() }
func (e *exitCodeError) Unwrap() error { return e.err }

// exitCodeFor ánh xạ lỗi của lệnh sang mã thoát: 2 dùng sai, mã riêng nếu có, còn lại 1.
func exitCodeFor(err error) int {
	if isUsageError(err) {
		return 2
	}
	if ec, ok := errors.AsType[*exitCodeError](err); ok {
		return ec.code
	}
	return 1
}

var routineCmd = &cobra.Command{
	Use:   "routine",
	Short: "Define, run and schedule routines (loops with a fixed goal and verify command)",
	Long: `A routine is a saved definition of a recurring job: a goal, the verify command that
defines "done", a repository, limits, and optionally a schedule. Running it uses the
same engine as "dew loop".

Definitions live in your own config directory (never inside a repository, so a
repository cannot add a scheduled job). Two modes:

  maintenance  mechanical, verifiable, endless work: may be scheduled
  goal         work with a target, like a feature: runs by hand only

Every run takes a lock (a second run exits 75), works in its OWN git worktree on a
routine/* branch (your checkout is never touched), runs with the unattended command
guard on (no push, merge, hard reset, recursive delete outside the worktree), and
writes a report outside the worktree. It never commits unless the definition says
"commit": true, never pushes, never merges: reviewing and merging stay yours. Before
each run, worktrees of older runs that are clean and have no commits are removed
(the newest is kept; see "routine gc").

Scheduling is print-only: "routine schedule NAME --print" shows systemd user units
or a crontab line for you to install. dew never edits your crontab or timers.`,
}

func init() {
	newCmd := &cobra.Command{
		Use:   "new NAME",
		Short: "Write a new routine definition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := cmd.Flags()
			d := routine.Definition{Name: args[0]}
			d.Mode, _ = f.GetString("mode")
			d.Repo, _ = f.GetString("repo")
			d.Goal, _ = f.GetString("goal")
			d.Verify, _ = f.GetString("verify")
			d.Model, _ = f.GetString("model")
			d.Commit, _ = f.GetBool("commit")
			if f.Changed("max-repairs") {
				n, _ := f.GetInt("max-repairs")
				d.MaxRepairs = &n
			}
			// Chuỗi nguyên văn: Definition.Validate (writeDefinition) kiểm như khi đọc tệp.
			d.RunTimeout, _ = f.GetString("run-timeout")
			d.MaxDuration, _ = f.GetString("max-duration")
			if f.Changed("stream-retries") {
				n, _ := f.GetInt("stream-retries")
				d.StreamRetries = &n
			}
			if d.Repo != "" {
				abs, err := filepath.Abs(d.Repo)
				if err != nil {
					return &usageError{err}
				}
				d.Repo = abs
			}
			// Không --model: đóng băng model mặc định HIỆN TẠI vào định nghĩa, để đổi mặc định
			// sau này không âm thầm đổi model của routine.
			frozen, why := "", "unknown"
			if d.Model == "" && d.Repo != "" {
				if m, err := defaultLargeModel(d.Repo); err != nil {
					why = "unknown: " + err.Error()
				} else {
					frozen, d.Model = m, m
				}
			}
			path, err := writeDefinition(routineDefsDir(), d)
			if err != nil {
				return &usageError{err}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\nrun it by hand:  dew routine run %s\n", path, d.Name)
			switch {
			case frozen != "":
				fmt.Fprintf(cmd.OutOrStdout(), "model: %s (no --model: froze the current default large model, so changing the default later does not change this routine; to use another, edit \"model\" in %s)\n", frozen, path)
			case d.Model == "":
				warnUnpinnedModel(cmd.ErrOrStderr(), why, path)
			}
			if d.Mode == routine.ModeMaintenance {
				fmt.Fprintf(cmd.OutOrStdout(), "to schedule it, add a \"schedule\" object to the file, then: dew routine schedule %s --print\n", d.Name)
			}
			return nil
		},
	}
	newCmd.Flags().String("mode", "", "maintenance (may be scheduled) or goal (by hand only) (required)")
	newCmd.Flags().String("repo", "", "Git repository the routine works on (required)")
	newCmd.Flags().String("goal", "", "What to do, self-contained (required)")
	newCmd.Flags().String("verify", "", "Command that defines \"done\": exit 0 = done (required)")
	newCmd.Flags().String("model", "", "Model to use (default: the current default large model, written into the definition)")
	newCmd.Flags().Int("max-repairs", routine.DefaultMaxRepairs, "Repair turns after the first agent turn (0-5)")
	newCmd.Flags().String("run-timeout", "", "Maximum time of one agent turn, like 30m (default "+routine.DefaultRunTimeout.String()+")")
	newCmd.Flags().String("max-duration", "", "Stop before starting a new agent turn after this long, like 2h (default: no limit)")
	newCmd.Flags().Int("stream-retries", routine.DefaultStreamRetries, fmt.Sprintf("Retries in a row of a turn cut off by a temporary provider error (0-%d, 0 = no retry)", routine.MaxStreamRetries))
	newCmd.Flags().Bool("commit", false, "Allow git commit on the routine/* branch (never push)")
	newCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return &usageError{err} })

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List routine definitions and their last run",
		Long: `List every routine definition with its newest run: state, the model it ran, how long it
took (or has run so far), verify, changed files and agent turns. RUNNING means the run's
process is alive (checked by pid, never read from report.json alone); a run whose process
died shows ABANDONED, one that cannot be checked (no pid, or Windows) UNKNOWN. verify is
PASS only when the run ended GREEN or NO_CHANGE; a green verify after any other stop shows
"PASS (not green)"; "not run" means verify never ran. --json adds the same fields.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries := listRoutines(routineDefsDir(), routineDataDir())
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(entries)
			}
			if len(entries) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "no routines in %s\n", routineDefsDir())
				return nil
			}
			printRoutineList(cmd.OutOrStdout(), entries)
			return nil
		},
	}
	listCmd.Flags().Bool("json", false, "Print the list as JSON")

	showCmd := &cobra.Command{
		Use:   "show NAME",
		Short: "Show a routine definition, its sha256 and where its files live",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, sum, err := loadRoutine(routineDefsDir(), args[0])
			if err != nil {
				return err
			}
			out := map[string]any{
				"definition": d, "sha256": sum,
				"lock":     routineLockPath(routineDataDir(), d.Name),
				"runs_dir": filepath.Join(routineDataDir(), d.Name, "runs"),
			}
			if runs := listRuns(routineDataDir(), d.Name); len(runs) > 0 {
				out["last_run"] = runs[0]
				out["runs"] = runs
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		},
	}

	scheduleCmd := &cobra.Command{
		Use:   "schedule NAME --print",
		Short: "Print the systemd units or crontab line that schedule a routine (never installs)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if print, _ := cmd.Flags().GetBool("print"); !print {
				return &usageError{errors.New("dew only prints the schedule for you to install; add --print")}
			}
			d, sum, err := loadRoutine(routineDefsDir(), args[0])
			if err != nil {
				return err
			}
			bin, err := os.Executable()
			if err != nil {
				return err
			}
			text, err := d.ScheduleText(bin, sum, schedulePATH(), scheduleEnv())
			if err != nil {
				return &usageError{err}
			}
			warnAllowlistNotScheduled(cmd.ErrOrStderr())
			_, err = io.WriteString(cmd.OutOrStdout(), text)
			return err
		},
	}
	scheduleCmd.Flags().Bool("print", false, "Print the schedule text (required: dew never edits timers or crontab)")

	runCmd := &cobra.Command{
		Use:   "run NAME|FILE",
		Short: "Run a routine once, in its own worktree, with a lock",
		Long: `Run a routine once. Exit codes: 0 GREEN (or NO_CHANGE for a maintenance routine:
nothing to do), 1 any other stop reason or failure, 2 bad usage (including
--scheduled on a goal routine or a changed definition), 75 another run of this
routine holds the lock, all routine slots are in use (DEW_ROUTINE_SLOTS runs
at once across all routines, default 2, 0 = no limit), less memory is available
than DEW_ROUTINE_MIN_MEM_MB (MemAvailable in MB; default 0 = no check; measured on
Linux only, elsewhere a warning), or the disk holding dew's data has less than
DEW_MIN_FREE_GB free (default 10, 0 = no check; "dew clean" shows what you can
delete). Before that check, a Go build cache over DEW_GOCACHE_MAX GB (default 20,
0 = no cap) is trimmed to 80% of it, oldest entries first, at most once an hour.
Set these in your shell or in your dew env file. At start, the newest run of any
other routine that is still marked RUNNING but whose process is provably gone
(Linux) is closed as ABANDONED.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rejectCwdFlag(cmd); err != nil {
				return err
			}
			d, sum, err := loadRoutine(routineDefsDir(), args[0])
			if err != nil {
				return &usageError{err}
			}
			opts := routineOpts{Def: d, Sum: sum, DataDir: routineDataDir()}
			if opts.Slots, err = routineSlots(); err != nil {
				return &usageError{err}
			}
			if opts.MinMemMB, err = routineMinMemMB(); err != nil {
				return &usageError{err}
			}
			if opts.GoCacheMax, err = envGB(goCacheMaxEnv, 20); err != nil {
				return &usageError{err}
			}
			if opts.MinFree, err = envGB(minFreeEnv, 10); err != nil {
				return &usageError{err}
			}
			opts.Scheduled, _ = cmd.Flags().GetBool("scheduled")
			opts.ExpectSHA, _ = cmd.Flags().GetString("expect-sha")

			ctx, cancel := signal.NotifyContext(context.Background(), stopSignals()...)
			defer cancel()
			event.SetNonInteractive(true)

			deps := routineDeps{
				Stderr: os.Stderr,
				NewAgent: func(worktree string) (func(context.Context, string) (loop.RunInfo, error), func(), func() string, error) {
					if err := os.Chdir(worktree); err != nil {
						return nil, nil, nil, fmt.Errorf("enter the worktree: %w", err)
					}
					return newLocalLoopAgent(ctx, cmd, loopParams{Model: d.Model}, io.Discard)
				},
			}
			out, err := executeRoutine(ctx, opts, deps)
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				if jerr := printRoutineRunJSON(cmd.OutOrStdout(), d.Name, out, err); jerr != nil {
					return errors.Join(err, jerr)
				}
			}
			if err != nil {
				return err
			}
			return routineExitErr(d.Mode, out.Result.Reason)
		},
	}
	runCmd.Flags().Bool("json", false, `Print one JSON object on stdout at the end: the final report, or {"refused":"lock_held|no_slot|low_memory|low_disk","routine":...,"detail":...} when the run was refused (exit 75), or {"routine":...,"error":...} when it failed before writing a report (exit 1); nothing on bad usage (exit 2)`)
	runCmd.Flags().Bool("scheduled", false, "Mark the run as started by a schedule (refused for goal routines)")
	runCmd.Flags().String("expect-sha", "", "Refuse to run unless the definition file has this sha256 (set by the printed schedule)")
	runCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return &usageError{err} })

	gcCmd := &cobra.Command{
		Use:   "gc [NAME]",
		Short: "Remove worktrees of finished runs that are clean and have no commits (keeps the newest)",
		Long: `Remove the worktree and branch of finished runs, for one routine or all of them.
A worktree is kept (one "keep" line says why) when it has uncommitted or untracked
changes, files hidden from git status (assume-unchanged or skip-worktree), commits not
in its base (also on a detached HEAD), gitignored files outside .crush (unless
--include-ignored; regenerable dirs such as node_modules, dist, build, target, .venv
or __pycache__ do not count and are dropped), a session database written after the
run finished (someone opened it in dew), or a report whose worktree or branch is not
the run's own. A branch whose deletion failed earlier is retried. A worktree holding
files you cannot delete (owned by another user, like root after a build in Docker,
or read-only directories, like a Go module cache) is kept; its keep line says whose
they are and how to fix that.
The newest run's worktree is always kept, and so are reports and journals. The run's
session database (.crush/crush.db, -wal, -shm) and .crush/logs are moved next to its
report first; if one cannot be moved the worktree stays and the report says why. A
routine whose lock is held is skipped. "routine run" does the same for its own
routine before every run, always keeping worktrees with gitignored files and not
retrying a run whose earlier cleanup failed.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var o gcOpts
			o.DryRun, _ = cmd.Flags().GetBool("dry-run")
			o.IncludeIgnored, _ = cmd.Flags().GetBool("include-ignored")
			return routineGC(cmd.Context(), routineDataDir(), args, o, cmd.OutOrStdout())
		},
	}
	gcCmd.Flags().Bool("dry-run", false, "Print what would be removed and the bytes it frees, change nothing")
	gcCmd.Flags().Bool("include-ignored", false, "Also remove worktrees that hold gitignored files outside .crush (listed by --dry-run)")

	routineCmd.AddCommand(newCmd, listCmd, showCmd, scheduleCmd, runCmd, gcCmd)
	rootCmd.AddCommand(routineCmd)
}

// printRoutineRunJSON là đầu ra của `routine run --json` (Q5): MỘT object. Loại từ chối khi lượt
// không được chạy (thoát 75); báo cáo cuối như report.json (cả ERROR) khi lượt đã có báo cáo; lỗi
// hạ tầng trước khi có báo cáo (repo không còn là repo git, lỗi ghi đĩa, lỗi khoá) là
// {"routine","error"} (thoát 1). Chỉ lỗi dùng sai (thoát 2: cờ, định nghĩa đã đổi) không in gì.
func printRoutineRunJSON(w io.Writer, name string, out routineOutcome, err error) error {
	var v any
	ec, ok := errors.AsType[*exitCodeError](err)
	switch {
	case ok && ec.refused != "":
		v = struct {
			Refused string `json:"refused"`
			Routine string `json:"routine"`
			Detail  string `json:"detail"`
		}{ec.refused, name, ec.err.Error()}
	case out.Report.Routine != "":
		v = out.Report
	case err != nil && !isUsageError(err):
		v = struct {
			Routine string `json:"routine"`
			Error   string `json:"error"`
		}{name, err.Error()}
	default:
		return nil
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// routineExitErr: NO_CHANGE ở routine bảo trì là "không có việc", kết quả hợp lệ của
// lượt theo lịch; ở goal mode là việc chưa được chứng minh nên thoát 1.
func routineExitErr(mode string, r loop.Stop) error {
	if r == loop.StopGreen || (r == loop.StopNoChange && mode == routine.ModeMaintenance) {
		return nil
	}
	return fmt.Errorf("routine stopped: %s", r)
}

// warnAllowlistNotScheduled: allowlist cố ý không vào lịch (biến export thắng tệp
// env, sẽ lấn việc siết allowlist sau này), nên nhắc người dùng chuyển nó vào tệp env.
func warnAllowlistNotScheduled(w io.Writer) {
	if execpolicy.OriginalEnv(execpolicy.AllowedModelsEnv) != "" {
		fmt.Fprintf(w, "warning: %s is set in your shell but is not copied into the schedule; put the allowlist in your dew env file (%s) so scheduled runs use it\n",
			execpolicy.AllowedModelsEnv, execpolicy.EnvFileEnv+" or ~/.config/dew/env")
	}
}

// rejectCwdFlag: `routine run` làm việc trong worktree của nó; --cwd/-c sẽ đưa
// agent thẳng vào checkout của người dùng (ResolveCwd chdir theo cờ này).
func rejectCwdFlag(cmd *cobra.Command) error {
	if cmd.Flags().Changed("cwd") {
		return &usageError{errors.New("routine run works in its own git worktree; --cwd would put the agent in your own checkout")}
	}
	return nil
}

// exportUnattendedEnv đặt biến công tắc cho MỌI tiến trình con của lượt chạy (agent
// chạy `crush loop`, hay CLI khác, từ bash): cờ atomic trong tiến trình không tới
// được chúng. Trả hàm khôi phục env cũ. commit=false đặt rõ "0" để một biến mở
// commit có sẵn trong env của người gọi không lọt sang routine không bật commit.
func exportUnattendedEnv(commit bool) func() {
	commitVal := "0"
	if commit {
		commitVal = "1"
	}
	type saved struct {
		key, val string
		had      bool
	}
	var prev []saved
	for _, kv := range [][2]string{{execpolicy.UnattendedEnv, "1"}, {execpolicy.UnattendedCommitEnv, commitVal}} {
		old, had := os.LookupEnv(kv[0])
		prev = append(prev, saved{kv[0], old, had})
		_ = os.Setenv(kv[0], kv[1])
	}
	return func() {
		for _, p := range prev {
			if p.had {
				_ = os.Setenv(p.key, p.val)
			} else {
				_ = os.Unsetenv(p.key)
			}
		}
	}
}

// Thư mục định nghĩa nằm cạnh cấu hình toàn cục của người dùng; dữ liệu chạy (khoá,
// worktree, báo cáo) nằm cạnh dữ liệu toàn cục.
func routineDefsDir() string { return filepath.Join(filepath.Dir(config.GlobalConfig()), "routines") }
func routineDataDir() string {
	return filepath.Join(filepath.Dir(config.GlobalConfigData()), "routines")
}

func routineLockPath(dataDir, name string) string {
	return filepath.Join(dataDir, name, name+".lock")
}

var safePATH = regexp.MustCompile(`^[A-Za-z0-9/._:+-]+$`)

// schedulePATH là PATH ghi vào lịch: PATH hiện tại nếu sạch, không thì mặc định.
func schedulePATH() string {
	if p := os.Getenv("PATH"); safePATH.MatchString(p) {
		return p
	}
	return "/usr/local/bin:/usr/bin:/bin"
}

// scheduleEnv là các biến mà launcher `dew` đặt để chỉ chỗ cấu hình/dữ liệu,
// cộng tệp env của người dùng: allowlist model đọc từ tệp đó, không chép vào lịch
// (biến export thắng tệp env, siết allowlist sau này sẽ bị bỏ qua). cron và systemd
// chạy với env gần rỗng: thiếu CRUSH_GLOBAL_CONFIG thì lần chạy theo lịch tìm định
// nghĩa ở ~/.config/crush và thoát 2. Đọc env GỐC: `.env`
// của thư mục hiện tại do repo kiểm soát, không được chen vào lịch.
func scheduleEnv() []string {
	var env []string
	for _, k := range []string{"CRUSH_GLOBAL_CONFIG", "CRUSH_GLOBAL_DATA", "CRUSH_CACHE_DIR",
		"CRUSH_DISABLE_METRICS", "CRUSH_DISABLE_PROVIDER_AUTO_UPDATE", "CRUSH_DISABLE_UPDATE_CHECK",
		execpolicy.EnvFileEnv} {
		if v := execpolicy.OriginalEnv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	return env
}

// loadRoutine đọc và kiểm một định nghĩa theo tên (trong defsDir) hoặc đường dẫn tệp.
func loadRoutine(defsDir, nameOrPath string) (routine.Definition, string, error) {
	path := nameOrPath
	byName := !strings.ContainsRune(nameOrPath, filepath.Separator) && !strings.HasSuffix(nameOrPath, ".json")
	if byName {
		path = filepath.Join(defsDir, nameOrPath+".json")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return routine.Definition{}, "", fmt.Errorf("no routine %q (looked for %s); create one with: dew routine new", nameOrPath, path)
		}
		return routine.Definition{}, "", err
	}
	d, sum, err := routine.Parse(data)
	if err != nil {
		return routine.Definition{}, "", fmt.Errorf("%s: %w", path, err)
	}
	if err := d.Validate(); err != nil {
		return routine.Definition{}, "", fmt.Errorf("%s: %w", path, err)
	}
	if want := strings.TrimSuffix(filepath.Base(path), ".json"); want != d.Name {
		return routine.Definition{}, "", fmt.Errorf("%s: the file name must match the definition name %q", path, d.Name)
	}
	return d, sum, nil
}

// writeDefinition kiểm rồi ghi một định nghĩa mới (0600, không ghi đè).
func writeDefinition(defsDir string, d routine.Definition) (string, error) {
	if err := d.Validate(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(defsDir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(defsDir, d.Name+".json")
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("%s already exists; edit it instead of overwriting", path)
		}
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return "", err
	}
	return path, nil
}

type routineListEntry struct {
	Name         string    `json:"name"`
	Mode         string    `json:"mode,omitempty"`
	Scheduled    bool      `json:"scheduled"`
	LastReason   string    `json:"last_reason,omitempty"`
	LastFinished time.Time `json:"last_finished,omitzero"`
	Error        string    `json:"error,omitempty"`
	// Lượt mới nhất, đọc như Home (routine.LatestRuns). State: RUNNING chỉ khi pid còn sống và
	// đúng là `routine run <tên>`; RUNNING mà tiến trình đã chết là ABANDONED, không kiểm được
	// (không pid) là UNKNOWN; báo cáo mất hay hỏng là UNREADABLE; còn lại là lý do dừng.
	State           string `json:"state,omitempty"`
	Model           string `json:"model,omitempty"`
	PID             int    `json:"pid,omitempty"`              // chỉ khi State là RUNNING
	DurationSeconds int64  `json:"duration_seconds,omitempty"` // đang chạy: tới bây giờ
	// Verify: PASS (run GREEN/NO_CHANGE), PASS_NOT_GREEN (verify xanh mà run không đạt), FAIL, NOT_RUN.
	Verify       string `json:"verify,omitempty"`
	ChangedCount int    `json:"changed_count,omitempty"`
	Attempts     int    `json:"attempts,omitempty"`
}

// printRoutineList in mỗi routine một dòng: chế độ, lịch, rồi lượt mới nhất (lastRunText).
func printRoutineList(w io.Writer, entries []routineListEntry) {
	for _, e := range entries {
		if e.Error != "" {
			fmt.Fprintf(w, "%-20s INVALID  %s\n", e.Name, e.Error)
			continue
		}
		sched := "manual"
		if e.Scheduled {
			sched = "scheduled"
		}
		fmt.Fprintf(w, "%-20s %-11s %-9s last: %s\n", e.Name, e.Mode, sched, lastRunText(e))
	}
}

// verifyText là cột verify của `routine list`: chỉ PASS khi run GREEN/NO_CHANGE.
var verifyText = map[string]string{"PASS": "PASS", routine.VerifyPassNotGreen: "PASS (not green)", "FAIL": "FAIL", "NOT_RUN": "not run"}

// lastRunText: trạng thái lượt mới nhất (kèm pid khi đang chạy, giờ xong khi đã xong), model,
// thời lượng, verify, số tệp đổi và số lượt agent.
func lastRunText(e routineListEntry) string {
	switch e.State {
	case "":
		return "never run"
	case "UNREADABLE":
		return "UNREADABLE (its report is missing or broken)"
	}
	s, dur := e.State, "-"
	d := (time.Duration(e.DurationSeconds) * time.Second).String()
	switch {
	case e.PID > 0:
		s, dur = fmt.Sprintf("%s (pid %d)", s, e.PID), "running "+d
	case e.DurationSeconds > 0:
		dur = "took " + d
	}
	if e.PID == 0 && !e.LastFinished.IsZero() {
		s += " " + e.LastFinished.Local().Format("2006-01-02 15:04")
	}
	return fmt.Sprintf("%s  model %s  %s  verify %s  changed %d  attempts %d",
		s, cmp.Or(e.Model, "unknown"), dur, verifyText[e.Verify], e.ChangedCount, e.Attempts)
}

// listRoutines liệt kê mọi *.json trong defsDir; tệp hỏng vẫn có một dòng kèm lỗi
// (không bỏ qua lặng lẽ: routine hỏng im lặng là cách hỏng nguy hiểm nhất).
func listRoutines(defsDir, dataDir string) []routineListEntry {
	files, _ := filepath.Glob(filepath.Join(defsDir, "*.json"))
	sort.Strings(files)
	latest := map[string]routine.RunSummary{}
	for _, s := range routine.LatestRuns(dataDir, time.Now()) {
		latest[s.Routine] = s
	}
	var out []routineListEntry
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".json")
		e := routineListEntry{Name: name}
		d, _, err := loadRoutine(defsDir, name)
		if err != nil {
			e.Error = err.Error()
			out = append(out, e)
			continue
		}
		e.Mode, e.Scheduled = d.Mode, d.Schedule != nil
		if rep, ok := lastReport(dataDir, name); ok {
			e.LastReason, e.LastFinished, e.Model = rep.Reason, rep.Finished, rep.Model
		}
		if r, ok := latest[name]; ok && r.Unreadable {
			e.State = "UNREADABLE"
		} else if ok {
			e.State, e.Verify, e.ChangedCount, e.Attempts = r.Reason, cmp.Or(r.Verify, "NOT_RUN"), r.Changed, r.Attempts
			e.DurationSeconds = int64(r.Duration.Round(time.Second) / time.Second)
			if r.Running {
				e.PID = r.PID
			}
		}
		out = append(out, e)
	}
	return out
}

// lastReport là trạng thái của lượt chạy mới nhất (tên thư mục là dấu thời gian).
// Lượt mới nhất luôn thắng, kể cả khi nó chết dở: không bao giờ lùi về lượt cũ hơn.
func lastReport(dataDir, name string) (routineReport, bool) {
	runs := listRuns(dataDir, name)
	if len(runs) == 0 {
		return routineReport{}, false
	}
	return runs[0], true
}

// listRuns trả mọi lượt chạy, mới nhất trước. Lượt chết giữa chừng (bị kill, OOM,
// máy khởi động lại) thành ABANDONED: hoặc không có báo cáo, hoặc báo cáo còn
// RUNNING mà không ai giữ khoá (khoá tự nhả khi tiến trình chết).
func listRuns(dataDir, name string) []routineReport {
	dirs, _ := filepath.Glob(filepath.Join(dataDir, name, "runs", "*"))
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	var out []routineReport
	for i, dir := range dirs {
		var rep routineReport
		raw, err := os.ReadFile(filepath.Join(dir, "report.json"))
		// Chỉ thử khoá khi cần (routineLockHeld có cửa sổ làm lượt khác thoát 75).
		if err != nil || json.Unmarshal(raw, &rep) != nil {
			// Lượt mới nhất giữ khoá mà chưa có báo cáo là vừa khởi động, chưa chết.
			if i == 0 && routineLockHeld(dataDir, name) {
				out = append(out, routineReport{Routine: name, Reason: "RUNNING",
					Detail: "starting: no report written yet; run dir " + dir})
				continue
			}
			rep = routineReport{Routine: name, Reason: "ABANDONED",
				Detail: "no report: the run died before writing one; run dir " + dir}
			out = append(out, rep)
			continue
		}
		if rep.Reason == "RUNNING" {
			if i > 0 || !routineLockHeld(dataDir, name) {
				rep.Reason = "ABANDONED"
				rep.Detail = "the run stopped before finishing (killed, crashed or the machine restarted); its worktree and branch stay until a later run or `dew routine gc` removes them, and only if they hold no work: " + rep.Worktree
			}
		}
		out = append(out, rep)
	}
	return out
}

// routineLockHeld thử khoá của routine rồi nhả ngay. shortcut: trong vài micro giây
// đó một lượt chạy mới khởi động cùng lúc sẽ thấy khoá bận và thoát 75; nâng lên
// kiểm PID khi điều này gây phiền thật.
func routineLockHeld(dataDir, name string) bool {
	release, err := lock.TryFile(routineLockPath(dataDir, name))
	if err != nil {
		return errors.Is(err, lock.ErrContended)
	}
	release()
	return false
}

// routineOpts là đầu vào một lượt chạy routine.
type routineOpts struct {
	Def       routine.Definition
	Sum       string // sha256 của tệp định nghĩa
	Scheduled bool
	ExpectSHA string
	DataDir   string
	Slots     int // số lượt routine chạy cùng lúc tối đa trên máy; 0 = không giới hạn
	Now       func() time.Time
	// GoCacheMax: cache build Go lớn hơn số byte này thì trim (routineHousekeep); 0 = không.
	GoCacheMax int64
	// MinFree: đĩa chứa DataDir còn ít hơn số byte này thì không chạy (thoát 75); 0 = không kiểm.
	MinFree int64
	// MinMemMB: RAM trống ít hơn số MB này thì không chạy (thoát 75); 0 = không kiểm.
	MinMemMB int
}

// routineModel là provider/model large mà lượt dùng. Không ghim (want rỗng): model large
// hiện hành (current), tức mặc định lúc chạy. Có ghim: model mà tiến trình đã áp (applied,
// override mà lượt agent đầu đặt như -m); chưa áp thì chính chuỗi ghim, nguyên văn: một
// model ID trần có thể có ở nhiều provider, chưa phân giải thì chưa biết provider nào.
func routineModel(current, applied config.SelectedModel, want string) string {
	m := applied
	if want == "" {
		m = current
	}
	if m.Model == "" {
		return want
	}
	return m.Provider + "/" + m.Model
}

// defaultLargeModel là provider/model large mặc định mà một lượt ở repo chạy khi không ghim
// (cấu hình toàn cục cộng crush.json của repo); "" khi chưa cấu hình model nào.
func defaultLargeModel(repo string) (string, error) {
	store, err := config.InitProject(config.ProjectRoot(repo), repo, "", false)
	if err != nil {
		return "", err
	}
	return routineModel(store.Config().Models[config.SelectedModelTypeLarge], config.SelectedModel{}, ""), nil
}

// warnUnpinnedModel: không phân giải được model mặc định lúc tạo thì định nghĩa không ghim
// model, và mỗi lượt chạy model mặc định LÚC CHẠY (20/10: 7 lượt chết vì hạn mức của model
// mặc định mà người dùng tưởng là model khác). now là lý do không phân giải được.
func warnUnpinnedModel(w io.Writer, now, defPath string) {
	fmt.Fprintf(w, "warning: no --model: every run uses whatever the default large model is when it runs (now %s); pin one with --model provider/model, or a \"model\" field in %s\n", now, defPath)
}

const (
	// routineSlotsEnv là số lượt routine (mọi routine cộng lại) được chạy cùng lúc.
	routineSlotsEnv = "DEW_ROUTINE_SLOTS"
	// routineMinMemEnv là RAM trống (MemAvailable, MB) tối thiểu để một lượt bắt đầu.
	routineMinMemEnv = "DEW_ROUTINE_MIN_MEM_MB"
)

// routineSlots đọc routineSlotsEnv; mặc định 2 (đo 21/10: lượt chạy lúc >=4 routine song song
// 0/14 GREEN, lúc <=1,5 song song 18/20).
func routineSlots() (int, error) {
	return envCount(routineSlotsEnv, 2, "the number of routine runs allowed at once (0 = no limit)")
}

// routineMinMemMB đọc routineMinMemEnv; mặc định 0 (không kiểm).
func routineMinMemMB() (int, error) {
	return envCount(routineMinMemEnv, 0, "the MB of available memory a run needs to start (0 = no check)")
}

// envCount đọc số nguyên không âm key: biến export, không có thì tệp env của dew; rỗng thì def.
func envCount(key string, def int, use string) (int, error) {
	v, err := execpolicy.FileEnv(key)
	if err != nil {
		return 0, err
	}
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid %s=%q: use %s", key, v, use)
	}
	return n, nil
}

// checkRoutineMemory: RAM trống dưới minMB thì không chạy (75), như khi hết chỗ. Không đo
// được (không phải Linux) thì vẫn chạy và cảnh báo một dòng, như bước đo đĩa.
func checkRoutineMemory(minMB int, read func() (uint64, error), w io.Writer) error {
	if minMB <= 0 {
		return nil
	}
	if read == nil {
		read = memAvailableKB
	}
	kb, err := read()
	if err != nil {
		fmt.Fprintf(w, "[routine] warning: %s=%d is not checked: %v\n", routineMinMemEnv, minMB, err)
		return nil
	}
	if kb < uint64(minMB)*1024 {
		return &exitCodeError{code: exitCodeBusy, refused: refusedLowMemory, err: fmt.Errorf("only %d MB of memory available (MemAvailable), %s=%d: try again later (0 = no check)", kb/1024, routineMinMemEnv, minMB)}
	}
	return nil
}

// memAvailableKB là MemAvailable (kB) của /proc/meminfo.
func memAvailableKB() (uint64, error) {
	if runtime.GOOS != "linux" {
		return 0, errors.New("available memory is only measured on Linux")
	}
	raw, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	return parseMemAvailable(raw)
}

func parseMemAvailable(raw []byte) (uint64, error) {
	for line := range strings.Lines(string(raw)) {
		if v, ok := strings.CutPrefix(line, "MemAvailable:"); ok {
			return strconv.ParseUint(strings.TrimSuffix(strings.TrimSpace(v), " kB"), 10, 64)
		}
	}
	return 0, errors.New("no MemAvailable line in /proc/meminfo")
}

// takeRoutineSlot giữ một trong n tệp khoá slots/<i>.lock (n=0: không giới hạn). Hết
// chỗ thì thoát 75 như khi khoá của routine bận.
func takeRoutineSlot(dataDir string, n int) (func(), error) {
	if n == 0 {
		return func() {}, nil
	}
	dir := filepath.Join(dataDir, "slots")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	for i := range n {
		release, err := lock.TryFile(filepath.Join(dir, strconv.Itoa(i)+".lock"))
		if !errors.Is(err, lock.ErrContended) {
			return release, err
		}
	}
	return nil, &exitCodeError{code: exitCodeBusy, refused: refusedNoSlot, err: fmt.Errorf("all %d routine slots are in use (%s=%d; 0 = no limit): try again later", n, routineSlotsEnv, n)}
}

// routineDeps là phần dựng agent (trong worktree) và nơi in tiến trình, được tiêm để test.
type routineDeps struct {
	Stderr io.Writer
	// NewAgent trả thêm model(): provider/model large mà lượt dùng (routineModel).
	NewAgent func(worktree string) (agent func(context.Context, string) (loop.RunInfo, error), cleanup func(), model func() string, err error)
	// FreeBytes đo chỗ trống của đĩa chứa một đường dẫn; nil là housekeep.FreeBytes.
	FreeBytes func(path string) (uint64, error)
	// MemAvailableKB đo RAM trống (kB); nil là memAvailableKB.
	MemAvailableKB func() (uint64, error)
	// RunAlive kiểm lượt của routine khác còn sống (closeGhostRuns); nil là routine.RunAlive.
	RunAlive func(pid int, name string) bool
}

// routineReport là báo cáo một lượt chạy, ghi NGOÀI worktree (xoá worktree không
// làm mất bằng chứng).
type routineReport struct {
	Routine          string    `json:"routine"`
	Mode             string    `json:"mode"`
	DefinitionSHA256 string    `json:"definition_sha256"`
	Scheduled        bool      `json:"scheduled"`
	Started          time.Time `json:"started"`
	Finished         time.Time `json:"finished,omitzero"` // không có khi còn RUNNING
	Repo             string    `json:"repo"`
	Worktree         string    `json:"worktree"`
	Branch           string    `json:"branch"`
	BaseSHA          string    `json:"base_sha"`
	Commit           bool      `json:"commit"`
	Reason           string    `json:"reason"`
	Detail           string    `json:"detail,omitempty"`
	Attempts         int       `json:"attempts"`
	SessionIDs       []string  `json:"session_ids,omitempty"`
	VerifyStatus     string    `json:"verify_status,omitempty"`
	VerifyExitCode   *int      `json:"verify_exit_code,omitempty"` // chỉ khi verify đã chạy sau một lượt agent
	Journal          string    `json:"journal,omitempty"`
	// PID của tiến trình đang chạy lượt (ghi cùng báo cáo RUNNING): `ps` kiểm nó còn
	// sống thay vì thử khoá.
	PID int `json:"pid,omitempty"`
	// Worktree đã được gỡ (sạch, không commit); SessionDB là crush.db (transcript) của
	// lượt, chuyển ra cạnh report.json trước khi gỡ.
	WorktreeRemoved time.Time `json:"worktree_removed,omitzero"`
	SessionDB       string    `json:"session_db,omitempty"`
	SessionLogs     string    `json:"session_logs,omitempty"` // .crush/logs của lượt, chuyển cùng crush.db
	// Model là provider/model large mà lượt dùng (đã phân giải), không phải trường "model"
	// của định nghĩa: không ghim model thì là mặc định toàn cục lúc chạy.
	Model string `json:"model,omitempty"`
	// GCError là lỗi của lần dọn trước (worktree được giữ); lượt sau không thử lại để
	// khỏi cảnh báo mỗi lượt, `routine gc` thì thử lại.
	GCError string `json:"gc_error,omitempty"`
	// Cờ "xanh mà chưa chắc đã làm": cổng xanh sẵn từ trước, hoặc cây không đổi.
	BaselinePass bool `json:"baseline_pass,omitempty"`
	Unchanged    bool `json:"unchanged_tree,omitempty"`
	// Tập "việc" của lượt (changedFiles, Q3: cả tệp bị ignore ngoài .crush/ và thư mục dựng lại
	// được), và cờ khi trong đó có test hay tệp mà lệnh verify gọi tới: GREEN nhờ sửa cổng là
	// thành công giả.
	ChangedFiles []string `json:"changed_files,omitempty"`
	TouchedTests bool     `json:"touched_tests,omitempty"`
	TouchedGate  bool     `json:"touched_gate,omitempty"`
}

// testPath nhận đường dẫn test theo quy ước phổ biến (Go, JS/TS, Python, thư mục test).
var testPath = regexp.MustCompile(`(_test\.go$|\.(test|spec)\.[cm]?[jt]sx?$|(^|/)test_[^/]*\.py$|_test\.py$|(^|/)(tests?|__tests__|spec|testdata)/)`)

// gateScript: tệp chạy được mà lệnh verify có thể gọi.
var gateScript = regexp.MustCompile(`\.(sh|bash|py|js|mjs|cjs|ts|rb|pl|mk)$`)

// changedFiles là tập "việc" của worktree (classifyWorktree): tệp đổi trong commit ngoài base
// và chưa commit, tệp chưa theo dõi, tệp bị ignore ngoài .crush/ và thư mục dựng lại được (thư
// mục bị ignore trọn hiện một lần, "out/"), tệp bị giấu khỏi status. Không đọc được thì rỗng.
func changedFiles(ctx context.Context, worktree, base, branch string) []string {
	st, err := classifyWorktree(ctx, worktree, base, branch)
	if err != nil {
		// Agent đổi tên hay xoá nhánh của lượt: rev-list trên nhánh hỏng. Báo cáo vẫn liệt kê việc
		// theo HEAD; bộ dọn thì vẫn giữ worktree khi lỗi (không bỏ qua nhánh mất ở classifyWorktree).
		st, err = classifyWorktree(ctx, worktree, base, "")
	}
	if err != nil {
		return nil
	}
	return st.workFiles()
}

// markChanges điền tệp đã đổi và hai cờ test/cổng vào báo cáo.
func markChanges(rep *routineReport, files []string, verify string) {
	rep.ChangedFiles = files
	for _, f := range files {
		if testPath.MatchString(f) {
			rep.TouchedTests = true
		}
		// Cổng là thứ verify CHẠY (script có tên trong lệnh, Makefile của `make`); tệp
		// được kiểm như `test -f done.txt` là sản phẩm, không phải cổng.
		if strings.Contains(verify, f) && gateScript.MatchString(f) ||
			filepath.Base(f) == "Makefile" && strings.HasPrefix(strings.TrimSpace(verify), "make") {
			rep.TouchedGate = true
		}
	}
}

type routineOutcome struct {
	Result      loop.Result
	Report      routineReport
	RunDir      string
	JournalPath string
}

// unattendedSuffix nói thẳng với agent rằng không ai trả lời được.
const unattendedSuffix = "\n\nThis is an unattended run in a throwaway git worktree (branch %s). " +
	"Nobody can answer questions: decide, and write your assumptions in your final answer. " +
	"Stay inside the current directory. Never push, merge or change remotes."

// executeRoutine chạy một routine MỘT lần: kiểm cổng, giữ khoá, dọn lượt cũ (gcRoutine),
// tạo worktree riêng, bật chế độ không người trực, chạy engine Loop, ghi báo cáo. Không
// bao giờ push, merge hay xoá worktree còn việc chưa duyệt. Lỗi trả về là lỗi dùng
// (usageError), bận (exit 75) hoặc hạ tầng; lý do dừng của loop nằm ở outcome.Result.Reason.
func executeRoutine(ctx context.Context, o routineOpts, d routineDeps) (routineOutcome, error) {
	def := o.Def
	if err := def.Validate(); err != nil {
		return routineOutcome{}, &usageError{err}
	}
	if o.Scheduled && def.Mode != routine.ModeMaintenance {
		return routineOutcome{}, &usageError{fmt.Errorf("routine %q has mode %q: only maintenance routines run from a schedule", def.Name, def.Mode)}
	}
	if o.ExpectSHA != "" && o.ExpectSHA != o.Sum {
		return routineOutcome{}, &usageError{fmt.Errorf("the definition of %q changed since the schedule was installed (expected sha256 %s, the file has %s); review it, then print and install the schedule again", def.Name, o.ExpectSHA, o.Sum)}
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	stderr := d.Stderr
	if stderr == nil {
		stderr = io.Discard
	}

	routineDir := filepath.Join(o.DataDir, def.Name)
	if err := os.MkdirAll(routineDir, 0o700); err != nil {
		return routineOutcome{}, err
	}
	release, err := lock.TryFile(routineLockPath(o.DataDir, def.Name))
	if err != nil {
		if errors.Is(err, lock.ErrContended) {
			return routineOutcome{}, &exitCodeError{code: exitCodeBusy, refused: refusedLockHeld, err: fmt.Errorf("routine %q is already running (lock held)", def.Name)}
		}
		return routineOutcome{}, err
	}
	defer release()

	// Giữ khoá rồi thì dọn lượt cũ của chính routine này; lỗi dọn không chặn lượt chạy.
	st, gcErr := gcRoutine(ctx, o.DataDir, def.Name, gcOpts{}, io.Discard)
	if st.Removed > 0 {
		fmt.Fprintf(stderr, "[routine] %s: removed %d old worktree(s) that held no work\n", def.Name, st.Removed)
	}
	if gcErr != nil {
		fmt.Fprintf(stderr, "[routine] %s: warning: cleaning old runs: %v\n", def.Name, gcErr)
	}
	closeGhostRuns(o.DataDir, def.Name, d.RunAlive, stderr)
	if err := routineHousekeep(ctx, o, d, now, stderr); err != nil {
		return routineOutcome{}, err
	}
	if err := checkRoutineMemory(o.MinMemMB, d.MemAvailableKB, stderr); err != nil {
		return routineOutcome{}, err
	}
	releaseSlot, err := takeRoutineSlot(o.DataDir, o.Slots)
	if err != nil {
		return routineOutcome{}, err
	}
	defer releaseSlot()

	git := func(args ...string) (string, error) {
		c := exec.CommandContext(ctx, "git", append([]string{"-C", def.Repo}, args...)...)
		out, err := c.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	base, err := git("rev-parse", "--verify", "HEAD")
	if err != nil {
		return routineOutcome{}, fmt.Errorf("repo %s is not a git repository with at least one commit: %s", def.Repo, base)
	}

	repairs, maxDur, runTO, verifyTO, err := def.Limits()
	if err != nil {
		return routineOutcome{}, &usageError{err}
	}
	retries, err := def.StreamRetryLimit()
	if err != nil {
		return routineOutcome{}, &usageError{err}
	}

	started := now()
	stamp := started.UTC().Format("20060102-150405")
	branch := "routine/" + def.Name + "-" + stamp
	worktree := filepath.Join(routineDir, "wt", stamp)
	runDir := filepath.Join(routineDir, "runs", stamp)
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return routineOutcome{}, err
	}
	// Báo cáo RUNNING ghi ngay khi có thư mục lượt, trước mọi bước có thể hỏng:
	// lượt bị kill vẫn để lại dấu (ABANDONED khi khoá đã nhả), lượt lỗi sớm có ERROR.
	running := routineReport{
		Routine: def.Name, Mode: def.Mode, DefinitionSHA256: o.Sum, Scheduled: o.Scheduled,
		Started: started, Repo: def.Repo, Worktree: worktree, Branch: branch, BaseSHA: base,
		Commit: def.Commit, Reason: "RUNNING", PID: os.Getpid(),
		Model: def.Model, // chuỗi ghim; agent dựng xong thì thành model() (routineModel)
	}
	// fail ghi đè RUNNING bằng ERROR mang lý do thật; worktree trống vì nó không còn.
	fail := func(err error) (routineOutcome, error) {
		rep := running
		rep.Worktree, rep.Finished, rep.Reason, rep.Detail = "", now(), "ERROR", err.Error()
		out := routineOutcome{Report: rep, RunDir: runDir}
		if werr := writeReport(runDir, rep); werr != nil {
			return out, errors.Join(err, werr)
		}
		return out, err
	}
	if err := writeReport(runDir, running); err != nil {
		return fail(err)
	}
	if err := os.MkdirAll(filepath.Dir(worktree), 0o700); err != nil {
		return fail(err)
	}
	if out, err := git("worktree", "add", "-b", branch, worktree, "HEAD"); err != nil {
		return fail(fmt.Errorf("create the worktree: %s", out))
	}
	fmt.Fprintf(stderr, "[routine] %s: worktree %s on branch %s (base %s)\n", def.Name, worktree, branch, shortSHA(base))

	execpolicy.SetUnattended(true, def.Commit)
	defer execpolicy.SetUnattended(false, false)
	defer exportUnattendedEnv(def.Commit)()

	agent, cleanup, model, err := d.NewAgent(worktree)
	if err != nil {
		// Chưa có việc nào được làm: dọn worktree và nhánh vừa tạo, nhưng để lại báo
		// cáo ERROR để routine hỏng không im lặng.
		_, _ = git("worktree", "remove", "--force", worktree)
		_, _ = git("branch", "-D", branch)
		return fail(err)
	}
	// Model của lượt vào báo cáo RUNNING (lượt bị kill vẫn nói nó chạy model nào) và stderr.
	running.Model = model()
	if err := writeReport(runDir, running); err != nil {
		fmt.Fprintf(stderr, "[routine] %s: warning: %v\n", def.Name, err)
	}
	source := "from the definition"
	if def.Model == "" {
		source = "the default at run time; pin one with \"model\" in the definition"
	}
	fmt.Fprintf(stderr, "[routine] %s: model %s (%s)\n", def.Name, running.Model, source)

	// Engine chỉ giữ chuỗi lỗi của lượt agent; giữ lại lỗi gốc để đọc mã HTTP của provider.
	var agentErr error
	res, loopErr := executeLoop(ctx, loopParams{
		Goal: def.Goal + fmt.Sprintf(unattendedSuffix, branch), Verify: def.Verify, Name: def.Name,
		MaxRepairs: repairs, MaxDuration: maxDur, RunTimeout: runTO, VerifyTimeout: verifyTO,
		StreamRetries: retries, Cwd: worktree, JournalDir: runDir, Model: def.Model,
	}, loopDeps{Agent: func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		info, err := agent(ctx, prompt)
		agentErr = err
		// Lượt đầu áp model ghim: báo cáo RUNNING nói provider/model thay vì chuỗi ghim. Lượt
		// agent chạy đồng bộ, nên lần ghi này luôn trước báo cáo cuối.
		if m := model(); m != running.Model {
			running.Model = m
			if werr := writeReport(runDir, running); werr != nil {
				fmt.Fprintf(stderr, "[routine] %s: warning: %v\n", def.Name, werr)
			}
		}
		return info, err
	}, Stderr: stderr, Now: now})
	usedModel := model() // sau lượt đầu: model ghim đã được áp
	// Đóng session (crush.db) TRƯỚC khi chốt finished: crush.db ghi sau mốc đó là của
	// người mở lại worktree bằng dew, và bước dọn dựa vào điều này để giữ worktree.
	cleanup()

	rep := routineReport{
		Routine: def.Name, Mode: def.Mode, DefinitionSHA256: o.Sum, Scheduled: o.Scheduled,
		Started: started, Finished: now(), Repo: def.Repo, Worktree: worktree, Branch: branch, Model: usedModel,
		BaseSHA: base, Commit: def.Commit, Reason: string(res.Result.Reason), Detail: res.Result.Detail,
		Attempts: res.Result.Attempts, SessionIDs: res.Result.SessionIDs,
		VerifyStatus: string(res.Result.Last.Status),
		Journal:      res.JournalPath, BaselinePass: res.Result.BaselinePass, Unchanged: res.Result.Unchanged,
	}
	if res.Result.Attempts > 0 { // baseline CHƯA ĐO cũng điền Last
		rep.VerifyExitCode = verifyExitCode(res.Result.Last)
	}
	// Lỗi tạm thời hết lần chạy lại và QUOTA: engine đã nêu mã HTTP trong chi tiết.
	if pe, ok := errors.AsType[*fantasy.ProviderError](agentErr); ok && res.Result.Reason == loop.StopRunError && pe.StatusCode != 0 && !strings.Contains(rep.Detail, "(HTTP ") {
		rep.Detail += fmt.Sprintf(" (HTTP %d)", pe.StatusCode)
	}
	if loopErr != nil {
		rep.Reason, rep.Detail = "ERROR", loopErr.Error()
	}
	// ctx đã huỷ (CANCELLED) vẫn phải liệt kê được việc agent đã làm.
	gitCtx, gitCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer gitCancel()
	markChanges(&rep, changedFiles(gitCtx, worktree, base, branch), def.Verify)
	if rep.Reason == string(loop.StopGreen) && (rep.TouchedTests || rep.TouchedGate) {
		fmt.Fprintf(stderr, "[routine] %s: warning: GREEN, but the run changed test files or files the verify command uses; review them before trusting it\n", def.Name)
	}
	out := routineOutcome{Result: res.Result, Report: rep, RunDir: runDir, JournalPath: res.JournalPath}
	if err := writeReport(runDir, rep); err != nil {
		return out, err
	}
	if loopErr != nil {
		return out, loopErr
	}
	fmt.Fprintf(stderr, "[routine] %s: %s. Review: git -C %s diff %s..%s\n[routine] clean up when done: git -C %s worktree remove %s && git -C %s branch -D %s\n",
		def.Name, rep.Reason, def.Repo, shortSHA(base), branch, def.Repo, worktree, def.Repo, branch)
	return out, nil
}

// readReport đọc report.json của một thư mục lượt.
func readReport(dir string) (routineReport, error) {
	var rep routineReport
	raw, err := os.ReadFile(filepath.Join(dir, "report.json"))
	if err == nil {
		err = json.Unmarshal(raw, &rep)
	}
	return rep, err
}

// abandonRun viết lại báo cáo RUNNING của một lượt đã chết thành ABANDONED. Lúc kết
// thúc là lần cuối lượt đó ghi gì: báo cáo, journal hay crush.db trong worktree (lượt
// bị kill giữa một lượt agent ghi crush.db sau journal; không tính thì worktree đó trông
// như có người mở lại).
func abandonRun(dir string, rep routineReport) routineReport {
	rep.Reason = "ABANDONED"
	rep.Detail = "the run stopped before finishing (killed, crashed or the machine restarted)"
	rep.Finished = time.Time{}
	js, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if len(js) > 0 {
		rep.Journal = cmp.Or(rep.Journal, js[len(js)-1])
	}
	for _, p := range append(js, filepath.Join(dir, "report.json")) {
		if info, err := os.Stat(p); err == nil && info.ModTime().After(rep.Finished) {
			rep.Finished = info.ModTime()
		}
	}
	if t := sessionModTime(rep.Worktree); rep.Worktree != "" && t.After(rep.Finished) {
		rep.Finished = t
	}
	return rep
}

// sessionModTime là lần ghi cuối vào crush.db (cả -wal, -shm) trong worktree.
func sessionModTime(wt string) time.Time {
	var last time.Time
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if info, err := os.Stat(filepath.Join(wt, ".crush", "crush.db"+suffix)); err == nil && info.ModTime().After(last) {
			last = info.ModTime()
		}
	}
	return last
}

// closeGhostRuns ghi lại thành ABANDONED lượt MỚI NHẤT của mỗi routine khác còn báo RUNNING
// mà tiến trình chắc chắn đã chết (hoặc pid đã thuộc tiến trình khác), để list/ps/Home không
// thấy run ma cho tới khi chính routine đó chạy lại. Chỉ trên Linux có /proc: nơi khác pid
// không chứng minh được là chết. Báo cáo không pid (bản cũ) để lại cho `routine gc`, lệnh giữ
// khoá của routine đó; không thử khoá ở đây (thử khoá làm một lượt đang khởi động thoát 75).
func closeGhostRuns(dataDir, self string, alive func(int, string) bool, w io.Writer) {
	if _, err := os.Stat("/proc/self/cmdline"); runtime.GOOS != "linux" || err != nil {
		return
	}
	if alive == nil {
		alive = routine.RunAlive
	}
	runsDirs, _ := filepath.Glob(filepath.Join(dataDir, "*", "runs"))
	for _, runs := range runsDirs {
		name := filepath.Base(filepath.Dir(runs))
		dirs, _ := filepath.Glob(filepath.Join(runs, "*"))
		if name == self || len(dirs) == 0 {
			continue
		}
		dir := slices.Max(dirs)
		rep, err := readReport(dir)
		if err != nil || rep.Reason != "RUNNING" || rep.PID <= 0 || alive(rep.PID, name) {
			continue
		}
		// Đọc lại SAU khi biết tiến trình đã chết: nó có thể đã ghi báo cáo cuối rồi thoát giữa
		// lần đọc trên và lúc kiểm; tiến trình chết rồi thì không ghi gì nữa.
		pid := rep.PID
		if rep, err = readReport(dir); err != nil || rep.Reason != "RUNNING" || rep.PID != pid {
			continue
		}
		if err := writeReport(dir, abandonRun(dir, rep)); err != nil {
			fmt.Fprintf(w, "[routine] warning: closing the dead run %s: %v\n", dir, err)
			continue
		}
		fmt.Fprintf(w, "[routine] closed %s as ABANDONED: it was still marked RUNNING but its process (pid %d) is gone\n", dir, rep.PID)
	}
}

// gcOpts là tuỳ chọn một lượt dọn. Giá trị rỗng là lượt dọn tự động đầu `routine run`.
type gcOpts struct {
	DryRun         bool // chỉ in, không đổi gì (kể cả báo cáo)
	IncludeIgnored bool // gỡ cả worktree có tệp bị gitignore ngoài .crush
	Manual         bool // `routine gc`: thử lại lượt dọn hỏng lần trước (gc_error), đo dung lượng
}

// gcStats đếm một lượt dọn.
type gcStats struct {
	Removed int   // worktree đã gỡ (hoặc sẽ gỡ, khi chạy thử)
	Kept    int   // giữ vì còn việc, đang dùng hoặc dọn hỏng (mỗi cái một dòng "keep")
	Freed   int64 // byte
}

// gcRoutine dọn các lượt cũ của một routine. Người gọi PHẢI giữ khoá của routine: khi
// đó báo cáo còn RUNNING là của lượt đã chết (trừ khi pid của nó vẫn là `routine run`
// cùng tên: flock không có tác dụng trên NFS/sshfs), nên được ghi lại thành ABANDONED.
// Worktree+nhánh của một lượt đã xong chỉ bị gỡ khi keepReason rỗng; worktree của lượt
// mới nhất còn worktree luôn được giữ để xem lại. Báo cáo và journal luôn còn. w nhận
// một dòng cho mỗi worktree gỡ hoặc giữ.
func gcRoutine(ctx context.Context, dataDir, name string, o gcOpts, w io.Writer) (gcStats, error) {
	var st gcStats
	var errs []error
	verb := "removed"
	if o.DryRun {
		verb = "would remove"
	}
	dirs, _ := filepath.Glob(filepath.Join(dataDir, name, "runs", "*"))
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	keptNewest := false
	for _, dir := range dirs {
		rep, err := readReport(dir)
		if err != nil {
			continue // chết trước khi ghi báo cáo, tức trước khi có worktree
		}
		if rep.Reason == "RUNNING" {
			// pid của chính tiến trình này là PID tái dùng: báo cáo của lượt này chưa ghi.
			if rep.PID != os.Getpid() && routine.RunAlive(rep.PID, name) {
				fmt.Fprintf(w, "skip %s: its run is still alive (pid %d)\n", dir, rep.PID)
				continue
			}
			rep = abandonRun(dir, rep)
			if !o.DryRun {
				errs = append(errs, writeReport(dir, rep))
			}
		}
		// Worktree đã gỡ mà `branch -D` hỏng (vd nhánh đang checkout để duyệt): thử lại,
		// kể cả lượt tự động (lỗi lặp lại chỉ báo ở `routine gc`), chỉ với nhánh của lượt.
		if !rep.WorktreeRemoved.IsZero() && rep.GCError != "" && !o.DryRun && rep.Branch == "routine/"+name+"-"+filepath.Base(dir) {
			err := deleteRunBranch(ctx, rep)
			rep.GCError = ""
			if err != nil {
				rep.GCError = err.Error()
				if o.Manual {
					errs = append(errs, err)
				}
			}
			errs = append(errs, writeReport(dir, rep))
			continue
		}
		if rep.Worktree == "" || !rep.WorktreeRemoved.IsZero() || (rep.GCError != "" && !o.Manual) {
			continue
		}
		if _, err := os.Stat(rep.Worktree); err != nil {
			continue
		}
		if !keptNewest {
			keptNewest = true
			continue
		}
		why, regen := keepReason(ctx, dataDir, name, dir, rep, o)
		if why != "" {
			st.Kept++
			fmt.Fprintf(w, "keep %s: %s\n", rep.Worktree, why)
			continue
		}
		detail := "branch " + rep.Branch
		var size int64
		if o.Manual { // đi cả cây (node_modules...): chỉ khi người dùng hỏi
			size = housekeep.DirSize(rep.Worktree)
			detail += ", " + humanize.Bytes(uint64(size))
		}
		if regen > 0 {
			detail += fmt.Sprintf(", drops %d regenerable ignored path(s)", regen)
		}
		if !o.DryRun {
			err := removeRunWorktree(ctx, dir, &rep)
			rep.GCError = ""
			if err != nil {
				rep.GCError = err.Error()
				errs = append(errs, fmt.Errorf("%s: %w", rep.Worktree, err))
			}
			errs = append(errs, writeReport(dir, rep))
			if rep.WorktreeRemoved.IsZero() {
				st.Kept++
				fmt.Fprintf(w, "keep %s: %v\n", rep.Worktree, err)
				continue
			}
		}
		st.Removed++
		st.Freed += size
		fmt.Fprintf(w, "%s %s (%s)\n", verb, rep.Worktree, detail)
	}
	return st, errors.Join(errs...)
}

// regenerableDirs là thư mục build/phụ thuộc mà công cụ tự dựng lại: tệp bị gitignore
// nằm dưới chúng không phải việc của agent (đo 21/10: mọi worktree routine thật đều có
// chúng, tính là việc thì không dọn được gì).
var regenerableDirs = map[string]bool{
	"node_modules": true, "dist": true, "build": true, "coverage": true, ".venv": true, "venv": true,
	"__pycache__": true, ".pytest_cache": true, ".mypy_cache": true, ".ruff_cache": true, ".tox": true,
	"target": true, ".next": true, ".nuxt": true, ".turbo": true, ".gradle": true, ".parcel-cache": true,
}

// regenerable: mục bị ignore là một THƯ MỤC (kết thúc bằng "/") mà tên cuối thuộc
// regenerableDirs, vd web/node_modules/. Với --ignored=matching, thư mục bị ignore trọn
// hiện đúng một lần; mục sâu hơn (build/x.env) chứng tỏ build/ không bị ignore trọn (repo
// theo dõi nó), nên tệp đó là việc của agent.
func regenerable(p string) bool {
	return strings.HasSuffix(p, "/") && regenerableDirs[path.Base(strings.TrimSuffix(p, "/"))]
}

const errRunPaths = "the report's worktree or branch is not where this run puts them"

// runPathsMatch: worktree và nhánh trong báo cáo đúng là chỗ lượt stamp của routine name đặt
// chúng (report.json sửa được từ bên ngoài, nên không tin đường dẫn trong đó).
func runPathsMatch(dataDir, name, stamp string, rep routineReport) bool {
	return filepath.Clean(rep.Worktree) == filepath.Join(dataDir, name, "wt", stamp) && rep.Branch == "routine/"+name+"-"+stamp
}

// keepReason trả lý do phải giữ worktree của lượt ở dir (rỗng khi gỡ được) và số mục bị
// gitignore dựng lại được (regenerableDirs) sẽ mất theo. Không đọc được thì giữ.
func keepReason(ctx context.Context, dataDir, name, dir string, rep routineReport, o gcOpts) (string, int) {
	// report.json sửa được từ bên ngoài (lớp chặn không phải sandbox): chỉ gỡ đúng
	// worktree và nhánh mà lượt này tạo ra.
	if !runPathsMatch(dataDir, name, filepath.Base(dir), rep) {
		return errRunPaths, 0
	}
	// Lượt đóng crush.db trước khi chốt finished: ghi sau mốc đó là người đã mở worktree bằng dew.
	// shortcut: lượt của bản cũ đóng DB SAU finished (đo: +6 ms), nên cho một phút; người mở lại
	// worktree trong phút đầu rồi bỏ đó thì không nhận ra (crush.db vẫn được chuyển ra, không mất).
	if sessionModTime(rep.Worktree).After(rep.Finished.Add(time.Minute)) {
		return "its session db changed after the run finished (opened in dew since?)", 0
	}
	// base_sha không phải SHA (sửa tay, hỏng) thì không đếm được commit ngoài base: giữ (fail closed).
	if !commitSHA.MatchString(rep.BaseSHA) {
		return "the report's base_sha is not a commit SHA", 0
	}
	// Chỉ gỡ khi tập "việc" (Q3) rỗng; --include-ignored cho bỏ cả tệp bị ignore ngoài .crush/.
	// Cả HEAD của worktree lẫn nhánh: commit trên HEAD tách rời mất theo worktree.
	st, err := classifyWorktree(ctx, rep.Worktree, rep.BaseSHA, rep.Branch)
	switch {
	case err != nil:
		return "cannot read the worktree: " + err.Error(), 0
	case len(st.Changed) > 0 || len(st.Untracked) > 0:
		return "uncommitted changes", 0
	case len(st.Hidden) > 0:
		return st.Hidden[0] + " is hidden from git status (assume-unchanged or skip-worktree)", 0
	case st.Commits > 0:
		return fmt.Sprintf("%d commit(s) not in base %s", st.Commits, shortSHA(rep.BaseSHA)), 0
	}
	if ignored := st.IgnoredWork; len(ignored) > 0 && !o.IncludeIgnored {
		size := ""
		if o.Manual {
			var n int64
			for _, p := range ignored {
				n += housekeep.DirSize(filepath.Join(rep.Worktree, p))
			}
			size = " (" + humanize.Bytes(uint64(n)) + ")"
		}
		more := ""
		if len(ignored) > 5 {
			more = ", ..."
		}
		return fmt.Sprintf("%d ignored path(s) outside .crush%s: %s%s; pass --include-ignored to remove them too",
			len(ignored), size, strings.Join(ignored[:min(5, len(ignored))], ", "), more), 0
	}
	// Thứ người dùng không xoá được làm `git worktree remove` hỏng giữa chừng, khi git đã xoá .git
	// và bỏ đăng ký worktree (đo 21/10): không lượt dọn nào gỡ tiếp được. Giữ và nói cách xử lý.
	if hint := housekeep.FindStuck(rep.Worktree, currentUID()).Hint(rep.Worktree); hint != "" {
		return hint, 0
	}
	return "", len(st.Regenerable)
}

// removeRunWorktree chuyển dữ liệu dew của lượt (crush.db, -wal, -shm, logs; .crush/.gitignore
// che chúng nên cây vẫn "sạch") ra cạnh report.json, rồi gỡ worktree và nhánh. Mỗi mục
// chuyển riêng, không ghi đè; một mục không chuyển được thì KHÔNG gỡ worktree (mục đó chỉ
// còn ở đó). Không ghi báo cáo: người gọi ghi.
// shortcut: chỉ tìm ở .crush (data_directory mặc định); nâng cấp khi có người đổi nó.
func removeRunWorktree(ctx context.Context, dir string, rep *routineReport) error {
	var errs []error
	for _, n := range []string{"crush.db", "crush.db-wal", "crush.db-shm", "logs"} {
		src, dst := filepath.Join(rep.Worktree, ".crush", n), filepath.Join(dir, n)
		if _, err := os.Lstat(src); errors.Is(err, fs.ErrNotExist) {
			continue // không có, hoặc đã chuyển ở lần trước
		}
		if _, err := os.Lstat(dst); err == nil {
			errs = append(errs, fmt.Errorf("move %s: %s already exists", n, dst))
			continue
		}
		if err := os.Rename(src, dst); err != nil {
			errs = append(errs, err)
			continue
		}
		switch n {
		case "crush.db":
			rep.SessionDB = dst
		case "logs":
			rep.SessionLogs = dst
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	// Lớp kiểm của chính git (không --force) cũng không được để cấu hình giấu tệp mới.
	if _, err := runGit(ctx, rep.Repo, "-c", "status.showUntrackedFiles=all", "worktree", "remove", rep.Worktree); err != nil {
		return err
	}
	rep.WorktreeRemoved = time.Now()
	return deleteRunBranch(ctx, *rep)
}

// deleteRunBranch xoá nhánh của lượt nếu nó còn và không có commit ngoài base (đếm lại:
// lần thử lại có thể sau khi ai đó đã commit lên nhánh).
func deleteRunBranch(ctx context.Context, rep routineReport) error {
	ref := "refs/heads/" + rep.Branch
	if out, err := runGit(ctx, rep.Repo, "for-each-ref", "--format=%(refname)", ref); err != nil || out == "" {
		return err // không còn nhánh, hoặc không đọc được
	}
	if n, err := runGit(ctx, rep.Repo, "rev-list", "--count", ref, "^"+rep.BaseSHA); err != nil {
		return err
	} else if n != "0" {
		return fmt.Errorf("branch %s has %s commit(s) not in base %s; not deleted", rep.Branch, n, shortSHA(rep.BaseSHA))
	}
	_, err := runGit(ctx, rep.Repo, "branch", "-D", rep.Branch)
	return err
}

// runGit chạy git -C dir (không chạy fsmonitor do repo cấu hình) và trả stdout đã cắt khoảng trắng.
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, "git", append([]string{"-C", dir, "-c", "core.fsmonitor=false"}, args...)...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// routineGC là `routine gc`: dọn từng routine (mọi routine có lượt chạy khi names
// rỗng) trong lúc giữ khoá của nó; routine đang chạy thì bỏ qua. In một dòng tổng.
func routineGC(ctx context.Context, dataDir string, names []string, o gcOpts, w io.Writer) error {
	_, err := routineGCStats(ctx, dataDir, names, o, w)
	return err
}

// routineGCStats là routineGC, trả thêm số đếm cộng dồn (cho `dew clean`).
func routineGCStats(ctx context.Context, dataDir string, names []string, o gcOpts, w io.Writer) (gcStats, error) {
	o.Manual = true
	dryRun := o.DryRun
	if len(names) == 0 {
		runs, _ := filepath.Glob(filepath.Join(dataDir, "*", "runs"))
		for _, r := range runs {
			names = append(names, filepath.Base(filepath.Dir(r)))
		}
	}
	var total gcStats
	var errs []error
	skipped := 0
	for _, name := range names {
		if filepath.Base(name) != name || strings.HasPrefix(name, ".") {
			return total, &usageError{fmt.Errorf("%q is not a routine name", name)}
		}
		if _, err := os.Stat(filepath.Join(dataDir, name, "runs")); err != nil {
			errs = append(errs, fmt.Errorf("routine %q has no runs in %s", name, dataDir))
			continue
		}
		release, err := lock.TryFile(routineLockPath(dataDir, name))
		if errors.Is(err, lock.ErrContended) {
			skipped++
			fmt.Fprintf(w, "skip %s: a run holds its lock\n", name)
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		st, err := gcRoutine(ctx, dataDir, name, o, w)
		release()
		errs = append(errs, err)
		total.Removed += st.Removed
		total.Kept += st.Kept
		total.Freed += st.Freed
	}
	verb := "removed"
	if dryRun {
		verb = "would remove"
	}
	fmt.Fprintf(w, "%s %d worktree(s), %s; kept %d; skipped %d running routine(s)\n",
		verb, total.Removed, humanize.Bytes(uint64(total.Freed)), total.Kept, skipped)
	return total, errors.Join(errs...)
}

func shortSHA(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

// writeReport ghi báo cáo qua tệp tạm rồi đổi tên: bị kill giữa lúc ghi thì báo
// cáo cũ (RUNNING) còn nguyên thay vì thành JSON cụt.
func writeReport(dir string, rep routineReport) error {
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	// Tệp tạm riêng mỗi lần ghi (0600): hai tiến trình có thể ghi cùng một báo cáo (closeGhostRuns
	// của routine này, gc của routine kia); tên tạm chung thì một bên đổi tên mất tệp của bên kia.
	f, err := os.CreateTemp(dir, "report.json.*.tmp")
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), filepath.Join(dir, "report.json"))
	}
	if err != nil {
		_ = os.Remove(f.Name())
	}
	return err
}
