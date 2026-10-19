package cmd

import (
	"testing"

	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/stretchr/testify/require"
)

// Tin giữ chỗ của prompt chưa chạy (provider "agent-cli-pending", model = ID
// tin user) không phải lời trả lời của model nào: `run --json` không được báo
// nó là provider/model. Các ca của message.LastReply ở gói message; ở đây chỉ
// kiểm đường server: protoReplies chép đủ trường để LastReply nhận ra tin giữ
// chỗ.
func TestLastReplyModelSkipsPromptPlaceholders(t *testing.T) {
	t.Parallel()
	msgs := []proto.Message{
		{ID: "u1", Role: proto.User},
		{Role: proto.Assistant, Provider: "openai", Model: "gpt-5"},
		{ID: "u2", Role: proto.User},
		{Role: proto.Assistant, Provider: message.PendingReplyProvider, Model: "u2"},
	}
	p, m := lastReplyModel(protoReplies(msgs))
	require.Equal(t, "openai", p)
	require.Equal(t, "gpt-5", m)
}
