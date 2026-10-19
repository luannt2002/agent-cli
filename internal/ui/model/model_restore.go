package model

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/ui/util"
)

// sessionModelNote báo trên thanh trạng thái khi lời trả lời thật cuối của
// session vừa mở (message.LastReply) chạy model khác model lớn của workspace.
// Mở session không ghi gì: model thuộc workspace và chỉ đổi khi người dùng chọn
// (alt+m, alt+p).
func (m *UI) sessionModelNote(msgs []message.Message) tea.Cmd {
	cfg := m.com.Config()
	if cfg == nil {
		return nil
	}
	reply, ok := message.LastReply(msgs)
	if !ok {
		return nil
	}
	last := config.SelectedModel{Provider: reply.Provider, Model: reply.Model}
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

// appliesNextNote là phần nói thêm khi đổi model/effort lúc có run: lượt đang
// chạy giữ thiết lập cũ, thiết lập mới áp cho tin nhắn gửi sau.
func appliesNextNote(busy bool) string {
	if !busy {
		return ""
	}
	return " (applies to your next message; running tasks keep their model)"
}
