package agent

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/stretchr/testify/require"
)

// Prompt đã ghi lúc nhận mà bị cancel mark phủ khi còn trong hàng đợi (hai
// đường bỏ theo mark: bước gộp drainQueueForStep và nextQueued lúc chuyển
// lượt) thì dấu vết của nó được đóng là huỷ, không bị bỏ im lặng.
func TestQueueMarkDropClosesRecordedPrompt(t *testing.T) {
	t.Parallel()
	for name, drop := range map[string]func(*sessionAgent, string){
		"drainQueueForStep": func(sa *sessionAgent, sid string) {
			fold, canceled := sa.drainQueueForStep(sid, config.SelectedModel{})
			require.Empty(t, fold)
			require.Len(t, canceled, 1)
			sa.publishCanceledQueueDrops(canceled)
		},
		"nextQueued": func(sa *sessionAgent, sid string) {
			_, _, ok := sa.nextQueued(sid)
			require.False(t, ok)
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			sa, env := newCancelTestAgent(t)
			sess, err := env.sessions.Create(t.Context(), "session")
			require.NoError(t, err)

			accept := sa.BeginAccepted(sess.ID)
			call := SessionAgentCall{SessionID: sess.ID, Prompt: "queued"}
			require.NoError(t, accept.Record(t.Context(), call))
			call.recorded, call.Accepted = accept.recorded, accept
			sa.enqueueCall(call)
			accept.Close()
			sa.cancelMark.Set(sess.ID, accept.seq)

			drop(sa, sess.ID)

			require.Zero(t, sa.QueuedPrompts(sess.ID))
			msgs, err := env.messages.List(t.Context(), sess.ID)
			require.NoError(t, err)
			require.Len(t, msgs, 2)
			require.Equal(t, message.User, msgs[0].Role)
			require.Equal(t, "queued", msgs[0].Content().Text)
			require.Equal(t, message.Assistant, msgs[1].Role)
			require.Equal(t, message.FinishReasonCanceled, msgs[1].FinishReason())
		})
	}
}

// ackRecorder là LLM giả ghi chữ các tin user của từng request rồi trả "done".
type ackRecorder struct {
	mu    sync.Mutex
	users [][]string
}

func (m *ackRecorder) Provider() string { return "fake" }
func (m *ackRecorder) Model() string    { return "fake-model" }
func (m *ackRecorder) Generate(context.Context, fantasy.Call) (*fantasy.Response, error) {
	return &fantasy.Response{Content: fantasy.ResponseContent{fantasy.TextContent{Text: "title"}}, FinishReason: fantasy.FinishReasonStop}, nil
}

func (m *ackRecorder) GenerateObject(context.Context, fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	return nil, errors.New("not supported")
}

func (m *ackRecorder) StreamObject(context.Context, fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return nil, errors.New("not supported")
}

func (m *ackRecorder) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	var users []string
	for _, msg := range call.Prompt {
		if msg.Role != fantasy.MessageRoleUser {
			continue
		}
		var b strings.Builder
		for _, part := range msg.Content {
			if tp, ok := fantasy.AsMessagePart[fantasy.TextPart](part); ok {
				b.WriteString(tp.Text)
			}
		}
		users = append(users, b.String())
	}
	m.mu.Lock()
	m.users = append(m.users, users)
	m.mu.Unlock()
	return (&finishStreamModel{text: "done"}).Stream(ctx, call)
}

// prompts trả, theo từng request, các chữ user nằm trong want (bỏ lời nhắc
// hệ thống chèn vào lịch sử).
func (m *ackRecorder) prompts(want ...string) [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([][]string, len(m.users))
	for i, r := range m.users {
		out[i] = []string{}
		for _, x := range r {
			if slices.Contains(want, x) {
				out[i] = append(out[i], x)
			}
		}
	}
	return out
}

