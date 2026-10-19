package backend

// Bất biến P1-a ("ack ⇒ dấu vết bền"): Backend.SendMessage đã trả nil (ack
// cho `send`/TUI) thì prompt đã có tin user trong DB; prompt đó sau này không
// chạy được (server tắt hay chết, huỷ, xoá hàng đợi) thì khi đọc lại session
// dấu vết hiện đúng trạng thái (huỷ hoặc Interrupted), không biến mất im lặng.
// Đối chứng: luồng thường không ghi trùng tin user. Mọi test đi qua cửa ack
// thật (Backend.SendMessage) và đọc lại bằng đường đọc thật
// (agent.ReadSettledMessages, như `session show` của tiến trình mới).

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent"
	"github.com/charmbracelet/crush/internal/agent/agenttest"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ackModel là LLM giả: lần Stream đầu báo entered rồi chờ release (hoặc ctx
// huỷ); toolFirst thì lần đầu trả tool call "noop" để lượt có bước 2, nơi
// prompt xếp hàng không RunID được gộp. Các lần sau trả "done". users ghi chữ
// các tin user của từng request.
type ackModel struct {
	entered   chan struct{}
	release   chan struct{}
	toolFirst bool
	n         atomic.Int32
	mu        sync.Mutex
	users     [][]string
}

func newAckModel(toolFirst bool) *ackModel {
	return &ackModel{entered: make(chan struct{}), release: make(chan struct{}), toolFirst: toolFirst}
}

func (m *ackModel) Provider() string { return "fake" }
func (m *ackModel) Model() string    { return "fake-model" }

func (m *ackModel) Generate(context.Context, fantasy.Call) (*fantasy.Response, error) {
	return &fantasy.Response{Content: fantasy.ResponseContent{fantasy.TextContent{Text: "title"}}, FinishReason: fantasy.FinishReasonStop}, nil
}

func (m *ackModel) GenerateObject(context.Context, fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	return nil, errors.New("not supported")
}

func (m *ackModel) StreamObject(context.Context, fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return nil, errors.New("not supported")
}

func (m *ackModel) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	var users []string
	for _, msg := range call.Prompt {
		if msg.Role != fantasy.MessageRoleUser {
			continue
		}
		for _, p := range msg.Content {
			if tp, ok := fantasy.AsMessagePart[fantasy.TextPart](p); ok {
				users = append(users, tp.Text)
			}
		}
	}
	m.mu.Lock()
	m.users = append(m.users, users)
	m.mu.Unlock()
	first := m.n.Add(1) == 1
	if first {
		close(m.entered)
		select {
		case <-m.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if first && m.toolFirst {
		return func(yield func(fantasy.StreamPart) bool) {
			_ = yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputStart, ID: "tc1", ToolCallName: "noop"}) &&
				yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputEnd, ID: "tc1"}) &&
				yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolCall, ID: "tc1", ToolCallName: "noop", ToolCallInput: "{}"}) &&
				yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeFinish, FinishReason: fantasy.FinishReasonToolCalls})
		}, nil
	}
	return func(yield func(fantasy.StreamPart) bool) {
		_ = yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextStart, ID: "1"}) &&
			yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextDelta, ID: "1", Delta: "done"}) &&
			yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextEnd, ID: "1"}) &&
			yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeFinish, FinishReason: fantasy.FinishReasonStop})
	}, nil
}

func (m *ackModel) requests() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.users)
}

// saCoordinator nối Backend với một sessionAgent thật (LLM giả): các phương
// thức backend gọi tới đi thẳng vào sessionAgent, phần còn lại không dùng.
type saCoordinator struct {
	agent.Coordinator
	sa agent.SessionAgent
}

func (c *saCoordinator) BeginAccepted(sessionID string) *agent.AcceptedRun {
	return c.sa.BeginAccepted(sessionID)
}

