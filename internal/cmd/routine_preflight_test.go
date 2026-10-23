package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/crush/internal/execpolicy"
	"github.com/stretchr/testify/require"
)

// writeProviderConfig ghi crush.json của repo: provider fake (models m1, m2; large m1) với apiKey
// nguyên văn trong JSON (rỗng: không khai api_key).
func writeProviderConfig(t *testing.T, repo, apiKey string) {
	t.Helper()
	key := ""
	if apiKey != "" {
		key = `"api_key":"` + apiKey + `",`
	}
	require.NoError(t, os.WriteFile(filepath.Join(repo, "crush.json"), []byte(`{"providers":{"fake":{"type":"openai-compat","base_url":"http://127.0.0.1:9/v1",`+key+
		`"models":[{"id":"m1","name":"m1","context_window":100000,"default_max_tokens":1000},{"id":"m2","name":"m2","context_window":100000,"default_max_tokens":1000}]}},`+
		`"models":{"large":{"provider":"fake","model":"m1"},"small":{"provider":"fake","model":"m1"}}}`), 0o600))
}

// Kiểm trước `worktree add` theo đúng luật của lượt in-process: model ghim phải có ở một provider đã
// cấu hình; api_key khai trong cấu hình mà phân giải ra rỗng là thiếu key (đo 22/10: `large model ...
// not found`, `No api key passed in`). Provider không khai api_key (model local) vẫn chạy. Lỗi nêu
// thứ thiếu, không bao giờ in giá trị key.
func TestRoutinePreflight(t *testing.T) {
	_, repo := sandboxConfig(t, "m1")
	t.Setenv("DEW_TEST_UNSET_KEY", "")
	wt := filepath.Join(t.TempDir(), "wt", "20261016-090001")
	for _, c := range []struct {
		name, key, model, wantErr string
	}{
		{"pinned model", "sk-secret-123", "fake/m2", ""},
		{"bare pinned model", "sk-secret-123", "m2", ""},
		{"default model", "sk-secret-123", "", ""},
		{"keyless local provider", "", "fake/m1", ""},
		{"unknown model", "sk-secret-123", "fake/nope", `large model "fake/nope" not found`},
		{"unknown provider", "sk-secret-123", "other/m1", `large model "other/m1" not found`},
		{"key resolves to empty", "$DEW_TEST_UNSET_KEY", "fake/m1", `provider "fake": its api_key`},
		{"key resolves to empty, default model", "$DEW_TEST_UNSET_KEY", "", `provider "fake": its api_key`},
	} {
		writeProviderConfig(t, repo, c.key)
		err := routinePreflight(repo, wt, c.model)
		if c.wantErr == "" {
			require.NoError(t, err, c.name)
			continue
		}
		require.ErrorContains(t, err, c.wantErr, c.name)
		require.NotContains(t, err.Error(), "sk-secret-123", c.name)
	}
	require.NoDirExists(t, filepath.Dir(wt), "the preflight creates nothing")
	require.NoDirExists(t, filepath.Join(repo, ".crush"), "nor anything in the repo")
}

// Kiểm trước thất bại: báo cáo ERROR nêu thứ thiếu (lỗi dựng lượt, như khi dựng agent hỏng), không
// worktree, không nhánh, agent không chạy. Kiểm đạt thì lượt chạy như thường.
func TestExecuteRoutine_PreflightFailureCreatesNoWorktreeOrBranch(t *testing.T) {
	t.Cleanup(func() { execpolicy.SetUnattended(false, false) })
	sandboxConfig(t, "m1")
	t.Setenv("DEW_TEST_UNSET_KEY", "")
	for _, c := range []struct{ key, model, want string }{
		{"x", "fake/nope", `large model "fake/nope" not found`},
		{"$DEW_TEST_UNSET_KEY", "fake/m1", `provider "fake": its api_key`},
	} {
		e := newRoutineEnv(t)
		writeProviderConfig(t, e.repo, c.key)
		e.opts.Def.Model = c.model
		d := e.deps(nil)
		d.Preflight = routinePreflight
		out, err := executeRoutine(t.Context(), e.opts, d)
		require.ErrorContains(t, err, c.want)
		require.Zero(t, e.runs, "the agent never starts")
		require.Equal(t, "ERROR", out.Report.Reason)
		rep, rerr := readReport(out.RunDir)
		require.NoError(t, rerr)
		require.Equal(t, "ERROR", rep.Reason)
		require.Contains(t, rep.Detail, c.want)
		require.Empty(t, routineBranches(t, e.repo))
		require.NoDirExists(t, filepath.Join(e.data, "nightly", "wt", filepath.Base(out.RunDir)))
		require.Len(t, splitLines(gitIn(t, e.repo, "worktree", "list")), 1)
	}

	e := newRoutineEnv(t)
	writeProviderConfig(t, e.repo, "x")
	e.opts.Def.Model = "fake/m1"
	d := e.deps(nil)
	d.Preflight = routinePreflight
	out, err := executeRoutine(t.Context(), e.opts, d)
	require.NoError(t, err)
	require.DirExists(t, out.Report.Worktree)
	require.Positive(t, e.runs)
}
