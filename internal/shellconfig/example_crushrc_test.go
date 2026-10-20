package shellconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// runExampleCrushrc chạy chính tệp dew.crushrc.example ở gốc repo với env
// cho sẵn. Mọi biến DEW_* khác bị xoá và tệp env trỏ tới chỗ không tồn
// tại, để test không bao giờ đọc cấu hình (và khoá) thật của người chạy.
func runExampleCrushrc(t *testing.T, env map[string]string) (map[string]any, error) {
	t.Helper()
	script, err := os.ReadFile(filepath.Join("..", "..", "dew.crushrc.example"))
	require.NoError(t, err)

	t.Setenv("DEW_ENV_FILE", filepath.Join(t.TempDir(), "no-env-file"))
	for _, k := range []string{
		"DEW_PROVIDER_ID", "DEW_PROVIDER_NAME", "DEW_PROVIDER_TYPE",
		"DEW_BASE_URL", "DEW_MODEL", "DEW_SMALL_MODEL", "DEW_API_KEY",
		"DEW_CONTEXT_WINDOW", "DEW_MAX_TOKENS", "DEW_CAN_REASON",
	} {
		t.Setenv(k, env[k])
	}

	out, err := LoadShellConfig(t.Context(), filepath.Join(t.TempDir(), "crushrc"), script)
	if err != nil {
		return nil, err
	}
	var cfg map[string]any
	require.NoError(t, json.Unmarshal(out, &cfg))
	return cfg, nil
}

// Provider dựng sẵn (vd zai cho GLM): chỉ ghi khoá, model chọn theo ID
// đó, alt+p liệt kê provider này đầu tiên.
func TestExampleCrushrc_CatalogProvider(t *testing.T) {
	cfg, err := runExampleCrushrc(t, map[string]string{
		"DEW_PROVIDER_ID": "zai",
		"DEW_MODEL":       "glm-5.3",
		"DEW_SMALL_MODEL": "glm-5.3-flash",
		"DEW_API_KEY":     "test-key",
	})
	require.NoError(t, err)

	providers := cfg["providers"].(map[string]any)
	require.NotContains(t, providers, "main")
	zai := providers["zai"].(map[string]any)
	require.Equal(t, "test-key", zai["api_key"])
	require.NotContains(t, zai, "name", "the catalog supplies the name")
	require.NotContains(t, zai, "base_url", "the catalog supplies the endpoint")

	models := cfg["models"].(map[string]any)
	require.Equal(t, "zai", models["large"].(map[string]any)["provider"])
	require.Equal(t, "glm-5.3", models["large"].(map[string]any)["model"])
	require.Equal(t, "glm-5.3-flash", models["small"].(map[string]any)["model"])

	opts := cfg["options"].(map[string]any)
	require.Equal(t, []any{"zai", "openai", "anthropic"}, opts["visible_providers"])
}

// Endpoint tự khai (không đặt DEW_PROVIDER_ID): provider "main" như cũ.
func TestExampleCrushrc_CustomEndpoint(t *testing.T) {
	cfg, err := runExampleCrushrc(t, map[string]string{
		"DEW_BASE_URL":   "http://127.0.0.1:8080/v1",
		"DEW_MODEL":      "local-model",
		"DEW_CAN_REASON": "true",
	})
	require.NoError(t, err)

	main := cfg["providers"].(map[string]any)["main"].(map[string]any)
	require.Equal(t, "http://127.0.0.1:8080/v1", main["base_url"])
	require.Equal(t, "main", cfg["models"].(map[string]any)["large"].(map[string]any)["provider"])
	require.Equal(t, []any{"main", "openai", "anthropic"}, cfg["options"].(map[string]any)["visible_providers"])
}

// Thiếu biến bắt buộc thì crushrc dừng (cả hai cách khai).
func TestExampleCrushrc_MissingRequired(t *testing.T) {
	_, err := runExampleCrushrc(t, map[string]string{})
	require.Error(t, err)
	_, err = runExampleCrushrc(t, map[string]string{"DEW_PROVIDER_ID": "zai", "DEW_MODEL": "glm-5.3"})
	require.Error(t, err, "a catalog provider still needs an API key")
}

// Provider đăng nhập bằng tài khoản (dew login ...): không có
// DEW_API_KEY vẫn nạp được, và crushrc KHÔNG phát ra trường api_key (một
// key rỗng không được phép đè lên token đăng nhập đã lưu).
func TestExampleCrushrc_LoginProviderWithoutKey(t *testing.T) {
	for _, id := range []string{"openai", "xai", "copilot", "hyper"} {
		t.Run(id, func(t *testing.T) {
			cfg, err := runExampleCrushrc(t, map[string]string{
				"DEW_PROVIDER_ID": id,
				"DEW_MODEL":       "some-model",
			})
			require.NoError(t, err)

			provider := cfg["providers"].(map[string]any)[id].(map[string]any)
			require.NotContains(t, provider, "api_key", "an empty key must not be emitted over a stored login token")
			models := cfg["models"].(map[string]any)
			require.Equal(t, id, models["large"].(map[string]any)["provider"])
			require.Equal(t, id, models["small"].(map[string]any)["provider"])
		})
	}
}

