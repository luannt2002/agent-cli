package agent

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/loop"
)

// requestTimeoutError reports that an LLM request exhausted its configured
// request_timeout budget. For streaming requests this is an idle timeout:
// it only fires when the provider sends nothing for the whole window, so a
// slow but actively streaming response is never killed. It wraps the
// underlying error so callers can still match [context.DeadlineExceeded]
// through the chain.
type requestTimeoutError struct {
	timeout time.Duration
	idle    bool
	cause   error
}

func (e *requestTimeoutError) Error() string {
	msg := fmt.Sprintf("LLM request timed out after %s", e.timeout)
	if e.idle {
		msg = fmt.Sprintf("LLM stream received no data for %s", e.timeout)
	}
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", msg, e.cause)
	}
	return msg
}

func (e *requestTimeoutError) Unwrap() error { return e.cause }

// userMessage explains the timeout in the UI, including how long the request
// ran before giving up and how to change the limit.
func (e *requestTimeoutError) userMessage() string {
	hint := "Increase the limit with \"option request-timeout SECONDS\" or set it to 0 to disable the timeout."
	if e.idle {
		return fmt.Sprintf("The model stopped sending data for %s. %s", e.timeout, hint)
	}
	return fmt.Sprintf("The model did not respond within %s. %s", e.timeout, hint)
}

// summaryRequestTimeoutScale nhân hạn chờ cho request tóm tắt: request đó đọc
// cả lịch sử (gần đầy context) nên model có thể im lâu trước token đầu; hạn
// chờ thường (mặc định 2 phút) từng cắt ngang và làm dừng cả lượt.
const summaryRequestTimeoutScale = 3

type requestTimeoutScaleKey struct{}

// withRequestTimeoutScale nhân hạn chờ của các request đi qua ctx.
func withRequestTimeoutScale(ctx context.Context, scale int) context.Context {
	return context.WithValue(ctx, requestTimeoutScaleKey{}, scale)
}

// budget là hạn chờ của request: hạn cấu hình, nhân hệ số trong ctx nếu có.
func (m requestTimeoutModel) budget(ctx context.Context) time.Duration {
	if scale, ok := ctx.Value(requestTimeoutScaleKey{}).(int); ok && scale > 1 {
		return m.timeout * time.Duration(scale)
	}
	return m.timeout
}

// requestTimeoutModel wraps a [fantasy.LanguageModel] so requests are
// bounded by the configured request_timeout. Non-streaming calls get a hard
// per-request deadline, applied per call so fantasy's retry loop gives every
// attempt a fresh budget — the same per-request semantics the provider SDKs
// expose. Streams instead get an idle timeout: the budget resets whenever a
// part arrives and only fires when the provider goes silent, so a slow but
// actively streaming response is never aborted.
type requestTimeoutModel struct {
	fantasy.LanguageModel
	timeout time.Duration
}

// newRequestTimeoutModel bounds each request to m with the given timeout. A
// timeout of zero or less, or a nil model, returns m unchanged.
func newRequestTimeoutModel(m fantasy.LanguageModel, timeout time.Duration) fantasy.LanguageModel {
	if m == nil || timeout <= 0 {
		return m
	}
	return requestTimeoutModel{LanguageModel: m, timeout: timeout}
}

// wrapTimedOut replaces err with the requestTimeoutError when this model's
// own deadline fired. Other errors — user cancellation, outer deadlines,
// provider failures — pass through unchanged. Cancellation errors caused by
// our own timer report as [context.DeadlineExceeded] so callers never
// mistake a timeout for a user cancellation.
func wrapTimedOut(ctx context.Context, timeoutErr *requestTimeoutError, err error) error {
	if err == nil || context.Cause(ctx) != timeoutErr {
		return err
	}
	// net/http trả thẳng context.Cause(ctx), tức chính timeoutErr (hoặc lỗi bọc nó): làm cause
	// thì vòng Unwrap khiến errors.Is/As chạy mãi.
	if errors.Is(err, context.Canceled) || errors.Is(err, timeoutErr) {
		timeoutErr.cause = context.DeadlineExceeded
	} else {
		timeoutErr.cause = err
	}
	return timeoutErr
}

// Generate implements [fantasy.LanguageModel]. The request gets a hard
// deadline: there is no incremental progress signal, so the whole call must
// finish within the budget.
func (m requestTimeoutModel) Generate(ctx context.Context, call fantasy.Call) (*fantasy.Response, error) {
	timeout := m.budget(ctx)
	timeoutErr := &requestTimeoutError{timeout: timeout}
	ctx, cancel := context.WithTimeoutCause(ctx, timeout, timeoutErr)
	defer cancel()
	resp, err := m.LanguageModel.Generate(ctx, call)
	return resp, wrapTimedOut(ctx, timeoutErr, err)
}

