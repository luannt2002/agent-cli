// Package loop là engine Loop: lặp "agent làm → lệnh verify tất định đo → sửa"
// cho tới khi đạt điều kiện dừng. Engine KHÔNG biết cách chạy agent hay verify:
// cả hai được tiêm vào (Engine.Agent, Engine.Verify), nên test được không cần
// server, và driver CLI/routine dùng chung một bộ luật dừng.
//
// Luật thiết kế (theo learn_claudecode): xong = lệnh verify thoát 0 (không phải
// lời agent); ba trạng thái PASS/FAIL/UNMEASURED; hai lần sửa là hết; cùng một
// lỗi mà cây làm việc không đổi thì dừng (không tiến triển); ghi journal TRƯỚC
// khi hành động; không có trần tiền (chốt thật là thời gian + số lần sửa). Lượt
// agent hỏng vì lỗi tạm thời của provider thì chạy lại (có trần), không tốn lượt sửa.
package loop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Status là kết quả một lần verify. Ba trạng thái, không hai: không chạy được
// verify là CHƯA ĐO, không được tính xanh.
type Status string

const (
	StatusPass       Status = "PASS"
	StatusFail       Status = "FAIL"
	StatusUnmeasured Status = "UNMEASURED"
)

// Stop là lý do loop dừng. Do engine tính từ lệnh verify, không từ mã thoát của agent.
type Stop string

const (
	StopGreen      Stop = "GREEN"       // verify thoát 0 sau một lượt agent
	StopNoChange   Stop = "NO_CHANGE"   // verify xanh từ trước VÀ cây không đổi: không chứng minh gì
	StopMaxRepairs Stop = "MAX_REPAIRS" // hết số lần sửa mà verify vẫn đỏ
	StopStalled    Stop = "STALLED"     // cùng lỗi, cây làm việc không đổi
	StopTimeout    Stop = "TIMEOUT"     // quá MaxDuration (kiểm trước mỗi lượt agent), hoặc một lượt quá hạn của nó (ErrTurnTimeout)
	StopRunError   Stop = "RUN_ERROR"   // lượt agent lỗi (lỗi tạm thời: đã chạy lại hết trần)
	StopCancelled  Stop = "CANCELLED"   // ctx bị huỷ
	StopUnmeasured Stop = "UNMEASURED"  // không chạy được verify
	// StopStreamStall: stream của LLM im (lớp ClassStreamStall) hết số lần chạy lại.
	StopStreamStall Stop = "STREAM_STALL"
	// StopQuota: provider báo hết hạn mức/usage limit (lớp ClassQuota): không chạy lại.
	StopQuota Stop = "QUOTA"
)

// Lớp lỗi mà Engine.Transient trả về và engine xử lý riêng.
const (
	ClassStreamStall = "stream_stall" // stream im quá hạn chờ: hết lần chạy lại thì STREAM_STALL
	ClassQuota       = "quota"        // hết hạn mức: KHÔNG chạy lại, dừng QUOTA ngay
)

// ErrTurnTimeout là lỗi mà driver bọc (%w) khi chính hạn chờ mỗi lượt của nó cắt một
// lượt agent (--run-timeout, run_timeout): engine dừng TIMEOUT sau khi đo verify một lần.
var ErrTurnTimeout = errors.New("agent turn timed out")

// Spec mô tả một lần chạy loop.
type Spec struct {
	Name   string
	Goal   string // việc cần làm, tự chứa đủ
	Verify string // lệnh định nghĩa "xong": thoát 0 là xong
	// VerifyDir là thư mục chạy Verify, tương đối theo working dir của agent
	// (vd "sub/"), khi hai thư mục khác nhau; rỗng khi trùng (prompt như cũ).
	VerifyDir string
	// MaxRepairs là số lượt SỬA sau lượt đầu (mặc định của driver: 2). Tổng số
	// lượt agent tối đa là MaxRepairs+1; không có lần vá thứ ba.
	MaxRepairs int
	// MaxDuration > 0 là trần thời gian cả lần chạy; kiểm trước mỗi lượt agent.
	MaxDuration time.Duration
	// StreamRetries là số lần chạy lại LIÊN TIẾP một lượt bị lỗi tạm thời cắt ngang
	// (mặc định của driver: 2); 0 = không chạy lại.
	StreamRetries int
}