func newAckAgent(t *testing.T, messages message.Service) (*sessionAgent, fakeEnv, *ackRecorder) {
	t.Helper()
	env := testEnv(t)
	if messages == nil {
		messages = env.messages
	}
	model := &ackRecorder{}
	cfg := catwalk.Model{ContextWindow: 200000, DefaultMaxTokens: 10000}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel:           Model{Model: model, CatwalkCfg: cfg},
		SmallModel:           Model{Model: &finishStreamModel{text: "title"}, CatwalkCfg: cfg},
		DisableAutoSummarize: true,
		IsYolo:               true,
		Sessions:             env.sessions,
		Messages:             messages,
	}).(*sessionAgent)
	return sa, env, model
}

// accept nhận và ghi một prompt như backend.SendMessage làm.
func accept(t *testing.T, sa *sessionAgent, sid, prompt string) SessionAgentCall {
	t.Helper()
	a := sa.BeginAccepted(sid)
	call := SessionAgentCall{SessionID: sid, Prompt: prompt, Accepted: a}
	require.NoError(t, a.Record(t.Context(), call))
	return call
}

// Prompt B được ghi (đã ack) trước khi lượt của A đọc lịch sử (server mới
// khởi động, A còn chờ readiness): B không được vào request của A, và A vẫn
// là prompt đầu nên session được đặt tiêu đề. B chạy sau thấy A rồi B.
func TestRecordedPromptStaysOutOfEarlierTurn(t *testing.T) {
	t.Parallel()
	sa, env, model := newAckAgent(t, nil)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)

	a := accept(t, sa, sess.ID, "prompt A")
	b := accept(t, sa, sess.ID, "prompt B")
	_, err = sa.Run(t.Context(), a)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		s, err := env.sessions.Get(t.Context(), sess.ID)
		return err == nil && s.Title != "session"
	}, 5*time.Second, 10*time.Millisecond, "the first prompt must still get a session title")
	_, err = sa.Run(t.Context(), b)
	require.NoError(t, err)

	require.Equal(t, [][]string{{"prompt A"}, {"prompt A", "prompt B"}}, model.prompts("prompt A", "prompt B"))
	msgs, err := env.messages.List(t.Context(), sess.ID)
	require.NoError(t, err)
	var order []string
	for _, m := range msgs {
		order = append(order, string(m.Role)+":"+m.Content().Text)
	}
	require.Equal(t, []string{"user:prompt A", "assistant:done", "user:prompt B", "assistant:done"}, order)
}

// Tóm tắt không được chép prompt còn chờ (đã ack, chưa chạy) vào bản tóm
// tắt; prompt ghi sau tin tóm tắt cũng không vào request của lượt khác.
func TestSummarizeLeavesOutPendingPrompts(t *testing.T) {
	t.Parallel()
	sa, env, model := newAckAgent(t, nil)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, Prompt: "prompt A"})
	require.NoError(t, err)

	accept(t, sa, sess.ID, "pending B")
	require.NoError(t, sa.Summarize(t.Context(), sess.ID, nil, nil))
	accept(t, sa, sess.ID, "pending C")
	d := accept(t, sa, sess.ID, "prompt D")
	_, err = sa.Run(t.Context(), d)
	require.NoError(t, err)

	got := model.prompts("prompt A", "pending B", "pending C", "prompt D")
	require.Len(t, got, 3)
	require.Equal(t, []string{"prompt A"}, got[1], "the summary request must not carry a pending prompt")
	require.Equal(t, []string{"prompt D"}, got[2], "a turn must not carry another pending prompt")
}

// failingListMessages làm ListFromSummary lỗi: lượt hỏng trước khi có lời trả lời.
type failingListMessages struct {
	message.Service
}

func (m failingListMessages) ListFromSummary(context.Context, string, string) ([]message.Message, error) {
	return nil, errors.New("disk on fire")
}

