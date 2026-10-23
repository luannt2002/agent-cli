package cmd

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/crush/internal/agent"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/event"
	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/charmbracelet/crush/internal/loop"
	"github.com/charmbracelet/crush/internal/routine"
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
verify output), so a failed attempt does not pollute the next one. A turn cut off
by a temporary provider error (5xx, 429, a stalled stream, a dropped connection)
is retried in the same session, at most --stream-retries times in a row (default
2; after 30s, then every 60s), without using up a repair turn; "attempts" in the
JSON and the journal's stop event count every agent turn, retries included. A
quota or usage-limit error (HTTP 402, or a 429/403 whose message says usage limit,
quota or billing) is never retried. When a turn is cut off (--run-timeout, or a
temporary error after the retries) the verify command still runs once on what the
agent left, and its status and exit code are reported. After a --run-timeout cut,
a failing verify with repair turns left starts the next repair turn (told that the
previous turn was cut off); a passing verify, or no repair turn left, stops TIMEOUT.
The verify right after a cut never makes the run GREEN, even when it passes: GREEN
only follows an agent turn that ran to its end. The loop stops with one of:

  GREEN         the verify command exited 0 after an agent turn that ran to its end
  NO_CHANGE     the verify command passed before any change and the working tree
                did not change: nothing proves the agent's work
  MAX_REPAIRS   still failing after --max-repairs repair turns (default 2)
  STALLED       the same failure and an unchanged working tree after a repair turn
  TIMEOUT       --max-duration reached (checked before each agent turn), or a turn
                cut off by --run-timeout whose verify then passed (review it) or
                that had no repair turn left (the detail says which)
  STREAM_STALL  the LLM stream sent no data (request_timeout), after the retries
  QUOTA         the provider reported a quota or usage limit (not retried)
  RUN_ERROR     an agent turn failed (a temporary error: after the retries)
  CANCELLED     Ctrl+C or SIGTERM
  UNMEASURED    the verify command could not run (not found, syntax error, timed out)

Every step is written to a journal (JSON Lines) BEFORE it happens, so a killed loop
still shows how far it got. The verify command runs in the workspace with your
environment, except that the trusted variables (DEW_ALLOWED_MODELS,
DEW_DELEGATION, ...) come from the original environment, never from the
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
		p.StreamRetries, _ = f.GetInt("stream-retries")
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
		runAgent, cleanup, err := newLoopAgent(ctx, cmd, p, out)
		if err != nil {
			return err
		}
		defer cleanup()

		outcome, err := executeLoop(ctx, p, loopDeps{Agent: runAgent, Stderr: os.Stderr})
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
	f.Int("stream-retries", routine.DefaultStreamRetries, "Retries in a row of a turn cut off by a temporary provider error (0-5, 0 = no retry)")
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
	MaxRepairs, StreamRetries int
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
	case p.StreamRetries < 0 || p.StreamRetries > routine.MaxStreamRetries:
		return &usageError{fmt.Errorf("--stream-retries must be between 0 and %d", routine.MaxStreamRetries)}
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

	runTurn := func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		rctx := ctx
		if p.RunTimeout > 0 {
			var cancel context.CancelFunc
			rctx, cancel = context.WithTimeout(ctx, p.RunTimeout)
			defer cancel()
		}
		info, err := d.Agent(rctx, prompt)
		// Lượt không dừng được (settleTurn) phải dừng cả loop, không được thành lượt bị cắt để chạy tiếp.
		if err != nil && !errors.Is(err, errTurnNotSettled) && ctx.Err() == nil && errors.Is(rctx.Err(), context.DeadlineExceeded) {
			err = fmt.Errorf("%w after %s (per-turn run timeout)", loop.ErrTurnTimeout, p.RunTimeout)
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
			MaxRepairs: p.MaxRepairs, MaxDuration: p.MaxDuration, StreamRetries: p.StreamRetries,
		},
		Agent:       runTurn,
		Verify:      shellVerify(p.Cwd, p.VerifyTimeout),
		Fingerprint: gitFingerprint(root),
		Journal:     progressJournal{inner: journal, w: stderr},
		Now:         now,
		Transient:   agent.TransientError,
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