// RunInfo là kết quả một lượt agent.
type RunInfo struct {
	SessionID string
	Text      string
}

// VerifyResult là kết quả lệnh verify. Output là phần đuôi đủ để đưa lại cho agent.
type VerifyResult struct {
	Status   Status
	ExitCode int
	Output   string
	Duration time.Duration
}

// Engine chạy loop. Mọi trường trừ Now và Journal là bắt buộc.
type Engine struct {
	Spec        Spec
	Agent       func(ctx context.Context, prompt string) (RunInfo, error)
	Verify      func(ctx context.Context, cmd string) (VerifyResult, error)
	Fingerprint func(ctx context.Context) (string, error) // dấu vân tay cây làm việc
	Journal     Journal
	Now         func() time.Time
	// Transient phân loại lỗi của lượt agent: class khác rỗng là lỗi tạm thời
	// (provider 5xx/429, stream im, mạng đứt) thì chạy lại lượt đó thay vì dừng, trừ
	// ClassQuota (dừng QUOTA ngay); status là mã HTTP (0 nếu không biết). nil: mọi lỗi
	// dừng RUN_ERROR như cũ.
	Transient func(error) (class string, status int)

	retryWaits []time.Duration // nil = transientWaits (retryWait); test rút ngắn
}

// Result là kết quả cả lần chạy.
type Result struct {
	Reason     Stop
	Detail     string
	Attempts   int // số lượt agent đã chạy
	SessionIDs []string
	Last       VerifyResult
	// BaselinePass: verify đã xanh trước mọi thay đổi (cổng có thể không đo mục tiêu).
	BaselinePass bool
	// Unchanged: verify xanh mà dấu vân tay cây làm việc y như lúc baseline (agent không
	// đổi gì). False khi không biết dấu vân tay.
	Unchanged bool
}

// transientWaits là thời gian chờ trước mỗi lần chạy lại sau lỗi tạm thời liên tiếp
// (retryWait); trần số lần chạy lại là Spec.StreamRetries.
var transientWaits = []time.Duration{30 * time.Second, time.Minute}

// interruptedNote mở đầu prompt của lượt chạy lại: lượt bị cắt có thể đã sửa tệp
// hoặc chạy lệnh dở dang.
const interruptedNote = "Your previous turn was cut off by a temporary provider error before it finished. Check the current state of the files before you continue."

type resumeKey struct{}

// ResumeSession là session mà hàm Agent nên chạy tiếp ("" = session mới). Engine
// đặt nó vào ctx khi chạy lại một lượt bị lỗi tạm thời cắt ngang; driver bỏ qua thì
// lượt chạy lại ở session mới, vẫn đủ vì prompt tự chứa đủ đề bài.
func ResumeSession(ctx context.Context) string {
	id, _ := ctx.Value(resumeKey{}).(string)
	return id
}

// retryWait là thời gian chờ trước lần chạy lại thứ i (từ 0): waits[i], quá cuối bảng
// thì giữ phần tử cuối. waits nil = transientWaits.
func retryWait(waits []time.Duration, i int) time.Duration {
	if waits == nil {
		waits = transientWaits
	}
	return waits[min(i, len(waits)-1)]
}

// tailLines là số dòng cuối của output verify đưa lại cho agent và ghi vào journal.
const tailLines = 60

