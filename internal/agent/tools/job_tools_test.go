package tools

import (
	"context"
	"encoding/json"
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/shell"
	"github.com/stretchr/testify/require"
)

// runJobTool gọi tool job_* như agent gọi: session nằm trong ctx.
func runJobTool(t *testing.T, tool fantasy.AgentTool, sessionID string, params any) fantasy.ToolResponse {
	t.Helper()

	input, err := json.Marshal(params)
	require.NoError(t, err)
	ctx := context.WithValue(t.Context(), SessionIDContextKey, sessionID)
	resp, err := tool.Run(ctx, fantasy.ToolCall{ID: "job-call", Name: tool.Info().Name, Input: string(input)})
	require.NoError(t, err)
	return resp
}

// Bảng job dùng chung cả tiến trình và ID job đánh số lại sau mỗi lần khởi
// động: session khác (kể cả session mở lại còn nhớ ID cũ) không được đọc hay
// dừng job không phải của nó, job không có chủ cũng vậy.
func TestJobTools_OnlyOwnerSessionCanReadOrKill(t *testing.T) {
	t.Parallel()

	bgManager := shell.GetBackgroundShellManager()
	owned, err := bgManager.StartFor(context.Background(), shell.JobOwner{SessionID: "owner"}, t.TempDir(), nil, "sleep 30", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = bgManager.Kill(owned.ID) })
	ownerless, err := bgManager.Start(context.Background(), t.TempDir(), nil, "sleep 30", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = bgManager.Kill(ownerless.ID) })

	outputTool := NewJobOutputTool(t.TempDir())
	killTool := NewJobKillTool()

	for _, tc := range []struct {
		name    string
		session string
		job     *shell.BackgroundShell
	}{
		{"other session", "intruder", owned},
		{"no session", "", owned},
		{"ownerless job, no session", "", ownerless},
	} {
		resp := runJobTool(t, outputTool, tc.session, JobOutputParams{ShellID: tc.job.ID})
		require.True(t, resp.IsError, "%s: job_output must not read the job", tc.name)
		require.Contains(t, resp.Content, "not found", tc.name)

		resp = runJobTool(t, killTool, tc.session, JobKillParams{ShellID: tc.job.ID})
		require.True(t, resp.IsError, "%s: job_kill must not kill the job", tc.name)
		require.Contains(t, resp.Content, "not found", tc.name)
		require.False(t, tc.job.IsDone(), "%s: the job must keep running", tc.name)
	}

	resp := runJobTool(t, outputTool, "owner", JobOutputParams{ShellID: owned.ID})
	require.False(t, resp.IsError, resp.Content)
	require.Contains(t, resp.Content, "Status: running")

	resp = runJobTool(t, killTool, "owner", JobKillParams{ShellID: owned.ID})
	require.False(t, resp.IsError, resp.Content)
	require.True(t, owned.IsDone(), "the owner session kills its job")
}

// Job đã xong mang mã thoát trong metadata (như lệnh bash chạy-rồi-chờ), để UI
// không vẽ ✓ cho job thoát lỗi; job còn chạy thì không có mã.
func TestJobOutput_MetadataCarriesExitCode(t *testing.T) {
	t.Parallel()

	bgManager := shell.GetBackgroundShellManager()
	tool := NewJobOutputTool(t.TempDir())
	owner := shell.JobOwner{SessionID: "s"}

	// Đọc metadata như UI đọc (JSON), không qua struct của tool.
	readMeta := func(command string, wait bool) (done bool, exitCode *int) {
		bg, err := bgManager.StartFor(context.Background(), owner, t.TempDir(), nil, command, "")
		require.NoError(t, err)
		t.Cleanup(func() { _ = bgManager.Kill(bg.ID) })
		if wait {
			bg.Wait()
		}
		resp := runJobTool(t, tool, "s", JobOutputParams{ShellID: bg.ID})
		require.False(t, resp.IsError, resp.Content)
		var meta struct {
			Done     bool `json:"done"`
			ExitCode *int `json:"exit_code"`
		}
		require.NoError(t, json.Unmarshal([]byte(resp.Metadata), &meta))
		return meta.Done, meta.ExitCode
	}

	for _, tc := range []struct {
		command string
		want    int
	}{
		{"true", 0},
		{"echo boom && exit 3", 3},
	} {
		done, exitCode := readMeta(tc.command, true)
		require.True(t, done, tc.command)
		require.NotNil(t, exitCode, "%s: metadata must carry exit_code once the job is done", tc.command)
		require.Equal(t, tc.want, *exitCode, tc.command)
	}

	done, exitCode := readMeta("sleep 30", false)
	require.False(t, done)
	require.Nil(t, exitCode, "a running job has no exit code")
}
