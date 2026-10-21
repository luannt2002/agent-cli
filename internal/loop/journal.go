package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event là một dòng journal. Journal là sổ ghi TRƯỚC khi hành động: bị kill giữa
// chừng thì lần sau vẫn biết đã tới đâu và verify cuối ra sao.
type Event struct {
	Time        time.Time `json:"time"`
	Kind        string    `json:"kind"`              // start, baseline, attempt_begin, llm_request, agent_end, verify, stop
	Attempt     int       `json:"attempt,omitempty"` // lượt chạy lại giữ số lượt bị cắt; ở stop: tổng số lượt agent, kể cả lượt chạy lại
	SessionID   string    `json:"session_id,omitempty"`
	Status      string    `json:"status,omitempty"`
	ExitCode    int       `json:"exit_code,omitempty"`
	Output      string    `json:"output,omitempty"` // đuôi output verify
	Fingerprint string    `json:"fingerprint,omitempty"`
	PromptLen   int       `json:"prompt_len,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	Note        string    `json:"note,omitempty"`
	// ErrorClass, HTTPStatus: lượt agent hỏng vì lỗi tạm thời (Engine.Transient), để
	// người đọc thấy vì sao lần chạy thử lại; HTTPStatus 0 khi không biết mã.
	ErrorClass string `json:"error_class,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
	// llm_request: một request LLM streaming của lượt (Time = lúc gửi; Status = "ok", "cancelled"
	// hoặc lớp lỗi của agent.TransientError, "other" khi không tạm thời). TTFTMS vắng khi không
	// phần nào tới. PromptTokens gồm cả phần prompt đọc từ cache: là cỡ context thật đã gửi.
	// Retry là lần chạy lại thứ mấy của dew (0 = lần đầu của lượt). fantasy tự thử lại một bước
	// bằng cách gọi lại Stream, nên mỗi lần thử của nó cũng là một dòng, cùng Retry.
	TTFTMS           *int64 `json:"ttft_ms,omitempty"`
	TotalMS          int64  `json:"total_ms,omitempty"`
	PromptTokens     int64  `json:"prompt_tokens,omitempty"`
	CompletionTokens int64  `json:"completion_tokens,omitempty"`
	Retry            int    `json:"retry,omitempty"`
}

type requestObserverKey struct{}

// WithRequestObserver gắn vào ctx hàm nhận số đo của mỗi request LLM streaming mà lượt agent
// chạy trong ctx gửi đi (agent điền Event, engine thêm Kind/Attempt/Retry). Engine gắn nó cho
// mỗi lượt; hàm có thể bị gọi từ goroutine khác (agent con, tiêu đề).
// shortcut: chỉ đường in-process (routine, `dew loop` không CRUSH_CLIENT_SERVER) mang được ctx tới
// request; qua client/server request chạy trong tiến trình server nên journal không có llm_request.
// Khi cần số đo ở đường đó, server gửi chúng theo sự kiện của lượt.
func WithRequestObserver(ctx context.Context, observe func(Event)) context.Context {
	return context.WithValue(ctx, requestObserverKey{}, observe)
}

// RequestObserver là hàm gắn bằng WithRequestObserver; nil khi ctx không thuộc lượt nào của loop.
func RequestObserver(ctx context.Context) func(Event) {
	observe, _ := ctx.Value(requestObserverKey{}).(func(Event))
	return observe
}

// Journal nhận sự kiện theo thứ tự.
type Journal interface {
	Append(Event) error
}

// FileJournal ghi mỗi sự kiện một dòng JSON và Sync ngay, nên sống sót khi
// tiến trình bị kill -9.
type FileJournal struct {
	mu sync.Mutex
	f  *os.File
}

// OpenFileJournal mở (tạo nếu chưa có) tệp journal ở chế độ nối thêm.
func OpenFileJournal(path string) (*FileJournal, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create journal dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open journal: %w", err)
	}
	return &FileJournal{f: f}, nil
}

// Append implements Journal.
func (j *FileJournal) Append(e Event) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if _, err := j.f.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write journal: %w", err)
	}
	return j.f.Sync()
}

// Close đóng tệp.
func (j *FileJournal) Close() error { return j.f.Close() }

// ReadJournal đọc lại mọi sự kiện của một tệp journal. Dòng cuối hỏng (bị cắt
// giữa chừng khi kill) bị bỏ qua, các dòng trước vẫn đọc được.
func ReadJournal(path string) ([]Event, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []Event
	for _, line := range splitLines(string(data)) {
		var e Event
		if json.Unmarshal([]byte(line), &e) != nil {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