// Run chạy loop tới khi có lý do dừng. Lỗi trả về chỉ dành cho hỏng nội bộ
// (không ghi được journal); mọi lý do dừng bình thường nằm ở Result.Reason.
func (e *Engine) Run(ctx context.Context) (Result, error) {
	now := e.Now
	if now == nil {
		now = time.Now
	}
	start := now()
	res := Result{}
	emit := func(ev Event) error {
		ev.Time = now()
		if e.Journal == nil {
			return nil
		}
		return e.Journal.Append(ev)
	}
	stop := func(reason Stop, detail string) (Result, error) {
		res.Reason, res.Detail = reason, detail
		return res, emit(Event{Kind: "stop", Attempt: res.Attempts, Reason: string(reason), Note: detail})
	}
	// cut dừng một lần chạy mà lượt agent bị cắt ngang (hết hạn lượt, lỗi tạm thời hết lần
	// chạy lại): đo verify MỘT lần trên cây agent bỏ lại để báo cáo có trạng thái thật.
	// Không bao giờ GREEN, kể cả khi verify xanh: không ai biết agent đã làm xong chưa.
	cut := func(attempt int, reason Stop, detail string) (Result, error) {
		fp, _ := e.fingerprint(ctx)
		v, _ := e.runVerify(ctx) // lỗi hạ tầng đã thành CHƯA ĐO trong v
		if ctx.Err() != nil {
			return stop(StopCancelled, "cancelled during the verify after the agent was cut off")
		}
		res.Last = v
		if err := emit(Event{Kind: "verify", Attempt: attempt, Status: string(v.Status), ExitCode: v.ExitCode,
			Output: tail(v.Output, tailLines), Fingerprint: fp, Note: "after the agent was cut off"}); err != nil {
			return res, err
		}
		switch v.Status {
		case StatusPass:
			detail += "; verify PASS after the agent was cut off — review before trusting"
		case StatusFail:
			detail += fmt.Sprintf("; verify FAIL (exit %d) after the agent was cut off", v.ExitCode)
		default:
			detail += "; verify could not run after the agent was cut off: " + strings.TrimSpace(tail(v.Output, 5))
		}
		return stop(reason, detail)
	}

	if err := emit(Event{Kind: "start", Note: e.Spec.Name}); err != nil {
		return res, err
	}
	if ctx.Err() != nil {
		return stop(StopCancelled, "cancelled before start")
	}

	// Baseline: đo verify TRƯỚC mọi thay đổi. Đỏ là bình thường (có việc để làm);
	// xanh nghĩa là cổng có thể không đo đúng mục tiêu; không chạy được thì dừng.
	base, err := e.runVerify(ctx)
	if ctx.Err() != nil {
		return stop(StopCancelled, "cancelled during the baseline verify")
	}
	if err != nil {
		return stop(StopUnmeasured, err.Error())
	}
	baseFP, _ := e.fingerprint(ctx)
	note := ""
	if base.Status == StatusPass {
		note = "verify already passes before any change: the gate may not measure this goal"
		res.BaselinePass = true
	}
	if err := emit(Event{Kind: "baseline", Status: string(base.Status), ExitCode: base.ExitCode,
		Output: tail(base.Output, tailLines), Fingerprint: baseFP, Note: note}); err != nil {
		return res, err
	}
	if base.Status == StatusUnmeasured {
		res.Last = base
		return stop(StopUnmeasured, "cannot run the verify command: "+tail(base.Output, 5))
	}

	prevSig, prevFP := signature(base.ExitCode, base.Output), baseFP
	if base.Status == StatusPass {
		prevSig = ""
	}
	deadline := time.Time{}
	if e.Spec.MaxDuration > 0 {
		deadline = start.Add(e.Spec.MaxDuration)
	}
	turn := goalPrompt(e.Spec) // đề bài của lượt hiện tại (lượt đầu hoặc lượt sửa)
	prompt, resume, retries := turn, "", 0

	for attempt := 1; ; {
		if ctx.Err() != nil {
			return stop(StopCancelled, "cancelled")
		}
		if !deadline.IsZero() && !now().Before(deadline) {
			return stop(StopTimeout, fmt.Sprintf("max duration %s reached after %d agent turn(s)", e.Spec.MaxDuration, res.Attempts))
		}

		// Ghi trước khi làm: bị kill giữa lượt thì journal vẫn nói đang ở lượt nào.
		if err := emit(Event{Kind: "attempt_begin", Attempt: attempt, PromptLen: len(prompt)}); err != nil {
			return res, err
		}
		actx := ctx
		if resume != "" {
			actx = context.WithValue(ctx, resumeKey{}, resume)
		}
		info, runErr := e.Agent(actx, prompt)
		res.Attempts++
		if info.SessionID != "" && (len(res.SessionIDs) == 0 || res.SessionIDs[len(res.SessionIDs)-1] != info.SessionID) {
			res.SessionIDs = append(res.SessionIDs, info.SessionID)
		}
		if runErr != nil {
			if ctx.Err() != nil {
				return stop(StopCancelled, runErr.Error())
			}
			ev := Event{Kind: "agent_end", Attempt: attempt, SessionID: info.SessionID, Note: "error: " + runErr.Error()}
			if e.Transient != nil {
				ev.ErrorClass, ev.HTTPStatus = e.Transient(runErr)
			}
			switch {
			case errors.Is(runErr, ErrTurnTimeout):
				_ = emit(ev)
				return cut(attempt, StopTimeout, runErr.Error())
			case ev.ErrorClass == ClassQuota:
				_ = emit(ev)
				detail := runErr.Error()
				if ev.HTTPStatus != 0 {
					detail += fmt.Sprintf(" (HTTP %d)", ev.HTTPStatus)
				}
				return stop(StopQuota, detail)
			case ev.ErrorClass == "":
				_ = emit(ev)
				return stop(StopRunError, runErr.Error())
			}
			class := ev.ErrorClass
			if ev.HTTPStatus != 0 {
				class += fmt.Sprintf(" error (HTTP %d)", ev.HTTPStatus)
			} else {
				class += " error"
			}
			if retries >= e.Spec.StreamRetries {
				_ = emit(ev)
				reason := StopRunError
				if ev.ErrorClass == ClassStreamStall {
					reason = StopStreamStall
				}
				return cut(attempt, reason, fmt.Sprintf("temporary %s persists after %d retries: %v", class, retries, runErr))
			}
			wait := retryWait(e.retryWaits, retries)
			// Lượt chạy lại không được phóng sau hạn: đừng ngủ hết thời gian chờ rồi mới dừng.
			if !deadline.IsZero() && !now().Add(wait).Before(deadline) {
				_ = emit(ev)
				return cut(attempt, StopTimeout, fmt.Sprintf("max duration %s leaves no time to retry after a temporary %s (%d agent turn(s))", e.Spec.MaxDuration, class, res.Attempts))
			}
			retries++
			ev.Note += fmt.Sprintf("; retry %d of %d in %s", retries, e.Spec.StreamRetries, wait)
			if err := emit(ev); err != nil {
				return res, err
			}
			if !sleep(ctx, wait) {
				return stop(StopCancelled, "cancelled while waiting to retry after a temporary error")
			}
			// Chạy lại không tốn lượt sửa: cùng attempt, cùng đề bài, tiếp session bị cắt.
			prompt, resume = interruptedNote+"\n\n"+turn, info.SessionID
			continue
		}
		retries, resume = 0, ""
		fp, _ := e.fingerprint(ctx)
		if err := emit(Event{Kind: "agent_end", Attempt: attempt, SessionID: info.SessionID, Fingerprint: fp}); err != nil {
			return res, err
		}

		v, err := e.runVerify(ctx)
		if ctx.Err() != nil {
			// Huỷ giữa lúc verify: kết quả đo dở không phải bằng chứng, đừng báo CHƯA ĐO.
			return stop(StopCancelled, "cancelled during verify")
		}
		if err != nil {
			return stop(StopUnmeasured, err.Error())
		}
		res.Last = v
		if err := emit(Event{Kind: "verify", Attempt: attempt, Status: string(v.Status), ExitCode: v.ExitCode,
			Output: tail(v.Output, tailLines), Fingerprint: fp}); err != nil {
			return res, err
		}

		switch v.Status {
		case StatusPass:
			// GREEN không đi kèm thay đổi nào thì phải nói ra, không chỉ nằm trong journal.
			res.Unchanged = fp != "" && fp == baseFP
			if res.BaselinePass && res.Unchanged {
				// Xanh trước và sau, cây y nguyên: cổng không đo việc này hoặc việc đã xong sẵn.
				return stop(StopNoChange, "verify already passed before any change and the working tree did not change: nothing proves the agent's work")
			}
			var warn []string
			if res.BaselinePass {
				warn = append(warn, "verify already passed before any change")
			}
			if res.Unchanged {
				warn = append(warn, "the working tree did not change")
			}
			return stop(StopGreen, strings.Join(warn, "; "))
		case StatusUnmeasured:
			return stop(StopUnmeasured, "cannot run the verify command: "+tail(v.Output, 5))
		}

		// FAIL. Cùng lỗi mà cây không đổi sau một lượt sửa nghĩa là agent không tiến triển.
		sig := signature(v.ExitCode, v.Output)
		if attempt > 1 && sig == prevSig && fp == prevFP {
			return stop(StopStalled, "same failure and unchanged working tree after a repair turn")
		}
		if attempt-1 >= e.Spec.MaxRepairs {
			return stop(StopMaxRepairs, fmt.Sprintf("still failing after %d repair turn(s)", e.Spec.MaxRepairs))
		}
		prevSig, prevFP = sig, fp
		turn = repairPrompt(e.Spec, v, attempt)
		prompt = turn
		attempt++
	}
}

