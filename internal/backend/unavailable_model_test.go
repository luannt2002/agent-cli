package backend_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/crush/internal/backend"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Model đang chọn ở provider dựng sẵn mà thiếu key: agent-cli không đổi provider
// (fail closed), nhưng workspace vẫn phải lên để `agent-cli login`, `session`… chạy
// được và TUI mở màn chọn model. Lượt nào cần agent vẫn gặp đúng lỗi nêu provider.
func TestCreateWorkspaceStartsWhenSelectedModelIsUnavailable(t *testing.T) {
	hostHome := t.TempDir()
	t.Setenv("HOME", hostHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(hostHome, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(hostHome, ".local", "share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(hostHome, ".cache"))
	global := t.TempDir()
	t.Setenv("CRUSH_GLOBAL_CONFIG", global)
	t.Setenv("CRUSH_GLOBAL_DATA", global)
	t.Setenv("CRUSH_SKILLS_DIR", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")

	wd := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(wd, "crush.json"), []byte(`{
  "options": {"disable_provider_auto_update": true},
  "providers": {"qwen": {"type": "openai-compat", "base_url": "http://127.0.0.1:9/qwen/v1", "api_key": "x",
    "models": [{"id": "qwen3-coder", "name": "q", "context_window": 8192, "default_max_tokens": 128}]}},
  "models": {"large": {"provider": "openai", "model": "gpt-4o"}}
}`), 0o600))

	srvCfg, err := config.Init(wd, "", false)
	require.NoError(t, err)
	b := backend.New(t.Context(), srvCfg, nil)
	cid := uuid.New().String()
	ws, _, err := b.CreateWorkspace(proto.Workspace{ClientID: cid, Path: wd, DataDir: filepath.Join(wd, ".crush")})
	require.NoError(t, err, "the workspace must come up so login and the model picker still work")
	t.Cleanup(func() { _ = b.DeleteWorkspace(ws.ID, cid) })
	require.Nil(t, ws.AgentCoordinator, "no agent built on another provider")

	err = ws.InitCoderAgentNonInteractive(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), `"openai"`)
	require.Contains(t, err.Error(), "does not switch to another provider")
}