// gitFingerprint băm trạng thái cây làm việc theo bộ phân loại (classifyWorktree, Q3): HEAD, diff
// của tệp đã theo dõi, tên và nội dung tệp chưa theo dõi và tệp bị giấu khỏi status, và CHỈ TÊN
// của mục bị ignore là việc (kể cả tên tệp bên trong thư mục bị ignore trọn): verify hay ghi lại
// artefact bị ignore (junit.xml, cover.out) với nội dung mới mỗi lần, băm nội dung thì STALLED
// không bao giờ bắt được; agent tạo tệp bị ignore mới thì có tên mới. Hai lượt cho cùng dấu vân
// tay nghĩa là agent không đổi gì. Không phải repo git (hoặc chưa có commit, hoặc git hỏng) thì
// không biết cây có đổi không: trả giá trị luôn khác để không dừng oan vì STALLED.
func gitFingerprint(dir string) func(ctx context.Context) (string, error) {
	// Cấu hình của repo có thể do agent vừa ghi: tắt fsmonitor, hook và chương trình diff ngoài
	// (--no-ext-diff, --no-textconv). Clean filter do repo khai vẫn chạy (xem classifyWorktree).
	git := func(ctx context.Context, args ...string) ([]byte, error) {
		c := gitCmd(ctx, args...)
		c.Dir = dir
		return c.Output()
	}
	return func(ctx context.Context) (string, error) {
		unknown := "unknown-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		head, err := git(ctx, "rev-parse", "HEAD")
		if err != nil {
			return unknown, nil
		}
		st, err := classifyWorktree(ctx, dir, "", "")
		if err != nil {
			return unknown, nil
		}
		diff, err := git(ctx, "-C", st.Root, "diff", "--no-ext-diff", "--no-textconv", "HEAD")
		if err != nil {
			return unknown, nil
		}
		h := sha256.New()
		h.Write(head)
		h.Write(diff)
		for _, class := range []struct {
			tag     string
			paths   []string
			content bool
		}{{"??", st.Untracked, true}, {"!!", st.IgnoredWork, false}, {"h", st.Hidden, true}} {
			for _, p := range class.paths {
				hashTree(h, st.Root, class.tag, p, class.content)
			}
		}
		return hex.EncodeToString(h.Sum(nil))[:16], nil
	}
}

// hashTree băm tên một mục "việc", và nội dung khi content: mỗi tệp tối đa maxVerifyOutput byte,
// chỉ tệp thường (FIFO hay thiết bị làm treo), symlink băm đích. Chỉ băm tên (mục bị ignore) thì đi
// cả thư mục, bỏ qua .git: tệp mới trong thư mục bị ignore trọn có tên mới. Băm nội dung thì không
// đi vào thư mục: với -uall thư mục chưa theo dõi chỉ là repo lồng, băm tên như bản cũ.
// shortcut: tên trong thư mục bị ignore lớn không thuộc regenerableDirs (vendor/ của PHP...) được
// liệt kê lại mỗi lượt (không đọc nội dung); thêm thư mục đó vào regenerableDirs khi việc này chậm thật.
func hashTree(h io.Writer, root, tag, p string, content bool) {
	_ = filepath.WalkDir(filepath.Join(root, p), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(root, path)
		fmt.Fprintf(h, "%s %s\x00", tag, filepath.ToSlash(rel))
		switch {
		case content && d.IsDir():
			return filepath.SkipDir
		case !content:
		case d.Type().IsRegular():
			if f, err := os.Open(path); err == nil {
				_, _ = io.CopyN(h, f, maxVerifyOutput)
				f.Close()
			}
		case d.Type()&fs.ModeSymlink != 0:
			target, _ := os.Readlink(path)
			_, _ = io.WriteString(h, target)
		}
		return nil
	})
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
	case "agent_end":
		switch e.ErrorClass {
		case "":
		case loop.ClassQuota:
			fmt.Fprintf(p.w, "[loop] attempt %d: provider quota or usage limit, not retried: %s\n", e.Attempt, e.Note)
		default:
			fmt.Fprintf(p.w, "[loop] attempt %d: temporary %s %s\n", e.Attempt, e.ErrorClass, e.Note)
		}
	case "verify":
		fmt.Fprintf(p.w, "[loop] attempt %d: verify %s (exit %d)\n", e.Attempt, e.Status, e.ExitCode)
	case "continue": // lượt bị cắt, verify FAIL, còn lượt sửa: Note nói lượt nào bị cắt và vì sao
		fmt.Fprintf(p.w, "[loop] attempt %d: %s\n", e.Attempt, e.Note)
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
	VerifyExit   *int     `json:"verify_exit_code,omitempty"` // chỉ khi verify đã cho ra mã thoát, như báo cáo routine
	Journal      string   `json:"journal"`
}

