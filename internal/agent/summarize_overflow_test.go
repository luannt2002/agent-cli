package agent

import (
	"cmp"
	"context"
	"errors"
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

// overflowModel: mỗi bước làm việc đều gọi tool với usage gần đầy context
// (phần cố định của prompt đã vượt ngưỡng tự tóm tắt), tóm tắt luôn thành công.
// Sau maxWork bước làm việc thì trả "done" để test không treo khi mã còn lặp.
type overflowModel struct {
	n       atomic.Int64
	maxWork int64
	// cut: bước làm việc bị cắt ở giới hạn output giữa tool call (không có phần
	// ToolCall, như provider openai-compat).
	cut   bool
	mu    sync.Mutex
	kinds []string
}

func (m *overflowModel) Provider() string { return "fake" }
func (m *overflowModel) Model() string    { return "fake-model" }
func (m *overflowModel) Generate(context.Context, fantasy.Call) (*fantasy.Response, error) {
	return &fantasy.Response{Content: fantasy.ResponseContent{fantasy.TextContent{Text: "title"}}, FinishReason: fantasy.FinishReasonStop}, nil
}

func (m *overflowModel) GenerateObject(context.Context, fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	return nil, errors.New("not supported")
}

func (m *overflowModel) StreamObject(context.Context, fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return nil, errors.New("not supported")
}

func (m *overflowModel) seen() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.kinds...)
}

func (m *overflowModel) Stream(_ context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	isSummary := false
	for _, msg := range call.Prompt {
		if msg.Role != fantasy.MessageRoleSystem {
			continue
		}
		for _, part := range msg.Content {
			if tp, ok := fantasy.AsMessagePart[fantasy.TextPart](part); ok && strings.Contains(tp.Text, "summar") {
				isSummary = true
			}
		}
	}
	text := func(s string, usage fantasy.Usage) fantasy.StreamResponse {
		return func(yield func(fantasy.StreamPart) bool) {
			if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextStart, ID: "1"}) {
				return
			}
			if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextDelta, ID: "1", Delta: s}) {
				return
			}
			if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextEnd, ID: "1"}) {
				return
			}
			yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeFinish, FinishReason: fantasy.FinishReasonStop, Usage: usage})
		}
	}
	m.mu.Lock()
	if isSummary {
		m.kinds = append(m.kinds, "summary")
	} else {
		m.kinds = append(m.kinds, "work")
	}
	m.mu.Unlock()
	if isSummary {
		return text("summary of the work", fantasy.Usage{InputTokens: 300, OutputTokens: 20, TotalTokens: 320}), nil
	}
	if m.n.Add(1) > m.maxWork {
		return text("done", fantasy.Usage{InputTokens: 900, OutputTokens: 10, TotalTokens: 910}), nil
	}
	if m.cut {
		return func(yield func(fantasy.StreamPart) bool) {
			if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputStart, ID: "tc", ToolCallName: "noop"}) ||
				!yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputDelta, ID: "tc", Delta: `{"content":"half`}) {
				return
			}
			yield(fantasy.StreamPart{
				Type: fantasy.StreamPartTypeFinish, FinishReason: fantasy.FinishReasonLength,
				Usage: fantasy.Usage{InputTokens: 900, OutputTokens: 60, TotalTokens: 960},
			})
		}, nil
	}
	return func(yield func(fantasy.StreamPart) bool) {
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputStart, ID: "tc", ToolCallName: "noop"}) {
			return
		}
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputEnd, ID: "tc"}) {
			return
		}
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolCall, ID: "tc", ToolCallName: "noop", ToolCallInput: "{}"}) {
			return
		}
		yield(fantasy.StreamPart{
			Type: fantasy.StreamPartTypeFinish, FinishReason: fantasy.FinishReasonToolCalls,
			Usage: fantasy.Usage{InputTokens: 900, OutputTokens: 60, TotalTokens: 960},
		})
	}, nil
}

// Phần cố định của prompt đã chiếm gần hết context: tóm tắt xong, bước đầu của
// lượt làm tiếp vẫn vượt ngưỡng. Trước đây vòng làm-việc ↔ tóm-tắt lặp vô hạn
// (probe P-04: 755 + 755 lời gọi trong 90 giây). Nay lượt dừng với lỗi nêu rõ
// context window quá nhỏ, chỉ sau một lần tóm tắt.
func TestAutoSummarizeStopsWhenSummaryCannotFreeRoom(t *testing.T) {
	for _, cut := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete tool calls", true: "tool calls cut at the output limit"}[cut], func(t *testing.T) {
			testSummaryCannotFreeRoom(t, &overflowModel{maxWork: 12, cut: cut})
		})
	}
}

