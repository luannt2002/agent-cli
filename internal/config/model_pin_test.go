package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/stretchr/testify/require"
)

// twoProviderConfig is a config file naming large as the given
// provider/model, with both providers available so either choice resolves.
func twoProviderConfig(provider, model string) string {
	return `{
		"models": {
			"large": {"provider": "` + provider + `", "model": "` + model + `"}
		},
		"providers": {
			"openai": {
				"api_key": "test-key",
				"models": [{"id": "gpt-4", "name": "GPT-4"}]
			},
			"anthropic": {
				"api_key": "test-key-2",
				"models": [{"id": "claude-3", "name": "Claude 3"}]
			}
		}
	}`
}

// TestModelSelectionSurvivesPeerWrite is a regression test for the model
// switching out from under the user when several Crush instances share the
// global config file. A sibling instance selecting a different model must
// not change ours, even though an unrelated config write (a token refresh,
// for example) reloads the file we both write to.
func TestModelSelectionSurvivesPeerWrite(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "crush.json")

	t.Setenv("CRUSH_GLOBAL_CONFIG", dir)
	t.Setenv("CRUSH_GLOBAL_DATA", dir)
	resetProviderState()
	t.Cleanup(resetProviderState)

	require.NoError(t, os.WriteFile(configPath, []byte(twoProviderConfig("openai", "gpt-4")), 0o600))

	store, err := Load(dir, dir, false)
	require.NoError(t, err)
	store.globalDataPath = configPath
	store.CaptureStalenessSnapshot([]string{configPath})

	// The user picks Claude in this instance.
	require.NoError(t, store.UpdatePreferredModel(ScopeGlobal, SelectedModelTypeLarge, SelectedModel{
		Provider: "anthropic",
		Model:    "claude-3",
	}))

	// A sibling instance then picks GPT-4 and writes it to the shared file.
	require.NoError(t, os.WriteFile(configPath, []byte(twoProviderConfig("openai", "gpt-4")), 0o600))

	// Any reload (here explicit; in practice triggered by an unrelated
	// write such as an OAuth token refresh) must leave our choice alone.
	require.NoError(t, store.ReloadFromDisk(context.Background()))

	large := store.Config().Models[SelectedModelTypeLarge]
	require.Equal(t, "anthropic", large.Provider)
	require.Equal(t, "claude-3", large.Model)
}

// TestModelSelectionYieldsToDiskWhenUnchosen verifies the other half of the
// rule: a model type this instance never selected still follows the config
// file, so external edits and `crush login` defaults keep working.
func TestModelSelectionYieldsToDiskWhenUnchosen(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "crush.json")

	t.Setenv("CRUSH_GLOBAL_CONFIG", dir)
	t.Setenv("CRUSH_GLOBAL_DATA", dir)
	resetProviderState()
	t.Cleanup(resetProviderState)

	require.NoError(t, os.WriteFile(configPath, []byte(twoProviderConfig("openai", "gpt-4")), 0o600))

	store, err := Load(dir, dir, false)
	require.NoError(t, err)
	store.globalDataPath = configPath
	store.CaptureStalenessSnapshot([]string{configPath})

	require.NoError(t, os.WriteFile(configPath, []byte(twoProviderConfig("anthropic", "claude-3")), 0o600))
	require.NoError(t, store.ReloadFromDisk(context.Background()))

	large := store.Config().Models[SelectedModelTypeLarge]
	require.Equal(t, "anthropic", large.Provider)
	require.Equal(t, "claude-3", large.Model)
}