func printLoopJSON(w io.Writer, o loopOutcome) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	j := loopJSON{
		Reason: string(o.Result.Reason), Detail: o.Result.Detail, Attempts: o.Result.Attempts,
		SessionIDs: o.Result.SessionIDs, VerifyStatus: string(o.Result.Last.Status), Journal: o.JournalPath,
	}
	j.VerifyExit = verifyExitCode(o.Result.Last)
	return enc.Encode(j)
}

// verifyExitCode là mã thoát của verify để báo, chỉ khi verify thật sự cho ra mã: PASS, FAIL,
// hoặc CHƯA ĐO với mã thật khác 0 (126/127). Quá verify_timeout hay không chạy được thì nil:
// 0 đọc như đã qua.
func verifyExitCode(v loop.VerifyResult) *int {
	if v.Status == loop.StatusPass || v.Status == loop.StatusFail || v.Status == loop.StatusUnmeasured && v.ExitCode != 0 {
		return &v.ExitCode
	}
	return nil
}

// newLoopAgent dựng hàm chạy MỘT lượt agent trong session mới (hoặc tiếp session
// loop.ResumeSession khi chạy lại sau lỗi tạm thời), qua server của dự án (mặc
// định) hoặc in-process, đúng như `run`. out nhận phần văn bản của agent.
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
		// Mỗi lượt runNonInteractive đóng luồng SSE của nó khi xong, và server gỡ
		// workspace khi client không còn luồng nào quá 10s: verify hay lúc chờ chạy
		// lại dài hơn thế làm lượt sau hỏng "workspace not found". Giữ một luồng
		// (cùng client id) suốt lần chạy loop.
		// shortcut: luồng giữ này rớt thì không mở lại; khi cần, nối lại trong goroutine đọc.
		holdCtx, stopHold := context.WithCancel(ctx)
		hold, err := c.SubscribeEvents(holdCtx, ws.ID)
		if err != nil {
			stopHold()
			cleanup()
			return nil, nil, err
		}
		go func() {
			for range hold {
			}
		}()
		release := func() {
			stopHold()
			cleanup()
		}
		return func(ctx context.Context, prompt string) (loop.RunInfo, error) {
			var (
				info loop.RunInfo
				text bytes.Buffer
			)
			err := runNonInteractive(ctx, c, ws, io.MultiWriter(out, &text),
				func(id string) { info.SessionID = id }, nil,
				prompt, p.Model, "", p.ReasoningEffort, true, loop.ResumeSession(ctx), false)
			info.Text = text.String()
			return info, err
		}, release, nil
	}

	agent, cleanup, _, err := newLocalLoopAgent(ctx, cmd, p, out)
	return agent, cleanup, err
}

// settleWait là thời gian tối đa chờ một lượt agent bị cắt (hay lỗi) dừng hẳn (settleTurn).
const settleWait = 30 * time.Second

var errTurnNotSettled = errors.New("the agent turn did not stop")

