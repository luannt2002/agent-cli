package app

import (
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/stretchr/testify/require"
)

// sessionModelApp ghi msgs vào một session mới trên DB thật, tin sau muộn hơn
// tin trước một giây (created_at tính bằng giây), và trả App có model lớn của
// workspace là qq/m-qq cùng ID session đó.
func sessionModelApp(t *testing.T, msgs ...message.CreateMessageParams) (*App, string) {
	t.Helper()
	conn, err := db.Connect(t.Context(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	q := db.New(conn)
	sess, err := session.NewService(q, conn).Create(t.Context(), "test")
	require.NoError(t, err)
	svc := message.NewService(q)
	for i, p := range msgs {
		m, err := svc.Create(t.Context(), sess.ID, p)
		require.NoError(t, err)
		_, err = conn.ExecContext(t.Context(), `UPDATE messages SET created_at = ? WHERE id = ?`, 1000+i, m.ID)
		require.NoError(t, err)
	}
	app := &App{
		config: config.NewTestStore(&config.Config{
			Models: map[config.SelectedModelType]config.SelectedModel{
				config.SelectedModelTypeLarge: {Provider: "qq", Model: "m-qq"},
			},
		}),
		Messages: svc,
	}
	return app, sess.ID
}

// Dòng báo của `run -s/-C` local: nêu model lượt cuối của session và model của
// workspace mà lượt này chạy; im khi trùng hoặc session chưa có trả lời. Tin
// giữ chỗ của prompt chưa chạy (DB do server ghi, local đọc lại) không phải
// lời trả lời: dòng báo dựa vào lời trả lời thật trước nó. Việc lượt local thật
// sự chạy model của workspace (không đổi trong RAM) do
// scripts/eval/session-model.sh kiểm trên binary thật.
func TestSessionModelNote(t *testing.T) {
	t.Parallel()

	const want = "last ran on pp/m-pp; continuing on the project model qq/m-qq"
	reply := func(provider, model string) message.CreateMessageParams {
		return message.CreateMessageParams{Role: message.Assistant, Provider: provider, Model: model}
	}
	user := message.CreateMessageParams{ID: "u2", Role: message.User}
	pending := message.CreateMessageParams{Role: message.Assistant, Provider: message.PendingReplyProvider, Model: "u2"}
	closed := pending
	closed.Parts = []message.ContentPart{message.Finish{Reason: message.FinishReasonCanceled, Message: "User canceled request"}}

	for name, tc := range map[string]struct {
		msgs []message.CreateMessageParams
		want string
	}{
		"other model":                        {msgs: []message.CreateMessageParams{reply("pp", "m-pp")}, want: want},
		"other model, prompt pending after":  {msgs: []message.CreateMessageParams{reply("pp", "m-pp"), user, pending}, want: want},
		"other model, prompt canceled after": {msgs: []message.CreateMessageParams{reply("pp", "m-pp"), user, closed}, want: want},
		"same model, prompt pending after":   {msgs: []message.CreateMessageParams{reply("qq", "m-qq"), user, pending}},
		"only a pending prompt":              {msgs: []message.CreateMessageParams{user, pending}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			app, sessionID := sessionModelApp(t, tc.msgs...)
			require.Equal(t, tc.want, app.sessionModelNote(t.Context(), sessionID))
		})
	}

	qq := config.SelectedModel{Provider: "qq", Model: "m-qq"}
	require.Empty(t, SessionModelNote(qq, qq), "same model: nothing to say")
	require.Empty(t, SessionModelNote(config.SelectedModel{}, qq), "no assistant reply yet")
}