// G-01 / PRB-04: model đang chọn nằm ở provider có sẵn bị thiếu khoá lúc khởi động.
// Khởi động vẫn phải thành công (để còn chạy `agent-cli login`), nhưng không đổi
// sang provider khác, kể cả trong RAM; và không ghi gì xuống đĩa: có khoá lại
// thì lần mở sau phải dùng đúng model đã cấu hình. Trước đây fallback được ghi đè
// vào tệp, nên một lần thiếu khoá thành đổi provider vĩnh viễn (và với allowlist
// thì mọi lần mở sau đều bị chặn MODEL_NOT_ALLOWED).
func TestStartupFallbackIsNotPersisted(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "crush.json")
	t.Setenv("CRUSH_GLOBAL_CONFIG", dir)
	t.Setenv("CRUSH_GLOBAL_DATA", dir)
	resetProviderState()
	t.Cleanup(resetProviderState)
	require.NoError(t, os.WriteFile(configPath, []byte(`{
		"models": {"large": {"provider": "openai", "model": "gpt-4"}},
		"providers": {
			"openai": {"api_key": "$S10_OPENAI_KEY", "models": [{"id": "gpt-4", "name": "GPT-4"}]},
			"anthropic": {"api_key": "test-key-2", "models": [{"id": "claude-3", "name": "Claude 3"}]}
		}
	}`), 0o600))

	t.Setenv("S10_OPENAI_KEY", "")
	store, err := Load(dir, dir, false)
	require.NoError(t, err, "startup must not fail without credentials")
	// Không lặng lẽ đổi sang provider khác (fail closed): lựa chọn giữ nguyên để
	// dựng agent báo lỗi nêu tên provider.
	require.Equal(t, "openai", store.Config().Models[SelectedModelTypeLarge].Provider, "no silent switch to another provider")
	require.Equal(t, "gpt-4", store.Config().Models[SelectedModelTypeLarge].Model)

	raw, err := os.ReadFile(configPath)
	require.NoError(t, err)
	var onDisk struct {
		Models map[string]SelectedModel `json:"models"`
	}
	require.NoError(t, json.Unmarshal(raw, &onDisk))
	require.Equal(t, "openai", onDisk.Models["large"].Provider, "the fallback must not be written to disk")
	require.Equal(t, "gpt-4", onDisk.Models["large"].Model)

	t.Setenv("S10_OPENAI_KEY", "test-key")
	resetProviderState()
	store, err = Load(dir, dir, false)
	require.NoError(t, err)
	require.Equal(t, "openai", store.Config().Models[SelectedModelTypeLarge].Provider, "with credentials back the configured model is used again")
	require.Equal(t, "gpt-4", store.Config().Models[SelectedModelTypeLarge].Model)
}