func (c *saCoordinator) RunAccepted(ctx context.Context, accept *agent.AcceptedRun, sessionID, prompt string, attachments ...message.Attachment) (*fantasy.AgentResult, error) {
	return c.sa.Run(ctx, agent.SessionAgentCall{
		SessionID:       sessionID,
		RunID:           agent.RunIDFromContext(ctx),
		Prompt:          prompt,
		Attachments:     attachments,
		Accepted:        accept,
		MaxOutputTokens: 1000,
	})
}

func (c *saCoordinator) Cancel(sessionID string)              { c.sa.Cancel(sessionID) }
func (c *saCoordinator) CancelRun(sessionID, runID string)    { c.sa.CancelRun(sessionID, runID) }
func (c *saCoordinator) CancelAll()                           { c.sa.CancelAll() }
func (c *saCoordinator) IsSessionBusy(sessionID string) bool  { return c.sa.IsSessionBusy(sessionID) }
func (c *saCoordinator) HasPendingPrompts(id string) bool     { return c.sa.HasPendingPrompts(id) }
func (c *saCoordinator) IsBusy() bool                         { return c.sa.IsBusy() }
func (c *saCoordinator) QueuedPrompts(sessionID string) int   { return c.sa.QueuedPrompts(sessionID) }
func (c *saCoordinator) ClearQueue(sessionID string)          { c.sa.ClearQueue(sessionID) }
func (c *saCoordinator) QueuedPromptsList(id string) []string { return c.sa.QueuedPromptsList(id) }

type ackHarness struct {
	conn     *sql.DB
	b        *Backend
	ws       *Workspace
	sa       agent.SessionAgent
	model    *ackModel
	messages message.Service
	sid      string
}

func newAckHarness(t *testing.T, toolFirst bool) *ackHarness {
	t.Helper()
	conn, err := db.Connect(t.Context(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	q := db.New(conn)
	sessions := session.NewService(q, conn)
	messages := message.NewService(q)

	model := newAckModel(toolFirst)
	title := newAckModel(false)
	close(title.release)
	cfg := catwalk.Model{ContextWindow: 200000, DefaultMaxTokens: 10000}
	noop := fantasy.NewAgentTool("noop", "test tool", func(context.Context, struct{}, fantasy.ToolCall) (fantasy.ToolResponse, error) {
		return fantasy.NewTextResponse("ok"), nil
	})
	sa := agent.NewSessionAgent(agent.SessionAgentOptions{
		LargeModel:           agent.Model{Model: model, CatwalkCfg: cfg},
		SmallModel:           agent.Model{Model: title, CatwalkCfg: cfg},
		DisableAutoSummarize: true,
		IsYolo:               true,
		Sessions:             sessions,
		Messages:             messages,
		Tools:                []fantasy.AgentTool{noop},
	})
	sess, err := sessions.Create(t.Context(), "session")
	require.NoError(t, err)

	b, _ := newTestBackend(t)
	// App thật: lượt lỗi phát thông báo qua broker của nó.
	ws := insertRunCompleteWorkspace(t, b, context.Background(), &saCoordinator{sa: sa})
	h := &ackHarness{conn: conn, b: b, ws: ws, sa: sa, model: model, messages: messages, sid: sess.ID}
	// Chạy trước khi đóng DB (Cleanup chạy ngược thứ tự đăng ký): không để
	// lượt nào còn ghi vào DB đã đóng.
	t.Cleanup(func() {
		sa.CancelAll()
		ws.cancel()
		ws.runWG.Wait()
	})
	return h
}

func (h *ackHarness) send(t *testing.T, runID, prompt string) {
	t.Helper()
	require.NoError(t, h.b.SendMessage(h.ws.ID, proto.AgentMessage{SessionID: h.sid, RunID: runID, Prompt: prompt}))
}

// startBusy gửi prompt A và chờ tới khi lượt của A đang ở LLM (session bận).
func (h *ackHarness) startBusy(t *testing.T, runID string) {
	t.Helper()
	h.send(t, runID, "A long turn")
	select {
	case <-h.model.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("turn A never reached the model")
	}
}

// sendQueued gửi prompt B khi session đang bận và chờ B vào hàng đợi.
func (h *ackHarness) sendQueued(t *testing.T, runID, prompt string) {
	t.Helper()
	h.send(t, runID, prompt)
	require.Eventually(t, func() bool { return h.sa.QueuedPrompts(h.sid) == 1 }, 5*time.Second, 5*time.Millisecond,
		"prompt B never reached the queue")
}

func (h *ackHarness) waitRuns(t *testing.T) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		h.ws.runWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("dispatched runs did not return")
	}
}

