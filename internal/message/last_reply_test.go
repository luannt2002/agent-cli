package message

import (
	"testing"

	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/stretchr/testify/require"
)

// Tin giữ chỗ của prompt chưa chạy (provider "agent-cli-pending", model = ID
// tin user) không phải lời trả lời của model nào; bản tóm tắt và tin không có
// model cũng không. LastReply lùi về lời trả lời thật trước chúng.
func TestLastReply(t *testing.T) {
	t.Parallel()
	placeholder := func(userID string) Message {
		return Message{Role: Assistant, Provider: PendingReplyProvider, Model: userID}
	}
	reply := Message{ID: "r1", Role: Assistant, Provider: "openai", Model: "gpt-5"}

	for name, tc := range map[string]struct {
		msgs []Message
		want string // ID lời trả lời; rỗng: không có.
	}{
		"no messages": {},
		// Lượt hỏng trước khi chạy (thiếu key, provider tắt): chỉ có tin giữ
		// chỗ đã đóng lỗi, nên không có lời trả lời nào.
		"only a placeholder": {msgs: []Message{{ID: "u1", Role: User}, placeholder("u1")}},
		// Sau lời trả lời thật đã có prompt khác được ghi.
		"placeholder after a reply": {msgs: []Message{{ID: "u1", Role: User}, reply, {ID: "u2", Role: User}, placeholder("u2")}, want: "r1"},
		"summary after a reply":     {msgs: []Message{reply, {ID: "s", Role: Assistant, Provider: "openai", Model: "gpt-5", IsSummaryMessage: true}}, want: "r1"},
		"no model after a reply":    {msgs: []Message{reply, {ID: "x", Role: Assistant}}, want: "r1"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, ok := LastReply(tc.msgs)
			require.Equal(t, tc.want != "", ok)
			require.Equal(t, tc.want, got.ID)
		})
	}
}

// GetLastAssistantMessage không trả tin giữ chỗ (DB do server ghi, chế độ local
// đọc lại): tin giữ chỗ sau lời trả lời thật thì hàm trả lời trả lời thật.
func TestGetLastAssistantMessageSkipsPromptPlaceholders(t *testing.T) {
	t.Parallel()
	conn, err := db.Connect(t.Context(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	q := db.New(conn)
	sess, err := session.NewService(q, conn).Create(t.Context(), "test")
	require.NoError(t, err)
	svc := NewService(q)

	// created_at tính bằng giây: đặt rõ thứ tự để tin giữ chỗ là tin mới nhất.
	for i, p := range []CreateMessageParams{
		{Role: Assistant, Provider: "pp", Model: "m-pp"},
		{ID: "u2", Role: User},
		{Role: Assistant, Provider: PendingReplyProvider, Model: "u2"},
	} {
		m, err := svc.Create(t.Context(), sess.ID, p)
		require.NoError(t, err)
		_, err = conn.ExecContext(t.Context(), `UPDATE messages SET created_at = ? WHERE id = ?`, 1000+i, m.ID)
		require.NoError(t, err)
	}

	last, err := svc.GetLastAssistantMessage(t.Context(), sess.ID)
	require.NoError(t, err)
	require.Equal(t, "pp", last.Provider)
	require.Equal(t, "m-pp", last.Model)
}
