package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/notify"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

func readLedger(t *testing.T, path string) []ledgerEntry {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	var out []ledgerEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e ledgerEntry
		require.NoError(t, json.Unmarshal(sc.Bytes(), &e))
		out = append(out, e)
	}
	return out
}

// Sổ ghi nói đúng model nào đã nhận lời gọi nào, với vai trò của lời gọi, và
// usage của CHÍNH lời gọi đó; không chứa nội dung prompt.
func TestLedgerRecordsEachCallWithRoleProviderAndUsage(t *testing.T) {
	env := testEnv(t)
	broker := pubsub.NewBroker[notify.RunComplete]()
	t.Cleanup(broker.Shutdown)
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	cw := catwalk.Model{ContextWindow: 1000, DefaultMaxTokens: 100}
	large := modelWith(newLedgerModel(&scriptModel{script: []string{"tool_small", "done"}, summaryText: "S"}, "qwen", "qwen3-coder", path), "qwen", "qwen3-coder", cw)
	small := modelWith(newLedgerModel(&finishStreamModel{text: "title"}, "glm", "glm-5.3-flash", path), "glm", "glm-5.3-flash", cw)
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel: large, SmallModel: small, IsYolo: true,
		Sessions: env.sessions, Messages: env.messages, RunComplete: broker,
		Tools: []fantasy.AgentTool{namedTool("noop")},
	}).(*sessionAgent)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ch := broker.Subscribe(t.Context())

	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "x", Prompt: "SECRET-PROMPT-TEXT fix it"})
	require.NoError(t, err)
	waitRunComplete(t, ch, "x")
	sa.GenerateTitle(t.Context(), sess.ID, "SECRET-PROMPT-TEXT fix it")

	entries := readLedger(t, path)
	byRole := map[string][]ledgerEntry{}
	for _, e := range entries {
		byRole[e.Role] = append(byRole[e.Role], e)
		require.NotEmpty(t, e.Time)
	}
	require.Len(t, byRole["main"], 2, "two main steps: tool, then done")
	for _, e := range byRole["main"] {
		require.Equal(t, "qwen", e.Provider)
		require.Equal(t, "qwen3-coder", e.Model)
		require.Equal(t, sess.ID, e.SessionID)
		require.True(t, e.Stream)
		require.Equal(t, "noop", e.ToolNames, "the tool palette offered to the model")
	}
	require.EqualValues(t, 10, byRole["main"][0].InputTokens, "usage of that call, not the session total")
	require.NotEmpty(t, byRole["title"])
	require.Equal(t, "glm", byRole["title"][0].Provider, "the ledger shows the cross-provider title call")

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "SECRET-PROMPT-TEXT", "sizes and hashes only, never prompt text")
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

type failingStreamModel struct{ finishStreamModel }

func (*failingStreamModel) Stream(context.Context, fantasy.Call) (fantasy.StreamResponse, error) {
	return nil, &fantasy.ProviderError{StatusCode: 400, Message: "Error rendering prompt: SECRET-PROMPT-TEXT"}
}

// Server OpenAI-compat local (llama.cpp, LM Studio) hay chép mẩu prompt vào thông
// báo lỗi: sổ ghi chỉ giữ loại lỗi và mã HTTP, không giữ văn bản lỗi.
func TestLedgerRecordsTheErrorKindNotItsText(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	m := newLedgerModel(&failingStreamModel{}, "local", "qwen", path)
	_, err := m.Stream(t.Context(), fantasy.Call{})
	require.Error(t, err)

	entries := readLedger(t, path)
	require.Len(t, entries, 1)
	require.Equal(t, "provider_error status=400", entries[0].Error)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "SECRET-PROMPT-TEXT")
}

// Tắt (path rỗng) thì không bọc gì: không chi phí, không tệp.
func TestLedgerOffReturnsTheModelUnchanged(t *testing.T) {
	t.Parallel()
	m := &finishStreamModel{text: "x"}
	require.Same(t, fantasy.LanguageModel(m), newLedgerModel(m, "p", "m", ""))
}

// ledgerWarnRecorder đếm bản ghi slog mức Warn trở lên nói về sổ ghi.
type ledgerWarnRecorder struct {
	mu sync.Mutex
	n  int
}