// reopen đọc session như một tiến trình mới mở lại nó: không còn lượt nào
// chạy, nên các dấu vết bỏ dở được settle trước khi đọc.
func reopen(t *testing.T, messages message.Service, sid string) []message.Message {
	t.Helper()
	msgs, err := agent.ReadSettledMessages(t.Context(), messages, nil, sid)
	require.NoError(t, err)
	return msgs
}

// promptTrace trả số tin user mang đúng prompt và tin ngay sau tin user đầu
// tiên đó (nil nếu không có): tin ngay sau là nơi ghi số phận của prompt.
func promptTrace(msgs []message.Message, prompt string) (users int, next *message.Message) {
	for i := range msgs {
		if msgs[i].Role != message.User || msgs[i].Content().Text != prompt {
			continue
		}
		if users == 0 && i+1 < len(msgs) {
			next = &msgs[i+1]
		}
		users++
	}
	return users, next
}

func countRole(msgs []message.Message, role message.MessageRole) int {
	n := 0
	for _, m := range msgs {
		if m.Role == role {
			n++
		}
	}
	return n
}

// requireClosedUnrun khẳng định prompt có đúng một tin user và tin ngay sau
// nó là lời trợ lý đã đóng với trạng thái huỷ hoặc Interrupted.
func requireClosedUnrun(t *testing.T, msgs []message.Message, prompt string) {
	t.Helper()
	users, next := promptTrace(msgs, prompt)
	require.Equal(t, 1, users, "acked prompt %q must have exactly one user message in the DB", prompt)
	require.NotNil(t, next, "acked prompt %q has nothing recording that it never ran", prompt)
	require.Equal(t, message.Assistant, next.Role)
	require.True(t, next.IsFinished(), "the record after %q must be closed", prompt)
	f := next.FinishPart()
	ok := f.Reason == message.FinishReasonCanceled || (f.Reason == message.FinishReasonError && f.Message == "Interrupted")
	require.True(t, ok, "prompt %q must read as canceled/Interrupted, got %q %q", prompt, f.Reason, f.Message)
}

// requireCanceledNow khẳng định, trên DB thô (chưa settle, server còn sống),
// prompt bị bỏ khỏi hàng đợi đã được đóng là huỷ ngay lúc bỏ.
func requireCanceledNow(t *testing.T, messages message.Service, sid, prompt string) {
	t.Helper()
	msgs, err := messages.List(t.Context(), sid)
	require.NoError(t, err)
	users, next := promptTrace(msgs, prompt)
	require.Equal(t, 1, users)
	require.NotNil(t, next)
	require.Equal(t, message.FinishReasonCanceled, next.FinishReason(), "a dropped prompt is closed as canceled when it is dropped")
}

