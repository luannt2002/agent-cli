package agent

import (
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/loop"
)

// TransientError phân loại lỗi kết thúc một lượt agent cho vòng lặp (dew loop,
// routine): class khác rỗng là lỗi tạm thời của provider hoặc đường mạng, chạy
// lại lượt sau một lúc có thể qua; status là mã HTTP khi biết (0 nếu không).
// Tạm thời: 5xx (kể cả 524), 429, stream im quá request_timeout (loop.ClassStreamStall),
// mạng đứt (reset, refused, EOF). Hết hạn mức (loop.ClassQuota: 402, hoặc 429/403/lỗi giữa
// stream mà lời provider là quotaMessage) KHÔNG chạy lại: vòng lặp dừng QUOTA. Không tạm thời: lỗi của tool, xác thực 401/403, tràn context, huỷ, mọi lỗi lạ.
// Có lỗi có kiểu thì chỉ xét các trường của nó; chữ chỉ dùng cho lỗi đi qua server
// (không còn kiểu).
func TransientError(err error) (class string, status int) {
	if err == nil || errors.Is(err, context.Canceled) {
		return "", 0
	}
	// Lỗi của tool (kể cả lỗi provider của agent con bên trong) là lỗi cục bộ:
	// fantasy cũng không bao giờ thử lại nó.
	var toolErr *fantasy.ToolExecutionError
	if errors.As(err, &toolErr) {
		return "", 0
	}
	// Hạn chờ của chính ta xét trước nhánh provider, như runAgent.
	var timeout *requestTimeoutError
	if errors.As(err, &timeout) {
		if timeout.idle {
			return loop.ClassStreamStall, 0
		}
		return "timeout", 0
	}
	var pe *fantasy.ProviderError
	if errors.As(err, &pe) {
		return providerTransient(pe)
	}
	var fe *fantasy.Error
	if errors.As(err, &fe) {
		return "", 0
	}
	if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return "network", 0
	}
	return transientFromText(err.Error())
}

func providerTransient(pe *fantasy.ProviderError) (string, int) {
	switch {
	case pe.IsContextTooLarge():
		return "", 0
	case pe.StatusCode == http.StatusPaymentRequired,
		quotaStatus(pe.StatusCode) && quotaMessage(pe.Message):
		return loop.ClassQuota, pe.StatusCode
	case pe.AuthError:
		return "", 0
	case pe.StatusCode == http.StatusTooManyRequests:
		return "rate_limit", pe.StatusCode
	case pe.StatusCode >= 500:
		return "server_error", pe.StatusCode
	case pe.StatusCode != 0:
		return "", 0
	case pe.TransientError:
		// Sự kiện lỗi giữa stream (overloaded, server_error) trong một phản hồi 200.
		return "server_error", 0
	case pe.Title == transportErrorTitle, errors.Is(pe.Cause, io.ErrUnexpectedEOF):
		// fantasy.WrapTransportError / NewIncompleteStreamError: http2 reset, stream đứt.
		return "network", 0
	}
	return "", 0
}

// quotaText là lời provider dùng khi hết hạn mức hay hết tiền: z.ai "Usage limit reached
// for 5 hour", OpenAI "exceeded your current quota" (insufficient_quota), "Insufficient
// Balance". Chữ "quota"/"billing" trần không đủ: giới hạn theo phút cũng dùng chúng.
var quotaText = []string{"insufficient_quota", "exceeded your current quota", "usage limit", "insufficient balance"}

// rateText là lời của giới hạn theo phút, qua được nếu chờ (Gemini "… limit: 15, per minute"
// kể cả khi mở đầu "You exceeded your current quota"): không phải hết hạn mức.
var rateText = []string{"per min", "rate limit", "rate-limit"}

// quotaMessage: lời của provider nói hết hạn mức hay hết tiền, không phải giới hạn theo phút.
func quotaMessage(msg string) bool {
	m := strings.ToLower(msg)
	return hasAny(m, quotaText) && !hasAny(m, rateText)
}

// quotaStatus: mã mà provider dùng cho hết hạn mức (429, 403), hoặc 0 (lỗi giữa stream).
func quotaStatus(code int) bool {
	return code == http.StatusTooManyRequests || code == http.StatusForbidden || code == 0
}

