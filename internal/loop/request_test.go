package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func ms(v int64) *int64 { return &v }

// requestEvents là các dòng llm_request của journal.
func requestEvents(j *memJournal) []Event {
	var out []Event
	for _, e := range j.events {
		if e.Kind == "llm_request" {
			out = append(out, e)
		}
	}
	return out
}

// Mỗi request LLM mà lượt agent báo qua ctx thành một dòng llm_request mang đúng lượt
// và chỉ số lần chạy lại của dew; số đo của agent giữ nguyên (cả thời điểm gửi).
func TestEngine_RecordsEachLLMRequestWithAttemptAndRetry(t *testing.T) {
	t.Parallel()
	s := &script{
		verifies:     []VerifyResult{fail("b"), pass()},
		fingerprints: []string{"fp1", "fp2"},
		errs:         []error{fmt.Errorf("cut: %w", tempErr{ClassStreamStall, 0})},
	}
	eng := s.engine(baseSpec())
	sent := time.Unix(42, 0).UTC()
	inner := eng.Agent
	calls := 0
	eng.Agent = func(ctx context.Context, prompt string) (RunInfo, error) {
		// Agent giả của script kiểm attempt_begin là dòng cuối: chạy nó trước rồi mới báo request.
		info, err := inner(ctx, prompt)
		calls++
		observe := RequestObserver(ctx)
		require.NotNil(t, observe, "engine phải gắn observer cho mọi lượt")
		if calls == 1 {
			observe(Event{Time: sent, TTFTMS: ms(250), TotalMS: 900, PromptTokens: 1200, CompletionTokens: 40, Status: "ok"})
			observe(Event{Time: sent, TotalMS: 120000, Status: ClassStreamStall})
		} else {
			observe(Event{Time: sent, TTFTMS: ms(300), TotalMS: 1000, Status: "ok"})
		}
		return info, err
	}
	res, err := eng.Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopGreen, res.Reason)

	ev := requestEvents(s.journal)
	require.Len(t, ev, 3)
	require.Equal(t, Event{Time: sent, Kind: "llm_request", Attempt: 1, TTFTMS: ms(250), TotalMS: 900, PromptTokens: 1200, CompletionTokens: 40, Status: "ok"}, ev[0])
	require.Equal(t, Event{Time: sent, Kind: "llm_request", Attempt: 1, TotalMS: 120000, Status: ClassStreamStall}, ev[1])
	require.Equal(t, 1, ev[2].Attempt, "lần chạy lại giữ số lượt")
	require.Equal(t, 1, ev[2].Retry, "lần chạy lại thứ nhất của dew")
	require.Equal(t, 0, ev[0].Retry)

	// Thứ tự: dòng request của một lượt nằm giữa attempt_begin và agent_end của nó.
	require.Equal(t, []string{"start", "baseline", "attempt_begin", "llm_request", "llm_request", "agent_end",
		"attempt_begin", "llm_request", "agent_end", "verify", "stop"}, s.journal.kinds())
}

// Request còn chạy khi lượt đã trả về (tiêu đề sinh nền) không được ghi chen sau
// agent_end/stop: dòng cuối journal vẫn là stop.
func TestEngine_DropsRequestsReportedAfterTheTurnReturned(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("b"), pass()}, fingerprints: []string{"fp1"}}
	eng := s.engine(baseSpec())
	inner := eng.Agent
	var late func(Event)
	eng.Agent = func(ctx context.Context, prompt string) (RunInfo, error) {
		late = RequestObserver(ctx)
		return inner(ctx, prompt)
	}
	_, err := eng.Run(t.Context())
	require.NoError(t, err)
	late(Event{TotalMS: 5, Status: "ok"})
	require.Empty(t, requestEvents(s.journal))
	require.Equal(t, "stop", s.journal.events[len(s.journal.events)-1].Kind)
}

func TestRequestObserver_NilOutsideALoop(t *testing.T) {
	t.Parallel()
	require.Nil(t, RequestObserver(context.Background()))
}

// Dòng llm_request là JSON hợp lệ, đọc lại đủ trường; ttft_ms vắng khi không có byte nào tới.
func TestFileJournal_LLMRequestLinesAreValidJSON(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "j.jsonl")
	j, err := OpenFileJournal(path)
	require.NoError(t, err)
	want := Event{Time: time.Unix(7, 0).UTC(), Kind: "llm_request", Attempt: 2, Retry: 1, TTFTMS: ms(0), TotalMS: 3, PromptTokens: 5, CompletionTokens: 6, Status: "ok"}
	require.NoError(t, j.Append(want))
	require.NoError(t, j.Append(Event{Time: time.Unix(8, 0).UTC(), Kind: "llm_request", Attempt: 2, TotalMS: 9, Status: ClassQuota, HTTPStatus: 429}))
	require.NoError(t, j.Close())

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := splitLines(string(raw))
	require.Len(t, lines, 2)
	for _, l := range lines {
		require.True(t, json.Valid([]byte(l)), l)
	}
	require.Contains(t, lines[0], `"ttft_ms":0`, "TTFT 0 ms vẫn là số đo, không bị bỏ")
	require.NotContains(t, lines[1], "ttft_ms", "không có byte nào tới thì không có TTFT")
	events, err := ReadJournal(path)
	require.NoError(t, err)
	require.Equal(t, want, events[0])
	require.Equal(t, 429, events[1].HTTPStatus)
}

// overlapJournal đếm số lần hai Append chạy chồng nhau. Journal không cần an toàn đồng thời:
// engine tuần tự hoá các lần ghi của observer.
type overlapJournal struct {
	inner    *memJournal
	inFlight atomic.Int32
	overlaps atomic.Int32
}

func (j *overlapJournal) Append(e Event) error {
	if j.inFlight.Add(1) > 1 {
		j.overlaps.Add(1)
	}
	time.Sleep(time.Millisecond) // nới cửa sổ chồng nhau
	err := j.inner.Append(e)
	j.inFlight.Add(-1)
	return err
}

// Agent con và tiêu đề gọi observer từ nhiều goroutine cùng lúc: mọi dòng phải vào journal,
// không lần ghi nào chồng lần khác.
func TestEngine_ObserverCalledConcurrentlyWritesEveryLineSerially(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("b"), pass()}, fingerprints: []string{"fp1"}}
	eng := s.engine(baseSpec())
	oj := &overlapJournal{inner: s.journal}
	eng.Journal = oj
	inner := eng.Agent
	const workers, perWorker = 8, 5
	eng.Agent = func(ctx context.Context, prompt string) (RunInfo, error) {
		info, err := inner(ctx, prompt)
		observe := RequestObserver(ctx)
		var wg sync.WaitGroup
		for range workers {
			wg.Go(func() {
				for range perWorker {
					observe(Event{TotalMS: 1, Status: "ok"})
				}
			})
		}
		wg.Wait()
		return info, err
	}
	res, err := eng.Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopGreen, res.Reason)
	require.Len(t, requestEvents(s.journal), workers*perWorker)
	require.Zero(t, oj.overlaps.Load(), "Append bị gọi chồng nhau")
}
