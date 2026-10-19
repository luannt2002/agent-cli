package tools

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/shell"
)

const (
	JobOutputToolName = "job_output"
)

//go:embed job_output.md
var jobOutputDescription string

type JobOutputParams struct {
	ShellID string `json:"shell_id" description:"The ID of the background shell to retrieve output from"`
	Wait    bool   `json:"wait" description:"If true, block until the background shell completes before returning output"`
}

type JobOutputResponseMetadata struct {
	ShellID          string `json:"shell_id"`
	Command          string `json:"command"`
	Description      string `json:"description"`
	Done             bool   `json:"done"`
	WorkingDirectory string `json:"working_directory"`
	// ExitCode chỉ có khi job đã xong (như BashResponseMetadata.ExitCode).
	ExitCode *int `json:"exit_code,omitempty"`
}

func NewJobOutputTool(spillDir string) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		JobOutputToolName,
		jobOutputDescription,
		func(ctx context.Context, params JobOutputParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.ShellID == "" {
				return fantasy.NewTextErrorResponse("missing shell_id"), nil
			}

			bgShell, ok := ownedJob(ctx, params.ShellID)
			if !ok {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("background shell not found: %s", params.ShellID)), nil
			}

			if params.Wait {
				bgShell.WaitContext(ctx)
			}

			stdout, stderr, done, err := bgShell.GetOutput()

			var outputParts []string
			if stdout != "" {
				outputParts = append(outputParts, stdout)
			}
			if stderr != "" {
				outputParts = append(outputParts, stderr)
			}

			status := "running"
			var exitCode *int
			if done {
				status = "completed"
				code := shell.ExitCode(err)
				exitCode = &code
				if code != 0 {
					outputParts = append(outputParts, fmt.Sprintf("Exit code %d", code))
				}
			}

			output := strings.Join(outputParts, "\n")
			output = TruncateOutput(output, spillDir)

			metadata := JobOutputResponseMetadata{
				ShellID:          params.ShellID,
				Command:          bgShell.Command,
				Description:      bgShell.Description,
				Done:             done,
				WorkingDirectory: bgShell.WorkingDir,
				ExitCode:         exitCode,
			}

			if output == "" {
				output = BashNoOutput
			}

			result := fmt.Sprintf("Status: %s\n\n%s", status, output)
			return fantasy.WithResponseMetadata(fantasy.NewTextResponse(result), metadata), nil
		},
	)
}

// ownedJob trả job id nếu nó thuộc session đang gọi tool. Bảng job dùng chung
// cả tiến trình và ID đánh số lại từ đầu sau mỗi lần khởi động, nên job của
// session khác (hay không có chủ) coi như không có: session mở lại còn nhớ ID
// cũ không đọc hay dừng nhầm job của session khác.
func ownedJob(ctx context.Context, id string) (*shell.BackgroundShell, bool) {
	bg, ok := shell.GetBackgroundShellManager().Get(id)
	sessionID := GetSessionFromContext(ctx)
	if !ok || sessionID == "" || bg.Owner.SessionID != sessionID {
		return nil, false
	}
	return bg, true
}