// Có key thì vẫn truyền như cũ, kể cả với provider đăng nhập được (openai dùng
// API key thay cho tài khoản ChatGPT).
func TestExampleCrushrc_LoginProviderWithKeyPassesIt(t *testing.T) {
	cfg, err := runExampleCrushrc(t, map[string]string{
		"DEW_PROVIDER_ID": "openai",
		"DEW_MODEL":       "some-model",
		"DEW_API_KEY":     "sk-test",
	})
	require.NoError(t, err)
	require.Equal(t, "sk-test", cfg["providers"].(map[string]any)["openai"].(map[string]any)["api_key"])
}

// Provider dựng sẵn không đăng nhập được thì key vẫn bắt buộc, và lỗi nêu đúng biến.
func TestExampleCrushrc_NonLoginCatalogProviderStillNeedsKey(t *testing.T) {
	for _, id := range []string{"zai", "anthropic", "openrouter"} {
		_, err := runExampleCrushrc(t, map[string]string{
			"DEW_PROVIDER_ID": id,
			"DEW_MODEL":       "some-model",
		})
		require.Error(t, err, id)
		require.Contains(t, err.Error(), "DEW_API_KEY", id)
	}
}

// DEW_CAN_REASON sai giá trị phải làm crushrc dừng với thông báo nêu đúng
// tên biến, không để lộ thông báo của lệnh `model add --can-reason`.
func TestExampleCrushrc_CanReasonInvalid(t *testing.T) {
	for _, bad := range []string{"on", "off", "maybe", "2", "enabled", "y", "t", "f"} {
		_, err := runExampleCrushrc(t, map[string]string{
			"DEW_BASE_URL":   "http://127.0.0.1:8080/v1",
			"DEW_MODEL":      "local-model",
			"DEW_CAN_REASON": bad,
		})
		require.Error(t, err, bad)
		require.Contains(t, err.Error(), "DEW_CAN_REASON", bad)
		require.Contains(t, err.Error(), `"`+bad+`"`, "the message must echo the bad value: %s", bad)
		require.NotContains(t, err.Error(), "--can-reason expects", "the builtin's own message must not surface: %s", bad)
	}
}

// Mọi giá trị crushrc cho qua phải được `model add --can-reason` chấp nhận và
// đi tới model đúng nghĩa (true/1/yes là true, false/0/no là false).
func TestExampleCrushrc_CanReasonValid(t *testing.T) {
	for value, want := range map[string]bool{
		"":      false, // không đặt = mặc định false
		"true":  true,
		"TRUE":  true,
		"True":  true,
		"1":     true,
		"yes":   true,
		"false": false,
		"False": false,
		"0":     false,
		"no":    false,
	} {
		cfg, err := runExampleCrushrc(t, map[string]string{
			"DEW_BASE_URL":   "http://127.0.0.1:8080/v1",
			"DEW_MODEL":      "local-model",
			"DEW_CAN_REASON": value,
		})
		require.NoError(t, err, "value %q", value)
		models := cfg["providers"].(map[string]any)["main"].(map[string]any)["models"].([]any)
		require.Equal(t, want, models[0].(map[string]any)["can_reason"], "value %q", value)
	}
}

// DEW_CAN_REASON không dùng ở chế độ provider dựng sẵn (catalog tự khai),
// nên giá trị lạ ở đó không được làm hỏng khởi động.
func TestExampleCrushrc_CanReasonIgnoredForCatalogProvider(t *testing.T) {
	_, err := runExampleCrushrc(t, map[string]string{
		"DEW_PROVIDER_ID": "zai",
		"DEW_MODEL":       "glm-5.3",
		"DEW_API_KEY":     "test-key",
		"DEW_CAN_REASON":  "on",
	})
	require.NoError(t, err)
}

// visible_providers không có phần tử lặp: provider chính là openai hoặc
// anthropic thì không thêm lần hai.
func TestExampleCrushrc_VisibleProvidersHaveNoDuplicates(t *testing.T) {
	for id, want := range map[string][]any{
		"openai":    {"openai", "anthropic"},
		"anthropic": {"anthropic", "openai"},
		"zai":       {"zai", "openai", "anthropic"},
	} {
		cfg, err := runExampleCrushrc(t, map[string]string{
			"DEW_PROVIDER_ID": id,
			"DEW_MODEL":       "some-model",
			"DEW_API_KEY":     "test-key",
		})
		require.NoError(t, err, id)
		require.Equal(t, want, cfg["options"].(map[string]any)["visible_providers"], id)
	}
}

// Mặc định upstream bảo model thêm `Assisted-by: Crush:<model>` và dòng
// `Generated with Crush` vào commit/PR; mẫu tắt cả hai (cả hai chế độ provider).
func TestExampleCrushrc_AttributionOff(t *testing.T) {
	for name, env := range map[string]map[string]string{
		"catalog": {"DEW_PROVIDER_ID": "zai", "DEW_MODEL": "glm-5.3", "DEW_API_KEY": "test-key"},
		"custom":  {"DEW_BASE_URL": "http://127.0.0.1:8080/v1", "DEW_MODEL": "local-model"},
	} {
		cfg, err := runExampleCrushrc(t, env)
		require.NoError(t, err, name)
		require.Equal(t, map[string]any{
			"trailer_style":  "none",
			"generated_with": false,
		}, cfg["options"].(map[string]any)["attribution"], name)
	}
}
