package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// Verify của `routine run` là tiến trình con của lượt không người trực: nó phải thấy
// DEW_UNATTENDED=1 (và DEW_UNATTENDED_COMMIT theo định nghĩa) như mọi tiến
// trình con khác. Ngoài routine, `.env` của repo không bật được chế độ đó cho verify.
// Chạy ở tiến trình con vì ảnh chụp env gốc là toàn cục, chỉ chụp một lần.
func TestVerifyCarriesTheCurrentUnattendedState(t *testing.T) {
	if os.Getenv("VERIFY_UNATTENDED_CHILD") == "1" {
		t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
		execpolicy.Snapshot()
		require.NoError(t, godotenv.Load())
		wd, err := os.Getwd()
		require.NoError(t, err)
		state := filepath.Join(wd, "state.txt")
		probe := `echo "on=${DEW_UNATTENDED:-unset} commit=${DEW_UNATTENDED_COMMIT:-unset}"`
		res, err := shellVerify(wd, 0)(context.Background(), probe)
		require.NoError(t, err)
		fmt.Print("LOOP " + res.Output)
		for _, commit := range []bool{false, true} {
			e := newRoutineEnv(t)
			e.opts.Def.Commit = commit
			e.opts.Def.Verify = probe + " > " + state + "; test -f done.txt"
			_, err := executeRoutine(t.Context(), e.opts, e.deps(func(wt string) error {
				return os.WriteFile(filepath.Join(wt, "done.txt"), []byte("x"), 0o644)
			}))
			require.NoError(t, err)
			got, err := os.ReadFile(state)
			require.NoError(t, err)
			fmt.Printf("ROUTINE commit=%v %s", commit, got)
		}
		return
	}
	dir := t.TempDir()
	// `.env` của repo cố bật cả hai công tắc.
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("DEW_UNATTENDED=1\nDEW_UNATTENDED_COMMIT=1\n"), 0o600))
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "DEW_") {
			env = append(env, kv)
		}
	}
	c := exec.Command(os.Args[0], "-test.run=^TestVerifyCarriesTheCurrentUnattendedState$")
	c.Dir, c.Env = dir, append(env, "VERIFY_UNATTENDED_CHILD=1")
	out, err := c.CombinedOutput()
	require.NoError(t, err, withoutEnvLines(string(out)))

	var got []string
	for line := range strings.SplitSeq(string(out), "\n") {
		if strings.HasPrefix(line, "LOOP ") || strings.HasPrefix(line, "ROUTINE ") {
			got = append(got, line)
		}
	}
	require.Equal(t, []string{
		"LOOP on=unset commit=unset",
		"ROUTINE commit=false on=1 commit=0",
		"ROUTINE commit=true on=1 commit=1",
	}, got)
}
