package model

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/ui/util"
)

// sessionModelNote báo trên thanh trạng thái khi tin assistant cuối của session
// vừa mở chạy model khác model lớn của workspace. Mở session không ghi gì: model
// thuộc workspace và chỉ đổi khi người dùng chọn (alt+m, alt+p).
func (m *UI) sessionModelNote(msgs []message.Message) tea.Cmd {
	cfg := m.com.Config()
	if cfg == nil {
		return nil
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != message.Assistant || msgs[i].IsSummaryMessage {
			continue
		}
		last := config.SelectedModel{Provider: msgs[i].Provider, Model: msgs[i].Model}
		current := cfg.Models[config.SelectedModelTypeLarge]
		note := app.SessionModelNote(last, current)
		if note == "" {
			return nil
		}
		// alt+m chỉ liệt kê model của provider đang dùng (keys.go).
		hint := " (alt+m to change model)"
		if last.Provider != current.Provider {
			hint = " (alt+p to change provider)"
		}
		return util.ReportInfo("This session " + note + hint)
	}
	return nil
}

// appliesNextNote là phần nói thêm khi đổi model/effort lúc có run: lượt đang
// chạy giữ thiết lập cũ, thiết lập mới áp cho tin nhắn gửi sau.
func appliesNextNote(busy bool) string {
	if !busy {
		return ""
	}
	return " (applies to your next message; running tasks keep their model)"
}
