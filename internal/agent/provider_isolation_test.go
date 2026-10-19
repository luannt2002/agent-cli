package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/stretchr/testify/require"
)

// Phiên chạy trên provider X: việc phụ (tiêu đề, agentic_fetch) không được gửi
// prompt sang provider Y chỉ vì model small cấu hình ở Y (probe P-01).
func TestSameProviderSmall(t *testing.T) {
	t.Parallel()
	qwen := config.SelectedModel{Provider: "qwen", Model: "qwen3-coder"}
	glmFlash := config.SelectedModel{Provider: "glm", Model: "glm-5.3-flash"}
	glm := config.SelectedModel{Provider: "glm", Model: "glm-5.3"}
	require.Equal(t, qwen, sameProviderSmall(qwen, glmFlash), "cross-provider small falls back to the main model")
	require.Equal(t, glmFlash, sameProviderSmall(glm, glmFlash), "same-provider small is kept")
}

// Cả đường dựng model thật của coordinator: large ở qwen, small ở glm thì model
// small trả ra là của qwen.
func TestBuildAgentModelsKeepsSmallOnTheMainProvider(t *testing.T) {
	env := testEnv(t)
	crushJSON := `{
  "options": {"disable_default_providers": true, "disable_provider_auto_update": true},
  "providers": {
    "qwen": {"id": "qwen", "name": "Qwen", "type": "openai-compat", "base_url": "http://127.0.0.1:9/qwen/v1", "api_key": "x",
      "models": [{"id": "qwen3-coder", "name": "q", "context_window": 8192, "default_max_tokens": 128}]},
    "glm": {"id": "glm", "name": "GLM", "type": "openai-compat", "base_url": "http://127.0.0.1:9/glm/v1", "api_key": "y",
      "models": [{"id": "glm-5.3-flash", "name": "g", "context_window": 8192, "default_max_tokens": 128}]}},
  "models": {"large": {"provider": "qwen", "model": "qwen3-coder"},
             "small": {"provider": "glm", "model": "glm-5.3-flash"}}
}`
	require.NoError(t, os.WriteFile(filepath.Join(env.workingDir, "crush.json"), []byte(crushJSON), 0o644))
	cfg, err := config.Init(env.workingDir, "", false)
	require.NoError(t, err)
	cfg.SetupAgents()
	coord := &coordinator{cfg: cfg, sessions: env.sessions, messages: env.messages, permissions: env.permissions, history: env.history, filetracker: *env.filetracker}

	large, small, err := coord.buildAgentModels(t.Context(), false)
	require.NoError(t, err)
	require.Equal(t, "qwen", large.ModelCfg.Provider)
	require.Equal(t, "qwen", small.ModelCfg.Provider)
	require.Equal(t, "qwen3-coder", small.ModelCfg.Model)
}

// F1: model large chọn ở provider dựng sẵn (openai) nhưng chưa có khoá, còn một
// provider khác (qwen) dùng được. Nạp cấu hình vẫn được (để còn `agent-cli login`),
// lựa chọn giữ ở openai, và dựng agent thất bại nêu tên provider, không lặng lẽ
// chạy trên qwen.
func TestBuildAgentModelsFailsClosedWhenSelectedProviderHasNoKey(t *testing.T) {
	env := testEnv(t)
	global := t.TempDir()
	t.Setenv("CRUSH_GLOBAL_CONFIG", global)
	t.Setenv("CRUSH_GLOBAL_DATA", global)
	t.Setenv("OPENAI_API_KEY", "")
	crushJSON := `{
  "options": {"disable_provider_auto_update": true},
  "providers": {
    "qwen": {"id": "qwen", "name": "Qwen", "type": "openai-compat", "base_url": "http://127.0.0.1:9/qwen/v1", "api_key": "x",
      "models": [{"id": "qwen3-coder", "name": "q", "context_window": 8192, "default_max_tokens": 128}]}},
  "models": {"large": {"provider": "openai", "model": "gpt-4o"}}
}`
	require.NoError(t, os.WriteFile(filepath.Join(env.workingDir, "crush.json"), []byte(crushJSON), 0o644))
	cfg, err := config.Init(env.workingDir, "", false)
	require.NoError(t, err, "loading must succeed so agent-cli login keeps working")
	large := cfg.Config().Models[config.SelectedModelTypeLarge]
	require.Equal(t, "openai", large.Provider, "no silent switch to another provider")
	require.Equal(t, "gpt-4o", large.Model)

	cfg.SetupAgents()
	coord := &coordinator{cfg: cfg, sessions: env.sessions, messages: env.messages, permissions: env.permissions, history: env.history, filetracker: *env.filetracker}
	_, _, err = coord.buildAgentModels(t.Context(), false)
	require.Error(t, err)
	require.ErrorIs(t, err, errLargeModelProviderNotConfigured)
	require.Contains(t, err.Error(), `"openai"`)
	require.Contains(t, err.Error(), `"gpt-4o"`)
	require.Contains(t, err.Error(), "does not switch to another provider")
	require.Contains(t, err.Error(), "agent-cli login")
}