// Lượt có prompt đã ghi hỏng trước khi có lời trả lời: tin giữ chỗ phải đóng
// bằng lỗi thật, không treo chờ rồi thành "Interrupted".
func TestRunEarlyErrorClosesRecordedPrompt(t *testing.T) {
	t.Parallel()
	env := testEnv(t)
	sa, _, _ := newAckAgent(t, failingListMessages{env.messages})
	sa.sessions = env.sessions
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)

	a := accept(t, sa, sess.ID, "prompt A")
	_, err = sa.Run(t.Context(), a)
	require.ErrorContains(t, err, "disk on fire")

	msgs, err := env.messages.List(t.Context(), sess.ID)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	f := msgs[1].FinishPart()
	require.NotNil(t, f, "the pending reply must be closed")
	require.Equal(t, message.FinishReasonError, f.Reason)
	require.Contains(t, f.Details, "disk on fire")
}

// lockProbeMessages kiểm, ở mỗi lần ghi Update, khoá dispatch của session
// có đang bị giữ không (bất biến: không IO DB dưới khoá này).
type lockProbeMessages struct {
	message.Service
	probe func()
}

func (m lockProbeMessages) Update(ctx context.Context, msg message.Message) error {
	m.probe()
	return m.Service.Update(ctx, msg)
}

// Đóng dấu vết của prompt bị bỏ khỏi hàng đợi là ghi DB: phải làm ngoài khoá
// dispatch của session, ở cả bốn đường bỏ.
func TestQueueDropWritesOutsideDispatchLock(t *testing.T) {
	t.Parallel()
	for name, drop := range map[string]func(*sessionAgent, string){
		"Cancel":     func(sa *sessionAgent, sid string) { sa.Cancel(sid) },
		"ClearQueue": func(sa *sessionAgent, sid string) { sa.ClearQueue(sid) },
		"CancelRun":  func(sa *sessionAgent, sid string) { sa.CancelRun(sid, "b") },
		"nextQueued": func(sa *sessionAgent, sid string) {
			sa.cancelMark.Set(sid, sa.acceptSeqGen)
			_, _, ok := sa.nextQueued(sid)
			require.False(t, ok)
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := testEnv(t)
			var sa *sessionAgent
			var sid string
			var held, probed atomic.Bool
			sa, _, _ = newAckAgent(t, lockProbeMessages{Service: env.messages, probe: func() {
				mu := sa.sessionMu(sid)
				if mu.TryLock() {
					mu.Unlock()
				} else {
					held.Store(true)
				}
				probed.Store(true)
			}})
			sa.sessions = env.sessions
			sess, err := env.sessions.Create(t.Context(), "session")
			require.NoError(t, err)
			sid = sess.ID

			b := accept(t, sa, sid, "queued B")
			b.RunID = "b"
			b.recorded = b.Accepted.recorded
			sa.enqueueCall(b)
			b.Accepted.Close()

			drop(sa, sid)

			require.True(t, probed.Load(), "the dropped prompt was not closed")
			require.False(t, held.Load(), "DB write under the session dispatch lock")
		})
	}
}

// busyOf đưa một sessionAgent vào SettleBusy như coordinator thật làm.
type busyOf struct {
	Coordinator
	sa *sessionAgent
}

func (c busyOf) IsSessionBusy(id string) bool     { return c.sa.IsSessionBusy(id) }
func (c busyOf) HasPendingPrompts(id string) bool { return c.sa.HasPendingPrompts(id) }

// Settle không đóng tin giữ chỗ của prompt còn sống (đã nhận chưa active,
// hay đang xếp hàng khi không có lượt nào active), nhưng vẫn đóng của prompt
// mà tiến trình hiện tại không còn giữ (tiến trình trước đã chết).
func TestSettleBusyKeepsLivePromptsOpen(t *testing.T) {
	t.Parallel()
	sa, env, _ := newAckAgent(t, nil)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	pendingFinished := func(busy SessionBusyFunc) bool {
		t.Helper()
		msgs, err := ReadSettledMessages(t.Context(), env.messages, busy, sess.ID)
		require.NoError(t, err)
		require.Len(t, msgs, 2)
		return msgs[1].IsFinished()
	}

	b := accept(t, sa, sess.ID, "queued B")
	require.False(t, pendingFinished(SettleBusy(busyOf{sa: sa})), "accepted, not yet active")
	b.recorded = b.Accepted.recorded
	sa.enqueueCall(b)
	b.Accepted.Close()
	require.False(t, sa.IsSessionBusy(sess.ID))
	require.False(t, pendingFinished(SettleBusy(busyOf{sa: sa})), "queued between turns")

	restarted, _, _ := newAckAgent(t, env.messages)
	require.True(t, pendingFinished(SettleBusy(busyOf{sa: restarted})), "a prompt no live process holds is closed")
}

