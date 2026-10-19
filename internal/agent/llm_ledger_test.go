package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/notify"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

func readLedger(t *testing.T, path string) []ledgerEntry {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	var out []ledgerEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e ledgerEntry
		require.NoError(t, json.Unmarshal(sc.Bytes(), &e))
		out = append(out, e)
	}
	return out
}

// Sổ ghi nói đúng model nào đã nhận lời gọi nào, với vai trò của lời gọi, và
// usage của CHÍNH lời gọi đó; không chứa nội dung prompt.
func TestLedgerRecordsEachCallWithRoleProviderAndUsage(t *testing.T) {
	env := testEnv(t)
	broker := pubsub.NewBroker[notify.RunComplete]()
	t.Cleanup(broker.Shutdown)
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	cw := catwalk.Model{ContextWindow: 1000, DefaultMaxTokens: 100}
	large := modelWith(newLedgerModel(&scriptModel{script: []string{"tool_small", "done"}, summaryText: "S"}, "qwen", "qwen3-coder", path), "qwen", "qwen3-coder", cw)
	small := modelWith(newLedgerModel(&finishStreamModel{text: "title"}, "glm", "glm-5.3-flash", path), "glm", "glm-5.3-flash", cw)
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel: large, SmallModel: small, IsYolo: true,
		Sessions: env.sessions, Messages: env.messages, RunComplete: broker,
		Tools: []fantasy.AgentTool{namedTool("noop")},
	}).(*sessionAgent)
	sess, err := env.sessions.Create(t.Context(), "session")
	require.NoError(t, err)
	ch := broker.Subscribe(t.Context())

	_, err = sa.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, RunID: "x", Prompt: "SECRET-PROMPT-TEXT fix it"})
	require.NoError(t, err)
	waitRunComplete(t, ch, "x")
	sa.GenerateTitle(t.Context(), sess.ID, "SECRET-PROMPT-TEXT fix it")

	entries := readLedger(t, path)
	byRole := map[string][]ledgerEntry{}
	for _, e := range entries {
		byRole[e.Role] = append(byRole[e.Role], e)
		require.NotEmpty(t, e.Time)
	}
	require.Len(t, byRole["main"], 2, "two main steps: tool, then done")
	for _, e := range byRole["main"] {
		require.Equal(t, "qwen", e.Provider)
		require.Equal(t, "qwen3-coder", e.Model)
		require.Equal(t, sess.ID, e.SessionID)
		require.True(t, e.Stream)
		require.Equal(t, "noop", e.ToolNames, "the tool palette offered to the model")
	}
	require.EqualValues(t, 10, byRole["main"][0].InputTokens, "usage of that call, not the session total")
	require.NotEmpty(t, byRole["title"])
	require.Equal(t, "glm", byRole["title"][0].Provider, "the ledger shows the cross-provider title call")

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "SECRET-PROMPT-TEXT", "sizes and hashes only, never prompt text")
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

type failingStreamModel struct{ finishStreamModel }

func (*failingStreamModel) Stream(context.Context, fantasy.Call) (fantasy.StreamResponse, error) {
	return nil, &fantasy.ProviderError{StatusCode: 400, Message: "Error rendering prompt: SECRET-PROMPT-TEXT"}
}

// Server OpenAI-compat local (llama.cpp, LM Studio) hay chép mẩu prompt vào thông
// báo lỗi: sổ ghi chỉ giữ loại lỗi và mã HTTP, không giữ văn bản lỗi.
func TestLedgerRecordsTheErrorKindNotItsText(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	m := newLedgerModel(&failingStreamModel{}, "local", "qwen", path)
	_, err := m.Stream(t.Context(), fantasy.Call{})
	require.Error(t, err)

	entries := readLedger(t, path)
	require.Len(t, entries, 1)
	require.Equal(t, "provider_error status=400", entries[0].Error)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "SECRET-PROMPT-TEXT")
}

// Tắt (path rỗng) thì không bọc gì: không chi phí, không tệp.
func TestLedgerOffReturnsTheModelUnchanged(t *testing.T) {
	t.Parallel()
	m := &finishStreamModel{text: "x"}
	require.Same(t, fantasy.LanguageModel(m), newLedgerModel(m, "p", "m", ""))
}
