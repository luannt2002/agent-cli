package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/agent/tools/mcp"
	"github.com/charmbracelet/crush/internal/config"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// connectedMCP trả trạng thái một MCP server đã kết nối (in-memory) có chỉ
// dẫn instr trong InitializeResult.
func connectedMCP(t *testing.T, name, instr string) mcp.ClientInfo {
	t.Helper()
	serverTransport, clientTransport := gomcp.NewInMemoryTransports()
	server := gomcp.NewServer(&gomcp.Implementation{Name: name}, &gomcp.ServerOptions{Instructions: instr})
	ss, err := server.Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := gomcp.NewClient(&gomcp.Implementation{Name: "crush-test"}, nil).Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })
	return mcp.ClientInfo{Name: name, State: mcp.StateConnected, Client: &mcp.ClientSession{ClientSession: cs}}
}

// Chỉ dẫn của một MCP server chỉ vào system prompt của lượt có đường tới
// server đó: agent con, agentic_fetch, plan (không có tool MCP) không nhận.
func TestMCPInstructionsFollowRunTools(t *testing.T) {
	t.Parallel()
	states := map[string]mcp.ClientInfo{
		"alpha": connectedMCP(t, "alpha", "ALPHA-INSTR"),
		"beta":  connectedMCP(t, "beta", "BETA-INSTR"),
		// Chưa kết nối: không có Client, không được đọc tới.
		"gamma": {Name: "gamma", State: mcp.StateError},
	}
	alpha := &channelTestTool{name: "mcp_alpha_x", mcpName: "alpha"}
	beta := &channelTestTool{name: "mcp_beta_x", mcpName: "beta"}
	gamma := &channelTestTool{name: "mcp_gamma_x", mcpName: "gamma"}
	view := &fakeTool{name: "view"}

	require.Empty(t, mcpInstructions(states, []fantasy.AgentTool{view}),
		"an agent with no MCP tool must not get <mcp-instructions>")
	require.Empty(t, mcpInstructions(states, nil))

	onlyAlpha := mcpInstructions(states, []fantasy.AgentTool{view, alpha})
	require.Contains(t, onlyAlpha, "<mcp-instructions>")
	require.Contains(t, onlyAlpha, "ALPHA-INSTR")
	require.NotContains(t, onlyAlpha, "BETA-INSTR", "beta's instructions without any beta tool")

	// Có PreToolUse hook thì tool của agent chính được bọc bởi hookedTool.
	hooked := mcpInstructions(states, []fantasy.AgentTool{newHookedTool(alpha, nil)})
	require.Contains(t, hooked, "ALPHA-INSTR")
	require.NotContains(t, hooked, "BETA-INSTR")

	// list/read_mcp_resource nhận tên server bất kỳ: lượt có chúng chạm tới
	// mọi server, như agent chính hiện nay.
	for _, name := range []string{tools.ListMCPResourcesToolName, tools.ReadMCPResourceToolName} {
		got := mcpInstructions(states, []fantasy.AgentTool{&fakeTool{name: name}})
		require.Contains(t, got, "ALPHA-INSTR", name)
		require.Contains(t, got, "BETA-INSTR", name)
	}

	// Đối chứng agent chính (đủ tool MCP): nhận đủ, server chưa kết nối bị bỏ.
	all := mcpInstructions(states, []fantasy.AgentTool{view, alpha, beta, gamma,
		&fakeTool{name: tools.ListMCPResourcesToolName}, &fakeTool{name: tools.ReadMCPResourceToolName}})
	require.Contains(t, all, "ALPHA-INSTR")
	require.Contains(t, all, "BETA-INSTR")

	// Định dạng khối giữ nguyên từng byte (một server để thứ tự cố định).
	one := map[string]mcp.ClientInfo{"alpha": states["alpha"]}
	require.Equal(t, "\n\n<mcp-instructions>\nALPHA-INSTR\n\n\n</mcp-instructions>",
		mcpInstructions(one, []fantasy.AgentTool{view, alpha}))
}