// T11 (đơn vị): SendMessage trả nil khi lượt còn chưa chạy (đang chờ
// trước RunAccepted) thì tin user đã phải nằm trong DB; tiến trình chết đúng
// lúc đó rồi mở lại thì prompt hiện là Interrupted.
func TestSendMessage_AckPersistsUserMessageBeforeRun(t *testing.T) {
	t.Parallel()
	b, _ := newTestBackend(t)
	coord, sessions, messages := newRealCoordinator(t)
	sess, err := sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ws := insertAgentWorkspace(t, b, coord)

	require.NoError(t, b.SendMessage(ws.ID, proto.AgentMessage{SessionID: sess.ID, RunID: "r1", Prompt: "hi"}))
	select {
	case <-coord.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("dispatched run never entered RunAccepted")
	}

	msgs, err := messages.List(t.Context(), sess.ID)
	require.NoError(t, err)
	users, _ := promptTrace(msgs, "hi")
	require.Equal(t, 1, users, "SendMessage acked before the user message was written")

	requireClosedUnrun(t, reopen(t, messages, sess.ID), "hi")

	// Dọn: lượt bị huỷ ngay khi vào, không ghi thêm tin user nào.
	require.NoError(t, b.CancelSession(ws.ID, sess.ID))
	close(coord.gate)
	ws.runWG.Wait()
	msgs, err = messages.List(t.Context(), sess.ID)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	requireClosedUnrun(t, msgs, "hi")
}

// T13 (đơn vị): B xếp hàng sau A rồi server tắt (SIGTERM): mở lại session
// thấy B, đóng là huỷ/Interrupted; B chưa tới LLM.
func TestSendMessage_QueuedPromptSurvivesShutdown(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, false)
	h.startBusy(t, "a")
	h.sendQueued(t, "b", "B queued prompt")

	// Đúng thứ tự Workspace.Shutdown (phần agent): chặn nhận, huỷ ctx
	// workspace, CancelAll, chờ các lượt.
	h.ws.runMu.Lock()
	h.ws.closing = true
	h.ws.runMu.Unlock()
	h.ws.cancel()
	h.sa.CancelAll()
	h.waitRuns(t)

	requireClosedUnrun(t, reopen(t, h.messages, h.sid), "B queued prompt")
	require.Len(t, h.model.requests(), 1, "B must not reach the model")
}

// Tiến trình chết (SIGKILL) khi B còn trong hàng đợi trong RAM: DB lúc đó
// phải đủ để tiến trình mới mở lại thấy B là Interrupted, và A (đang ở LLM,
// tin giữ chỗ của nó đã nhường cho lời trả lời thật) cũng vậy.
func TestSendMessage_QueuedPromptSurvivesCrash(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, false)
	h.startBusy(t, "a")
	h.sendQueued(t, "b", "B queued prompt")

	msgs := reopen(t, h.messages, h.sid)
	requireClosedUnrun(t, msgs, "B queued prompt")
	requireClosedUnrun(t, msgs, "A long turn")
	require.Equal(t, 2, countRole(msgs, message.Assistant), "one record per prompt, no leftover placeholder")
}

// Esc trên TUI (Cancel, prompt không RunID): B đang chờ bị bỏ thì vẫn hiện,
// đóng là huỷ.
func TestSendMessage_QueuedPromptCanceledLeavesTrace(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, false)
	h.startBusy(t, "")
	h.sendQueued(t, "", "B queued prompt")

	require.NoError(t, h.b.CancelSession(h.ws.ID, h.sid))
	h.waitRuns(t)

	requireCanceledNow(t, h.messages, h.sid, "B queued prompt")
	requireClosedUnrun(t, reopen(t, h.messages, h.sid), "B queued prompt")
	require.Len(t, h.model.requests(), 1, "B must not reach the model")
}

// ClearQueue: B bị xoá khỏi hàng đợi thì vẫn hiện, đóng là huỷ; A chạy tiếp
// bình thường.
func TestSendMessage_ClearedQueueLeavesTrace(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, false)
	h.startBusy(t, "")
	h.sendQueued(t, "", "B queued prompt")

	require.NoError(t, h.b.ClearQueue(h.ws.ID, h.sid))
	close(h.model.release)
	h.waitRuns(t)

	requireCanceledNow(t, h.messages, h.sid, "B queued prompt")
	msgs := reopen(t, h.messages, h.sid)
	requireClosedUnrun(t, msgs, "B queued prompt")
	users, next := promptTrace(msgs, "A long turn")
	require.Equal(t, 1, users)
	require.NotNil(t, next)
	require.Equal(t, message.FinishReasonEndTurn, next.FinishReason(), "A must finish normally")
	require.Len(t, h.model.requests(), 1, "B must not reach the model")
}