// quotaTitles là tiêu đề ProviderError (theo mã HTTP) mà lời hết hạn mức có thể theo sau
// trên đường server; "stream error" là lỗi giữa stream (mã 0).
var quotaTitles = map[string]int{
	fantasy.ErrorTitleForStatusCode(http.StatusTooManyRequests): http.StatusTooManyRequests,
	fantasy.ErrorTitleForStatusCode(http.StatusForbidden):       http.StatusForbidden,
	"stream error": 0,
}

func hasAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// transportErrorTitle là tiêu đề fantasy đặt cho lỗi transport nó bọc lại.
const transportErrorTitle = "stream transport error"

// remoteTurnPrefix mở đầu lỗi của một lượt chạy qua server: runStream (cmd/run.go)
// dựng nó từ RunComplete.Error, tức err.Error() của lượt.
const remoteTurnPrefix = "agent run failed: "

// contextOverflowText là các câu provider dùng khi tràn context (fantasy
// parseContextTooLargeError của openai, anthropic, google, kronk; contextOverflowError).
var contextOverflowText = []string{
	"maximum context length", "context window", "context length", "prompt is too long",
	"input too long", "range of input length", "maximum allowed input length", "input token count",
}

// cloudflareErrorBody là thân trả lời của Cloudflare cho client không nhận HTML (524...).
var cloudflareErrorBody = regexp.MustCompile(`^error code: (5\d\d)\b`)

// transientFromText phân loại lỗi đi qua server, chỉ còn chữ. Chỉ nhận dạng
// đúng đầu chuỗi lỗi của lượt (tiêu đề ProviderError, lỗi net/http, hạn chờ),
// không tìm chữ ở giữa thông điệp.
// shortcut: đoán theo chữ của requestTimeoutError, fantasy và net/http; khi
// RunComplete mang lớp lỗi và mã HTTP thì đọc thẳng các trường đó.
func transientFromText(msg string) (string, int) {
	r, ok := strings.CutPrefix(msg, remoteTurnPrefix)
	if !ok {
		return "", 0
	}
	r = strings.TrimPrefix(r, "retry error: ")
	lower := strings.ToLower(r)
	for _, s := range contextOverflowText {
		if strings.Contains(lower, s) {
			return "", 0
		}
	}
	switch {
	case strings.HasPrefix(r, "LLM stream received no data for "):
		return loop.ClassStreamStall, 0
	case strings.HasPrefix(r, "LLM request timed out after "):
		return "timeout", 0
	case strings.HasPrefix(r, transportErrorTitle+": "):
		return "network", 0
	case strings.HasPrefix(r, `Post "`) && (strings.HasSuffix(r, ": EOF") || strings.HasSuffix(r, ": unexpected EOF") ||
		strings.HasSuffix(r, ": connection reset by peer") || strings.HasSuffix(r, ": connection refused")):
		return "network", 0
	}
	// ProviderError in "tiêu đề: thông điệp", tiêu đề là tên mã HTTP viết thường.
	title, rest, _ := strings.Cut(r, ": ")
	// Hết hạn mức: 402, hoặc quotaMessage ngay sau tiêu đề 429/403/lỗi giữa stream (như đường có kiểu).
	if title == fantasy.ErrorTitleForStatusCode(http.StatusPaymentRequired) {
		return loop.ClassQuota, http.StatusPaymentRequired
	}
	if code, ok := quotaTitles[title]; ok && quotaMessage(rest) {
		return loop.ClassQuota, code
	}
	if title == fantasy.ErrorTitleForStatusCode(http.StatusTooManyRequests) {
		return "rate_limit", http.StatusTooManyRequests
	}
	for code := 500; code < 600; code++ {
		if t := fantasy.ErrorTitleForStatusCode(code); t != "" && title == t {
			return "server_error", code
		}
	}
	// Mã ngoài chuẩn thành "provider request failed": chỉ tính khi thân lỗi nói rõ 5xx.
	if title == "provider request failed" {
		if m := cloudflareErrorBody.FindStringSubmatch(rest); m != nil {
			code, _ := strconv.Atoi(m[1])
			return "server_error", code
		}
	}
	return "", 0
}