// sleep chờ d; false khi ctx bị huỷ trước đó.
func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// runVerify chạy verify; lỗi hạ tầng của chính hàm Verify thành CHƯA ĐO.
func (e *Engine) runVerify(ctx context.Context) (VerifyResult, error) {
	v, err := e.Verify(ctx, e.Spec.Verify)
	if err != nil {
		return VerifyResult{Status: StatusUnmeasured, Output: err.Error()}, fmt.Errorf("verify could not run: %w", err)
	}
	return v, nil
}

func (e *Engine) fingerprint(ctx context.Context) (string, error) {
	if e.Fingerprint == nil {
		return "", nil
	}
	return e.Fingerprint(ctx)
}

// noEditRule là câu cấm đường tắt cám dỗ nhất: sửa test/cổng cho xanh.
const noEditRule = "Do not modify the tests, the verify command or any gate to make it pass. " +
	"If you believe a test is wrong, say so in your final answer instead of editing it."

func goalPrompt(s Spec) string {
	in := ""
	if s.VerifyDir != "" {
		in = " in `" + s.VerifyDir + "`"
	}
	return fmt.Sprintf("%s\n\nDefinition of done: running `%s`%s exits with code 0.\n%s", strings.TrimSpace(s.Goal), s.Verify, in, noEditRule)
}