// CancelRun theo RunID của B đang chờ: B vẫn hiện, đóng là huỷ.
func TestSendMessage_QueuedPromptCancelRunLeavesTrace(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, false)
	h.startBusy(t, "a")
	h.sendQueued(t, "b", "B queued prompt")

	require.NoError(t, h.b.CancelRun(h.ws.ID, h.sid, "b"))
	close(h.model.release)
	h.waitRuns(t)

	requireCanceledNow(t, h.messages, h.sid, "B queued prompt")
	requireClosedUnrun(t, reopen(t, h.messages, h.sid), "B queued prompt")
	require.Len(t, h.model.requests(), 1, "B must not reach the model")
}

// Đối chứng: một prompt chạy xong để lại đúng một tin user và một lời trả
// lời đã xong, không còn gì dở.
func TestSendMessage_SinglePromptNoDuplicate(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, false)
	close(h.model.release)
	h.send(t, "a", "A long turn")
	h.waitRuns(t)

	msgs := reopen(t, h.messages, h.sid)
	require.Len(t, msgs, 2)
	users, next := promptTrace(msgs, "A long turn")
	require.Equal(t, 1, users)
	require.NotNil(t, next)
	require.Equal(t, message.FinishReasonEndTurn, next.FinishReason())
	require.Equal(t, [][]string{{"A long turn"}}, onlyPrompts(h.model.requests(), "A long turn"))
}

// Đối chứng (gộp, đường TUI không RunID): B chờ trong lúc A ở bước 1, được
// gộp vào bước 2 của A. Đúng hai tin user (A, B), B tới LLM đúng một lần, ở
// bước 2, không còn lời trợ lý dở hay bị huỷ.
func TestSendMessage_FoldedPromptNoDuplicate(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, true)
	h.startBusy(t, "")
	h.sendQueued(t, "", "B queued prompt")
	close(h.model.release)
	h.waitRuns(t)

	msgs := reopen(t, h.messages, h.sid)
	require.Equal(t, 2, countRole(msgs, message.User), "one user message per prompt")
	users, _ := promptTrace(msgs, "B queued prompt")
	require.Equal(t, 1, users)
	for _, m := range msgs {
		if m.Role == message.Assistant {
			require.NotEqual(t, message.FinishReasonCanceled, m.FinishReason())
			require.NotEqual(t, message.FinishReasonError, m.FinishReason())
		}
	}
	reqs := h.model.requests()
	require.Len(t, reqs, 2)
	require.NotContains(t, reqs[0], "B queued prompt")
	require.Equal(t, 1, countText(reqs[1], "B queued prompt"), "B folded into step 2 exactly once")
	last := msgs[len(msgs)-1]
	require.Equal(t, message.Assistant, last.Role)
	require.Equal(t, message.FinishReasonEndTurn, last.FinishReason())
}

// Đối chứng (lượt riêng, prompt có RunID): B chạy sau khi A xong; DB theo
// đúng thứ tự A user, A trả lời, B user, B trả lời, không trùng, không dở.
func TestSendMessage_QueuedRunIDPromptRunsInOrder(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, false)
	h.startBusy(t, "a")
	h.sendQueued(t, "b", "B queued prompt")
	close(h.model.release)
	h.waitRuns(t)

	msgs := reopen(t, h.messages, h.sid)
	require.Len(t, msgs, 4)
	require.Equal(t, message.User, msgs[0].Role)
	require.Equal(t, "A long turn", msgs[0].Content().Text)
	require.Equal(t, message.Assistant, msgs[1].Role)
	require.Equal(t, message.FinishReasonEndTurn, msgs[1].FinishReason())
	require.Equal(t, message.User, msgs[2].Role)
	require.Equal(t, "B queued prompt", msgs[2].Content().Text)
	require.Equal(t, message.Assistant, msgs[3].Role)
	require.Equal(t, message.FinishReasonEndTurn, msgs[3].FinishReason())
	reqs := h.model.requests()
	require.Len(t, reqs, 2)
	require.Equal(t, 1, countText(reqs[1], "B queued prompt"), "B sent to the model exactly once")
}

