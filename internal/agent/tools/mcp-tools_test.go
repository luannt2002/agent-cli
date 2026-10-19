package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/tools/mcp"
	"github.com/charmbracelet/crush/internal/config"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// runMCPStub dựng một MCP server HTTP thật trong tiến trình, có một tool trả
// đúng text và cờ isError cho trước, nối nó qua InitializeSingle như lúc
// chạy thật, gọi tool đó một lần qua *Tool rồi trả về phản hồi mà agent sẽ
// gửi cho model, kèm thư mục data của cấu hình.
func runMCPStub(t *testing.T, name, text string, isError bool) (fantasy.ToolResponse, string) {
	t.Helper()

	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: name}, nil)
	mcpsdk.AddTool(
		server,
		&mcpsdk.Tool{Name: "out", Description: "test tool"},
		func(context.Context, *mcpsdk.CallToolRequest, struct{}) (*mcpsdk.CallToolResult, any, error) {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: text}},
				IsError: isError,
			}, nil, nil
		},
	)
	srv := httptest.NewServer(mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return server }, nil,
	))
	t.Cleanup(srv.Close)

	dataDir := t.TempDir()
	cfg := config.NewTestStore(&config.Config{
		MCP:     config.MCPs{name: {Type: config.MCPHttp, URL: srv.URL}},
		Options: &config.Options{DataDirectory: dataDir},
	})
	// Registry MCP là toàn cục: các test này chạy tuần tự (không t.Parallel)
	// và gỡ hẳn server khi xong, để test song song đọc trạng thái MCP (vd
	// crush_info) không thấy server của test này. Đăng ký dọn TRƯỚC khi khởi
	// tạo: khởi tạo lỗi vẫn để lại trạng thái Starting/Error cần gỡ.
	t.Cleanup(func() { mcp.Reinitialize(context.Background(), config.NewTestStore(&config.Config{})) })
	require.NoError(t, mcp.InitializeSingle(t.Context(), name, cfg))

	var tool *Tool
	for _, tl := range GetMCPTools(&mockPermissionService{}, cfg, t.TempDir()) {
		if tl.MCP() == name {
			tool = tl
		}
	}
	require.NotNil(t, tool, "MCP tool must be registered after InitializeSingle")

	ctx := context.WithValue(t.Context(), SessionIDContextKey, "s")
	resp, err := tool.Run(ctx, fantasy.ToolCall{ID: "c1", Name: tool.Name(), Input: "{}"})
	require.NoError(t, err)
	return resp, dataDir
}

// Kết quả MCP rất lớn phải có trần như bash: model chỉ nhận đầu + đuôi, bản
// đầy đủ nằm trong tệp spill cùng thư mục với bash (<data>/shell-output).
func TestMCPTool_LargeTextIsCappedAndSpilled(t *testing.T) {
	var b strings.Builder
	for i := 0; b.Len() < 5<<20; i++ {
		fmt.Fprintf(&b, "line %07d %s\n", i, strings.Repeat("x", 48))
	}
	big := b.String()

	resp, dataDir := runMCPStub(t, "p1b-big", big, false)

	require.False(t, resp.IsError)
	require.Less(t, len(resp.Content), MaxOutputLength+1024, "MCP output sent to the model must be capped like bash")
	m := regexp.MustCompile(`full output: (\S+)\]`).FindStringSubmatch(resp.Content)
	require.NotNil(t, m, "capped MCP output must point at the spill file")
	require.Equal(t, filepath.Join(dataDir, "shell-output"), filepath.Dir(m[1]), "spill must reuse the bash spill directory")
	full, err := os.ReadFile(m[1])
	require.NoError(t, err)
	require.Equal(t, big, string(full), "spill file must hold the full MCP output")
}

// Đối chứng: kết quả MCP nhỏ đi nguyên vẹn, không bị cắt.
func TestMCPTool_SmallTextIsUnchanged(t *testing.T) {
	resp, dataDir := runMCPStub(t, "p1b-small", "echo", false)

	require.False(t, resp.IsError)
	require.Equal(t, "echo", resp.Content)
	require.NoDirExists(t, filepath.Join(dataDir, "shell-output"))
}

// Tool MCP báo isError=true thì phản hồi gửi model cũng phải là lỗi, như tool
// nội bộ lỗi (view tệp không có), để tool_result lưu is_error=true.
func TestMCPTool_IsErrorReachesToolResponse(t *testing.T) {
	resp, _ := runMCPStub(t, "p1b-fail", "V3-MCP-FAIL boom", true)

	require.True(t, resp.IsError)
	require.Equal(t, "V3-MCP-FAIL boom", resp.Content)
}