// repairPrompt tự chứa đủ (đề bài + lỗi vừa đo), vì mỗi lượt có thể là session mới.
func repairPrompt(s Spec, v VerifyResult, attempt int) string {
	in := ""
	if s.VerifyDir != "" {
		in = " (run in `" + s.VerifyDir + "`)"
	}
	return fmt.Sprintf("%s\n\nThe previous attempt did not pass verification.\n"+
		"Verify command: `%s`%s\nResult: exit code %d\nOutput (last %d lines):\n%s\n\n"+
		"Find and fix the cause in the code. %s\nRepair %d of %d.",
		strings.TrimSpace(s.Goal), s.Verify, in, v.ExitCode, tailLines, tail(v.Output, tailLines), noEditRule, attempt, s.MaxRepairs)
}

var (
	reHex    = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	reNumber = regexp.MustCompile(`[0-9]+(\.[0-9]+)?`)
)

// signature là chữ ký của một lỗi verify: mã thoát cộng 20 dòng cuối đã bỏ số,
// địa chỉ và thời gian. Hai lượt cùng "FAIL TestFoo (12ms)" và "(907ms)" là cùng lỗi.
func signature(exit int, output string) string {
	var lines []string
	for _, l := range strings.Split(tail(output, 20), "\n") {
		l = reHex.ReplaceAllString(l, "0xN")
		l = reNumber.ReplaceAllString(l, "N")
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d|%s", exit, strings.Join(lines, "\n"))))
	return hex.EncodeToString(sum[:8])
}

// tail giữ n dòng cuối của s (bỏ dòng trống ở đuôi).
func tail(s string, n int) string {
	s = strings.TrimRight(s, "\n")
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
