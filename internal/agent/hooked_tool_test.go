package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/agent/tools/mcp"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/hooks"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/session"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// fakeTool records the context it was invoked with so tests can assert on
// values stamped onto it by the hookedTool decorator.
type fakeTool struct {
	name   string
	called bool
	gotCtx context.Context
	resp   fantasy.ToolResponse
}

func (f *fakeTool) Info() fantasy.ToolInfo {
	return fantasy.ToolInfo{Name: f.name}
}

func (f *fakeTool) Run(ctx context.Context, _ fantasy.ToolCall) (fantasy.ToolResponse, error) {
	f.called = true
	f.gotCtx = ctx
	return f.resp, nil
}

func (f *fakeTool) ProviderOptions() fantasy.ProviderOptions     { return nil }
func (f *fakeTool) SetProviderOptions(_ fantasy.ProviderOptions) {}

// newRunner builds a hooks.Runner from a single HookConfig, running the
// config-loader path that compiles the matcher regex.
func newRunner(t *testing.T, cmd string) *hooks.Runner {
	t.Helper()
	cfg := &config.Config{
		Hooks: map[string][]config.HookConfig{
			hooks.EventPreToolUse: {{Command: cmd}},
		},
	}
	require.NoError(t, cfg.ValidateHooks())
	return hooks.NewRunner(cfg.Hooks[hooks.EventPreToolUse], t.TempDir(), t.TempDir())
}

func TestHookedTool_AllowStampsHookApproval(t *testing.T) {
	t.Parallel()

	inner := &fakeTool{name: "view", resp: fantasy.NewTextResponse("ok")}
	runner := newRunner(t, `echo '{"decision":"allow"}'`)
	tool := newHookedTool(inner, runner)

	_, err := tool.Run(t.Context(), fantasy.ToolCall{ID: "call-1", Name: "view"})
	require.NoError(t, err)
	require.True(t, inner.called, "inner tool should have run")

	// The inner tool's permission service can now treat call-1 as pre-approved.
	svc := permission.NewPermissionService(t.TempDir(), false, nil)
	granted, err := svc.Request(inner.gotCtx, permission.CreatePermissionRequest{
		SessionID:  "s1",
		ToolCallID: "call-1",
		ToolName:   "view",
		Action:     "read",
		Path:       t.TempDir(),
	})
	require.NoError(t, err)
	require.True(t, granted, "hook allow should bypass the permission prompt")
}

func TestHookedTool_SilentDoesNotStampApproval(t *testing.T) {
	t.Parallel()

	inner := &fakeTool{name: "view", resp: fantasy.NewTextResponse("ok")}
	runner := newRunner(t, `exit 0`) // no stdout, no decision
	tool := newHookedTool(inner, runner)

	_, err := tool.Run(t.Context(), fantasy.ToolCall{ID: "call-2", Name: "view"})
	require.NoError(t, err)
	require.True(t, inner.called)

	// With no hook opinion, a fresh permission request has nothing stamped
	// and must fall through to the normal flow. We verify by checking that
	// the context does not look pre-approved for this call ID: sending a
	// request that no subscriber resolves will block until cancelled.
	svc := permission.NewPermissionService(t.TempDir(), false, nil)
	ctx, cancel := context.WithCancel(inner.gotCtx)
	cancel()
	granted, err := svc.Request(ctx, permission.CreatePermissionRequest{
		SessionID:  "s1",
		ToolCallID: "call-2",
		ToolName:   "view",
		Action:     "read",
		Path:       t.TempDir(),
	})
	require.Error(t, err, "no approval stamped => request should reach the prompt path")
	require.False(t, granted)
}

func TestHookedTool_DenySkipsInnerTool(t *testing.T) {
	t.Parallel()

	inner := &fakeTool{name: "bash"}
	runner := newRunner(t, `echo "blocked" >&2; exit 2`)
	tool := newHookedTool(inner, runner)

	resp, err := tool.Run(t.Context(), fantasy.ToolCall{ID: "call-3", Name: "bash"})
	require.NoError(t, err)
	require.False(t, inner.called, "denied call must not reach the inner tool")
	require.True(t, resp.IsError)
	require.Contains(t, resp.Content, "blocked")
}

