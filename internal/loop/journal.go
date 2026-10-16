package loop

import (
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
	Kind        string    `json:"kind"` // start, baseline, attempt_begin, agent_end, verify, stop
	Attempt     int       `json:"attempt,omitempty"`
	SessionID   string    `json:"session_id,omitempty"`
	Status      string    `json:"status,omitempty"`
	ExitCode    int       `json:"exit_code,omitempty"`
	Output      string    `json:"output,omitempty"` // đuôi output verify
	Fingerprint string    `json:"fingerprint,omitempty"`
	PromptLen   int       `json:"prompt_len,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	Note        string    `json:"note,omitempty"`
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
