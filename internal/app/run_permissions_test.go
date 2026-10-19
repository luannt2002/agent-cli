package app

import (
	"context"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/stretchr/testify/require"
)

// `run` cục bộ phải duyệt cả yêu cầu quyền của phiên con (agent con, agentic_fetch)
// như server-mode run, không thì chúng chờ một câu trả lời không ai đưa tới hết giờ
// (PRB-63). Phiên không thuộc lượt run thì không được đụng tới.
func TestRunNonInteractiveGrantsDescendantSessions(t *testing.T) {
	t.Parallel()
	perms := permission.NewPermissionService(t.TempDir(), false, nil)
	app := &App{Permissions: perms, Sessions: &mockSessionService{sessions: []session.Session{
		{ID: "run"},
		{ID: "msg-1$$call-1", ParentSessionID: "run"},
		{ID: "msg-2$$call-2", ParentSessionID: "msg-1$$call-1"},
		{ID: "other"},
		{ID: "other-child", ParentSessionID: "other"},
	}}}
	app.autoApproveRun(t.Context(), "run")

	request := func(sessionID string, wait time.Duration) bool {
		ctx, cancel := context.WithTimeout(t.Context(), wait)
		defer cancel()
		ok, _ := perms.Request(ctx, permission.CreatePermissionRequest{
			SessionID: sessionID, ToolCallID: "c-" + sessionID, ToolName: "bash", Action: "execute", Path: t.TempDir(),
		})
		return ok
	}
	for _, id := range []string{"run", "msg-1$$call-1", "msg-2$$call-2"} {
		require.Truef(t, request(id, 5*time.Second), "%s must be granted, not left waiting", id)
	}
	for _, id := range []string{"other", "other-child", "unknown"} {
		require.Falsef(t, request(id, 200*time.Millisecond), "%s is not part of the run", id)
	}
}