func TestWrapToolsWithHooks(t *testing.T) {
	t.Parallel()

	runner := newRunner(t, `exit 0`)
	inputs := []fantasy.AgentTool{&fakeTool{name: "a"}, &fakeTool{name: "b"}}

	t.Run("top-level agent wraps every tool", func(t *testing.T) {
		t.Parallel()
		out := wrapToolsWithHooks(inputs, runner, false)
		require.Len(t, out, len(inputs))
		for i, tool := range out {
			_, ok := tool.(*hookedTool)
			require.Truef(t, ok, "tool %d should be a *hookedTool", i)
		}
	})

	t.Run("sub-agent skips the wrap", func(t *testing.T) {
		t.Parallel()
		out := wrapToolsWithHooks(inputs, runner, true)
		require.Equal(t, inputs, out, "sub-agent tools should be returned unwrapped")
		for _, tool := range out {
			_, isHooked := tool.(*hookedTool)
			require.False(t, isHooked, "sub-agent tool should not be wrapped")
		}
	})

	t.Run("nil runner skips the wrap for both agent kinds", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, inputs, wrapToolsWithHooks(inputs, nil, false))
		require.Equal(t, inputs, wrapToolsWithHooks(inputs, nil, true))
	})
}

// Có hook PreToolUse thì mọi tool của agent cấp cao bị bọc trong
// *hookedTool. Lượt từ channel "signal" vẫn không được thấy tool gửi tin của
// channel "telegram". Ca "no hook" là đối chứng: hành vi như cũ.
func TestFilterToolsForChannel_SeesThroughHookedTool(t *testing.T) {
	t.Parallel()

	states := map[string]mcp.ClientInfo{"signal": {Channel: true}, "telegram": {Channel: true}}
	raw := []fantasy.AgentTool{
		&channelTestTool{name: "signal_send", mcpName: "signal"},
		&channelTestTool{name: "telegram_send", mcpName: "telegram"},
		&fakeTool{name: "plain"},
	}
	cases := map[string][]fantasy.AgentTool{
		"no hook":   raw,
		"with hook": wrapToolsWithHooks(raw, newRunner(t, `exit 0`), false),
	}
	for name, ts := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := filterToolsForChannel(ts, "signal", states)
			require.Equal(t, []string{"signal_send", "plain"}, toolNames(got))
			require.Len(t, filterToolsForChannel(ts, "", states), 3, "a local turn keeps every tool")
			requireHookKept(t, name, got)
		})
	}
}

// Có hook PreToolUse, server đã tắt bằng "Toggle MCPs" vẫn không được đưa
// tool của nó cho model. Dùng *tools.Tool thật (MCP server HTTP trong tiến
// trình) vì bộ lọc nhận diện tool MCP theo đúng kiểu đó. Ca "no hook" là đối
// chứng. Registry MCP là toàn cục nên test này không t.Parallel và gỡ hẳn
// server khi xong.
func TestFilterDisabledMCPTools_SeesThroughHookedTool(t *testing.T) {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "p2d-off"}, nil)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "send", Description: "test tool"},
		func(context.Context, *mcpsdk.CallToolRequest, struct{}) (*mcpsdk.CallToolResult, any, error) {
			return &mcpsdk.CallToolResult{}, nil, nil
		})
	srv := httptest.NewServer(mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server { return server }, nil))
	t.Cleanup(srv.Close)
	cfg := config.NewTestStore(&config.Config{
		MCP:     config.MCPs{"p2d-off": {Type: config.MCPHttp, URL: srv.URL}},
		Options: &config.Options{DataDirectory: t.TempDir()},
	})
	t.Cleanup(func() { mcp.Reinitialize(context.Background(), config.NewTestStore(&config.Config{})) })
	require.NoError(t, mcp.InitializeSingle(t.Context(), "p2d-off", cfg))
	raw := []fantasy.AgentTool{&fakeTool{name: "plain"}}
	for _, tl := range tools.GetMCPTools(nil, cfg, t.TempDir()) {
		raw = append(raw, tl)
	}
	require.Equal(t, []string{"plain", "mcp_p2d-off_send"}, toolNames(raw))

	dataDir := t.TempDir()
	t.Cleanup(func() { require.NoError(t, db.Release(dataDir)) })
	conn, err := db.Connect(t.Context(), dataDir)
	require.NoError(t, err)
	sessions := session.NewService(db.New(conn), conn)
	require.NoError(t, sessions.SetMCPServerDisabled(t.Context(), "p2d-off", true))
	a := &sessionAgent{sessions: sessions}

	cases := map[string][]fantasy.AgentTool{
		"no hook":   raw,
		"with hook": wrapToolsWithHooks(raw, newRunner(t, `exit 0`), false),
	}
	for name, ts := range cases {
		t.Run(name, func(t *testing.T) {
			got := a.filterDisabledMCPTools(t.Context(), ts)
			require.Equal(t, []string{"plain"}, toolNames(got))
			requireHookKept(t, name, got)
		})
	}
}

// requireHookKept: bộ lọc chỉ được nhìn xuyên lớp bọc để nhận diện, không được
// trả tool đã bóc, kẻo tool lọt qua mà không chạy hook PreToolUse.
func requireHookKept(t *testing.T, name string, got []fantasy.AgentTool) {
	t.Helper()
	if name != "with hook" {
		return
	}
	for _, x := range got {
		require.IsType(t, &hookedTool{}, x)
	}
}
