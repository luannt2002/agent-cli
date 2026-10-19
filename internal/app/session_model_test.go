package app

import (
	"context"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/stretchr/testify/require"
)

// lastAssistantMessages là message.Service giả chỉ trả tin assistant cuối.
type lastAssistantMessages struct {
	message.Service
	last message.Message
}

func (m lastAssistantMessages) GetLastAssistantMessage(context.Context, string) (message.Message, error) {
	return m.last, nil
}

// Dòng báo của `run -s/-C` local: nêu model lượt cuối của session và model của
// workspace mà lượt này chạy; im khi trùng hoặc session chưa có trả lời. Việc
// lượt local thật sự chạy model của workspace (không đổi trong RAM) do
// scripts/eval/session-model.sh kiểm trên binary thật.
func TestSessionModelNote(t *testing.T) {
	t.Parallel()

	qq := config.SelectedModel{Provider: "qq", Model: "m-qq"}
	app := &App{
		config: config.NewTestStore(&config.Config{
			Models: map[config.SelectedModelType]config.SelectedModel{config.SelectedModelTypeLarge: qq},
		}),
		Messages: lastAssistantMessages{last: message.Message{Role: message.Assistant, Provider: "pp", Model: "m-pp"}},
	}

	require.Equal(t, "last ran on pp/m-pp; continuing on the project model qq/m-qq", app.sessionModelNote(t.Context(), "x"))
	require.Empty(t, SessionModelNote(qq, qq), "same model: nothing to say")
	require.Empty(t, SessionModelNote(config.SelectedModel{}, qq), "no assistant reply yet")
}
