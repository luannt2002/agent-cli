package agent

import (
	"strings"
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/stretchr/testify/require"
)

// Bản tóm tắt bị cắt vì hết ngân sách output là bản tóm tắt HỎNG: nhận nó làm
// điểm cắt thì toàn bộ lịch sử bị thay bằng nửa bản tóm tắt mà không ai hay
// (audit g1: finish=length bị ghi đè thành end_turn). Giữ lịch sử, làm tiếp.
func TestTruncatedSummaryKeepsHistory(t *testing.T) {
	h := newContHarness(t, &scriptModel{
		script:      []string{"tool", "done"},
		summaryText: "## Current State\n- half of the brie", summaryFinish: fantasy.FinishReasonLength,
	})

	_, err := h.sa.Run(t.Context(), SessionAgentCall{SessionID: h.sid, RunID: "x", Prompt: "refactor the scanner"})
	require.NoError(t, err)
	rc := waitRunComplete(t, h.ch, "x")
	require.Empty(t, rc.Error)
	require.Equal(t, "done", rc.Text)

	sess, err := h.env.sessions.Get(t.Context(), h.sid)
	require.NoError(t, err)
	require.Empty(t, sess.SummaryMessageID, "a truncated summary must not become the history cut point")
	msgs, err := h.env.messages.List(t.Context(), h.sid)
	require.NoError(t, err)
	var failed bool
	for _, m := range msgs {
		if m.IsSummaryMessage {
			f := m.FinishPart()
			require.NotNil(t, f)
			failed = f.Reason == message.FinishReasonError
		}
	}
	require.True(t, failed, "the truncated summary is recorded as failed, not end_turn")
}