// pendingPair ghi một tin của prompt đã ghi theo đúng dạng lưu trên đĩa (tin
// user; tin giữ chỗ: provider "agent-cli-pending", model = ID tin user), để
// dựng thẳng các thứ tự xen kẽ mà ghi đồng thời có thể tạo ra.
func pendingUser(t *testing.T, ms message.Service, sid, prompt string) message.Message {
	t.Helper()
	m, err := ms.Create(t.Context(), sid, message.CreateMessageParams{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: prompt}}})
	require.NoError(t, err)
	return m
}

func pendingReply(t *testing.T, ms message.Service, sid string, user message.Message) message.Message {
	t.Helper()
	m, err := ms.Create(t.Context(), sid, message.CreateMessageParams{Role: message.Assistant, Parts: []message.ContentPart{}, Provider: "agent-cli-pending", Model: user.ID})
	require.NoError(t, err)
	return m
}

// Hai `send` đồng thời trên một kết nối DB có thể ra thứ tự [U_A, U_B, P_A,
// P_B]: luật lọc không được dựa vào vị trí. Mỗi request chứa mỗi prompt đúng
// một lần, đúng thứ tự.
func TestUnsentPromptsInterleavedPairs(t *testing.T) {
	t.Parallel()
	sa, env, model := newAckAgent(t, nil)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ua := pendingUser(t, env.messages, sess.ID, "prompt A")
	ub := pendingUser(t, env.messages, sess.ID, "prompt B")
	pa := pendingReply(t, env.messages, sess.ID, ua)
	pb := pendingReply(t, env.messages, sess.ID, ub)

	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, Prompt: "prompt A", recorded: &recordedPrompt{ua.ID, pa.ID}})
	require.NoError(t, err)
	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, Prompt: "prompt B", recorded: &recordedPrompt{ub.ID, pb.ID}})
	require.NoError(t, err)

	require.Equal(t, [][]string{{"prompt A"}, {"prompt A", "prompt B"}}, model.prompts("prompt A", "prompt B"))
	msgs, err := env.messages.List(t.Context(), sess.ID)
	require.NoError(t, err)
	var order []string
	for _, m := range msgs {
		order = append(order, string(m.Role)+":"+m.Content().Text)
	}
	require.Equal(t, []string{"user:prompt A", "assistant:done", "user:prompt B", "assistant:done"}, order)
}

// Tin của lượt khác (tool result) chen giữa tin user và tin giữ chỗ của B; B
// bị huỷ khi còn chờ: B không bao giờ vào lịch sử gửi model.
func TestUnsentPromptWithToolResultBetween(t *testing.T) {
	t.Parallel()
	sa, env, model := newAckAgent(t, nil)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ub := pendingUser(t, env.messages, sess.ID, "prompt B")
	_, err = env.messages.Create(t.Context(), sess.ID, message.CreateMessageParams{Role: message.Tool, Parts: []message.ContentPart{
		message.ToolResult{ToolCallID: "tc1", Name: "noop", Content: "ok"},
	}})
	require.NoError(t, err)
	pb := pendingReply(t, env.messages, sess.ID, ub)
	pb.AddFinish(message.FinishReasonCanceled, "User canceled request", "")
	require.NoError(t, env.messages.Update(t.Context(), pb))

	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, Prompt: "prompt D"})
	require.NoError(t, err)
	require.Equal(t, [][]string{{"prompt D"}}, model.prompts("prompt B", "prompt D"))
}
