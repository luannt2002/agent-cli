package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/charmbracelet/crush/internal/client"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/csync"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/proto"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/stretchr/testify/require"
)

// sessionModelRun là một lần `run` qua server giả trên session "x" mà lượt
// cuối chạy bằng pp/m-pp, trong khi model của workspace là qq/m-qq.
type sessionModelRun struct {
	useLast    bool   // `run -C` thay cho `run -s x`.
	largeModel string // `-m`.
	effort     string // `--reasoning-effort`.
	lastModel  string // Model lượt cuối của x (mặc định m-pp trên pp).

	after   []proto.Message // Tin sau lời trả lời cuối (prompt khác đã ghi).
	noReply bool            // x chưa có lời trả lời nào, chỉ có after.
}

// runOnSessionX chạy runNonInteractive trên server giả và trả các lần ghi
// model (POST config/model) cùng stderr của run.
func runOnSessionX(t *testing.T, o sessionModelRun) ([]config.SelectedModel, string) {
	t.Helper()
	providers := csync.NewMap[string, config.ProviderConfig]()
	providers.Set("pp", config.ProviderConfig{ID: "pp", Models: []catwalk.Model{{ID: "m-pp"}}})
	providers.Set("qq", config.ProviderConfig{ID: "qq", Models: []catwalk.Model{{ID: "m-qq", ReasoningLevels: []string{"low", "high"}}}})
	cfg := &config.Config{
		Options:   &config.Options{},
		Providers: providers,
		Models: map[config.SelectedModelType]config.SelectedModel{
			config.SelectedModelTypeLarge: {Provider: "qq", Model: "m-qq"},
			config.SelectedModelTypeSmall: {Provider: "qq", Model: "m-qq"},
		},
	}
	last := proto.Message{Role: proto.Assistant, Provider: "pp", Model: "m-pp"}
	if o.lastModel != "" {
		last.Provider, last.Model, _ = strings.Cut(o.lastModel, "/")
	}
	msgs := append([]proto.Message{last}, o.after...)
	if o.noReply {
		msgs = o.after
	}

	var (
		mu     sync.Mutex
		writes []config.SelectedModel
		runID  string
	)
	sent := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/v1/workspaces/ws")
		switch {
		case p == "/config/model" && r.Method == http.MethodPost:
			var req struct {
				Model config.SelectedModel `json:"model"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			mu.Lock()
			writes = append(writes, req.Model)
			mu.Unlock()
		case p == "/config":
			_ = json.NewEncoder(w).Encode(cfg)
		case p == "/agent" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(proto.AgentInfo{IsReady: true})
		case p == "/agent" && r.Method == http.MethodPost:
			var msg proto.AgentMessage
			_ = json.NewDecoder(r.Body).Decode(&msg)
			runID = msg.RunID
			close(sent)
		case p == "/sessions/x":
			_ = json.NewEncoder(w).Encode(proto.Session{ID: "x"})
		case p == "/sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode([]proto.Session{{ID: "x"}})
		case p == "/sessions/x/messages":
			_ = json.NewEncoder(w).Encode(msgs)
		case p == "/events":
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			<-sent
			ev, _ := json.Marshal(pubsub.Event[proto.RunComplete]{
				Type: pubsub.UpdatedEvent, Payload: proto.RunComplete{SessionID: "x", RunID: runID, Text: "done"},
			})
			env, _ := json.Marshal(pubsub.Payload{Type: pubsub.PayloadTypeRunComplete, Payload: ev})
			fmt.Fprintf(w, "data: %s\n\n", env)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		}
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	c, err := client.NewClient("", "tcp", u.Host)
	require.NoError(t, err)

	var stderr bytes.Buffer
	prev := runStderr
	runStderr = &stderr
	t.Cleanup(func() { runStderr = prev })

	continueID := "x"
	if o.useLast {
		continueID = ""
	}
	ws := &proto.Workspace{ID: "ws", Config: cfg}
	err = runNonInteractive(t.Context(), c, ws, io.Discard, nil, "P", o.largeModel, "", o.effort, true, continueID, o.useLast)
	require.NoError(t, err)
	mu.Lock()
	defer mu.Unlock()
	return writes, stderr.String()
}

// Model thuộc workspace và chỉ đổi khi người dùng chọn: tiếp một session
// cũ (`run -s`, `run -C`) không ghi model của session đó vào cấu hình dự án;
// lượt chạy bằng model của workspace và stderr nói rõ cả hai model. `-m` và
// `--reasoning-effort` là lựa chọn tường minh nên vẫn ghi, và effort áp cho model
// của workspace chứ không cho model cũ của session.
func TestContinuedRunKeepsTheWorkspaceModel(t *testing.T) {
	// Not parallel: swaps runStderr.
	const note = "pp/m-pp"
	qqHigh := config.SelectedModel{Provider: "qq", Model: "m-qq", ReasoningEffort: "high"}
	// Tin giữ chỗ của prompt u2 (đang chờ, hoặc đã đóng khi bị huỷ/lỗi) không
	// phải lời trả lời: dòng báo dựa vào lời trả lời thật trước nó.
	pending := []proto.Message{{ID: "u2", Role: proto.User}, {Role: proto.Assistant, Provider: message.PendingReplyProvider, Model: "u2"}}
	closed := []proto.Message{pending[0], pending[1]}
	closed[1].AddFinish(proto.FinishReasonCanceled, "User canceled request", "")
	cases := map[string]struct {
		run        sessionModelRun
		wantWrites []config.SelectedModel
		wantNote   bool
	}{
		"run -s":                    {wantNote: true},
		"run -C":                    {run: sessionModelRun{useLast: true}, wantNote: true},
		"run -s --reasoning-effort": {run: sessionModelRun{effort: "high"}, wantWrites: []config.SelectedModel{qqHigh}, wantNote: true},
		"run -s -m":                 {run: sessionModelRun{largeModel: "pp/m-pp"}, wantWrites: []config.SelectedModel{{Provider: "pp", Model: "m-pp"}}},
		"run -s same model":         {run: sessionModelRun{lastModel: "qq/m-qq"}},

		"run -s, prompt pending after":            {run: sessionModelRun{after: pending}, wantNote: true},
		"run -C, prompt canceled after":           {run: sessionModelRun{useLast: true, after: closed}, wantNote: true},
		"run -s same model, prompt pending after": {run: sessionModelRun{lastModel: "qq/m-qq", after: pending}},
		"run -s, only a pending prompt":           {run: sessionModelRun{noReply: true, after: pending}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			writes, stderr := runOnSessionX(t, tc.run)
			require.Equal(t, tc.wantWrites, writes, "model writes to the project config")
			require.NotContains(t, stderr, message.PendingReplyProvider, "a placeholder is not a model")
			if tc.wantNote {
				require.Contains(t, stderr, note, "stderr names the session's last model")
				require.Contains(t, stderr, "qq/m-qq", "stderr names the model this run uses")
				require.Equal(t, 1, strings.Count(stderr, "\n"), "one line: %q", stderr)
			} else {
				require.NotContains(t, stderr, "last ran on")
			}
		})
	}
}