func (h *ledgerWarnRecorder) Enabled(_ context.Context, l slog.Level) bool {
	return l >= slog.LevelWarn
}

func (h *ledgerWarnRecorder) Handle(_ context.Context, r slog.Record) error {
	if strings.Contains(r.Message, "ledger") {
		h.mu.Lock()
		h.n++
		h.mu.Unlock()
	}
	return nil
}
func (h *ledgerWarnRecorder) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *ledgerWarnRecorder) WithGroup(string) slog.Handler      { return h }
func (h *ledgerWarnRecorder) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.n
}

// Sổ ghi bật mà ghi lỗi (đĩa đầy; /dev/full): mỗi lượt bị ảnh hưởng có đúng một
// cảnh báo (không phải một cảnh báo mỗi lời gọi), lỗi đi theo RunComplete và
// LedgerWriteError của session cho run --json, và lượt vẫn chạy xong. Tệp ghi
// được thì im lặng. Đổi slog mặc định nên không chạy song song.
func TestLedgerWriteFailureIsReportedOncePerTurn(t *testing.T) {
	f, err := os.OpenFile("/dev/full", os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("no /dev/full: %v", err)
	}
	_ = f.Close()
	for _, tc := range []struct {
		name, path string
		fails      bool
	}{
		{"writable", filepath.Join(t.TempDir(), "ledger.jsonl"), false},
		{"dev-full", "/dev/full", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := &ledgerWarnRecorder{}
			prev := slog.Default()
			slog.SetDefault(slog.New(rec))
			t.Cleanup(func() { slog.SetDefault(prev) })
			env := testEnv(t)
			broker := pubsub.NewBroker[notify.RunComplete]()
			t.Cleanup(broker.Shutdown)
			cw := catwalk.Model{ContextWindow: 100000, DefaultMaxTokens: 100}
			// Hai lời gọi mỗi lượt (tool rồi trả lời): cả hai cùng ghi lỗi.
			script := &scriptModel{script: []string{"tool_small", "done", "tool_small", "done"}}
			sa := NewSessionAgent(SessionAgentOptions{
				LargeModel: modelWith(newLedgerModel(script, "prov", "model", tc.path), "prov", "model", cw),
				SmallModel: modelWith(&finishStreamModel{text: "title"}, "prov", "small", cw),
				IsYolo:     true, Sessions: env.sessions, Messages: env.messages, RunComplete: broker,
				Tools: []fantasy.AgentTool{namedTool("noop")},
			}).(*sessionAgent)
			sess, err := env.sessions.Create(t.Context(), "session")
			require.NoError(t, err)
			ch := broker.Subscribe(t.Context())
			for turn := 1; turn <= 2; turn++ {
				before := rec.count()
				runID := fmt.Sprintf("%s-%d", tc.name, turn)
				_, err := sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: runID, Prompt: "go " + runID})
				require.NoError(t, err, "a broken ledger must not fail the turn")
				rc := waitRunComplete(t, ch, runID)
				require.Empty(t, rc.Error)
				if !tc.fails {
					require.Zero(t, rec.count()-before)
					require.Empty(t, rc.LedgerError)
					require.Empty(t, LedgerWriteError(sess.ID))
					continue
				}
				require.Equal(t, 1, rec.count()-before, "turn %d: one warning per affected turn, not one per call", turn)
				require.Contains(t, rc.LedgerError, "no space left on device")
				require.Equal(t, rc.LedgerError, LedgerWriteError(sess.ID))
			}
			if !tc.fails {
				require.Len(t, readLedger(t, tc.path), 4, "both calls of both turns reach the ledger")
			}
		})
	}
}

