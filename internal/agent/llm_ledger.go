package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/execpolicy"
)

// Sổ ghi lời gọi LLM (opt-in, AGENT_CLI_LLM_LEDGER=<tệp>): mỗi lời gọi một dòng
// JSON nói vai trò, provider/model THẬT đã gọi, kích thước prompt và usage của
// chính lời gọi đó. Chỉ có kích thước và hash, không chép nội dung prompt. Dùng để
// chứng minh "phiên chọn provider X không gọi provider khác" và để đo token thật
// (bộ đếm token của session chỉ giữ bước cuối).

type llmRoleKey struct{}

// withLLMRole gắn vai trò cho mọi lời gọi LLM đi qua ctx.
func withLLMRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, llmRoleKey{}, role)
}

func llmRole(ctx context.Context) string {
	if r, ok := ctx.Value(llmRoleKey{}).(string); ok && r != "" {
		return r
	}
	return "main"
}

type ledgerEntry struct {
	Time            string `json:"time"`
	DurationMS      int64  `json:"duration_ms"`
	Role            string `json:"role"`
	SessionID       string `json:"session_id,omitempty"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	Stream          bool   `json:"stream"`
	Messages        int    `json:"messages"`
	SystemBytes     int    `json:"system_bytes"`
	SystemSHA       string `json:"system_sha,omitempty"`
	Tools           int    `json:"tools"`
	ToolNames       string `json:"tool_names,omitempty"`
	InputTokens     int64  `json:"input_tokens"`
	OutputTokens    int64  `json:"output_tokens"`
	CacheReadTokens int64  `json:"cache_read_tokens,omitempty"`
	Finish          string `json:"finish,omitempty"`
	Error           string `json:"error,omitempty"`
}

var (
	ledgerMu    sync.Mutex
	ledgerFiles = map[string]*os.File{}
	// ledgerWriteErrs: session ID → lỗi ghi sổ đầu tiên của prompt gần nhất, gồm
	// các lượt tự làm tiếp của nó (xoá khi session bắt đầu prompt mới), để cảnh
	// báo một lần mỗi prompt và để lượt báo lỗi qua RunComplete / run --json.
	// Mỗi session lỗi giữ một mục tới prompt kế của nó.
	ledgerWriteErrs = map[string]string{}
)

// ledgerPath là tệp sổ ghi đang bật, đọc từ env tin cậy (chụp đầu main) nên
// `.env` của repo không chuyển được sổ ra chỗ khác.
func ledgerPath() string { return execpolicy.TrustedEnv(execpolicy.LLMLedgerEnv) }

// writeLedger nối một dòng vào tệp path (mở một lần mỗi tiến trình, 0600).
func writeLedger(path string, e ledgerEntry) {
	line, err := json.Marshal(e)
	if err != nil {
		return
	}
	ledgerMu.Lock()
	defer ledgerMu.Unlock()
	f, ok := ledgerFiles[path]
	if !ok {
		f, err = os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
		if err != nil {
			slog.Warn("Cannot open the LLM ledger", "path", path, "error", err)
		}
		ledgerFiles[path] = f // nil khi lỗi: chỉ cảnh báo một lần
	}
	if f == nil {
		return
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		// Lời gọi không gắn session (tiêu đề) thì cảnh báo mỗi lần: mỗi session
		// mới chỉ có một lời gọi như vậy.
		if _, seen := ledgerWriteErrs[e.SessionID]; seen && e.SessionID != "" {
			return
		}
		if e.SessionID != "" {
			ledgerWriteErrs[e.SessionID] = err.Error()
		}
		slog.Warn("Failed to write the LLM ledger; calls of this turn are missing from it",
			"path", path, "session_id", e.SessionID, "role", e.Role, "error", err)
	}
}

// resetLedgerTurn quên lỗi ghi sổ của prompt trước khi session bắt đầu prompt mới.
func resetLedgerTurn(sessionID string) {
	ledgerMu.Lock()
	defer ledgerMu.Unlock()
	delete(ledgerWriteErrs, sessionID)
}

// LedgerWriteError trả lỗi ghi sổ đầu tiên trong prompt gần nhất của session
// ("" khi sổ tắt hoặc ghi đủ). Chỉ tính lời gọi gắn chính session đó: lời gọi
// của session con (agent con) và lời gọi tiêu đề chỉ có cảnh báo trong log.
func LedgerWriteError(sessionID string) string {
	ledgerMu.Lock()
	defer ledgerMu.Unlock()
	return ledgerWriteErrs[sessionID]
}

// ledgerModel bọc model để ghi sổ mỗi lời gọi. provider/model là nhãn cấu hình
// của chính model được gọi (sau mọi lần chọn/khoá), không phải model "đang chọn".
type ledgerModel struct {
	fantasy.LanguageModel
	provider, model, path string
}

// newLedgerModel trả m nguyên vẹn khi sổ ghi tắt (path rỗng).
func newLedgerModel(m fantasy.LanguageModel, provider, model, path string) fantasy.LanguageModel {
	if m == nil || path == "" {
		return m
	}
	return ledgerModel{LanguageModel: m, provider: provider, model: model, path: path}
}

func (m ledgerModel) entry(ctx context.Context, call fantasy.Call, stream bool, start time.Time) ledgerEntry {
	var system strings.Builder
	for _, msg := range call.Prompt {
		if msg.Role != fantasy.MessageRoleSystem {
			continue
		}
		for _, part := range msg.Content {
			if tp, ok := fantasy.AsMessagePart[fantasy.TextPart](part); ok {
				system.WriteString(tp.Text)
			}
		}
	}
	e := ledgerEntry{
		Time: start.UTC().Format(time.RFC3339Nano), Role: llmRole(ctx), Provider: m.provider, Model: m.model,
		Stream: stream, Messages: len(call.Prompt), SystemBytes: system.Len(), Tools: len(call.Tools),
	}
	if sid, ok := ctx.Value(tools.SessionIDContextKey).(string); ok {
		e.SessionID = sid
	}
	names := make([]string, 0, len(call.Tools))
	for _, t := range call.Tools {
		names = append(names, t.GetName())
	}
	sort.Strings(names)
	e.ToolNames = strings.Join(names, ",")
	if system.Len() > 0 {
		sum := sha256.Sum256([]byte(system.String()))
		e.SystemSHA = hex.EncodeToString(sum[:6])
	}
	return e
}

func (m ledgerModel) Generate(ctx context.Context, call fantasy.Call) (*fantasy.Response, error) {
	start := time.Now()
	e := m.entry(ctx, call, false, start)
	resp, err := m.LanguageModel.Generate(ctx, call)
	e.DurationMS = time.Since(start).Milliseconds()
	if err != nil {
		e.Error = ledgerErrorKind(err)
	} else if resp != nil {
		e.InputTokens, e.OutputTokens, e.CacheReadTokens = resp.Usage.InputTokens, resp.Usage.OutputTokens, resp.Usage.CacheReadTokens
		e.Finish = string(resp.FinishReason)
	}
	writeLedger(m.path, e)
	return resp, err
}

func (m ledgerModel) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	start := time.Now()
	e := m.entry(ctx, call, true, start)
	inner, err := m.LanguageModel.Stream(ctx, call)
	if err != nil {
		e.DurationMS = time.Since(start).Milliseconds()
		e.Error = ledgerErrorKind(err)
		writeLedger(m.path, e)
		return nil, err
	}
	return func(yield func(fantasy.StreamPart) bool) {
		defer func() {
			e.DurationMS = time.Since(start).Milliseconds()
			writeLedger(m.path, e)
		}()
		inner(func(part fantasy.StreamPart) bool {
			if part.Type == fantasy.StreamPartTypeFinish {
				e.InputTokens, e.OutputTokens, e.CacheReadTokens = part.Usage.InputTokens, part.Usage.OutputTokens, part.Usage.CacheReadTokens
				e.Finish = string(part.FinishReason)
			}
			if part.Error != nil && e.Error == "" {
				e.Error = ledgerErrorKind(part.Error)
			}
			return yield(part)
		})
	}, nil
}

// ledgerErrorKind chỉ ghi loại lỗi (và mã HTTP): server local hay chép mẩu prompt
// vào thông báo lỗi, nên văn bản lỗi không vào sổ.
func ledgerErrorKind(err error) string {
	var pe *fantasy.ProviderError
	switch {
	case errors.As(err, &pe):
		return fmt.Sprintf("provider_error status=%d", pe.StatusCode)
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	}
	return fmt.Sprintf("%T", err)
}