// Stream implements [fantasy.LanguageModel].
//
// The stream is consumed after Stream returns, so the timer must outlive
// this call: it fires only after timeout seconds without any part arriving,
// and is released when iteration ends, whether the stream finishes, breaks,
// or the idle timeout aborts it. Both the initial connection and gaps
// between parts share the same budget.
//
// Inside a loop turn (loop.RequestObserver) each stream is also measured for the
// journal's llm_request event; see requestRecord.
func (m requestTimeoutModel) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	timeout := m.budget(ctx)
	timeoutErr := &requestTimeoutError{timeout: timeout, idle: true}
	rec := newRequestRecord(ctx)
	ctx, cancel := context.WithCancelCause(ctx)
	timer := time.AfterFunc(timeout, func() { cancel(timeoutErr) })

	inner, err := m.LanguageModel.Stream(ctx, call)
	if err != nil {
		timer.Stop()
		cancel(nil)
		err = wrapTimedOut(ctx, timeoutErr, err)
		rec.done(err)
		return nil, err
	}
	return func(yield func(fantasy.StreamPart) bool) {
		defer timer.Stop()
		defer cancel(nil)
		defer rec.done(nil)
		inner(func(part fantasy.StreamPart) bool {
			timer.Reset(timeout)
			part.Error = wrapTimedOut(ctx, timeoutErr, part.Error)
			rec.part(part)
			return yield(part)
		})
	}, nil
}

// requestRecord đo một request streaming cho journal của vòng lặp: thời gian tới phần đầu
// tiên (TTFT), tổng thời gian, usage và lớp lỗi. nil khi ctx không có observer (TUI, run):
// không tốn gì.
// shortcut: đo trong wrapper của request_timeout nên request_timeout 0 (không có wrapper) thì
// không có llm_request; khi cần đo cả lúc đó, tách requestRecord thành wrapper luôn bật.
type requestRecord struct {
	observe func(loop.Event)
	parent  context.Context // ctx của người gọi: nó huỷ là "cancelled", không phải lỗi provider
	start   time.Time
	ev      loop.Event
	err     error // lỗi đầu tiên giữa stream
}

func newRequestRecord(ctx context.Context) *requestRecord {
	observe := loop.RequestObserver(ctx)
	if observe == nil {
		return nil
	}
	now := time.Now()
	return &requestRecord{observe: observe, parent: ctx, start: now, ev: loop.Event{Time: now}}
}

func (r *requestRecord) part(p fantasy.StreamPart) {
	if r == nil {
		return
	}
	// Phần Warnings do fantasy tự sinh trước khi đọc mạng; phần lỗi không phải dữ liệu.
	if r.ev.TTFTMS == nil && p.Type != fantasy.StreamPartTypeWarnings && p.Error == nil {
		ttft := time.Since(r.start).Milliseconds()
		r.ev.TTFTMS = &ttft
	}
	if p.Type == fantasy.StreamPartTypeFinish {
		u := p.Usage
		// openai tách phần đọc cache khỏi InputTokens, anthropic tách cả phần ghi cache: cộng
		// lại mới là cỡ prompt đã gửi.
		r.ev.PromptTokens = u.InputTokens + u.CacheReadTokens + u.CacheCreationTokens
		r.ev.CompletionTokens = u.OutputTokens
	}
	if p.Error != nil && r.err == nil {
		r.err = p.Error
	}
}

// done ghi số đo; err là lỗi mở stream (nil thì dùng lỗi đầu tiên giữa stream).
func (r *requestRecord) done(err error) {
	if r == nil {
		return
	}
	r.ev.TotalMS = time.Since(r.start).Milliseconds()
	r.ev.Status, r.ev.HTTPStatus = requestStatus(r.parent, cmp.Or(err, r.err))
	r.observe(r.ev)
}

// requestStatus: "cancelled" khi chính người gọi huỷ (Ctrl+C, hạn mỗi lượt), xét trước: người
// gọi thôi đọc vì huỷ thì không có phần lỗi nào nhưng request không xong; rồi "ok"; lớp của
// TransientError; hoặc "other". Mã HTTP khi provider trả về.
func requestStatus(parent context.Context, err error) (string, int) {
	code := 0
	if pe, ok := errors.AsType[*fantasy.ProviderError](err); ok {
		code = pe.StatusCode
	}
	switch {
	case parent.Err() != nil:
		return "cancelled", code
	case err == nil:
		return "ok", 0
	}
	class, _ := TransientError(err)
	return cmp.Or(class, "other"), code
}