// Prompt xếp hàng sau lượt đang chạy: RunComplete của lượt trước do handOff phát
// (trước khi lượt sau bắt đầu và xoá lỗi), nên cả hai lượt đều mang lỗi ghi sổ.
func TestLedgerWriteErrorReachesRunCompleteOfAHandedOffTurn(t *testing.T) {
	t.Parallel()
	f, err := os.OpenFile("/dev/full", os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("no /dev/full: %v", err)
	}
	_ = f.Close()
	env := testEnv(t)
	broker := pubsub.NewBroker[notify.RunComplete]()
	t.Cleanup(broker.Shutdown)
	large := &gatedStreamModel{text: "done", gate: make(chan struct{}), entered: make(chan struct{})}
	cw := catwalk.Model{ContextWindow: 200000, DefaultMaxTokens: 10000}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel: modelWith(newLedgerModel(large, "prov", "model", "/dev/full"), "prov", "model", cw),
		SmallModel: modelWith(&finishStreamModel{text: "title"}, "prov", "small", cw),
		IsYolo:     true, Sessions: env.sessions, Messages: env.messages, RunComplete: broker,
	}).(*sessionAgent)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ch := broker.Subscribe(t.Context())

	mainDone := make(chan error, 1)
	go func() {
		_, err := sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "first", Prompt: "first"})
		mainDone <- err
	}()
	waitEntered(t, large.entered)
	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "queued", Prompt: "queued"})
	require.NoError(t, err)
	require.Equal(t, 1, sa.QueuedPrompts(sess.ID))
	close(large.gate)
	require.NoError(t, <-mainDone)

	for _, runID := range []string{"first", "queued"} {
		rc := waitRunComplete(t, ch, runID)
		require.Contains(t, rc.LedgerError, "no space left on device", "run %s", runID)
	}
}

// fixLedgerBeforeSecondCall đổi tệp sổ của path sang tệp ghi được ngay trước lời
// gọi thứ hai: chỉ lời gọi đầu ghi lỗi (lỗi thoáng qua).
type fixLedgerBeforeSecondCall struct {
	fantasy.LanguageModel
	path  string
	calls atomic.Int64
}

func (m *fixLedgerBeforeSecondCall) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	if m.calls.Add(1) == 2 {
		good, err := os.OpenFile(m.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
		if err != nil {
			return nil, err
		}
		ledgerMu.Lock()
		bad := ledgerFiles[m.path]
		ledgerFiles[m.path] = good
		ledgerMu.Unlock()
		_ = bad.Close()
	}
	return m.LanguageModel.Stream(ctx, call)
}

// Lượt tự làm tiếp (trả lời bị cắt vì hết output) là phần của cùng một prompt:
// lỗi ghi sổ của đoạn trước phải còn trong RunComplete cuối, kể cả khi đoạn sau
// ghi được.
func TestLedgerWriteErrorSurvivesTheTurnsOwnContinuation(t *testing.T) {
	t.Parallel()
	full, err := os.OpenFile("/dev/full", os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("no /dev/full: %v", err)
	}
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	ledgerMu.Lock()
	ledgerFiles[path] = full
	ledgerMu.Unlock()
	t.Cleanup(func() {
		ledgerMu.Lock()
		f := ledgerFiles[path]
		delete(ledgerFiles, path)
		ledgerMu.Unlock()
		_ = f.Close()
	})
	env := testEnv(t)
	broker := pubsub.NewBroker[notify.RunComplete]()
	t.Cleanup(broker.Shutdown)
	script := &scriptModel{script: []string{"cut", "done"}}
	cw := catwalk.Model{ContextWindow: 100000, DefaultMaxTokens: 100}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel: modelWith(newLedgerModel(&fixLedgerBeforeSecondCall{LanguageModel: script, path: path}, "prov", "model", path), "prov", "model", cw),
		SmallModel: modelWith(&finishStreamModel{text: "title"}, "prov", "small", cw),
		IsYolo:     true, Sessions: env.sessions, Messages: env.messages, RunComplete: broker,
	}).(*sessionAgent)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ch := broker.Subscribe(t.Context())

	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "r", Prompt: "write it"})
	require.NoError(t, err)
	rc := waitRunComplete(t, ch, "r")
	kinds, _ := script.seen()
	require.Equal(t, []string{"work", "work"}, kinds, "the cut reply was continued in a second call")
	require.Len(t, readLedger(t, path), 1, "only the continuation reached the ledger")
	require.Contains(t, rc.LedgerError, "no space left on device")
	require.Equal(t, rc.LedgerError, LedgerWriteError(sess.ID))
}
