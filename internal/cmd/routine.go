package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/event"
	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/lock"
	"github.com/charmbracelet/crush/internal/loop"
	"github.com/charmbracelet/crush/internal/routine"
	"github.com/spf13/cobra"
)

// exitCodeBusy là EX_TEMPFAIL: có một lượt chạy khác của routine đang giữ khoá.
const exitCodeBusy = 75

// exitCodeError mang một mã thoát riêng (ngoài 1 và 2).
type exitCodeError struct {
	code int
	err  error
}

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
"commit": true, never pushes, never merges: reviewing and merging stay yours.

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
			if d.Repo != "" {
				abs, err := filepath.Abs(d.Repo)
				if err != nil {
					return &usageError{err}
				}
				d.Repo = abs
			}
			path, err := writeDefinition(routineDefsDir(), d)
			if err != nil {
				return &usageError{err}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\nrun it by hand:  dew routine run %s\n", path, d.Name)
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
	newCmd.Flags().String("model", "", "Model to use")
	newCmd.Flags().Int("max-repairs", routine.DefaultMaxRepairs, "Repair turns after the first agent turn (0-5)")
	newCmd.Flags().Bool("commit", false, "Allow git commit on the routine/* branch (never push)")
	newCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return &usageError{err} })

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List routine definitions and their last run",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries := listRoutines(routineDefsDir(), routineDataDir())
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(entries)
			}
			if len(entries) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "no routines in %s\n", routineDefsDir())
				return nil
			}
			for _, e := range entries {
				switch {
				case e.Error != "":
					fmt.Fprintf(cmd.OutOrStdout(), "%-20s INVALID  %s\n", e.Name, e.Error)
				default:
					last := "never run"
					if e.LastReason != "" {
						last = e.LastReason + " " + e.LastFinished.Local().Format("2006-01-02 15:04")
					}
					sched := "manual"
					if e.Scheduled {
						sched = "scheduled"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-11s %-9s last: %s\n", e.Name, e.Mode, sched, last)
				}
			}
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
routine holds the lock.`,
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
			opts.Scheduled, _ = cmd.Flags().GetBool("scheduled")
			opts.ExpectSHA, _ = cmd.Flags().GetString("expect-sha")

			ctx, cancel := signal.NotifyContext(context.Background(), stopSignals()...)
			defer cancel()
			event.SetNonInteractive(true)

			deps := routineDeps{
				Stderr: os.Stderr,
				NewAgent: func(worktree string) (func(context.Context, string) (loop.RunInfo, error), func(), error) {
					if err := os.Chdir(worktree); err != nil {
						return nil, nil, fmt.Errorf("enter the worktree: %w", err)
					}
					return newLocalLoopAgent(cmd, loopParams{Model: d.Model}, io.Discard)
				},
			}
			out, err := executeRoutine(ctx, opts, deps)
			if err != nil {
				return err
			}
			return routineExitErr(d.Mode, out.Result.Reason)
		},
	}
	runCmd.Flags().Bool("scheduled", false, "Mark the run as started by a schedule (refused for goal routines)")
	runCmd.Flags().String("expect-sha", "", "Refuse to run unless the definition file has this sha256 (set by the printed schedule)")
	runCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return &usageError{err} })

	routineCmd.AddCommand(newCmd, listCmd, showCmd, scheduleCmd, runCmd)
	rootCmd.AddCommand(routineCmd)
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
			execpolicy.AllowedModelsEnv, execpolicy.EnvFileEnv+" or ~/.config/agent-cli/env")
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
	LastFinished time.Time `json:"last_finished,omitempty"`
	Error        string    `json:"error,omitempty"`
}

// listRoutines liệt kê mọi *.json trong defsDir; tệp hỏng vẫn có một dòng kèm lỗi
// (không bỏ qua lặng lẽ: routine hỏng im lặng là cách hỏng nguy hiểm nhất).
func listRoutines(defsDir, dataDir string) []routineListEntry {
	files, _ := filepath.Glob(filepath.Join(defsDir, "*.json"))
	sort.Strings(files)
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
			e.LastReason, e.LastFinished = rep.Reason, rep.Finished
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
				rep.Detail = "the run stopped before finishing (killed, crashed or the machine restarted); its worktree and branch are kept: " + rep.Worktree
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
	Now       func() time.Time
}

// routineDeps là phần dựng agent (trong worktree) và nơi in tiến trình, được tiêm để test.
type routineDeps struct {
	Stderr   io.Writer
	NewAgent func(worktree string) (agent func(context.Context, string) (loop.RunInfo, error), cleanup func(), err error)
}

// routineReport là báo cáo một lượt chạy, ghi NGOÀI worktree (xoá worktree không
// làm mất bằng chứng).
type routineReport struct {
	Routine          string    `json:"routine"`
	Mode             string    `json:"mode"`
	DefinitionSHA256 string    `json:"definition_sha256"`
	Scheduled        bool      `json:"scheduled"`
	Started          time.Time `json:"started"`
	Finished         time.Time `json:"finished"`
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
	VerifyExitCode   int       `json:"verify_exit_code"`
	Journal          string    `json:"journal,omitempty"`
	// Cờ "xanh mà chưa chắc đã làm": cổng xanh sẵn từ trước, hoặc cây không đổi.
	BaselinePass bool `json:"baseline_pass,omitempty"`
	Unchanged    bool `json:"unchanged_tree,omitempty"`
	// Tệp đổi so với base (đã theo dõi + chưa theo dõi), và cờ khi trong đó có
	// test hay tệp mà lệnh verify gọi tới: GREEN nhờ sửa cổng là thành công giả.
	ChangedFiles []string `json:"changed_files,omitempty"`
	TouchedTests bool     `json:"touched_tests,omitempty"`
	TouchedGate  bool     `json:"touched_gate,omitempty"`
}

// testPath nhận đường dẫn test theo quy ước phổ biến (Go, JS/TS, Python, thư mục test).
var testPath = regexp.MustCompile(`(_test\.go$|\.(test|spec)\.[cm]?[jt]sx?$|(^|/)test_[^/]*\.py$|_test\.py$|(^|/)(tests?|__tests__|spec|testdata)/)`)

// gateScript: tệp chạy được mà lệnh verify có thể gọi.
var gateScript = regexp.MustCompile(`\.(sh|bash|py|js|mjs|cjs|ts|rb|pl|mk)$`)

// changedFiles liệt kê tệp trong worktree khác với base: diff (cả commit lẫn chưa
// commit) cộng tệp chưa theo dõi. Không chạy chương trình do cấu hình repo chỉ định.
func changedFiles(ctx context.Context, worktree, base string) []string {
	run := func(args ...string) []string {
		c := exec.CommandContext(ctx, "git", append([]string{"-C", worktree, "-c", "core.fsmonitor=false"}, args...)...)
		out, err := c.Output()
		if err != nil {
			return nil
		}
		return strings.Fields(strings.TrimSpace(string(out)))
	}
	files := append(run("diff", "--no-ext-diff", "--no-textconv", "--name-only", base),
		run("ls-files", "--others", "--exclude-standard")...)
	sort.Strings(files)
	return slices.Compact(files)
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

// executeRoutine chạy một routine MỘT lần: kiểm cổng, giữ khoá, tạo worktree riêng,
// bật chế độ không người trực, chạy engine Loop, ghi báo cáo. Không bao giờ push,
// merge hay xoá worktree. Lỗi trả về là lỗi dùng (usageError), bận (exit 75) hoặc
// hạ tầng; lý do dừng của loop nằm ở outcome.Result.Reason.
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
			return routineOutcome{}, &exitCodeError{code: exitCodeBusy, err: fmt.Errorf("routine %q is already running (lock held)", def.Name)}
		}
		return routineOutcome{}, err
	}
	defer release()

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
		Commit: def.Commit, Reason: "RUNNING",
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

	agent, cleanup, err := d.NewAgent(worktree)
	if err != nil {
		// Chưa có việc nào được làm: dọn worktree và nhánh vừa tạo, nhưng để lại báo
		// cáo ERROR để routine hỏng không im lặng. (Không có bộ dọn tự động nào khác:
		// worktree của lượt đã chạy được giữ lại để duyệt.)
		_, _ = git("worktree", "remove", "--force", worktree)
		_, _ = git("branch", "-D", branch)
		return fail(err)
	}
	defer cleanup()

	res, loopErr := executeLoop(ctx, loopParams{
		Goal: def.Goal + fmt.Sprintf(unattendedSuffix, branch), Verify: def.Verify, Name: def.Name,
		MaxRepairs: repairs, MaxDuration: maxDur, RunTimeout: runTO, VerifyTimeout: verifyTO,
		Cwd: worktree, JournalDir: runDir, Model: def.Model,
	}, loopDeps{Agent: agent, Stderr: stderr, Now: now})

	rep := routineReport{
		Routine: def.Name, Mode: def.Mode, DefinitionSHA256: o.Sum, Scheduled: o.Scheduled,
		Started: started, Finished: now(), Repo: def.Repo, Worktree: worktree, Branch: branch,
		BaseSHA: base, Commit: def.Commit, Reason: string(res.Result.Reason), Detail: res.Result.Detail,
		Attempts: res.Result.Attempts, SessionIDs: res.Result.SessionIDs,
		VerifyStatus: string(res.Result.Last.Status), VerifyExitCode: res.Result.Last.ExitCode,
		Journal: res.JournalPath, BaselinePass: res.Result.BaselinePass, Unchanged: res.Result.Unchanged,
	}
	if loopErr != nil {
		rep.Reason, rep.Detail = "ERROR", loopErr.Error()
	}
	markChanges(&rep, changedFiles(ctx, worktree, base), def.Verify)
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
	tmp := filepath.Join(dir, "report.json.tmp")
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "report.json"))
}
