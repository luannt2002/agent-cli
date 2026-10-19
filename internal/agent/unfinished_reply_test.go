package agent

import (
	"testing"

	"github.com/charmbracelet/crush/internal/message"
	"github.com/stretchr/testify/require"
)

// Tiến trình chết (SIGKILL, OOM) giữa lúc model đang viết chữ, trước token đầu,
// hay giữa lúc tóm tắt: lời trả lời không có tool call nào nên trước đây không
// bao giờ được đóng — spinner quay mãi, bản tóm tắt dở bị gửi lại như lịch sử
// (audit g1/g2). Mở session (không bận) phải đóng chúng là "Interrupted".
func TestSettleInterruptedCalls_ClosesUnfinishedTextReplies(t *testing.T) {
	env := testEnv(t)
	ctx := t.Context()
	sess, err := env.sessions.Create(ctx, "killed mid-text")
	require.NoError(t, err)

	partial, err := env.messages.Create(ctx, sess.ID, message.CreateMessageParams{
		Role:  message.Assistant,
		Parts: []message.ContentPart{message.TextContent{Text: "Half of an answ"}},
	})
	require.NoError(t, err)
	empty, err := env.messages.Create(ctx, sess.ID, message.CreateMessageParams{Role: message.Assistant})
	require.NoError(t, err)
	summary, err := env.messages.Create(ctx, sess.ID, message.CreateMessageParams{
		Role:             message.Assistant,
		IsSummaryMessage: true,
		Parts:            []message.ContentPart{message.TextContent{Text: "## Current State\n- half"}},
	})
	require.NoError(t, err)

	require.NoError(t, SettleInterruptedCalls(ctx, env.messages, sess.ID))

	for _, id := range []string{partial.ID, empty.ID, summary.ID} {
		got, err := env.messages.Get(ctx, id)
		require.NoError(t, err)
		require.True(t, got.IsFinished(), "an unfinished reply of a dead process must be closed")
		f := got.FinishPart()
		require.Equal(t, message.FinishReasonError, f.Reason)
		require.Equal(t, interruptedTurnMessage, f.Message)
	}
	kept, err := env.messages.Get(ctx, partial.ID)
	require.NoError(t, err)
	require.Equal(t, "Half of an answ", kept.Content().String(), "what was written stays")
}

// Lời trả lời đã xong không bị đụng tới.
func TestSettleInterruptedCalls_LeavesFinishedRepliesAlone(t *testing.T) {
	env := testEnv(t)
	ctx := t.Context()
	sess, err := env.sessions.Create(ctx, "fine")
	require.NoError(t, err)
	done, err := env.messages.Create(ctx, sess.ID, message.CreateMessageParams{
		Role:  message.Assistant,
		Parts: []message.ContentPart{message.TextContent{Text: "all done"}, message.Finish{Reason: message.FinishReasonEndTurn}},
	})
	require.NoError(t, err)

	require.NoError(t, SettleInterruptedCalls(ctx, env.messages, sess.ID))

	got, err := env.messages.Get(ctx, done.ID)
	require.NoError(t, err)
	require.Equal(t, message.FinishReasonEndTurn, got.FinishPart().Reason)
}
