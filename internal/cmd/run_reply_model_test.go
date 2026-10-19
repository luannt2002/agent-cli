package cmd

import (
	"testing"

	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/stretchr/testify/require"
)

// Tin giữ chỗ của prompt chưa chạy (provider "agent-cli-pending", model = ID
// tin user) không phải lời trả lời của model nào: `run --json` không được báo
// nó là provider/model.
func TestLastReplyModelSkipsPromptPlaceholders(t *testing.T) {
	t.Parallel()
	placeholder := func(userID string) message.Message {
		return message.Message{Role: message.Assistant, Provider: message.PendingReplyProvider, Model: userID}
	}

	// Lượt hỏng trước khi chạy (thiếu key, provider tắt): chỉ có tin giữ chỗ
	// đã đóng lỗi, nên không có provider/model nào để báo.
	p, m := lastReplyModel([]message.Message{{ID: "u1", Role: message.User}, placeholder("u1")})
	require.Empty(t, p)
	require.Empty(t, m)

	// Sau lời trả lời thật đã có prompt khác được ghi: báo nhãn của lời trả lời thật.
	msgs := []message.Message{
		{ID: "u1", Role: message.User},
		{Role: message.Assistant, Provider: "openai", Model: "gpt-5"},
		{ID: "u2", Role: message.User},
		placeholder("u2"),
	}
	p, m = lastReplyModel(msgs)
	require.Equal(t, "openai", p)
	require.Equal(t, "gpt-5", m)

	// Đường server: cùng kết quả qua tin proto.
	var protoMsgs []proto.Message
	for _, x := range msgs {
		protoMsgs = append(protoMsgs, proto.Message{Role: proto.MessageRole(x.Role), Provider: x.Provider, Model: x.Model})
	}
	p, m = lastReplyModel(protoReplies(protoMsgs))
	require.Equal(t, "openai", p)
	require.Equal(t, "gpt-5", m)
}
