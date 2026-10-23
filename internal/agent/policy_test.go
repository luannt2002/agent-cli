package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/stretchr/testify/require"
)

// Model chính (large/small) ngoài danh sách bị chặn trước khi dựng provider:
// mọi lượt, sub-agent và đặt tiêu đề dựng model qua đây. Không đặt danh sách
// thì như cũ.
func TestCheckModelPolicy(t *testing.T) {
	t.Parallel()
	pol, err := execpolicy.Parse("zai:glm-5.3 zai:glm-5.3-flash main:qwen3.8")
	require.NoError(t, err)

	glm := config.SelectedModel{Provider: "zai", Model: "glm-5.3"}
	flash := config.SelectedModel{Provider: "zai", Model: "glm-5.3-flash"}
	require.NoError(t, checkModelPolicy(pol, glm, flash))
	require.NoError(t, checkModelPolicy(pol, config.SelectedModel{Provider: "main", Model: "qwen3.8"}))

	for _, bad := range []config.SelectedModel{
		{Provider: "zai", Model: "glm-4.6"},
		{Provider: "zai", Model: "glm-5.4"},
		{Provider: "openai", Model: "gpt-6.1-sol"},
	} {
		err := checkModelPolicy(pol, glm, bad)
		var denied *execpolicy.Denied
		require.ErrorAsf(t, err, &denied, "%v", bad)
		require.Equal(t, execpolicy.CodeModelNotAllowed, denied.Code)
	}

	require.NoError(t, checkModelPolicy(execpolicy.Policy{}, config.SelectedModel{Provider: "any", Model: "thing"}))
}

// Chốt allowlist trên ĐƯỜNG THẬT của coordinator (buildAgentModels): execpolicy trả
// chính sách rỗng trong test binary, nên trước đây xoá lời gọi checkModelPolicy mà
// `go test ./...` vẫn xanh (audit 22/10). Danh sách thiếu model large hay small đang
// chọn thì MODEL_NOT_ALLOWED; có đủ thì dựng được.
func TestBuildAgentModelsEnforcesTheModelAllowlist(t *testing.T) {
	env := testEnv(t)
	crushJSON := `{
  "options": {"disable_default_providers": true, "disable_provider_auto_update": true},
  "providers": {
    "qwen": {"id": "qwen", "name": "Qwen", "type": "openai-compat", "base_url": "http://127.0.0.1:9/qwen/v1", "api_key": "x",
      "models": [{"id": "qwen3-coder", "name": "q", "context_window": 8192, "default_max_tokens": 128},
                 {"id": "qwen3-mini", "name": "m", "context_window": 8192, "default_max_tokens": 128}]}},
  "models": {"large": {"provider": "qwen", "model": "qwen3-coder"},
             "small": {"provider": "qwen", "model": "qwen3-mini"}}
}`
	require.NoError(t, os.WriteFile(filepath.Join(env.workingDir, "crush.json"), []byte(crushJSON), 0o644))
	cfg, err := config.Init(env.workingDir, "", false)
	require.NoError(t, err)
	cfg.SetupAgents()
	build := func(spec string) error {
		coord := &coordinator{cfg: cfg, sessions: env.sessions, messages: env.messages, permissions: env.permissions, history: env.history, filetracker: *env.filetracker,
			policy: func() (execpolicy.Policy, error) { return execpolicy.Parse(spec) }}
		_, _, err := coord.buildAgentModels(t.Context(), false)
		return err
	}

	for name, spec := range map[string]string{
		"large not listed": "qwen:qwen3-mini",
		"small not listed": "qwen:qwen3-coder",
		"other provider":   "zai:glm-5.3",
	} {
		err := build(spec)
		var denied *execpolicy.Denied
		require.ErrorAsf(t, err, &denied, "%s", name)
		require.Equalf(t, execpolicy.CodeModelNotAllowed, denied.Code, "%s", name)
		require.Equalf(t, "qwen", denied.Provider, "%s", name)
	}
	require.NoError(t, build("qwen:qwen3-coder qwen:qwen3-mini"), "both selected models listed")
	require.NoError(t, build(""), "no list: unrestricted as before")
}
