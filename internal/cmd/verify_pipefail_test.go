package cmd

import (
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/loop"
	"github.com/stretchr/testify/require"
)

// Lệnh verify có ống dẫn: đuôi `| cat`/`| tee log` không được che mã lỗi của lệnh
// đứng trước (audit g3: `false | cat` từng là PASS).
func TestShellVerify_PipelineFailureIsFail(t *testing.T) {
	t.Parallel()
	verify := shellVerify(t.TempDir(), 2*time.Second)
	for _, tc := range []struct {
		cmd    string
		status loop.Status
	}{
		{"false | cat", loop.StatusFail},
		{"exit 3 | tee /dev/null", loop.StatusFail},
		{"echo ok | cat", loop.StatusPass},
	} {
		res, err := verify(t.Context(), tc.cmd)
		require.NoError(t, err)
		require.Equal(t, tc.status, res.Status, tc.cmd)
	}
}

// Với pipefail, `cmd | head -1` có thể đỏ 141 vì cmd chết SIGPIPE dù kiểm đã đạt
// (soát F1). Vẫn là FAIL, nhưng output phải nói lý do để người/agent bỏ đuôi đó.
func TestShellVerify_SigpipeFailureExplainsItself(t *testing.T) {
	t.Parallel()
	verify := shellVerify(t.TempDir(), 5*time.Second)
	res, err := verify(t.Context(), "yes | head -1")
	require.NoError(t, err)
	require.Equal(t, loop.StatusFail, res.Status)
	require.Equal(t, 141, res.ExitCode)
	require.Contains(t, res.Output, "SIGPIPE")

	res, err = verify(t.Context(), "exit 3")
	require.NoError(t, err)
	require.NotContains(t, res.Output, "SIGPIPE", "the hint is only for exit 141")
}
