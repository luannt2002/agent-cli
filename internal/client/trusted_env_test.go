package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/charmbracelet/crush/internal/proto"
	"github.com/stretchr/testify/require"
)

// Lệnh chạy model gửi chính sách tin cậy (đủ khoá) để server so; lệnh không
// prompt thì không đọc chính sách (tệp env sai cú pháp không làm hỏng
// `jobs kill`/`logout`) và không gửi.
func TestCreateWorkspaceSendsTrustedEnvOnlyForAgentRuns(t *testing.T) {
	t.Parallel()
	var got proto.Workspace
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = proto.Workspace{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		_ = json.NewEncoder(w).Encode(proto.Workspace{ID: "w1"})
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	c, err := NewClient(t.TempDir(), "tcp", u.Host)
	require.NoError(t, err)

	_, err = c.CreateWorkspace(t.Context(), proto.Workspace{Path: "/p"})
	require.NoError(t, err)
	require.Len(t, got.TrustedEnv, 11)

	_, err = c.CreateWorkspace(t.Context(), proto.Workspace{Path: "/p", NoAgentRuns: true})
	require.NoError(t, err)
	require.Nil(t, got.TrustedEnv)
	require.True(t, got.NoAgentRuns)
}
