package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/event"
	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/loop"
	"github.com/charmbracelet/crush/internal/shell"
	"github.com/charmbracelet/crush/internal/workspace"
	"github.com/spf13/cobra"
	"mvdan.cc/sh/v3/interp"
)

var loopCmd = &cobra.Command{
	Use:   "loop [goal...]",
	Short: "Run the agent in a loop until a verify command passes",
	Long: `Run the agent, then a verify command, and repair until the verify command
exits 0 or a stop condition hits. "Done" is the exit code of --verify, never the
agent's own claim.

Each turn is a fresh session with a self-contained prompt (the goal plus the last
verify output), so a failed attempt does not pollute the next one. The loop stops
with one of:

  GREEN        the verify command exited 0 after an agent turn
  NO_CHANGE    the verify command passed before any change and the working tree
               did not change: nothing proves the agent's work
  MAX_REPAIRS  still failing after --max-repairs repair turns (default 2)
  STALLED      the same failure and an unchanged working tree after a repair turn
  TIMEOUT      --max-duration reached (checked before each agent turn)
  RUN_ERROR    an agent turn failed or exceeded --run-timeout
  CANCELLED    Ctrl+C or SIGTERM
  UNMEASURED   the verify command could not run (not found, syntax error, timed out)

Every step is written to a journal (JSON Lines) BEFORE it happens, so a killed loop
still shows how far it got. The verify command runs in the workspace with your
environment, except that the trusted variables (AGENT_CLI_ALLOWED_MODELS,
AGENT_CLI_DELEGATION, ...) come from the original environment, never from the
repo's .env.

SAFETY: like "run", loop auto-approves the agent's tool requests. It is meant for
work you start and watch. It never commits, pushes or merges.

Exit codes: 0 GREEN, 1 any other stop reason, 2 bad usage.`,
	Example: `
# Make the tests pass, at most 2 repair turns
dew loop --verify "go test ./..." "Fix the failing TestParse"

# Bound the whole run to 2 hours and print one JSON object at the end
dew loop --verify "npm test" --max-duration 2h --json "Make the new endpoint pass its tests"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		f := cmd.Flags()
		p := loopParams{}
		p.Verify, _ = f.GetString("verify")
		p.Name, _ = f.GetString("name")
		p.MaxRepairs, _ = f.GetInt("max-repairs")
		p.MaxDuration, _ = f.GetDuration("max-duration")
		p.RunTimeout, _ = f.GetDuration("run-timeout")
		p.VerifyTimeout, _ = f.GetDuration("verify-timeout")
		p.Model, _ = f.GetString("model")
		p.ReasoningEffort, _ = f.GetString("reasoning-effort")
		p.AsJSON, _ = f.GetBool("json")

		goal, err := MaybePrependStdin(strings.Join(args, " "))
		if err != nil {
			return err
		}
		p.Goal = goal
		if err := validateLoopParams(p); err != nil {
			return err
		}

		cwd, err := ResolveCwd(cmd)
		if err != nil {
			return err
		}
		p.Cwd = cwd
		p.JournalDir = filepath.Join(filepath.Dir(config.GlobalConfigData()), "loops")

		ctx, cancel := signal.NotifyContext(context.Background(), stopSignals()...)
		defer cancel()

		event.SetNonInteractive(true)
		out := io.Writer(os.Stdout)
		if p.AsJSON {
			out = io.Discard
		}
		agent, cleanup, err := newLoopAgent(ctx, cmd, p, out)
		if err != nil {
			return err
		}
		defer cleanup()

		outcome, err := executeLoop(ctx, p, loopDeps{Agent: agent, Stderr: os.Stderr})
		if err != nil {
			return err
		}
		if p.AsJSON {
			if err := printLoopJSON(os.Stdout, outcome); err != nil {
				return err
			}
		}
		return loopExitErr(outcome.Result.Reason)
	},
}

// loopExitErr: mọi lý do dừng trừ GREEN là thoát 1 (kể cả NO_CHANGE: không chứng minh gì).
func loopExitErr(r loop.Stop) error {
	if r != loop.StopGreen {
		return fmt.Errorf("loop stopped: %s", r)
	}
	return nil
}

func init() {
	f := loopCmd.Flags()
	f.String("verify", "", "Command that defines \"done\": the loop is green when it exits 0 (required)")
	f.String("name", "loop", "Name used for the journal file")
	f.Int("max-repairs", 2, "Repair turns after the first agent turn (0 = a single turn)")
	f.Duration("max-duration", 0, "Stop before starting a new agent turn after this long (0 = no limit)")
	f.Duration("run-timeout", 45*time.Minute, "Maximum time of one agent turn")
	f.Duration("verify-timeout", 30*time.Minute, "Maximum time of one verify command run")
	f.StringP("model", "m", "", "Model to use. Accepts 'model' or 'provider/model'")
	f.String("reasoning-effort", "", "Reasoning effort for the model (e.g. low, medium, high)")
	f.Bool("json", false, "Print one JSON object with the stop reason, attempts and journal path to stdout")
	addYOLOFlag(loopCmd)
	loopCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return &usageError{err} })
	rootCmd.AddCommand(loopCmd)
}

// loopParams là mọi tham số của một lần chạy loop.
type loopParams struct {
	Goal, Verify, Name        string
	MaxRepairs                int
	MaxDuration               time.Duration
	RunTimeout, VerifyTimeout time.Duration
	Cwd, JournalDir           string
	Model, ReasoningEffort    string
	AsJSON                    bool
}

// loopDeps là phần chạy agent và nơi in tiến trình, được tiêm để test.
type loopDeps struct {
	Agent  func(ctx context.Context, prompt string) (loop.RunInfo, error)
	Stderr io.Writer
	Now    func() time.Time
}

type loopOutcome struct {
	Result      loop.Result
	JournalPath string
}

// validateLoopParams là cổng L0: không nhận việc chưa phát biểu được "xong".
func validateLoopParams(p loopParams) error {
	switch {
	case strings.TrimSpace(p.Goal) == "":
		return &usageError{errors.New("no goal provided (pass it as arguments or on stdin)")}
	case strings.TrimSpace(p.Verify) == "":
		return &usageError{errors.New("loop needs --verify: the command that defines \"done\" (exit 0 = done)")}
	case p.MaxRepairs < 0:
		return &usageError{errors.New("--max-repairs must be 0 or more")}
	}
	return nil
}

// executeLoop dựng journal và engine rồi chạy. Lỗi trả về chỉ dành cho hỏng
// hạ tầng (không mở được journal...); lý do dừng nằm ở outcome.Result.Reason.
func executeLoop(ctx context.Context, p loopParams, d loopDeps) (loopOutcome, error) {
	if err := validateLoopParams(p); err != nil {
		return loopOutcome{}, err
	}
	now := d.Now
	if now == nil {
		now = time.Now
	}
	stderr := d.Stderr
	if stderr == nil {
		stderr = io.Discard
	}
	path := filepath.Join(p.JournalDir, fmt.Sprintf("%s-%s.jsonl", safeName(p.Name), now().UTC().Format("20060102-150405")))
	journal, err := loop.OpenFileJournal(path)
	if err != nil {
		return loopOutcome{}, err
	}
	defer journal.Close()
	fmt.Fprintf(stderr, "[loop] journal: %s\n", path)

	agent := func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		rctx := ctx
		if p.RunTimeout > 0 {
			var cancel context.CancelFunc
			rctx, cancel = context.WithTimeout(ctx, p.RunTimeout)
			defer cancel()
		}
		info, err := d.Agent(rctx, prompt)
		if err != nil && ctx.Err() == nil && errors.Is(rctx.Err(), context.DeadlineExceeded) {
			err = fmt.Errorf("agent turn timed out after %s", p.RunTimeout)
		}
		return info, err
	}
	// Agent làm việc ở gốc dự án (cùng ProjectRoot với workspace), verify chạy ở
	// thư mục mở: prompt nêu thư mục verify khi hai nơi khác nhau, và dấu vân
	// tay tính trên cả repo để thấy tệp agent đổi ngoài thư mục mở.
	root := config.ProjectRoot(p.Cwd)
	verifyDir := ""
	phys, err := filepath.EvalSymlinks(p.Cwd)
	if err != nil {
		phys = p.Cwd
	}
	if rel, err := filepath.Rel(root, phys); err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		verifyDir = filepath.ToSlash(rel) + "/"
	}
	eng := &loop.Engine{
		Spec: loop.Spec{
			Name: p.Name, Goal: p.Goal, Verify: p.Verify, VerifyDir: verifyDir,
			MaxRepairs: p.MaxRepairs, MaxDuration: p.MaxDuration,
		},
		Agent:       agent,
		Verify:      shellVerify(p.Cwd, p.VerifyTimeout),
		Fingerprint: gitFingerprint(root),
		Journal:     progressJournal{inner: journal, w: stderr},
		Now:         now,
	}
	res, err := eng.Run(ctx)
	return loopOutcome{Result: res, JournalPath: path}, err
}

var unsafeNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func safeName(s string) string {
	s = strings.Trim(unsafeNameChars.ReplaceAllString(s, "_"), "_")
	if s == "" {
		return "loop"
	}
	return s
}

// maxVerifyOutput giới hạn bộ nhớ giữ cho output verify (giữ phần đuôi, nơi lỗi nằm).
const maxVerifyOutput = 1 << 20

type tailBuffer struct{ b []byte }

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.b = append(t.b, p...)
	if len(t.b) > maxVerifyOutput {
		t.b = t.b[len(t.b)-maxVerifyOutput:]
	}
	return len(p), nil
}

// shellVerify chạy lệnh verify bằng đúng trình thông dịch mà hook dùng, trong
// thư mục workspace với env của người dùng. Ba kết quả: thoát 0 = PASS; thoát khác
// 0 = FAIL; không chạy được (không tìm thấy lệnh, lỗi cú pháp, quá hạn, rỗng) =
// UNMEASURED: không biết đỏ hay xanh thì không được tính là xanh.
func shellVerify(cwd string, timeout time.Duration) func(ctx context.Context, cmd string) (loop.VerifyResult, error) {
	return func(ctx context.Context, cmd string) (loop.VerifyResult, error) {
		if strings.TrimSpace(cmd) == "" {
			return loop.VerifyResult{Status: loop.StatusUnmeasured, Output: "empty verify command"}, nil
		}
		vctx := ctx
		if timeout > 0 {
			var cancel context.CancelFunc
			vctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		var buf tailBuffer
		start := time.Now()
		// pipefail: đuôi `| cat`/`| tee log` không được che mã lỗi của lệnh trước nó.
		// ChildEnv: verify có thể chạy dew; biến tin cậy không được lấy từ `.env` của repo.
		err := shell.Run(vctx, shell.RunOptions{Command: "set -o pipefail\n" + cmd, Cwd: cwd, Env: execpolicy.ChildEnv(os.Environ()), Stdout: &buf, Stderr: &buf})
		res := loop.VerifyResult{Output: string(buf.b), Duration: time.Since(start)}
		switch exit, isExit := errors.AsType[interp.ExitStatus](err); {
		case err == nil:
			res.Status = loop.StatusPass
		case ctx.Err() == nil && errors.Is(vctx.Err(), context.DeadlineExceeded):
			res.Status = loop.StatusUnmeasured
			res.Output += fmt.Sprintf("\nverify command timed out after %s", timeout)
		case isExit && exit != 126 && exit != 127:
			res.Status, res.ExitCode = loop.StatusFail, int(exit)
			if exit == 141 {
				// Vẫn FAIL (pipefail là cố ý), nhưng nói rõ: thường là đuôi đọc rồi thoát sớm.
				res.Output += "\nexit 141 = SIGPIPE: a reader like head or grep -q exited early; with pipefail that counts as failure; drop the early-exit reader"
			}
		case isExit:
			res.Status, res.ExitCode = loop.StatusUnmeasured, int(exit)
		default:
			res.Status = loop.StatusUnmeasured
			res.Output += "\n" + err.Error()
		}
		return res, nil
	}
}

// gitFingerprint băm trạng thái cây làm việc: HEAD, diff của tệp đã theo dõi, và
// nội dung tệp chưa theo dõi. Hai lượt cho cùng dấu vân tay nghĩa là agent không
// đổi gì. Không phải repo git (hoặc chưa có commit) thì không biết cây có đổi
// không: trả giá trị luôn khác để không dừng oan vì STALLED.
func gitFingerprint(dir string) func(ctx context.Context) (string, error) {
	// Cấu hình của repo có thể do agent vừa ghi (core.fsmonitor, diff.external): tắt
	// mọi chương trình do cấu hình chỉ định để dấu vân tay không chạy mã của nó.
	git := func(ctx context.Context, args ...string) ([]byte, error) {
		c := exec.CommandContext(ctx, "git", append([]string{"-c", "core.fsmonitor=false"}, args...)...)
		c.Dir = dir
		return c.Output()
	}
	return func(ctx context.Context) (string, error) {
		head, err := git(ctx, "rev-parse", "HEAD")
		if err != nil {
			return "unknown-" + strconv.FormatInt(time.Now().UnixNano(), 10), nil
		}
		h := sha256.New()
		h.Write(head)
		if diff, err := git(ctx, "diff", "--no-ext-diff", "--no-textconv", "HEAD"); err == nil {
			h.Write(diff)
		}
		if names, err := git(ctx, "ls-files", "--others", "--exclude-standard", "-z"); err == nil {
			for _, name := range bytes.Split(names, []byte{0}) {
				if len(name) == 0 {
					continue
				}
				h.Write(name)
				if f, err := os.Open(filepath.Join(dir, string(name))); err == nil {
					_, _ = io.CopyN(h, f, maxVerifyOutput)
					f.Close()
				}
			}
		}
		return hex.EncodeToString(h.Sum(nil))[:16], nil
	}
}

// progressJournal in một dòng người đọc được cho mỗi sự kiện quan trọng rồi
// chuyển tiếp cho journal thật.
type progressJournal struct {
	inner loop.Journal
	w     io.Writer
}

func (p progressJournal) Append(e loop.Event) error {
	if err := p.inner.Append(e); err != nil {
		return err
	}
	switch e.Kind {
	case "baseline":
		fmt.Fprintf(p.w, "[loop] baseline: verify %s (exit %d)\n", e.Status, e.ExitCode)
		if e.Note != "" {
			fmt.Fprintf(p.w, "[loop] warning: %s\n", e.Note)
		}
	case "attempt_begin":
		fmt.Fprintf(p.w, "[loop] attempt %d: running the agent\n", e.Attempt)
	case "verify":
		fmt.Fprintf(p.w, "[loop] attempt %d: verify %s (exit %d)\n", e.Attempt, e.Status, e.ExitCode)
	case "stop":
		if e.Note != "" {
			fmt.Fprintf(p.w, "[loop] stop: %s (%s)\n", e.Reason, e.Note)
		} else {
			fmt.Fprintf(p.w, "[loop] stop: %s\n", e.Reason)
		}
	}
	return nil
}

type loopJSON struct {
	Reason       string   `json:"reason"`
	Detail       string   `json:"detail,omitempty"`
	Attempts     int      `json:"attempts"`
	SessionIDs   []string `json:"session_ids,omitempty"`
	VerifyStatus string   `json:"verify_status"`
	VerifyExit   int      `json:"verify_exit_code"`
	Journal      string   `json:"journal"`
}

func printLoopJSON(w io.Writer, o loopOutcome) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(loopJSON{
		Reason: string(o.Result.Reason), Detail: o.Result.Detail, Attempts: o.Result.Attempts,
		SessionIDs: o.Result.SessionIDs, VerifyStatus: string(o.Result.Last.Status),
		VerifyExit: o.Result.Last.ExitCode, Journal: o.JournalPath,
	})
}

// newLoopAgent dựng hàm chạy MỘT lượt agent trong session mới, qua server của dự
// án (mặc định) hoặc in-process, đúng như `run`. out nhận phần văn bản của agent.
func newLoopAgent(ctx context.Context, cmd *cobra.Command, p loopParams, out io.Writer) (func(context.Context, string) (loop.RunInfo, error), func(), error) {
	if useClientServer() {
		c, ws, cleanup, err := connectToServer(cmd)
		if err != nil {
			return nil, nil, err
		}
		event.AppInitialized()
		if !ws.Config.IsConfigured() {
			cleanup()
			return nil, nil, fmt.Errorf("no providers configured - please run 'dew' to set up a provider interactively")
		}
		if err := ensureAgent(ctx, c, ws.ID); err != nil {
			cleanup()
			return nil, nil, err
		}
		return func(ctx context.Context, prompt string) (loop.RunInfo, error) {
			var (
				info loop.RunInfo
				text bytes.Buffer
			)
			err := runNonInteractive(ctx, c, ws, io.MultiWriter(out, &text),
				func(id string) { info.SessionID = id }, nil,
				prompt, p.Model, "", p.ReasoningEffort, true, "", false)
			info.Text = text.String()
			return info, err
		}, cleanup, nil
	}

	return newLocalLoopAgent(cmd, p, out)
}

// newLocalLoopAgent chạy agent in-process (không qua server), trong thư mục hiện
// tại của tiến trình. Routine luôn dùng đường này: lớp chặn không người trực là
// trạng thái của chính tiến trình, nên không thể bị một server dùng chung bỏ qua.
func newLocalLoopAgent(cmd *cobra.Command, p loopParams, out io.Writer) (func(context.Context, string) (loop.RunInfo, error), func(), error) {
	ws, cleanup, err := setupLocalWorkspace(cmd)
	if err != nil {
		return nil, nil, err
	}
	event.AppInitialized()
	if !ws.Config().IsConfigured() {
		cleanup()
		return nil, nil, fmt.Errorf("no providers configured - please run 'dew' to set up a provider interactively")
	}
	appWs := ws.(*workspace.AppWorkspace)
	return func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		var (
			info loop.RunInfo
			text bytes.Buffer
		)
		err := appWs.App().RunNonInteractive(ctx, io.MultiWriter(out, &text),
			func(id string) { info.SessionID = id },
			prompt, p.Model, "", p.ReasoningEffort, true, "", false)
		info.Text = text.String()
		return info, err
	}, cleanup, nil
}
