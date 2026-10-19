package agent

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"time"

	"charm.land/fantasy"

	"github.com/charmbracelet/crush/internal/agent/prompt"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/permission"
)

//go:embed templates/agentic_fetch.md
var agenticFetchToolDescription string

// agenticFetchValidationResult holds the validated parameters from the tool call context.
type agenticFetchValidationResult struct {
	SessionID      string
	AgentMessageID string
}

// validateAgenticFetchParams validates the tool call parameters and extracts required context values.
func validateAgenticFetchParams(ctx context.Context, params tools.AgenticFetchParams) (agenticFetchValidationResult, error) {
	if params.Prompt == "" {
		return agenticFetchValidationResult{}, errors.New("prompt is required")
	}

	sessionID := tools.GetSessionFromContext(ctx)
	if sessionID == "" {
		return agenticFetchValidationResult{}, errors.New("session id missing from context")
	}

	agentMessageID := tools.GetMessageFromContext(ctx)
	if agentMessageID == "" {
		return agenticFetchValidationResult{}, errors.New("agent message id missing from context")
	}

	return agenticFetchValidationResult{
		SessionID:      sessionID,
		AgentMessageID: agentMessageID,
	}, nil
}

//go:embed templates/agentic_fetch_prompt.md.tpl
var agenticFetchPromptTmpl []byte

func (c *coordinator) agenticFetchTool(_ context.Context, client *http.Client) (fantasy.AgentTool, error) {
	if client == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.MaxIdleConns = 100
		transport.MaxIdleConnsPerHost = 10
		transport.IdleConnTimeout = 90 * time.Second

		client = &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		}
	}

	return fantasy.NewParallelAgentTool(
		tools.AgenticFetchToolName,
		agenticFetchToolDescription,
		func(ctx context.Context, params tools.AgenticFetchParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			validationResult, err := validateAgenticFetchParams(ctx, params)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			// Chế độ URL tự tải URL trước khi agent con chạy, nên tắt web_fetch phải
			// chặn ở đây; chế độ tìm kiếm vẫn được nếu web_search còn bật.
			if params.URL != "" && slices.Contains(agenticFetchDisabled(c.cfg.Config().Options.DisabledTools), tools.WebFetchToolName) {
				return fantasy.NewTextErrorResponse("URL mode is unavailable: the web_fetch tool is disabled (fetch or web_fetch is in disabled_tools). Omit url to search the web instead."), nil
			}

			// Determine description based on mode.
			var description string
			if params.URL != "" {
				description = fmt.Sprintf("Fetch and analyze content from URL: %s", params.URL)
			} else {
				description = "Search the web and analyze results"
			}

			p, err := c.permissions.Request(
				ctx,
				permission.CreatePermissionRequest{
					SessionID:   validationResult.SessionID,
					Path:        c.cfg.WorkingDir(),
					ToolCallID:  call.ID,
					ToolName:    tools.AgenticFetchToolName,
					Action:      "fetch",
					Description: description,
					Params:      tools.AgenticFetchPermissionsParams(params),
				},
			)
			if err != nil {
				return fantasy.ToolResponse{}, err
			}
			if !p {
				return tools.NewPermissionDeniedResponse(), nil
			}

			tmpDir, err := os.MkdirTemp(c.cfg.Config().Options.DataDirectory, "crush-fetch-*")
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("Failed to create temporary directory: %s", err)), nil
			}
			defer os.RemoveAll(tmpDir)

			var fullPrompt string

			if params.URL != "" {
				// URL mode: fetch the URL content first.
				content, err := tools.FetchURLAndConvert(ctx, client, params.URL)
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("Failed to fetch URL: %s", err)), nil
				}

				hasLargeContent := len(content) > tools.LargeContentThreshold

				if hasLargeContent {
					tempFile, err := os.CreateTemp(tmpDir, "page-*.md")
					if err != nil {
						return fantasy.NewTextErrorResponse(fmt.Sprintf("Failed to create temporary file: %s", err)), nil
					}
					tempFilePath := tempFile.Name()

					if _, err := tempFile.WriteString(content); err != nil {
						tempFile.Close()
						return fantasy.NewTextErrorResponse(fmt.Sprintf("Failed to write content to file: %s", err)), nil
					}
					tempFile.Close()

					fullPrompt = fmt.Sprintf("%s\n\nThe web page from %s has been saved to: %s\n\nUse the view and grep tools to analyze this file and extract the requested information.", params.Prompt, params.URL, tempFilePath)
				} else {
					fullPrompt = fmt.Sprintf("%s\n\nWeb page URL: %s\n\n<webpage_content>\n%s\n</webpage_content>", params.Prompt, params.URL, content)
				}
			} else {
				// Search mode: let the sub-agent search and fetch as needed.
				fullPrompt = fmt.Sprintf("%s\n\nUse the web_search tool to find relevant information. Break down the question into smaller, focused searches if needed. After searching, use web_fetch to get detailed content from the most relevant results.", params.Prompt)
			}

			promptOpts := []prompt.Option{
				prompt.WithWorkingDir(tmpDir),
			}

			promptTemplate, err := prompt.NewPrompt("agentic_fetch", string(agenticFetchPromptTmpl), promptOpts...)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("error creating prompt: %s", err)
			}

			_, small, err := c.buildAgentModels(ctx, true)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("error building models: %s", err)
			}

			systemPrompt, err := promptTemplate.Build(ctx, small.Model.Provider(), small.Model.Model(), c.cfg)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("error building system prompt: %s", err)
			}

			smallProviderCfg, ok := c.cfg.Config().Providers.Get(small.ModelCfg.Provider)
			if !ok {
				return fantasy.ToolResponse{}, errors.New("small model provider not configured")
			}

			fetchTools := c.agenticFetchInnerTools(tmpDir, client)

			// Sub-agent tools run without hook interception. The top-level
			// `agentic_fetch` call itself is already wrapped from the coder's
			// side; firing hooks again for every inner tool call would run
			// the user's hooks N times per delegated turn.

			agent := NewSessionAgent(SessionAgentOptions{
				LargeModel:           small, // Use small model for both (fetch doesn't need large)
				SmallModel:           small,
				SystemPromptPrefix:   smallProviderCfg.SystemPromptPrefix,
				SystemPrompt:         systemPrompt,
				DisableAutoSummarize: c.cfg.Config().Options.DisableAutoSummarize,
				IsYolo:               c.permissions.SkipRequests(),
				Sessions:             c.sessions,
				Messages:             c.messages,
				Tools:                fetchTools,
			})

			return c.runSubAgent(withLLMRole(ctx, "agentic_fetch"), subAgentParams{
				Agent:          agent,
				SessionID:      validationResult.SessionID,
				AgentMessageID: validationResult.AgentMessageID,
				ToolCallID:     call.ID,
				Prompt:         fullPrompt,
				SessionTitle:   "Fetch Analysis",
				// Không tự duyệt session con: nội dung web có thể tiêm lệnh bảo agent
				// đọc tệp ngoài thư mục tạm (rồi gửi đi bằng web_fetch). Việc thường
				// của nó (web_fetch/web_search, view/grep trong thư mục tạm) không cần
				// quyền. Đọc ra ngoài: TUI hỏi người dùng; `run` không người hỏi nên
				// duyệt mọi phiên con như phiên gốc (app.autoApproveRun) (audit g5).
			})
		},
	), nil
}

