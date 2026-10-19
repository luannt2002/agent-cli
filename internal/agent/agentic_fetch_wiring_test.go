package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/stretchr/testify/require"
)

// fetchTestEnv: coordinator với disabled_tools cho trước, provider giả trả 400 ngay
// (agent con thất bại nhanh, không chạm mạng thật) và một trang /page đếm lượt tải.
type fetchTestEnv struct {
	coord     *coordinator
	srv       *httptest.Server
	pageHits  atomic.Int32
	modelHits atomic.Int32
}

func newFetchTestEnv(t *testing.T, disabled []string, perms permission.Service) *fetchTestEnv {
	t.Helper()
	e := &fetchTestEnv{}
	e.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/page" {
			e.pageHits.Add(1)
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "hello")
			return
		}
		e.modelHits.Add(1)
		http.Error(w, `{"error":{"message":"bad request"}}`, http.StatusBadRequest)
	}))
	t.Cleanup(e.srv.Close)

	env := testEnv(t)
	disabledJSON, err := json.Marshal(disabled)
	require.NoError(t, err)
	crushJSON := fmt.Sprintf(`{
  "options": {"disable_default_providers": true, "disable_provider_auto_update": true, "disabled_tools": %s},
  "providers": {"mock": {"id": "mock", "name": "Mock", "type": "openai",
    "base_url": %q, "api_key": "test-key",
    "models": [{"id": "mock-model", "name": "Mock", "context_window": 8192, "default_max_tokens": 128}]}},
  "models": {"large": {"provider": "mock", "model": "mock-model"},
             "small": {"provider": "mock", "model": "mock-model"}}
}`, disabledJSON, e.srv.URL+"/v1")
	require.NoError(t, os.WriteFile(filepath.Join(env.workingDir, "crush.json"), []byte(crushJSON), 0o644))
	cfg, err := config.Init(env.workingDir, "", false)
	require.NoError(t, err)
	cfg.SetupAgents()
	require.NoError(t, os.MkdirAll(cfg.Config().Options.DataDirectory, 0o755))
	if perms == nil {
		perms = env.permissions
	}
	e.coord = &coordinator{
		cfg:         cfg,
		sessions:    env.sessions,
		messages:    env.messages,
		permissions: perms,
		history:     env.history,
		filetracker: *env.filetracker,
		agents:      make(map[string]SessionAgent),
	}
	return e
}

func toolNames(ts []fantasy.AgentTool) []string {
	var names []string
	for _, tl := range ts {
		names = append(names, tl.Info().Name)
	}
	return names
}

// runAgenticFetch gọi agentic_fetch như agent chính gọi, trong một phiên cha mới.
func (e *fetchTestEnv) runAgenticFetch(t *testing.T, params tools.AgenticFetchParams) (parentID, childID string, resp fantasy.ToolResponse) {
	t.Helper()
	parent, err := e.coord.sessions.Create(t.Context(), "Parent")
	require.NoError(t, err)
	tool, err := e.coord.agenticFetchTool(t.Context(), e.srv.Client())
	require.NoError(t, err)
	input, err := json.Marshal(params)
	require.NoError(t, err)
	ctx := context.WithValue(t.Context(), tools.SessionIDContextKey, parent.ID)
	ctx = context.WithValue(ctx, tools.MessageIDContextKey, "msg-1")
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	resp, err = tool.Run(ctx, fantasy.ToolCall{ID: "call-1", Name: tools.AgenticFetchToolName, Input: string(input)})
	require.NoError(t, err)
	return parent.ID, e.coord.sessions.CreateAgentToolSessionID("msg-1", "call-1"), resp
}

// Tắt cả web_fetch lẫn web_search thì agentic_fetch không được đăng ký; còn một
// trong hai thì vẫn có.
func TestBuildToolsGatesAgenticFetchOnWebTools(t *testing.T) {
	for _, tc := range []struct {
		disabled []string
		want     bool
	}{
		{[]string{tools.WebFetchToolName, tools.WebSearchToolName}, false},
		{[]string{tools.WebFetchToolName}, true},
		{nil, true},
	} {
		t.Run(fmt.Sprint(tc.disabled), func(t *testing.T) {
			e := newFetchTestEnv(t, tc.disabled, nil)
			agentCfg := e.coord.cfg.Config().Agents[config.AgentCoder]
			require.Contains(t, agentCfg.AllowedTools, tools.AgenticFetchToolName, "disabled_tools does not name agentic_fetch itself")
			ts, err := e.coord.buildTools(t.Context(), agentCfg, false)
			require.NoError(t, err)
			require.Equal(t, tc.want, contains(toolNames(ts), tools.AgenticFetchToolName))
		})
	}
}