// Đối chứng (lượt riêng sau một lượt nhiều bước): A còn ghi tin sau lúc B
// được nhận, nên tin user của B phải được ghi lại sau lời trả lời cuối của A
// (DB đúng thứ tự model thấy), không nằm kẹt giữa lượt A, không trùng.
func TestSendMessage_QueuedRunIDPromptAfterMultiStepTurn(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, true)
	h.startBusy(t, "a")
	h.sendQueued(t, "b", "B queued prompt")
	close(h.model.release)
	h.waitRuns(t)

	msgs := reopen(t, h.messages, h.sid)
	require.Equal(t, 2, countRole(msgs, message.User))
	users, next := promptTrace(msgs, "B queued prompt")
	require.Equal(t, 1, users)
	require.NotNil(t, next)
	require.Equal(t, message.FinishReasonEndTurn, next.FinishReason())
	bAt := slices.IndexFunc(msgs, func(m message.Message) bool { return m.Content().Text == "B queued prompt" })
	require.Equal(t, len(msgs)-2, bAt, "B and its reply must come after every message of turn A")
	reqs := h.model.requests()
	require.Len(t, reqs, 3)
	require.Zero(t, countText(reqs[1], "B queued prompt"), "B must not be folded into A")
	require.Equal(t, 1, countText(reqs[2], "B queued prompt"))
}

func countText(texts []string, s string) int {
	n := 0
	for _, x := range texts {
		if x == s {
			n++
		}
	}
	return n
}

// onlyPrompts lọc mỗi request còn các chữ bằng một trong prompts (bỏ lời
// nhắc hệ thống chèn vào lịch sử).
func onlyPrompts(reqs [][]string, prompts ...string) [][]string {
	out := make([][]string, len(reqs))
	for i, r := range reqs {
		out[i] = []string{}
		for _, x := range r {
			if slices.Contains(prompts, x) {
				out[i] = append(out[i], x)
			}
		}
	}
	return out
}

// sendMore gửi thêm một prompt khi session đang bận và chờ hàng đợi đủ n.
func (h *ackHarness) sendMore(t *testing.T, runID, prompt string, n int) {
	t.Helper()
	h.send(t, runID, prompt)
	require.Eventually(t, func() bool { return h.sa.QueuedPrompts(h.sid) == n }, 5*time.Second, 5*time.Millisecond,
		"prompt %q never reached the queue", prompt)
}

// Hai prompt chờ (TUI, không RunID) sau lượt A đang ở bước cuối: prompt đã
// ghi mà chưa tới lượt không được lọt vào request của lượt khác. Mỗi request
// chứa mỗi prompt đúng một lần, đúng thứ tự gửi.
func TestSendMessage_PendingPromptsFoldInOrder(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, false)
	h.startBusy(t, "")
	h.sendMore(t, "", "B queued prompt", 1)
	h.sendMore(t, "", "C queued prompt", 2)
	close(h.model.release)
	h.waitRuns(t)

	require.Equal(t, [][]string{{"A long turn"}, {"A long turn", "B queued prompt", "C queued prompt"}},
		onlyPrompts(h.model.requests(), "A long turn", "B queued prompt", "C queued prompt"))
}

// Như trên với prompt có RunID (`send`, `run -s`): mỗi prompt là một lượt
// riêng và lượt của B không thấy C.
func TestSendMessage_PendingRunIDPromptsInOrder(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, false)
	h.startBusy(t, "a")
	h.sendMore(t, "b", "B queued prompt", 1)
	h.sendMore(t, "c", "C queued prompt", 2)
	close(h.model.release)
	h.waitRuns(t)

	require.Equal(t, [][]string{
		{"A long turn"},
		{"A long turn", "B queued prompt"},
		{"A long turn", "B queued prompt", "C queued prompt"},
	}, onlyPrompts(h.model.requests(), "A long turn", "B queued prompt", "C queued prompt"))
}

