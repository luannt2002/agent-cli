package backend

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Dòng log "flag mismatch" từng ghi NGUYÊN env của cả hai client (API key đã
// export) vào crush.log, giữ tới 30 ngày (audit A-09, g4). Chỉ được ghi TÊN biến
// lệch, không bao giờ ghi giá trị.
func TestFirstWinsMismatch_NeverLogsEnvValues(t *testing.T) {
	xdgIsolated(t)
	cwd, dataDir := t.TempDir(), t.TempDir()
	buf := captureDebugLogs(t)
	b := New(context.Background(), nil, func() {})
	b.SetCreateGrace(2 * time.Second)
	t.Cleanup(func() { drainBackend(t, b) })

	argsA := protoWS(cwd, dataDir, uuid.New().String())
	argsA.Env = []string{"OPENAI_API_KEY=sk-secret-AAA", "SAME=1"}
	_, _, err := b.CreateWorkspace(argsA)
	require.NoError(t, err)
	argsB := protoWS(cwd, dataDir, uuid.New().String())
	argsB.Env = []string{"OPENAI_API_KEY=sk-secret-BBB", "SAME=1", "ONLY_B=x"}
	argsB.YOLO = true
	_, _, err = b.CreateWorkspace(argsB)
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, "Workspace flag mismatch on duplicate create")
	require.NotContains(t, out, "sk-secret-AAA")
	require.NotContains(t, out, "sk-secret-BBB")
	require.Contains(t, out, "OPENAI_API_KEY", "the name of a differing variable is useful and safe")
	require.Contains(t, out, "ONLY_B")
	require.NotContains(t, out, "SAME", "identical variables are not listed")
}
