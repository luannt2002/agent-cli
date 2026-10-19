package agent

import (
	"errors"
	"fmt"
)

var (
	ErrRequestCancelled = errors.New("request canceled by user")
	ErrSessionBusy      = errors.New("session is currently processing another request")
	ErrEmptyPrompt      = errors.New("prompt is empty")
	ErrSessionMissing   = errors.New("session id is missing")

	// errLoopDetected: lượt bị dừng vì model lặp lại cùng một tool call (cùng
	// tham số, cùng kết quả) quá nhiều lần trong cửa sổ gần nhất.
	errLoopDetected = fmt.Errorf("stopped: the model repeated the same tool call more than %d times in its last %d steps", loopDetectionMaxRepeats, loopDetectionWindowSize)

	// errEmptyReply: chuỗi của prompt kết thúc mà chưa ra chữ hay tool call nào.
	errEmptyReply = errors.New("the model returned an empty reply (no text and no tool call)")

	// errRefused: bộ lọc nội dung của provider dừng câu trả lời trước khi có gì.
	errRefused = errors.New("the provider's content filter stopped the reply before any text or tool call")

	// errSummaryLimit: một prompt đã tự tóm tắt maxAutoSummaries lần mà việc chưa xong.
	errSummaryLimit = fmt.Errorf("stopped: auto-summarized %d times for one prompt without finishing; send a new prompt to keep going", maxAutoSummaries)
)

// contextOverflowError: sau một lần tự tóm tắt, bước đầu của lượt làm tiếp vẫn
// vượt ngưỡng, nên tóm tắt thêm không giúp được (phần cố định của prompt quá
// lớn so với context window).
type contextOverflowError struct {
	used, window, threshold int64
}

func (e *contextOverflowError) Error() string {
	return fmt.Sprintf("context window too small: right after auto-summarizing, the next request's input alone already uses %d of %d tokens (auto-summarize keeps %d free); raise the model's context_window or shrink the fixed prompt (tools, MCP servers, context files)", e.used, e.window, e.threshold)
}