// Thứ tự khối chỉ dẫn cố định theo tên server: system prompt đổi giữa các
// lượt thì prompt cache của provider trượt.
func TestMCPInstructionsOrderIsStable(t *testing.T) {
	t.Parallel()
	states := map[string]mcp.ClientInfo{
		"beta":  connectedMCP(t, "beta", "BETA-INSTR"),
		"alpha": connectedMCP(t, "alpha", "ALPHA-INSTR"),
		"gamma": connectedMCP(t, "gamma", "GAMMA-INSTR"),
	}
	reach := []fantasy.AgentTool{&fakeTool{name: tools.ReadMCPResourceToolName}}
	want := "\n\n<mcp-instructions>\nALPHA-INSTR\n\nBETA-INSTR\n\nGAMMA-INSTR\n\n\n</mcp-instructions>"
	for i := range 50 {
		require.Equal(t, want, mcpInstructions(states, reach), "call %d", i)
	}
}

// promptRecorder ghi prompt của mỗi lần Stream rồi trả "done".
type promptRecorder struct {
	finishStreamModel
	mu      sync.Mutex
	prompts []fantasy.Prompt
}

func (m *promptRecorder) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	m.mu.Lock()
	m.prompts = append(m.prompts, call.Prompt)
	m.mu.Unlock()
	return m.finishStreamModel.Stream(ctx, call)
}

// Run lấy chỉ dẫn MCP theo bộ tool của chính lượt đó, đi qua trạng thái MCP
// thật (mcp.GetStates). Không chạy song song: trạng thái MCP là của cả
// tiến trình.
func TestRunMCPInstructionsFollowRunTools(t *testing.T) {
	const name = "p2a-run-instr"
	server := gomcp.NewServer(&gomcp.Implementation{Name: name}, &gomcp.ServerOptions{Instructions: "P2A-RUN-INSTR"})
	hs := httptest.NewServer(gomcp.NewStreamableHTTPHandler(func(*http.Request) *gomcp.Server { return server }, nil))
	t.Cleanup(hs.Close)
	cfg := config.NewTestStore(&config.Config{MCP: config.MCPs{name: {Type: config.MCPHttp, URL: hs.URL}}})
	// Dọn trước khi khởi tạo: khởi tạo lỗi vẫn để lại trạng thái cần gỡ.
	t.Cleanup(func() { _ = mcp.DisableSingle(cfg, name) })
	require.NoError(t, mcp.InitializeSingle(t.Context(), name, cfg))
	state, ok := mcp.GetState(name)
	require.True(t, ok)
	require.Equal(t, mcp.StateConnected, state.State, "precondition: the MCP server is connected")

	for _, tc := range []struct {
		name  string
		tools []fantasy.AgentTool
		want  bool
	}{
		{"no MCP tool", []fantasy.AgentTool{namedTool("view")}, false},
		{"tool of the server", []fantasy.AgentTool{namedTool("view"), &channelTestTool{name: "mcp_" + name + "_x", mcpName: name}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := testEnv(t)
			rec := &promptRecorder{finishStreamModel: finishStreamModel{text: "done"}}
			cw := catwalk.Model{ContextWindow: 100000, DefaultMaxTokens: 100}
			sa := NewSessionAgent(SessionAgentOptions{
				LargeModel:   modelWith(rec, "prov", "model", cw),
				SmallModel:   modelWith(&finishStreamModel{text: "title"}, "prov", "small", cw),
				SystemPrompt: "base system prompt",
				IsYolo:       true,
				Sessions:     env.sessions,
				Messages:     env.messages,
				Tools:        tc.tools,
			})
			sess, err := env.sessions.Create(t.Context(), "session")
			require.NoError(t, err)
			_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, Prompt: "hello"})
			require.NoError(t, err)

			rec.mu.Lock()
			defer rec.mu.Unlock()
			require.NotEmpty(t, rec.prompts, "the model was never called")
			var system strings.Builder
			for _, msg := range rec.prompts[0] {
				if msg.Role != fantasy.MessageRoleSystem {
					continue
				}
				for _, part := range msg.Content {
					if tp, ok := fantasy.AsMessagePart[fantasy.TextPart](part); ok {
						system.WriteString(tp.Text)
					}
				}
			}
			require.Contains(t, system.String(), "base system prompt", "test setup: system prompt not found")
			require.Equal(t, tc.want, strings.Contains(system.String(), "P2A-RUN-INSTR"), "system prompt: %q", system.String())
		})
	}
}
