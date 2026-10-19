package agent

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/notify"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

// titleModel đếm số lời gọi; trả text với finish đã cho, hoặc lỗi.
type titleModel struct {
	fastModel
	calls  atomic.Int64
	text   string
	finish fantasy.FinishReason
	err    error
}

func (m *titleModel) Stream(context.Context, fantasy.Call) (fantasy.StreamResponse, error) {
	m.calls.Add(1)
	if m.err != nil {
		return nil, m.err
	}
	return textStream(m.text, m.finish, fantasy.Usage{InputTokens: 200, OutputTokens: 40}), nil
}

func titleAgent(t *testing.T, small, large fantasy.LanguageModel) (*sessionAgent, fakeEnv) {
	t.Helper()
	env := testEnv(t)
	cw := catwalk.Model{ContextWindow: 200000, DefaultMaxTokens: 1000}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel: modelWith(large, "prov", "large", cw), SmallModel: modelWith(small, "prov", "small", cw),
		IsYolo: true, Sessions: env.sessions, Messages: env.messages,
	}).(*sessionAgent)
	return sa, env
}

// Bị cắt vì hết token: model suy luận đã dùng hết ngân sách; thử model large
// chỉ tốn thêm một lời gọi (baseline thật: 26/26 lời gọi tiêu đề `length`).
func TestTitleStopsAfterTokenLimit(t *testing.T) {
	t.Parallel()
	small := &titleModel{text: "<think>let me", finish: fantasy.FinishReasonLength}
	large := &titleModel{text: "Large Title", finish: fantasy.FinishReasonStop}
	sa, env := titleAgent(t, small, large)
	sess, err := env.sessions.Create(t.Context(), DefaultSessionName)
	require.NoError(t, err)

	sa.GenerateTitle(t.Context(), sess.ID, "fix the login bug")

	require.EqualValues(t, 1, small.calls.Load())
	require.EqualValues(t, 0, large.calls.Load(), "no second call after a token-limit cut")
	saved, err := env.sessions.Get(t.Context(), sess.ID)
	require.NoError(t, err)
	require.Equal(t, "fix the login bug", saved.Title, "the prompt-derived title stays")
}

// Lỗi provider ở model small thì vẫn thử model large (khác model).
func TestTitleRetriesTheLargeModelAfterAnError(t *testing.T) {
	t.Parallel()
	small := &titleModel{err: errors.New("small provider down")}
	large := &titleModel{text: "Large Title", finish: fantasy.FinishReasonStop}
	sa, env := titleAgent(t, small, large)
	sess, err := env.sessions.Create(t.Context(), DefaultSessionName)
	require.NoError(t, err)

	sa.GenerateTitle(t.Context(), sess.ID, "fix the login bug")

	require.EqualValues(t, 1, large.calls.Load())
	saved, err := env.sessions.Get(t.Context(), sess.ID)
	require.NoError(t, err)
	require.Equal(t, "Large Title", saved.Title)
}

// Session con (agent con) ẩn khỏi danh sách: không có lời gọi LLM đặt tên.
func TestNoTitleCallForChildSessions(t *testing.T) {
	t.Parallel()
	small := &titleModel{text: "Child Title", finish: fantasy.FinishReasonStop}
	env := testEnv(t)
	broker := pubsub.NewBroker[notify.RunComplete]()
	t.Cleanup(broker.Shutdown)
	cw := catwalk.Model{ContextWindow: 200000, DefaultMaxTokens: 1000}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel: modelWith(&finishStreamModel{text: "done"}, "prov", "large", cw), SmallModel: modelWith(small, "prov", "small", cw),
		IsYolo: true, Sessions: env.sessions, Messages: env.messages, RunComplete: broker,
	}).(*sessionAgent)
	parent, err := env.sessions.Create(t.Context(), "parent")
	require.NoError(t, err)
	child, err := env.sessions.CreateTaskSession(t.Context(), "msg$$call", parent.ID, "New Agent Session")
	require.NoError(t, err)
	ch := broker.Subscribe(t.Context())

	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: child.ID, RunID: "c", Prompt: "list the files"})
	require.NoError(t, err)
	waitRunComplete(t, ch, "c")
	// Tiêu đề chạy trong goroutine riêng: đợi một lúc để chắc không có lời gọi muộn.
	require.Never(t, func() bool { return small.calls.Load() > 0 }, 300*time.Millisecond, 10*time.Millisecond, "child sessions get no LLM title")
}
