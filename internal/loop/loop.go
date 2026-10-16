// Package loop là engine Loop: lặp "agent làm → lệnh verify tất định đo → sửa"
// cho tới khi đạt điều kiện dừng. Engine KHÔNG biết cách chạy agent hay verify:
// cả hai được tiêm vào (Engine.Agent, Engine.Verify), nên test được không cần
// server, và driver CLI/routine dùng chung một bộ luật dừng.
//
// Luật thiết kế (theo learn_claudecode): xong = lệnh verify thoát 0 (không phải
// lời agent); ba trạng thái PASS/FAIL/UNMEASURED; hai lần sửa là hết; cùng một
// lỗi mà cây làm việc không đổi thì dừng (không tiến triển); ghi journal TRƯỚC
// khi hành động; không có trần tiền (chốt thật là thời gian + số lần sửa).
package loop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	StopMaxRepairs Stop = "MAX_REPAIRS" // hết số lần sửa mà verify vẫn đỏ
	StopStalled    Stop = "STALLED"     // cùng lỗi, cây làm việc không đổi
	StopTimeout    Stop = "TIMEOUT"     // quá MaxDuration, kiểm trước mỗi lượt agent
	StopRunError   Stop = "RUN_ERROR"   // lượt agent lỗi
	StopCancelled  Stop = "CANCELLED"   // ctx bị huỷ
	StopUnmeasured Stop = "UNMEASURED"  // không chạy được verify
)

// Spec mô tả một lần chạy loop.
type Spec struct {
	Name   string
	Goal   string // việc cần làm, tự chứa đủ
	Verify string // lệnh định nghĩa "xong": thoát 0 là xong
	// MaxRepairs là số lượt SỬA sau lượt đầu (mặc định của driver: 2). Tổng số
	// lượt agent tối đa là MaxRepairs+1; không có lần vá thứ ba.
	MaxRepairs int
	// MaxDuration > 0 là trần thời gian cả lần chạy; kiểm trước mỗi lượt agent.
	MaxDuration time.Duration
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
}

// Result là kết quả cả lần chạy.
type Result struct {
	Reason     Stop
	Detail     string
	Attempts   int // số lượt agent đã chạy
	SessionIDs []string
	Last       VerifyResult
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
	prompt := goalPrompt(e.Spec)

	for attempt := 1; ; attempt++ {
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
		info, runErr := e.Agent(ctx, prompt)
		res.Attempts = attempt
		if info.SessionID != "" {
			res.SessionIDs = append(res.SessionIDs, info.SessionID)
		}
		if runErr != nil {
			if ctx.Err() != nil {
				return stop(StopCancelled, runErr.Error())
			}
			_ = emit(Event{Kind: "agent_end", Attempt: attempt, SessionID: info.SessionID, Note: "error: " + runErr.Error()})
			return stop(StopRunError, runErr.Error())
		}
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
			return stop(StopGreen, "")
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
		prompt = repairPrompt(e.Spec, v, attempt)
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
	return fmt.Sprintf("%s\n\nDefinition of done: running `%s` exits with code 0.\n%s", strings.TrimSpace(s.Goal), s.Verify, noEditRule)
}

// repairPrompt tự chứa đủ (đề bài + lỗi vừa đo), vì mỗi lượt có thể là session mới.
func repairPrompt(s Spec, v VerifyResult, attempt int) string {
	return fmt.Sprintf("%s\n\nThe previous attempt did not pass verification.\n"+
		"Verify command: `%s`\nResult: exit code %d\nOutput (last %d lines):\n%s\n\n"+
		"Find and fix the cause in the code. %s\nRepair %d of %d.",
		strings.TrimSpace(s.Goal), s.Verify, v.ExitCode, tailLines, tail(v.Output, tailLines), noEditRule, attempt, s.MaxRepairs)
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