func testSummaryCannotFreeRoom(t *testing.T, m *overflowModel) {
	env := testEnv(t)
	broker := pubsub.NewBroker[notify.RunComplete]()
	t.Cleanup(broker.Shutdown)
	cw := catwalk.Model{ContextWindow: 1000, DefaultMaxTokens: 100}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel:  modelWith(m, "prov", "model", cw),
		SmallModel:  modelWith(&finishStreamModel{text: "title"}, "prov", "small", cw),
		IsYolo:      true,
		Sessions:    env.sessions,
		Messages:    env.messages,
		RunComplete: broker,
		Tools:       []fantasy.AgentTool{namedTool("noop")},
	}).(*sessionAgent)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ch := broker.Subscribe(t.Context())

	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "x", Prompt: "refactor everything"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "context window")

	rc := waitRunComplete(t, ch, "x")
	require.Contains(t, rc.Error, "context window")
	require.Equal(t, []string{"work", "summary", "work"}, m.seen(), "one summary, then stop instead of summarizing again")
}

// Tóm tắt giải phóng được chỗ (bước đầu sau tóm tắt dưới ngưỡng) thì việc dài
// vẫn được tóm tắt nhiều lần như trước; chỉ ca không tiến triển mới bị chặn.
func TestAutoSummarizeStillRepeatsWhenItFreesRoom(t *testing.T) {
	env := testEnv(t)
	broker := pubsub.NewBroker[notify.RunComplete]()
	t.Cleanup(broker.Shutdown)
	m := &roomModel{}
	cw := catwalk.Model{ContextWindow: 1000, DefaultMaxTokens: 100}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel:  modelWith(m, "prov", "model", cw),
		SmallModel:  modelWith(&finishStreamModel{text: "title"}, "prov", "small", cw),
		IsYolo:      true,
		Sessions:    env.sessions,
		Messages:    env.messages,
		RunComplete: broker,
		Tools:       []fantasy.AgentTool{namedTool("noop")},
	}).(*sessionAgent)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ch := broker.Subscribe(t.Context())

	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "x", Prompt: "long job"})
	require.NoError(t, err)
	rc := waitRunComplete(t, ch, "x")
	require.Empty(t, rc.Error)
	require.Equal(t, "done", rc.Text)
	require.Equal(t, 2, strings.Count(strings.Join(m.seen(), ","), "summary"), "two productive summaries are allowed")
}

// roomModel: bước làm việc xen kẽ usage thấp (sau tóm tắt) rồi cao (đầy lại),
// hai vòng như vậy rồi xong. Mỗi lần tóm tắt đều giải phóng được chỗ.
type roomModel struct {
	overflowModel
	work atomic.Int64
	// rounds: số vòng (đầy lại → tóm tắt) trước khi trả lời xong; 0 = 2.
	rounds int64
	// cutBig: bước làm đầy context bị cắt ở giới hạn output giữa tool call (không
	// có phần ToolCall, như provider openai-compat).
	cutBig bool
}

func (m *roomModel) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	resp, err := m.overflowModel.Stream(ctx, call)
	kinds := m.seen()
	if kinds[len(kinds)-1] == "summary" {
		return resp, err
	}
	w := m.work.Add(1)
	usage := fantasy.Usage{InputTokens: 300, OutputTokens: 60, TotalTokens: 360}
	if w%2 == 0 {
		usage = fantasy.Usage{InputTokens: 900, OutputTokens: 60, TotalTokens: 960}
	}
	finish, name := fantasy.FinishReasonToolCalls, "noop"
	if w > 2*cmp.Or(m.rounds, 2) {
		return func(yield func(fantasy.StreamPart) bool) {
			if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextStart, ID: "1"}) {
				return
			}
			if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextDelta, ID: "1", Delta: "done"}) {
				return
			}
			if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextEnd, ID: "1"}) {
				return
			}
			yield(fantasy.StreamPart{
				Type: fantasy.StreamPartTypeFinish, FinishReason: fantasy.FinishReasonStop,
				Usage: fantasy.Usage{InputTokens: 300, OutputTokens: 10, TotalTokens: 310},
			})
		}, nil
	}
	if m.cutBig && w%2 == 0 {
		return func(yield func(fantasy.StreamPart) bool) {
			if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputStart, ID: "tc", ToolCallName: name}) {
				return
			}
			yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeFinish, FinishReason: fantasy.FinishReasonLength, Usage: usage})
		}, nil
	}
	return func(yield func(fantasy.StreamPart) bool) {
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputStart, ID: "tc", ToolCallName: name}) {
			return
		}
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputEnd, ID: "tc"}) {
			return
		}
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolCall, ID: "tc", ToolCallName: name, ToolCallInput: "{}"}) {
			return
		}
		yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeFinish, FinishReason: finish, Usage: usage})
	}, nil
}

// Model cứ làm tiếp sau mỗi lần tóm tắt mà không bao giờ xong (model yếu làm lại
// việc từ đầu sau tóm tắt): không có trần thì một prompt cháy token vô hạn
// (faults: 1.338 lời gọi main / 120 s). Dừng có lỗi sau maxAutoSummaries lần.
func TestAutoSummarizeHasACapPerPrompt(t *testing.T) {
	for _, cutBig := range []bool{false, true} {
		t.Run(map[bool]string{false: "tool calls", true: "cut tool call fills the window"}[cutBig], func(t *testing.T) {
			testSummaryCap(t, &roomModel{rounds: 1000, cutBig: cutBig})
		})
	}
}

