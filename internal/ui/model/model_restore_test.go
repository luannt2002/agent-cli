package model

import (
	"strings"
	"sync"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/csync"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	"github.com/charmbracelet/crush/internal/ui/util"
	"github.com/stretchr/testify/require"
)

// modelLogWorkspace ghi lại model lớn được ghi, theo thứ tự.
type modelLogWorkspace struct {
	*countingWorkspace
	mu      sync.Mutex
	written []string
}

func (w *modelLogWorkspace) UpdatePreferredModel(scope config.Scope, t config.SelectedModelType, sel config.SelectedModel) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if t == config.SelectedModelTypeLarge {
		w.written = append(w.written, sel.Provider+"/"+sel.Model)
	}
	return w.countingWorkspace.UpdatePreferredModel(scope, t, sel)
}

func (w *modelLogWorkspace) log() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.written...)
}

func restoreUI(t *testing.T) (*UI, *modelLogWorkspace) {
	t.Helper()
	m, cw := newModelScopeUI(t)
	cw.cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{Provider: "p", Model: "a"}
	cw.cfg.Models[config.SelectedModelTypeSmall] = config.SelectedModel{Provider: "p", Model: "small"}
	ws := &modelLogWorkspace{countingWorkspace: cw}
	m.com.Workspace = ws
	return m, ws
}

func lastAssistantFor(model string) []message.Message {
	return []message.Message{{Role: message.Assistant, Provider: "p", Model: model}}
}

func modelWithID(id string) catwalk.Model { return catwalk.Model{ID: id} }

// Mở một session cũ (alt+s, alt+. / alt+,, --session, --continue) không ghi
// model của session đó vào cấu hình dự án: model thuộc workspace và chỉ đổi khi
// người dùng chọn. Model khác thì thanh trạng thái nói rõ cả hai model, kèm
// phím đổi đúng: alt+m chỉ liệt kê model của provider đang dùng, nên khác
// provider thì gợi ý alt+p. Nạp lại session đang xem (sau reconnect) không báo
// lại, để không đè "Reconnected…".
func TestOpeningSessionKeepsTheWorkspaceModel(t *testing.T) {
	t.Parallel()

	// Tin giữ chỗ của prompt u2 (đang chờ, hoặc đã đóng khi bị huỷ/lỗi) không
	// phải lời trả lời: thông báo dựa vào lời trả lời thật trước nó.
	pending := []message.Message{{ID: "u2", Role: message.User}, {Role: message.Assistant, Provider: message.PendingReplyProvider, Model: "u2"}}
	closed := []message.Message{pending[0], pending[1]}
	closed[1].AddFinish(message.FinishReasonCanceled, "User canceled request", "")

	for name, tc := range map[string]struct {
		session  string
		last     message.Message
		wantHint string // Rỗng: không có thông báo.

		after []message.Message // Tin sau last.
	}{
		"other provider":          {session: "s2", last: message.Message{Role: message.Assistant, Provider: "q", Model: "x"}, wantHint: "alt+p"},
		"other model":             {session: "s2", last: lastAssistantFor("b")[0], wantHint: "alt+m"},
		"same model":              {session: "s2", last: lastAssistantFor("a")[0]},
		"reload of shown session": {session: "s1", last: message.Message{Role: message.Assistant, Provider: "q", Model: "x"}},

		"other model, prompt pending after":  {session: "s2", last: lastAssistantFor("b")[0], after: pending, wantHint: "alt+m"},
		"other model, prompt canceled after": {session: "s2", last: lastAssistantFor("b")[0], after: closed, wantHint: "alt+m"},
		"same model, prompt pending after":   {session: "s2", last: lastAssistantFor("a")[0], after: pending},
		"only a pending prompt":              {session: "s2", last: pending[0], after: pending[1:]},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m, ws := newJumpUI()
			ws.cfg = &config.Config{
				Providers: csync.NewMap[string, config.ProviderConfig](),
				Models: map[config.SelectedModelType]config.SelectedModel{
					config.SelectedModelTypeLarge: {Provider: "p", Model: "a"},
					config.SelectedModelTypeSmall: {Provider: "p", Model: "small"},
				},
			}
			ws.cfg.Providers.Set("p", config.ProviderConfig{ID: "p", Models: []catwalk.Model{modelWithID("a"), modelWithID("b")}})
			m.userThemeSelected = true

			msgs := append([]message.Message{tc.last}, tc.after...)
			_, next := m.Update(loadSessionMsg{session: &session.Session{ID: tc.session}, messages: msgs})
			var notes []string
			for _, msg := range leafMsgs(next) {
				if info, ok := msg.(util.InfoMsg); ok {
					require.NotContains(t, info.Msg, message.PendingReplyProvider, "a placeholder is not a model")
					if strings.Contains(info.Msg, "last ran on") {
						notes = append(notes, info.Msg)
					}
				}
			}

			require.Empty(t, ws.prefModelScopes, "opening a session must not write the project model")
			require.Equal(t, tc.session, m.currentSessionID())
			if tc.wantHint == "" {
				require.Empty(t, notes)
				return
			}
			require.Len(t, notes, 1)
			require.Contains(t, notes[0], tc.last.Provider+"/"+tc.last.Model, "the note names the session's last model")
			require.Contains(t, notes[0], "p/a", "the note names the model the next message uses")
			require.Contains(t, notes[0], tc.wantHint)
		})
	}
}

// Đổi model khi có run (ở bất kỳ session nào) không còn bị chặn; thông báo
// nói rõ thiết lập mới áp cho tin nhắn sau.
func TestModelSwitchAllowedWhileAnySessionBusy(t *testing.T) {
	m, ws := restoreUI(t)
	m.agentBusyCache.set(true)

	cmd := m.handleSelectModel(dialog.ActionSelectModel{
		Model:     config.SelectedModel{Provider: "p", Model: "b"},
		ModelType: config.SelectedModelTypeLarge,
	})
	require.NotNil(t, cmd)
	require.Equal(t, []string{"p/b"}, ws.log(), "the selection is written even while a run is in flight")
	require.Contains(t, appliesNextNote(true), "next message")
	require.Empty(t, appliesNextNote(false))
}