// Prompt bị bỏ khi còn xếp hàng (Esc, xoá hàng đợi) chưa từng tới model: nó
// vẫn hiện là huỷ nhưng không vào lịch sử gửi model của lượt sau (như V2,
// khi nó biến mất).
func TestSendMessage_DroppedQueuedPromptNotInHistory(t *testing.T) {
	t.Parallel()
	for name, drop := range map[string]func(h *ackHarness){
		"cancel": func(h *ackHarness) { require.NoError(t, h.b.CancelSession(h.ws.ID, h.sid)) },
		"clear_queue": func(h *ackHarness) {
			require.NoError(t, h.b.ClearQueue(h.ws.ID, h.sid))
			close(h.model.release)
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := newAckHarness(t, false)
			h.startBusy(t, "")
			h.sendQueued(t, "", "B queued prompt")
			drop(h)
			h.waitRuns(t)
			h.send(t, "", "D next prompt")
			h.waitRuns(t)

			reqs := onlyPrompts(h.model.requests(), "A long turn", "B queued prompt", "D next prompt")
			require.Equal(t, []string{"A long turn", "D next prompt"}, reqs[len(reqs)-1])
			requireClosedUnrun(t, reopen(t, h.messages, h.sid), "B queued prompt")
		})
	}
}

// Prompt đã nhận mà lượt chưa active (đứng trước RunAccepted, như lúc chờ
// readiness): người đọc session qua server (settle) không được đóng tin giữ
// chỗ còn sống là "Interrupted".
func TestSendMessage_SettleLeavesAcceptedPromptOpen(t *testing.T) {
	t.Parallel()
	b, _ := newTestBackend(t)
	coord, sessions, messages := newRealCoordinator(t)
	sess, err := sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ws := insertAgentWorkspace(t, b, coord)
	ws.App.Sessions, ws.App.Messages = sessions, messages

	require.NoError(t, b.SendMessage(ws.ID, proto.AgentMessage{SessionID: sess.ID, RunID: "r1", Prompt: "hi"}))
	select {
	case <-coord.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("dispatched run never entered RunAccepted")
	}
	_, err = b.ListSessionMessages(t.Context(), ws.ID, sess.ID)
	require.NoError(t, err)

	msgs, err := messages.List(t.Context(), sess.ID)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	require.False(t, msgs[1].IsFinished(), "a reader settled the reply of a prompt that is still accepted")

	require.NoError(t, b.CancelSession(ws.ID, sess.ID))
	close(coord.gate)
	ws.runWG.Wait()
}

