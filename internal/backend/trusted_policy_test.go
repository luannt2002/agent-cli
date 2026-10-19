package backend

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// withTrusted trả bản sao env tin cậy với key đổi thành value.
func withTrusted(env []string, key, value string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if k, _, _ := strings.Cut(kv, "="); k == key {
			kv = key + "=" + value
		}
		out = append(out, kv)
	}
	return out
}

// Chính sách tin cậy (execpolicy) nạp một lần mỗi tiến trình server: client
// sau có chính sách khác thì bị từ chối rõ ràng (nêu tên biến, không nêu giá
// trị, nói cách sửa), không lặng lẽ chạy theo chính sách của client đầu. Khác
// ở biến thường thì vẫn gắn vào workspace như cũ.
func TestCreateWorkspace_TrustPolicyMismatch(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*proto.Workspace)
		wantKey string // rỗng: được gắn vào
	}{
		{name: "same policy", mutate: func(*proto.Workspace) {}},
		{name: "plain env differs", mutate: func(p *proto.Workspace) { p.Env = []string{"V3_ROUTE=c2"} }},
		{
			name: "allowlist differs",
			mutate: func(p *proto.Workspace) {
				p.TrustedEnv = withTrusted(p.TrustedEnv, "AGENT_CLI_ALLOWED_MODELS", "qwen:m-qwen")
			},
			wantKey: "AGENT_CLI_ALLOWED_MODELS",
		},
		{
			name:    "delegation differs",
			mutate:  func(p *proto.Workspace) { p.TrustedEnv = withTrusted(p.TrustedEnv, "AGENT_CLI_DELEGATION", "on") },
			wantKey: "AGENT_CLI_DELEGATION",
		},
		{
			name:    "other trusted var differs",
			mutate:  func(p *proto.Workspace) { p.TrustedEnv = withTrusted(p.TrustedEnv, "AGENT_CLI_UNATTENDED", "1") },
			wantKey: "AGENT_CLI_UNATTENDED",
		},
		{
			name:    "yolo differs",
			mutate:  func(p *proto.Workspace) { p.YOLO = true },
			wantKey: "--yolo (this project is open without --yolo: drop --yolo)",
		},
		{
			name:    "older client sends no policy",
			mutate:  func(p *proto.Workspace) { p.TrustedEnv = nil },
			wantKey: "not sent",
		},
		// Lệnh không gửi prompt (jobs, login, logout): không chạy model nên
		// chính sách và --yolo không áp, không bị từ chối oan.
		{name: "no-agent client, yolo differs", mutate: func(p *proto.Workspace) { p.NoAgentRuns, p.YOLO = true, true }},
		{
			// Tên khoá lạ do client gửi không được lặp lại trong lỗi (chỉ đếm).
			name:    "client sends a key the server does not know",
			mutate:  func(p *proto.Workspace) { p.TrustedEnv = append(p.TrustedEnv, "INJECTED_NAME_X=1") },
			wantKey: "1 key(s) this server does not know",
		},
		{
			name: "no-agent client, allowlist differs",
			mutate: func(p *proto.Workspace) {
				p.NoAgentRuns = true
				p.TrustedEnv = withTrusted(p.TrustedEnv, "AGENT_CLI_ALLOWED_MODELS", "qwen:m-qwen")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			xdgIsolated(t)
			cwd, dataDir := t.TempDir(), t.TempDir()
			b := New(context.Background(), nil, func() {})
			b.SetCreateGrace(2 * time.Second)
			t.Cleanup(func() { drainBackend(t, b) })

			argsA := protoWS(cwd, dataDir, uuid.New().String())
			argsA.Env = []string{"V3_ROUTE=c1"}
			wsA, _, err := b.CreateWorkspace(argsA)
			require.NoError(t, err)

			argsB := protoWS(cwd, dataDir, uuid.New().String())
			argsB.Env = argsA.Env
			tc.mutate(&argsB)
			wsB, protoB, err := b.CreateWorkspace(argsB)
			if tc.wantKey == "" {
				require.NoError(t, err)
				require.Equal(t, wsA.ID, protoB.ID)
				return
			}
			require.ErrorIs(t, err, ErrPolicyMismatch)
			require.Nil(t, wsB)
			require.Contains(t, err.Error(), tc.wantKey)
			require.Contains(t, err.Error(), "exit every agent-cli client of this project")
			require.NotContains(t, err.Error(), "qwen:m-qwen", "names only, never values")
			require.NotContains(t, err.Error(), "INJECTED", "never echo key names the client made up")
			wsA.clientsMu.Lock()
			require.Len(t, wsA.clients, 1, "refused client must not be registered")
			wsA.clientsMu.Unlock()
		})
	}
}

