package message

import (
	"testing"

	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/stretchr/testify/require"
)

// Tin giữ chỗ của prompt không chạy (provider "agent-cli-pending") không phải
// lời trả lời của model nào: thống kê theo model và thời gian phản hồi trung
// bình không được tính nó.
func TestStatsSkipPromptPlaceholders(t *testing.T) {
	t.Parallel()
	conn, err := db.Connect(t.Context(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	q := db.New(conn)
	sess, err := session.NewService(q, conn).Create(t.Context(), "test")
	require.NoError(t, err)
	svc := NewService(q)

	user, err := svc.Create(t.Context(), sess.ID, CreateMessageParams{Role: User, Parts: []ContentPart{TextContent{Text: "B"}}})
	require.NoError(t, err)
	pending, err := svc.Create(t.Context(), sess.ID, CreateMessageParams{Role: Assistant, Provider: "agent-cli-pending", Model: user.ID})
	require.NoError(t, err)
	pending.AddFinishAt(FinishReasonCanceled, "User canceled request", "", pending.CreatedAt+1000)
	require.NoError(t, svc.Update(t.Context(), pending))
	require.NoError(t, svc.FlushAll(t.Context()))

	rows, err := q.GetUsageByModel(t.Context())
	require.NoError(t, err)
	require.Empty(t, rows, "a prompt placeholder is not a model reply")
	avg, err := q.GetAverageResponseTime(t.Context())
	require.NoError(t, err)
	require.Zero(t, avg)
}

// WriteBatch là một transaction: lần ghi sau hỏng thì lần ghi trước không
// còn (không bao giờ có tin user mà thiếu tin giữ chỗ, hay hai bản prompt).
func TestWriteBatchIsAllOrNothing(t *testing.T) {
	t.Parallel()
	conn, err := db.Connect(t.Context(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	q := db.New(conn)
	sess, err := session.NewService(q, conn).Create(t.Context(), "test")
	require.NoError(t, err)
	svc := NewService(q)
	_, err = conn.ExecContext(t.Context(), `CREATE TRIGGER fail_boom BEFORE INSERT ON messages
		WHEN NEW.provider = 'boom' BEGIN SELECT RAISE(ABORT, 'injected'); END`)
	require.NoError(t, err)

	old, err := svc.Create(t.Context(), sess.ID, CreateMessageParams{Role: User, Parts: []ContentPart{TextContent{Text: "old"}}})
	require.NoError(t, err)
	_, err = svc.WriteBatch(t.Context(), sess.ID, []CreateMessageParams{
		{Role: User, Parts: []ContentPart{TextContent{Text: "new"}}},
		{Role: Assistant, Provider: "boom"},
	}, []string{old.ID})
	require.ErrorContains(t, err, "injected")

	msgs, err := svc.List(t.Context(), sess.ID)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Equal(t, "old", msgs[0].Content().Text, "a failed batch leaves nothing behind and deletes nothing")
}

// WriteBatch phải mở transaction bằng câu ghi: lô không có gì để tạo bị từ chối.
func TestWriteBatchNeedsCreate(t *testing.T) {
	t.Parallel()
	svc, sessionID := newTestService(t)
	_, err := svc.WriteBatch(t.Context(), sessionID, nil, []string{"x"})
	require.ErrorContains(t, err, "at least one")
}