// settleTurn bọc một lượt agent in-process: lượt lỗi hay bị cắt thì chỉ trả về khi app hết bận
// (busy). RunNonInteractive trả về ngay khi ctx của lượt hết hạn, không chờ coordinator dọn xong
// (tool, ghi DB); lượt sau (chạy tiếp sau khi cắt, chạy lại sau lỗi tạm thời) dựng coordinator mới
// trong lúc lượt cũ còn chạy. Chờ tối đa limit, theo outer (Ctrl+C); không dừng được thì trả
// errTurnNotSettled (lỗi gốc chỉ ghi chữ, không bọc: không được phân loại thành lỗi tạm thời rồi
// chạy lại): dừng cả loop còn hơn chạy chồng.
func settleTurn(outer context.Context, run func(context.Context, string) (loop.RunInfo, error), busy func() bool, limit time.Duration) func(context.Context, string) (loop.RunInfo, error) {
	return func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		info, err := run(ctx, prompt)
		cause := cmp.Or(err, ctx.Err())
		if cause == nil {
			return info, nil
		}
		deadline := time.NewTimer(limit)
		defer deadline.Stop()
		tick := time.NewTicker(50 * time.Millisecond)
		defer tick.Stop()
		for busy() {
			select {
			case <-outer.Done():
				return info, err
			case <-deadline.C:
				return info, fmt.Errorf("%w within %s after it ended with: %v; stopping instead of starting another turn over it", errTurnNotSettled, limit, cause)
			case <-tick.C:
			}
		}
		return info, err
	}
}

// newLocalLoopAgent chạy agent in-process (không qua server), trong thư mục hiện
// tại của tiến trình. Routine luôn dùng đường này: lớp chặn không người trực là
// trạng thái của chính tiến trình, nên không thể bị một server dùng chung bỏ qua.
// model() trả provider/model large mà lượt dùng (routineModel), cho báo cáo routine.
func newLocalLoopAgent(outer context.Context, cmd *cobra.Command, p loopParams, out io.Writer) (func(context.Context, string) (loop.RunInfo, error), func(), func() string, error) {
	ws, cleanup, err := setupLocalWorkspace(cmd)
	if err != nil {
		return nil, nil, nil, err
	}
	event.AppInitialized()
	if !ws.Config().IsConfigured() {
		cleanup()
		return nil, nil, nil, fmt.Errorf("no providers configured - please run 'dew' to set up a provider interactively")
	}
	appWs := ws.(*workspace.AppWorkspace)
	model := func() string {
		store := appWs.App().Store()
		return routineModel(store.Config().Models[config.SelectedModelTypeLarge], store.Overrides().Models[config.SelectedModelTypeLarge], p.Model)
	}
	// Model ghim trùng model large mặc định (`routine new` đóng băng nó) thì đừng áp như -m: áp
	// sẽ thay cả SelectedModel (mất max_tokens, temperature, reasoning_effort...) và đổi model
	// nhỏ sang model nhỏ mặc định của provider, tức chạy KHÁC cái mặc định đã đóng băng.
	large := p.Model
	if large == routineModel(appWs.App().Store().Config().Models[config.SelectedModelTypeLarge], config.SelectedModel{}, "") {
		large = ""
	}
	run := func(ctx context.Context, prompt string) (loop.RunInfo, error) {
		var (
			info loop.RunInfo
			text bytes.Buffer
		)
		err := appWs.App().RunNonInteractive(ctx, io.MultiWriter(out, &text),
			func(id string) { info.SessionID = id },
			prompt, large, "", p.ReasoningEffort, true, loop.ResumeSession(ctx), false)
		info.Text = text.String()
		return info, err
	}
	// Coordinator của lượt vừa xong: lượt sau mới thay nó (initCoderAgent), sau khi settleTurn trả về.
	busy := func() bool {
		c := appWs.App().AgentCoordinator
		return c != nil && c.IsBusy()
	}
	return settleTurn(outer, run, busy, settleWait), cleanup, model, nil
}
