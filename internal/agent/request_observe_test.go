package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openaicompat"
	"github.com/charmbracelet/crush/internal/loop"
	"github.com/stretchr/testify/require"
)

// requestLog gom các sự kiện mà observer của loop nhận (có thể từ goroutine khác).
type requestLog struct {
	mu     sync.Mutex
	events []loop.Event
}

func (l *requestLog) observe(e loop.Event) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, e)
}

func (l *requestLog) all() []loop.Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]loop.Event(nil), l.events...)
}

// sseChunk là một dòng SSE của OpenAI chat completions.
func sseChunk(w http.ResponseWriter, body string) {
	fmt.Fprintf(w, "data: %s\n\n", body)
	w.(http.Flusher).Flush()
}

const (
	chunkText   = `{"id":"1","object":"chat.completion.chunk","created":0,"model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":"hi"},"finish_reason":null}]}`
	chunkFinish = `{"id":"1","object":"chat.completion.chunk","created":0,"model":"m","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
	// 30 token prompt, 10 trong đó đọc từ cache (fantasy tách thành Input 20 + CacheRead 10).
	chunkUsage = `{"id":"1","object":"chat.completion.chunk","created":0,"model":"m","choices":[],"usage":{"prompt_tokens":30,"completion_tokens":7,"total_tokens":37,"prompt_tokens_details":{"cached_tokens":10}}}`
)

// fakeOpenAI dựng provider openai-compat THẬT của fantasy trỏ vào handler, bọc bằng
// requestTimeoutModel như coordinator.
func fakeOpenAI(t *testing.T, timeout time.Duration, h http.HandlerFunc) fantasy.LanguageModel {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	p, err := openaicompat.New(openaicompat.WithBaseURL(srv.URL+"/v1"), openaicompat.WithAPIKey("x"))
	require.NoError(t, err)
	m, err := p.LanguageModel(t.Context(), "m")
	require.NoError(t, err)
	return newRequestTimeoutModel(m, timeout)
}

// drain chạy một request streaming tới hết, như vòng lặp của fantasy.
func drain(ctx context.Context, m fantasy.LanguageModel) {
	stream, err := m.Stream(ctx, fantasy.Call{Prompt: fantasy.Prompt{fantasy.NewUserMessage("hi")}})
	if err != nil {
		return
	}
	for range stream {
	}
}

// Server giả có độ trễ biết trước tới byte đầu: TTFT ghi được phải nằm trong
// [độ trễ, độ trễ + tolerance], tổng thời gian gồm cả phần đuôi 50ms. Tolerance 250ms
// (máy dùng chung, GOMAXPROCS=2); lệch đơn vị ms/s cho 0 hoặc gấp 1000 nên vẫn đỏ.
func TestRequestTimeoutModel_ObservesTTFTTotalAndUsage(t *testing.T) {
	t.Parallel()
	delays := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond, 400 * time.Millisecond, 500 * time.Millisecond}
	const tailDelay = 50 * time.Millisecond
	const tolerance = 250 * time.Millisecond
	var n atomic.Int32
	m := fakeOpenAI(t, time.Minute, func(w http.ResponseWriter, r *http.Request) {
		d := delays[n.Add(1)-1]
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		time.Sleep(d)
		sseChunk(w, chunkText)
		time.Sleep(tailDelay)
		sseChunk(w, chunkFinish)
		sseChunk(w, chunkUsage)
		sseChunk(w, "[DONE]")
	})
	log := &requestLog{}
	ctx := loop.WithRequestObserver(t.Context(), log.observe)
	for range delays {
		before := time.Now()
		drain(ctx, m)
		events := log.all()
		require.NotEmpty(t, events)
		require.WithinDuration(t, before, events[len(events)-1].Time, 50*time.Millisecond, "Time là lúc gửi request")
	}

	events := log.all()
	require.Len(t, events, len(delays))
	for i, ev := range events {
		d := delays[i]
		require.NotNil(t, ev.TTFTMS, "request %d", i)
		ttft := time.Duration(*ev.TTFTMS) * time.Millisecond
		total := time.Duration(ev.TotalMS) * time.Millisecond
		require.GreaterOrEqual(t, ttft, d, "request %d: TTFT %s < độ trễ %s", i, ttft, d)
		require.Less(t, ttft, d+tolerance, "request %d: TTFT %s", i, ttft)
		require.GreaterOrEqual(t, total, d+tailDelay, "request %d: total %s", i, total)
		require.Less(t, total, d+tailDelay+tolerance, "request %d: total %s", i, total)
		require.Equal(t, "ok", ev.Status)
		require.Zero(t, ev.HTTPStatus)
		require.Equal(t, int64(30), ev.PromptTokens, "cỡ prompt thật gồm cả phần cache")
		require.Equal(t, int64(7), ev.CompletionTokens)
		require.Empty(t, ev.Kind, "Kind/Attempt/Retry là việc của engine")
	}
}

// Lỗi được ghi theo lớp của TransientError: hết hạn mức (không có byte nào: không TTFT),
// stream im quá request_timeout (sau byte đầu: có TTFT), 5xx, lỗi không tạm thời, và huỷ
// từ phía người gọi.
func TestRequestTimeoutModel_ObservesErrorClasses(t *testing.T) {
	t.Parallel()
	errorBody := func(status int, msg string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			fmt.Fprintf(w, `{"error":{"message":%q,"type":"error"}}`, msg)
		}
	}
	stall := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		sseChunk(w, chunkText)
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}
	for _, tc := range []struct {
		name     string
		h        http.HandlerFunc
		cancel   bool
		status   string
		httpCode int
		ttft     bool
	}{
		{name: "quota", h: errorBody(http.StatusTooManyRequests, "Usage limit reached for 5 hour"), status: loop.ClassQuota, httpCode: 429},
		{name: "rate limit", h: errorBody(http.StatusTooManyRequests, "Rate limit reached for requests"), status: "rate_limit", httpCode: 429},
		{name: "server error", h: errorBody(http.StatusServiceUnavailable, "overloaded"), status: "server_error", httpCode: 503},
		{name: "auth", h: errorBody(http.StatusUnauthorized, "bad key"), status: "other", httpCode: 401},
		{name: "stall", h: stall, status: loop.ClassStreamStall, ttft: true},
		{name: "cancelled by the caller", h: stall, cancel: true, status: "cancelled", ttft: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			timeout := 300 * time.Millisecond
			if tc.cancel {
				timeout = time.Minute
			}
			m := fakeOpenAI(t, timeout, tc.h)
			log := &requestLog{}
			ctx, cancel := context.WithCancel(loop.WithRequestObserver(t.Context(), log.observe))
			defer cancel()
			if tc.cancel {
				time.AfterFunc(300*time.Millisecond, cancel)
			}
			drain(ctx, m)
			events := log.all()
			require.Len(t, events, 1)
			require.Equal(t, tc.status, events[0].Status)
			require.Equal(t, tc.httpCode, events[0].HTTPStatus)
			require.Equal(t, tc.ttft, events[0].TTFTMS != nil)
			require.GreaterOrEqual(t, events[0].TotalMS, int64(0))
		})
	}
}

// Lần tự thử lại của fantasy gọi lại Stream của model đã bọc, nên mỗi lần thử là một
// dòng riêng (dòng lỗi rồi dòng ok), không phải vô hình.
func TestRequestTimeoutModel_FantasyRetriesAreSeparateRequests(t *testing.T) {
	t.Parallel()
	var n atomic.Int32
	m := fakeOpenAI(t, time.Minute, func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After-Ms", "10")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"error":{"message":"overloaded","type":"error"}}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		sseChunk(w, chunkText)
		sseChunk(w, chunkFinish)
		sseChunk(w, chunkUsage)
		sseChunk(w, "[DONE]")
	})
	log := &requestLog{}
	ctx := loop.WithRequestObserver(t.Context(), log.observe)
	_, err := fantasy.NewAgent(m).Stream(ctx, fantasy.AgentStreamCall{Prompt: "hi"})
	require.NoError(t, err)
	events := log.all()
	require.Len(t, events, 2)
	require.Equal(t, "server_error", events[0].Status)
	require.Equal(t, 503, events[0].HTTPStatus)
	require.Equal(t, "ok", events[1].Status)
}

// Ngoài vòng lặp (TUI, run) ctx không có observer: không ghi gì, request vẫn chạy như cũ.
func TestRequestTimeoutModel_NoObserverOutsideALoop(t *testing.T) {
	t.Parallel()
	m := fakeOpenAI(t, time.Minute, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		sseChunk(w, chunkText)
		sseChunk(w, chunkFinish)
		sseChunk(w, "[DONE]")
	})
	stream, err := m.Stream(t.Context(), fantasy.Call{Prompt: fantasy.Prompt{fantasy.NewUserMessage("hi")}})
	require.NoError(t, err)
	var text string
	for part := range stream {
		text += part.Delta
	}
	require.Equal(t, "hi", text)
}

// Stream thật qua HTTP im sau byte đầu: net/http trả thẳng context.Cause(ctx), tức chính
// requestTimeoutError. Lỗi đó không được tự làm cause của mình (vòng Unwrap làm errors.Is/As
// chạy mãi: TransientError treo, fantasy treo khi xét có thử lại không).
func TestRequestTimeoutModel_RealStallErrorHasNoUnwrapCycle(t *testing.T) {
	t.Parallel()
	m := fakeOpenAI(t, 300*time.Millisecond, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		sseChunk(w, chunkText)
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	})
	stream, err := m.Stream(t.Context(), fantasy.Call{Prompt: fantasy.Prompt{fantasy.NewUserMessage("hi")}})
	require.NoError(t, err)
	var streamErr error
	for part := range stream {
		if part.Error != nil && streamErr == nil {
			streamErr = part.Error
		}
	}
	require.Error(t, streamErr)
	// Kiểm vòng bằng tay trước (không treo khi hỏng), rồi mới gọi hàm dùng errors.Is/As.
	seen := map[error]bool{}
	for e := streamErr; e != nil; e = errors.Unwrap(e) {
		require.False(t, seen[e], "vòng Unwrap ở %T", e)
		seen[e] = true
	}
	class, _ := TransientError(streamErr)
	require.Equal(t, loop.ClassStreamStall, class)
	require.ErrorIs(t, streamErr, context.DeadlineExceeded)
	require.Contains(t, streamErr.Error(), "LLM stream received no data for 300ms")
}

// Người gọi huỷ ctx rồi thôi đọc stream trước khi phần lỗi nào tới: request bị cắt từ phía
// người gọi là "cancelled", không phải "ok".
func TestRequestTimeoutModel_EarlyStopAfterCallerCancelIsCancelled(t *testing.T) {
	t.Parallel()
	m := fakeOpenAI(t, time.Minute, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		sseChunk(w, chunkText)
		<-r.Context().Done()
	})
	log := &requestLog{}
	ctx, cancel := context.WithCancel(loop.WithRequestObserver(t.Context(), log.observe))
	defer cancel()
	stream, err := m.Stream(ctx, fantasy.Call{Prompt: fantasy.Prompt{fantasy.NewUserMessage("hi")}})
	require.NoError(t, err)
	for part := range stream {
		require.NoError(t, part.Error)
		cancel()
		break
	}
	events := log.all()
	require.Len(t, events, 1)
	require.Equal(t, "cancelled", events[0].Status)
}