func testSummaryCap(t *testing.T, m *roomModel) {
	env := testEnv(t)
	broker := pubsub.NewBroker[notify.RunComplete]()
	t.Cleanup(broker.Shutdown)
	cw := catwalk.Model{ContextWindow: 1000, DefaultMaxTokens: 100}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel:  modelWith(m, "prov", "model", cw),
		SmallModel:  modelWith(&finishStreamModel{text: "title"}, "prov", "small", cw),
		IsYolo:      true,
		Sessions:    env.sessions,
		Messages:    env.messages,
		RunComplete: broker,
		Tools:       []fantasy.AgentTool{namedTool("noop")},
	}).(*sessionAgent)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ch := broker.Subscribe(t.Context())

	_, _ = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "x", Prompt: "endless job"})
	rc := waitRunComplete(t, ch, "x")
	require.Contains(t, rc.Error, "auto-summarized")
	require.Equal(t, maxAutoSummaries, strings.Count(strings.Join(m.seen(), ","), "summary"))
}

// outputHeavyModel: bước đầu đầy context bằng đầu vào; bước đầu SAU tóm tắt có đầu
// vào nhỏ nhưng ra một tool call rất dài (ghi tệp lớn) nên tổng vượt ngưỡng; rồi xong.
type outputHeavyModel struct {
	overflowModel
	work atomic.Int64
}

func (m *outputHeavyModel) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	resp, err := m.overflowModel.Stream(ctx, call)
	kinds := m.seen()
	if kinds[len(kinds)-1] == "summary" {
		return resp, err
	}
	usage := fantasy.Usage{InputTokens: 900, OutputTokens: 60, TotalTokens: 960}
	switch m.work.Add(1) {
	case 1:
	case 2:
		usage = fantasy.Usage{InputTokens: 300, OutputTokens: 700, TotalTokens: 1000}
	default:
		return textStream("done", fantasy.FinishReasonStop, fantasy.Usage{InputTokens: 300, OutputTokens: 10, TotalTokens: 310}), nil
	}
	return func(yield func(fantasy.StreamPart) bool) {
		if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputStart, ID: "tc", ToolCallName: "noop"}) ||
			!yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolInputEnd, ID: "tc"}) ||
			!yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeToolCall, ID: "tc", ToolCallName: "noop", ToolCallInput: "{}"}) {
			return
		}
		yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeFinish, FinishReason: fantasy.FinishReasonToolCalls, Usage: usage})
	}, nil
}

// Ngay sau tóm tắt mà chỉ phần OUTPUT của bước (ghi tệp lớn) đẩy tổng qua ngưỡng thì
// tóm tắt thêm vẫn giải phóng được chỗ: không phải "context window too small" (red
// team cuối, L1). Chỉ khi phần ĐẦU VÀO đã vượt ngưỡng mới là tràn thật.
func TestLargeOutputRightAfterSummaryIsNotAnOverflow(t *testing.T) {
	env := testEnv(t)
	broker := pubsub.NewBroker[notify.RunComplete]()
	t.Cleanup(broker.Shutdown)
	m := &outputHeavyModel{}
	cw := catwalk.Model{ContextWindow: 1000, DefaultMaxTokens: 100}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel:  modelWith(m, "prov", "model", cw),
		SmallModel:  modelWith(&finishStreamModel{text: "title"}, "prov", "small", cw),
		IsYolo:      true,
		Sessions:    env.sessions,
		Messages:    env.messages,
		RunComplete: broker,
		Tools:       []fantasy.AgentTool{namedTool("noop")},
	}).(*sessionAgent)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ch := broker.Subscribe(t.Context())

	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "x", Prompt: "write the big file"})
	require.NoError(t, err)
	rc := waitRunComplete(t, ch, "x")
	require.Empty(t, rc.Error)
	require.Equal(t, "done", rc.Text)
}

// Bước đầu sau tóm tắt vượt ngưỡng nhưng là câu trả lời cuối (không gọi tool):
// lượt kết thúc bình thường với câu trả lời đó, không báo lỗi, không tóm tắt lại.
func TestAutoSummarizeKeepsFinalAnswerAfterSummary(t *testing.T) {
	env := testEnv(t)
	broker := pubsub.NewBroker[notify.RunComplete]()
	t.Cleanup(broker.Shutdown)
	m := &overflowModel{maxWork: 1}
	cw := catwalk.Model{ContextWindow: 1000, DefaultMaxTokens: 100}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel:  modelWith(m, "prov", "model", cw),
		SmallModel:  modelWith(&finishStreamModel{text: "title"}, "prov", "small", cw),
		IsYolo:      true,
		Sessions:    env.sessions,
		Messages:    env.messages,
		RunComplete: broker,
		Tools:       []fantasy.AgentTool{namedTool("noop")},
	}).(*sessionAgent)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ch := broker.Subscribe(t.Context())

	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "x", Prompt: "small fix"})
	require.NoError(t, err)
	rc := waitRunComplete(t, ch, "x")
	require.Empty(t, rc.Error)
	require.Equal(t, "done", rc.Text)
	require.Equal(t, []string{"work", "summary", "work"}, m.seen())
}