func contains(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

// Bộ tool bên trong agentic_fetch lấy disabled_tools từ cấu hình thật.
func TestAgenticFetchInnerToolsRespectDisabledTools(t *testing.T) {
	e := newFetchTestEnv(t, []string{tools.WebSearchToolName, tools.SourcegraphToolName}, nil)
	names := toolNames(e.coord.agenticFetchInnerTools(t.TempDir(), nil))
	require.NotContains(t, names, tools.WebSearchToolName)
	require.NotContains(t, names, tools.SourcegraphToolName)
	require.Contains(t, names, tools.WebFetchToolName)
	require.Contains(t, names, tools.ViewToolName)
}

// Tắt web_fetch (web_search còn) thì chế độ URL không tự tải URL; chế độ tìm kiếm
// vẫn chạy tới agent con.
func TestAgenticFetchURLModeRefusedWhenWebFetchDisabled(t *testing.T) {
	e := newFetchTestEnv(t, []string{tools.WebFetchToolName}, nil)
	_, _, resp := e.runAgenticFetch(t, tools.AgenticFetchParams{URL: e.srv.URL + "/page", Prompt: "summarise"})
	require.True(t, resp.IsError, resp.Content)
	require.Contains(t, resp.Content, tools.WebFetchToolName)
	require.Zero(t, e.pageHits.Load(), "the URL was fetched although web_fetch is disabled")

	_, _, resp = e.runAgenticFetch(t, tools.AgenticFetchParams{Prompt: "find something"})
	require.NotContains(t, resp.Content, "disabled")
	require.NotZero(t, e.modelHits.Load(), "search mode must still reach the sub-agent")
}

// Có người trực: phiên con của agentic_fetch không được tự duyệt; yêu cầu quyền của
// nó phải hiện hộp thoại (nội dung web có thể tiêm lệnh đọc tệp ngoài thư mục tạm).
func TestAgenticFetchChildSessionNotAutoApproved(t *testing.T) {
	perms := permission.NewPermissionService(t.TempDir(), false, nil)
	e := newFetchTestEnv(t, nil, perms)
	events := perms.Subscribe(t.Context())
	prompted := make(chan permission.PermissionRequest, 8)
	go func() {
		for ev := range events {
			if ev.Payload.ToolName == tools.AgenticFetchToolName {
				perms.Grant(ev.Payload) // người dùng cho phép lệnh agentic_fetch ở phiên cha
				continue
			}
			prompted <- ev.Payload
		}
	}()
	_, childID, _ := e.runAgenticFetch(t, tools.AgenticFetchParams{URL: e.srv.URL + "/page", Prompt: "summarise"})
	require.NotZero(t, e.modelHits.Load(), "the sub-agent ran")
	_, err := e.coord.sessions.Get(t.Context(), childID)
	require.NoError(t, err, "the child session exists")

	reqCtx, reqCancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer reqCancel()
	ok, _ := perms.Request(reqCtx, permission.CreatePermissionRequest{
		SessionID: childID, ToolCallID: "inner-1", ToolName: tools.ViewToolName, Action: "read", Path: "/etc",
	})
	require.False(t, ok, "the agentic_fetch child session must not be auto-approved")
	select {
	case p := <-prompted:
		require.Equal(t, childID, p.SessionID)
	default:
		t.Fatal("the child's request did not reach the permission dialog")
	}
}

// L5 qua đường thật: disabled_tools: [fetch] bỏ web_fetch khỏi tool bên trong, chặn
// chế độ URL (không tải trang), và cùng web_search thì agentic_fetch không được
// đăng ký.
func TestDisabledFetchAlsoGatesAgenticFetch(t *testing.T) {
	e := newFetchTestEnv(t, []string{tools.FetchToolName}, nil)
	names := toolNames(e.coord.agenticFetchInnerTools(t.TempDir(), nil))
	require.NotContains(t, names, tools.WebFetchToolName)
	require.Contains(t, names, tools.WebSearchToolName)

	_, _, resp := e.runAgenticFetch(t, tools.AgenticFetchParams{URL: e.srv.URL + "/page", Prompt: "summarise"})
	require.True(t, resp.IsError, resp.Content)
	require.Zero(t, e.pageHits.Load(), "the URL was fetched although fetch is disabled")

	e = newFetchTestEnv(t, []string{tools.FetchToolName, tools.WebSearchToolName}, nil)
	agentCfg := e.coord.cfg.Config().Agents[config.AgentCoder]
	ts, err := e.coord.buildTools(t.Context(), agentCfg, false)
	require.NoError(t, err)
	require.NotContains(t, toolNames(ts), tools.AgenticFetchToolName)
}