// Server đã chốt tắt thì trả ErrServerShuttingDown (client dựng server mới,
// server đó áp chính sách của chính client) trước phép so chính sách.
func TestCreateWorkspace_ClosingServerBeatsPolicyMismatch(t *testing.T) {
	xdgIsolated(t)
	b := New(context.Background(), nil, func() {})
	b.mu.Lock()
	b.closing = true
	b.mu.Unlock()
	args := protoWS(t.TempDir(), t.TempDir(), uuid.New().String())
	args.TrustedEnv = withTrusted(args.TrustedEnv, "AGENT_CLI_ALLOWED_MODELS", "qwen:m-qwen")
	_, _, err := b.CreateWorkspace(args)
	require.ErrorIs(t, err, ErrServerShuttingDown)
}

// Chính sách thuộc cả tiến trình server, không riêng một workspace: client có
// chính sách khác mở dự án KHÁC trên cùng server cũng bị từ chối, không dựng
// workspace nào.
func TestCreateWorkspace_TrustPolicyMismatchOnNewWorkspace(t *testing.T) {
	xdgIsolated(t)
	b := New(context.Background(), nil, func() {})
	b.SetCreateGrace(2 * time.Second)
	t.Cleanup(func() { drainBackend(t, b) })

	_, _, err := b.CreateWorkspace(protoWS(t.TempDir(), t.TempDir(), uuid.New().String()))
	require.NoError(t, err)

	args := protoWS(t.TempDir(), t.TempDir(), uuid.New().String())
	args.TrustedEnv = withTrusted(args.TrustedEnv, "AGENT_CLI_ALLOWED_MODELS", "qwen:m-qwen")
	_, _, err = b.CreateWorkspace(args)
	require.ErrorIs(t, err, ErrPolicyMismatch)
	require.Len(t, b.ListWorkspaces(), 1, "no workspace is built for a refused client")
}

// So --yolo lúc dựng workspace, không so YOLO bật lúc chạy (ctrl+y áp cho cả
// workspace theo thiết kế): bật bằng ctrl+y thì client sau không --yolo (run,
// send, jobs) vẫn gắn vào được.
func TestCreateWorkspace_RuntimeYOLOToggleNotRefused(t *testing.T) {
	xdgIsolated(t)
	cwd, dataDir := t.TempDir(), t.TempDir()
	b := New(context.Background(), nil, func() {})
	b.SetCreateGrace(2 * time.Second)
	t.Cleanup(func() { drainBackend(t, b) })

	wsA, _, err := b.CreateWorkspace(protoWS(cwd, dataDir, uuid.New().String()))
	require.NoError(t, err)
	require.NoError(t, b.SetPermissionsSkip(wsA.ID, true))
	_, protoB, err := b.CreateWorkspace(protoWS(cwd, dataDir, uuid.New().String()))
	require.NoError(t, err)
	require.Equal(t, wsA.ID, protoB.ID)
}

// Hai client cùng dựng một workspace một lúc với --yolo khác nhau: đúng một
// client được giữ, client kia bị từ chối (cả khi nó thua cuộc đua ở nhánh
// kiểm lại sau lúc khởi tạo chậm).
func TestCreateWorkspace_ParallelCreatesRefuseOtherYOLO(t *testing.T) {
	xdgIsolated(t)
	cwd, dataDir := t.TempDir(), t.TempDir()
	b := New(context.Background(), nil, func() {})
	b.SetCreateGrace(2 * time.Second)
	t.Cleanup(func() { drainBackend(t, b) })

	errs := make(chan error, 2)
	start := make(chan struct{})
	for _, yolo := range []bool{false, true} {
		go func() {
			<-start
			args := protoWS(cwd, dataDir, uuid.New().String())
			args.YOLO = yolo
			ws, _, err := b.CreateWorkspace(args)
			if err == nil && ws.Permissions.SkipRequests() != yolo {
				err = errSharedAcrossYOLO
			}
			errs <- err
		}()
	}
	close(start)
	e1, e2 := <-errs, <-errs
	require.NotErrorIs(t, e1, errSharedAcrossYOLO)
	require.NotErrorIs(t, e2, errSharedAcrossYOLO)
	require.True(t, (e1 == nil) != (e2 == nil), "exactly one create wins: %v / %v", e1, e2)
	if e1 != nil {
		require.ErrorIs(t, e1, ErrPolicyMismatch)
	} else {
		require.ErrorIs(t, e2, ErrPolicyMismatch)
	}
}

var errSharedAcrossYOLO = errors.New("workspace shared with a client that asked for another --yolo")