func (c *coordinator) agenticFetchInnerTools(tmpDir string, client *http.Client) []fantasy.AgentTool {
	return filterDisabledTools([]fantasy.AgentTool{
		tools.NewWebFetchTool(tmpDir, client),
		tools.NewWebSearchTool(client),
		tools.NewGlobTool(c.permissions, tmpDir, c.cfg.Config().Tools.Glob),
		tools.NewGrepTool(c.permissions, tmpDir, c.cfg.Config().Tools.Grep),
		tools.NewSourcegraphTool(client),
		tools.NewViewTool(c.lspManager, c.permissions, c.filetracker, nil, tmpDir),
	}, agenticFetchDisabled(c.cfg.Config().Options.DisabledTools))
}

// filterDisabledTools bỏ các tool có tên trong disabled_tools của người dùng: luật
// tắt tool áp cho cả agent con, không chỉ agent chính.
func filterDisabledTools(ts []fantasy.AgentTool, disabled []string) []fantasy.AgentTool {
	out := ts[:0:0]
	for _, t := range ts {
		if !slices.Contains(disabled, t.Info().Name) {
			out = append(out, t)
		}
	}
	return out
}

// agenticFetchUsable: tắt cả web_fetch lẫn web_search thì agentic_fetch không còn
// cách nào lên web, nên không đưa nó cho model.
func agenticFetchUsable(disabled []string) bool {
	disabled = agenticFetchDisabled(disabled)
	return !slices.Contains(disabled, tools.WebFetchToolName) || !slices.Contains(disabled, tools.WebSearchToolName)
}

// agenticFetchDisabled: danh sách tool người dùng thấy chỉ có "fetch", không có
// web_fetch; tắt "fetch" mà agentic_fetch vẫn tải được URL là lọt.
func agenticFetchDisabled(disabled []string) []string {
	if slices.Contains(disabled, tools.FetchToolName) && !slices.Contains(disabled, tools.WebFetchToolName) {
		return append(slices.Clone(disabled), tools.WebFetchToolName)
	}
	return disabled
}