// Lượt hỏng trước khi tới sessionAgent.Run (provider tắt hay thiếu key:
// lỗi của coordinator): tin giữ chỗ đóng bằng lỗi thật thay vì quay mãi rồi
// thành "Interrupted".
func TestSendMessage_PreRunErrorClosesPromptWithError(t *testing.T) {
	t.Parallel()
	conn, err := db.Connect(t.Context(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	q := db.New(conn)
	sessions := session.NewService(q, conn)
	messages := message.NewService(q)
	coord, cfg, err := agenttest.NewCoordinatorWithConfig(t.Context(), t.TempDir(), sessions, messages)
	require.NoError(t, err)
	for id, p := range cfg.Config().Providers.Seq2() {
		p.Disable = true
		cfg.Config().Providers.Set(id, p)
	}
	sess, err := sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	b, _ := newTestBackend(t)
	ws := insertRunCompleteWorkspace(t, b, context.Background(), coord)

	require.NoError(t, b.SendMessage(ws.ID, proto.AgentMessage{SessionID: sess.ID, RunID: "r1", Prompt: "hi"}))
	ws.runWG.Wait()

	msgs, err := messages.List(t.Context(), sess.ID)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	f := msgs[1].FinishPart()
	require.NotNil(t, f, "the pending reply of a run that failed before starting must be closed")
	require.Equal(t, message.FinishReasonError, f.Reason)
	require.Contains(t, f.Details, "unavailable")
}

// failMessageDeletes làm mọi lệnh xoá tin hỏng (trigger SQLite), như khi tiến
// trình chết giữa hai lần ghi: các bước "ghi mới + xoá cũ" phải trọn vẹn hoặc
// không gì.
func failMessageDeletes(t *testing.T, conn *sql.DB) {
	t.Helper()
	_, err := conn.ExecContext(t.Context(), `CREATE TRIGGER fail_message_delete BEFORE DELETE ON messages
		BEGIN SELECT RAISE(ABORT, 'injected delete failure'); END`)
	require.NoError(t, err)
}

// Tạo lời trả lời thật và xoá tin giữ chỗ là một lần ghi: xoá hỏng thì không
// được để lại hai bản trả lời cho một prompt.
func TestSendMessage_ReplyReplacesPlaceholderAtomically(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, false)
	failMessageDeletes(t, h.conn)
	close(h.model.release)
	h.send(t, "a", "A long turn")
	h.waitRuns(t)

	msgs := reopen(t, h.messages, h.sid)
	require.Equal(t, 1, countRole(msgs, message.User))
	require.Equal(t, 1, countRole(msgs, message.Assistant), "one record per prompt")
}

// Dời cặp tin (ghi mới + xoá cũ) là một lần ghi: xoá hỏng thì không được để
// lại hai bản prompt; prompt mang lỗi thật.
func TestSendMessage_MoveRecordedPromptAtomically(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, true)
	h.startBusy(t, "a")
	h.sendQueued(t, "b", "B queued prompt")
	failMessageDeletes(t, h.conn)
	close(h.model.release)
	h.waitRuns(t)

	msgs := reopen(t, h.messages, h.sid)
	users, next := promptTrace(msgs, "B queued prompt")
	require.Equal(t, 1, users, "the prompt must not be duplicated")
	require.NotNil(t, next)
	require.Equal(t, message.FinishReasonError, next.FinishReason(), "the failed move is reported on the prompt")
	require.Equal(t, "Prompt not run", next.FinishPart().Message, "with the real error, not a later Interrupted")
}

// Hai `send` thật cùng lúc khi session bận: mỗi request chứa mỗi prompt nhiều
// nhất một lần, mỗi prompt tới model đúng một lần, không prompt nào trùng.
func TestSendMessage_ConcurrentSendsStayDistinct(t *testing.T) {
	t.Parallel()
	h := newAckHarness(t, false)
	h.startBusy(t, "a")
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, p := range []string{"B queued prompt", "C queued prompt"} {
		wg.Go(func() {
			<-start
			assert.NoError(t, h.b.SendMessage(h.ws.ID, proto.AgentMessage{SessionID: h.sid, RunID: p, Prompt: p}))
		})
	}
	close(start)
	wg.Wait()
	require.Eventually(t, func() bool { return h.sa.QueuedPrompts(h.sid) == 2 }, 5*time.Second, 5*time.Millisecond)
	close(h.model.release)
	h.waitRuns(t)

	prompts := []string{"A long turn", "B queued prompt", "C queued prompt"}
	reqs := onlyPrompts(h.model.requests(), prompts...)
	require.Len(t, reqs, 3)
	for i, r := range reqs {
		for _, p := range prompts {
			require.LessOrEqual(t, countText(r, p), 1, "request %d carries %q twice: %v", i, p, r)
		}
	}
	require.Equal(t, []string{"A long turn"}, reqs[0])
	require.Len(t, reqs[1], 2, "the second turn sees A and its own prompt only")
	require.Len(t, reqs[2], 3)
	msgs := reopen(t, h.messages, h.sid)
	require.Equal(t, 3, countRole(msgs, message.User))
	require.Equal(t, 3, countRole(msgs, message.Assistant))
}
