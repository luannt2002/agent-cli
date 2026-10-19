package config

import (
	"context"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/charmbracelet/crush/internal/csync"
	"github.com/charmbracelet/crush/internal/env"
	"github.com/stretchr/testify/require"
)

// ID "agent-cli-pending" đánh dấu tin giữ chỗ của prompt đã nhận
// (message.PendingReplyProvider): provider của người dùng không được mang nó,
// kẻo mọi lời trả lời của provider đó bị coi là prompt chưa chạy.
func TestConfig_configureProvidersRejectsReservedID(t *testing.T) {
	cfg := &Config{
		Providers: csync.NewMapFrom(map[string]ProviderConfig{
			"agent-cli-pending": {
				BaseURL: "https://api.custom.com/v1",
				Models:  []catwalk.Model{{ID: "test-model"}},
			},
		}),
	}
	cfg.setDefaults("/tmp", "")
	e := env.NewFromMap(map[string]string{})
	err := cfg.configureProviders(context.Background(), testStore(cfg), e, NewShellVariableResolver(e), []catwalk.Provider{})
	require.ErrorContains(t, err, "reserved")
}
