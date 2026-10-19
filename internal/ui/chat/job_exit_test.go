package chat

import (
	"testing"

	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

// Job nền thoát mã ≠ 0 (lệnh run_in_background lỗi ngay trong giây đầu, hay
// job_output của job đã xong) không được mang dấu ✓; job còn chạy, thoát mã 0
// và tin nhắn cũ (không có exit_code) giữ nguyên hiển thị.
func TestJobTools_NonZeroExitIsNotShownAsSuccess(t *testing.T) {
	t.Parallel()

	sty := styles.CharmtonePantera()
	render := func(r ToolRenderer, input, metadata string) string {
		return ansi.Strip(r.RenderTool(&sty, 120, &ToolRenderOpts{
			ToolCall: message.ToolCall{ID: "job", Input: input, Finished: true},
			Result:   &message.ToolResult{ToolCallID: "job", Content: "out", Metadata: metadata},
			Status:   ToolStatusSuccess,
		}))
	}
	bashStart := func(metadata string) string {
		return render(&BashToolRenderContext{}, `{"command":"make serve","run_in_background":true}`, metadata)
	}
	jobOutput := func(metadata string) string {
		return render(&JobOutputToolRenderContext{}, `{"shell_id":"007"}`, metadata)
	}

	for name, out := range map[string]string{
		"bash background fast fail": bashStart(`{"background":true,"output":"boom","exit_code":2}`),
		"job_output done, exit 2":   jobOutput(`{"shell_id":"007","done":true,"exit_code":2}`),
	} {
		require.Contains(t, out, styles.ToolError, name)
		require.Contains(t, out, "exit=2", name)
		require.NotContains(t, out, styles.ToolSuccess, name)
	}

	for name, out := range map[string]string{
		"bash background started":   bashStart(`{"background":true,"shell_id":"007"}`),
		"bash background exit 0":    bashStart(`{"background":true,"output":"ok","exit_code":0}`),
		"job_output running":        jobOutput(`{"shell_id":"007","done":false}`),
		"job_output done, exit 0":   jobOutput(`{"shell_id":"007","done":true,"exit_code":0}`),
		"job_output legacy message": jobOutput(`{"shell_id":"007","done":true}`),
	} {
		require.Contains(t, out, styles.ToolSuccess, name)
		require.NotContains(t, out, "exit=", name)
	}
}