// Model trả lời không chữ, không tool call (finish=stop): lượt không làm gì cả.
// Không có lỗi thì `run` thoát 0 với `text` rỗng (faults.sh ca empty-reply).
func TestEmptyTurnIsAnError(t *testing.T) {
	h := newContHarness(t, &scriptModel{script: []string{"empty"}, summaryText: "SUMMARY"})

	_, err := h.sa.Run(t.Context(), SessionAgentCall{SessionID: h.sid, RunID: "x", Prompt: "fix it"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty reply")
	rc := waitRunComplete(t, h.ch, "x")
	require.Contains(t, rc.Error, "empty reply")
}

// Làm việc bằng tool rồi kết thúc im lặng vẫn là một lượt có làm: không phải lỗi.
func TestToolWorkThenSilentEndIsNotAnError(t *testing.T) {
	h := newContHarness(t, &scriptModel{script: []string{"tool_small", "empty"}, summaryText: "SUMMARY"})

	_, err := h.sa.Run(t.Context(), SessionAgentCall{SessionID: h.sid, RunID: "x", Prompt: "fix it"})
	require.NoError(t, err)
	rc := waitRunComplete(t, h.ch, "x")
	require.Empty(t, rc.Error)
}

// Bị cắt ở giới hạn output giữa tool call viết dở (không có phần ToolCall) hay
// khi đang nghĩ: đó là việc của nhánh tự làm tiếp, không phải "trả lời rỗng".
func TestLengthCutWithoutTextIsContinuedNotEmpty(t *testing.T) {
	for _, step := range []string{"cut_tool", "think_cut"} {
		t.Run(step, func(t *testing.T) {
			h := newContHarness(t, &scriptModel{script: []string{step, "done"}, summaryText: "SUMMARY"})

			_, err := h.sa.Run(t.Context(), SessionAgentCall{SessionID: h.sid, RunID: "x", Prompt: "write the big file"})
			require.NoError(t, err)
			rc := waitRunComplete(t, h.ch, "x")
			require.Empty(t, rc.Error)
			require.Equal(t, "done", rc.Text, "the length continuation ran")
		})
	}
}

// Bộ lọc nội dung từ chối mà không có chữ: giữ lý do từ chối (TUI hiện REFUSED),
// không đổi thành "trả lời rỗng".
func TestContentFilterRefusalIsNotReportedAsEmpty(t *testing.T) {
	h := newContHarness(t, &scriptModel{script: []string{"refused"}, summaryText: "SUMMARY"})

	_, _ = h.sa.Run(t.Context(), SessionAgentCall{SessionID: h.sid, RunID: "x", Prompt: "fix it"})
	rc := waitRunComplete(t, h.ch, "x")
	require.NotContains(t, rc.Error, "empty reply")
	require.Contains(t, rc.Error, "content filter", "a refusal without any output is not a success")
	msgs, err := h.env.messages.List(t.Context(), h.sid)
	require.NoError(t, err)
	last := msgs[len(msgs)-1]
	require.NotNil(t, last.FinishPart())
	require.Equal(t, message.FinishReasonContentFilter, last.FinishPart().Reason)
	// TUI tự viết lời hướng dẫn khi bị từ chối (cmp.Or với Message/Details): không đè.
	require.Empty(t, last.FinishPart().Message)
	require.Empty(t, last.FinishPart().Details)
}

// Trả lời rỗng kết thúc bằng finish khác stop (openai-compat ánh xạ finish lạ như
// "network_error"/"sensitive" của GLM thành unknown; google có error/other) vẫn
// là trả lời rỗng, không phải thành công (soát lượt 3, F1).
func TestEmptyTurnWithOtherFinishReasonsIsAnError(t *testing.T) {
	for _, step := range []string{"empty_unknown", "empty_error", "empty_other"} {
		t.Run(step, func(t *testing.T) {
			h := newContHarness(t, &scriptModel{script: []string{step}, summaryText: "SUMMARY"})
			_, err := h.sa.Run(t.Context(), SessionAgentCall{SessionID: h.sid, RunID: "x", Prompt: "fix it"})
			require.Error(t, err)
			rc := waitRunComplete(t, h.ch, "x")
			require.Contains(t, rc.Error, "empty reply")
		})
	}
}

// Chuỗi bị cắt mà chưa từng ra chữ hay chạy xong tool call nào: lượt làm tiếp
// rỗng, hay hết trần làm tiếp, đều là thất bại (soát lượt 3, F2).
func TestLengthCutChainThatNeverProducesIsAnError(t *testing.T) {
	cap := make([]string, maxLengthContinues+1)
	for i := range cap {
		cap[i] = "think_cut"
	}
	for name, script := range map[string][]string{
		"think_cut then empty": {"think_cut", "empty"},
		"cut_tool then empty":  {"cut_tool", "empty"},
		"think_cut until cap":  cap,
	} {
		t.Run(name, func(t *testing.T) {
			h := newContHarness(t, &scriptModel{script: script, summaryText: "SUMMARY"})
			_, _ = h.sa.Run(t.Context(), SessionAgentCall{SessionID: h.sid, RunID: "x", Prompt: "write the big file"})
			rc := waitRunComplete(t, h.ch, "x")
			require.Contains(t, rc.Error, "empty reply")
		})
	}
}

// Nghĩ hết ngân sách đúng lúc vượt ngưỡng tóm tắt: nhánh tóm tắt không xếp lượt
// làm tiếp (không có tool call), nhánh tự làm tiếp bị bỏ qua khi đang tóm tắt; chuỗi
// chưa ra gì nên là lỗi, không phải "xong" rỗng (soát lượt 4, R4-A1).
func TestThinkingCutWhileSummarizingIsAnError(t *testing.T) {
	h := newContHarness(t, &scriptModel{script: []string{"think_cut_big"}, summaryText: "SUMMARY"})
	_, _ = h.sa.Run(t.Context(), SessionAgentCall{SessionID: h.sid, RunID: "x", Prompt: "fix it"})
	rc := waitRunComplete(t, h.ch, "x")
	require.Contains(t, rc.Error, "empty reply")
}

// Câu trả lời thật nằm ở khối chữ thứ hai (khối đầu chỉ có khoảng trắng): vẫn là có
// trả lời (soát lượt 4, R4-A2).
func TestAnswerInALaterTextBlockIsNotEmpty(t *testing.T) {
	h := newContHarness(t, &scriptModel{script: []string{"two_text_blocks"}, summaryText: "SUMMARY"})
	_, err := h.sa.Run(t.Context(), SessionAgentCall{SessionID: h.sid, RunID: "x", Prompt: "fix it"})
	require.NoError(t, err)
	rc := waitRunComplete(t, h.ch, "x")
	require.Empty(t, rc.Error)
	require.Contains(t, rc.Text, "the real answer")
}

// Lượt làm tiếp sau tự tóm tắt kết thúc im lặng: chuỗi của prompt đã làm việc,
// nên không phải lỗi.
func TestSilentContinuationAfterSummaryIsNotAnError(t *testing.T) {
	h := newContHarness(t, &scriptModel{script: []string{"tool", "empty"}, summaryText: "SUMMARY"})

	_, err := h.sa.Run(t.Context(), SessionAgentCall{SessionID: h.sid, RunID: "x", Prompt: "fix it"})
	require.NoError(t, err)
	rc := waitRunComplete(t, h.ch, "x")
	require.Empty(t, rc.Error)
	kinds, _ := h.model.seen()
	require.Equal(t, []string{"work", "summary", "work"}, kinds)
}

// Loop detection dừng lượt vì model lặp cùng một tool call: đó là thất bại, không
// phải "xong". RunComplete phải mang lỗi để `run` thoát khác 0 (audit C-06).
func TestLoopDetectionStopIsAnError(t *testing.T) {
	script := make([]string, 20)
	for i := range script {
		script[i] = "tool_small"
	}
	h := newContHarness(t, &scriptModel{script: script, summaryText: "SUMMARY"})

	_, err := h.sa.Run(t.Context(), SessionAgentCall{SessionID: h.sid, RunID: "x", Prompt: "fix it"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "repeated the same tool call")
	rc := waitRunComplete(t, h.ch, "x")
	require.Contains(t, rc.Error, "repeated the same tool call")
	kinds, _ := h.model.seen()
	require.Less(t, strings.Count(strings.Join(kinds, ","), "work"), 20, "the loop is cut before the script runs out")
}