// Provider đang chọn vẫn dùng được, chỉ mất model ID: rơi về model mặc định của
// CHÍNH provider đó (ở đây: model đầu, vì provider tự khai không có catalog), không
// sang provider khác dù provider kia đứng trước theo thứ tự; và không ghi xuống đĩa.
func TestSelectedModelGoneFallsBackWithinProvider(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "crush.json")
	t.Setenv("CRUSH_GLOBAL_CONFIG", dir)
	t.Setenv("CRUSH_GLOBAL_DATA", dir)
	resetProviderState()
	t.Cleanup(resetProviderState)
	require.NoError(t, os.WriteFile(configPath, []byte(`{
		"options": {"disable_default_providers": true, "disable_provider_auto_update": true},
		"models": {
			"large": {"provider": "zeta", "model": "gone"},
			"small": {"provider": "zeta", "model": "gone-too"}
		},
		"providers": {
			"alpha": {"type": "openai-compat", "base_url": "http://127.0.0.1:9/a/v1", "api_key": "k",
				"models": [{"id": "a1", "name": "A1"}]},
			"zeta": {"type": "openai-compat", "base_url": "http://127.0.0.1:9/z/v1", "api_key": "k",
				"models": [{"id": "z1", "name": "Z1"}, {"id": "z2", "name": "Z2"}]}
		}
	}`), 0o600))

	store, err := Load(dir, dir, false)
	require.NoError(t, err)
	large, small := store.Config().Models[SelectedModelTypeLarge], store.Config().Models[SelectedModelTypeSmall]
	require.Equal(t, "zeta", large.Provider)
	require.Equal(t, "z1", large.Model)
	require.Equal(t, "zeta", small.Provider)
	require.Equal(t, "z1", small.Model)

	raw, err := os.ReadFile(configPath)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"model": "gone"`, "the fallback must not be written to disk")
}

// Provider có catalog: mất model ID thì dùng DefaultLarge/DefaultSmallModelID của
// chính provider đó (không phải model đầu, không phải provider đứng trước).
func TestResolveSelectedModels_SameProviderCatalogDefault(t *testing.T) {
	known := []catwalk.Provider{
		{ID: "openai", APIKey: "k", DefaultLargeModelID: "o-large", DefaultSmallModelID: "o-small",
			Models: []catwalk.Model{{ID: "o-large"}, {ID: "o-small"}}},
		{ID: "anthropic", APIKey: "k", DefaultLargeModelID: "a-large", DefaultSmallModelID: "a-small",
			Models: []catwalk.Model{{ID: "a-other"}, {ID: "a-large", DefaultMaxTokens: 700}, {ID: "a-small"}}},
	}
	cfg, err := configureWith(t, map[string]ProviderConfig{}, map[SelectedModelType]SelectedModel{
		SelectedModelTypeLarge: {Provider: "anthropic", Model: "gone"},
		SelectedModelTypeSmall: {Provider: "anthropic", Model: "gone-too"},
	}, known)
	require.NoError(t, err)

	resolved, err := resolveSelectedModels(cfg, known)
	require.NoError(t, err)
	require.True(t, resolved.LargeFallback)
	require.True(t, resolved.SmallFallback)
	require.Equal(t, SelectedModel{Provider: "anthropic", Model: "a-large", MaxTokens: 700}, resolved.Large)
	require.Equal(t, "anthropic", resolved.Small.Provider)
	require.Equal(t, "a-small", resolved.Small.Model)
}

// Provider đang chọn không dùng được (không có trong cấu hình): giữ nguyên lựa
// chọn, không thay bằng model của provider khác; cả large lẫn small.
func TestResolveSelectedModels_NoCrossProviderFallback(t *testing.T) {
	known := []catwalk.Provider{
		{ID: "openai", APIKey: "k", DefaultLargeModelID: "o-large", DefaultSmallModelID: "o-small",
			Models: []catwalk.Model{{ID: "o-large"}, {ID: "o-small"}}},
	}
	cfg, err := configureWith(t, map[string]ProviderConfig{}, map[SelectedModelType]SelectedModel{
		SelectedModelTypeLarge: {Provider: "ghost", Model: "g-large"},
		SelectedModelTypeSmall: {Provider: "ghost", Model: "g-small"},
	}, known)
	require.NoError(t, err)

	resolved, err := resolveSelectedModels(cfg, known)
	require.NoError(t, err)
	require.False(t, resolved.LargeFallback)
	require.False(t, resolved.SmallFallback)
	require.Equal(t, SelectedModel{Provider: "ghost", Model: "g-large"}, resolved.Large)
	require.Equal(t, SelectedModel{Provider: "ghost", Model: "g-small"}, resolved.Small)
}

// Nạp lại cấu hình (TUI đang chạy, tệp đổi) sau khi provider đang chọn mất key: cùng
// luật như lúc khởi động, không đổi sang provider khác. Test do người soát vòng 6 viết
// (đường reload trước đó không có test nào gọi ReloadFromDisk).
func TestReloadDoesNotSwitchProviderWhenTheKeyDisappears(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "crush.json")
	t.Setenv("CRUSH_GLOBAL_CONFIG", dir)
	t.Setenv("CRUSH_GLOBAL_DATA", dir)
	resetProviderState()
	t.Cleanup(resetProviderState)
	require.NoError(t, os.WriteFile(configPath, []byte(`{
		"models": {"large": {"provider": "openai", "model": "gpt-4"}},
		"providers": {
			"openai": {"api_key": "$RELOAD_OPENAI_KEY", "models": [{"id": "gpt-4", "name": "GPT-4"}]},
			"anthropic": {"api_key": "test-key-2", "models": [{"id": "claude-3", "name": "Claude 3"}]}
		}
	}`), 0o600))
	t.Setenv("RELOAD_OPENAI_KEY", "k")
	store, err := Load(dir, dir, false)
	require.NoError(t, err)
	require.Equal(t, "openai", store.Config().Models[SelectedModelTypeLarge].Provider)
	store.globalDataPath = configPath
	store.CaptureStalenessSnapshot([]string{configPath})

	t.Setenv("RELOAD_OPENAI_KEY", "")
	require.NoError(t, store.ReloadFromDisk(t.Context()))
	large := store.Config().Models[SelectedModelTypeLarge]
	require.Equal(t, "openai", large.Provider, "reload must not switch provider")
	require.Equal(t, "gpt-4", large.Model)
	_, ok := store.Config().Providers.Get("openai")
	require.False(t, ok, "precondition: openai is unconfigured after reload")
}