// Provider bị tắt (`Disable`) mà model vẫn còn: người dùng tắt để không gửi code tới đó,
// nên dựng agent phải dừng có lỗi, không gọi nó (soát vòng 6: bản trước vẫn gửi 8 request
// tới provider dựng sẵn đã tắt). Provider dựng sẵn bị tắt vẫn nằm trong Providers với
// Disable=true; đặt cờ trực tiếp để test không phụ thuộc catalog nạp theo tiến trình.
func TestBuildAgentModelsRefusesADisabledProvider(t *testing.T) {
	env := testEnv(t)
	crushJSON := `{
  "options": {"disable_default_providers": true, "disable_provider_auto_update": true},
  "providers": {
    "qwen": {"id": "qwen", "name": "Qwen", "type": "openai-compat", "base_url": "http://127.0.0.1:9/qwen/v1", "api_key": "x",
      "models": [{"id": "qwen3-coder", "name": "q", "context_window": 8192, "default_max_tokens": 128}]}},
  "models": {"large": {"provider": "qwen", "model": "qwen3-coder"}}
}`
	require.NoError(t, os.WriteFile(filepath.Join(env.workingDir, "crush.json"), []byte(crushJSON), 0o644))
	cfg, err := config.Init(env.workingDir, "", false)
	require.NoError(t, err)
	cfg.SetupAgents()
	pc, ok := cfg.Config().Providers.Get("qwen")
	require.True(t, ok)
	pc.Disable = true
	cfg.Config().Providers.Set("qwen", pc)
	coord := &coordinator{cfg: cfg, sessions: env.sessions, messages: env.messages, permissions: env.permissions, history: env.history, filetracker: *env.filetracker}

	_, _, err = coord.buildAgentModels(t.Context(), false)
	require.ErrorIs(t, err, errLargeModelProviderNotConfigured)
	require.Contains(t, err.Error(), `"qwen"`)
	require.True(t, SelectedModelUnavailable(err), "the app still starts so another model can be picked")
}

// Lỗi không tìm thấy model cũng nêu provider/model và vẫn errors.Is được.
func TestNoFallbackErrorKeepsSentinel(t *testing.T) {
	t.Parallel()
	err := noFallbackError(errLargeModelNotFound, config.SelectedModel{Provider: "glm", Model: "glm-9"})
	require.ErrorIs(t, err, errLargeModelNotFound)
	require.Contains(t, err.Error(), `"glm"`)
	require.Contains(t, err.Error(), `"glm-9"`)
	require.Contains(t, err.Error(), "alt+m")
	// `run -m` không gỡ được: lỗi chặn lúc dựng app, trước khi -m được áp (đo bằng binary).
	require.NotContains(t, err.Error(), "-m provider/model")
}

// Chỉ lỗi "model đang chọn không dùng được" mới cho app khởi động không agent; allowlist
// chặn (MODEL_NOT_ALLOWED), chưa chọn model hay lỗi khác vẫn dừng khởi động như cũ.
func TestSelectedModelUnavailableCoversOnlyUnusableSelections(t *testing.T) {
	t.Parallel()
	m := config.SelectedModel{Provider: "openai", Model: "gpt-4o"}
	for _, sentinel := range []error{errLargeModelProviderNotConfigured, errSmallModelProviderNotConfigured, errLargeModelNotFound, errSmallModelNotFound} {
		require.True(t, SelectedModelUnavailable(noFallbackError(sentinel, m)), sentinel.Error())
	}
	require.False(t, SelectedModelUnavailable(errLargeModelNotSelected))
	require.False(t, SelectedModelUnavailable(errors.New("boom")))
	denied, err := execpolicy.Parse("main:qwen3.8")
	require.NoError(t, err)
	require.False(t, SelectedModelUnavailable(denied.CheckModel("openai", "gpt-4o")))
}
