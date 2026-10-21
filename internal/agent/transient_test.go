package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"syscall"
	"testing"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/loop"
	"github.com/stretchr/testify/require"
)

// Mỗi lớp lỗi ở cả hai dạng vòng lặp gặp: lỗi có kiểu (chạy in-process, routine)
// và chuỗi chữ (dew loop qua server: RunComplete chỉ mang err.Error()).
func TestTransientError(t *testing.T) {
	t.Parallel()

	idle := &requestTimeoutError{timeout: 2 * time.Minute, idle: true, cause: context.DeadlineExceeded}
	total := &requestTimeoutError{timeout: 2 * time.Minute, cause: context.DeadlineExceeded}
	// Bọc đúng như app.RunNonInteractive bọc lỗi của coordinator.
	local := func(err error) error {
		return fmt.Errorf("agent processing failed: %w", fmt.Errorf("failed to start agent processing stream: %w", err))
	}
	// Dạng chữ mà runStream (cmd/run.go) dựng từ RunComplete.Error.
	remote := func(err error) error { return errors.New("agent run failed: " + err.Error()) }
	pe := func(title string, code int) *fantasy.ProviderError {
		return &fantasy.ProviderError{Title: title, Message: "upstream said no", StatusCode: code}
	}
	retried := func(err error) error { return &fantasy.RetryError{Errors: []error{err, err, err, err}} }
	// Lỗi mạng như net/http trả: *url.Error bọc *net.OpError.
	post := func(err error) error {
		return &url.Error{Op: "Post", URL: "https://gw/v1/chat/completions", Err: err}
	}
	netErr := func(op string, errno syscall.Errno) error {
		return &net.OpError{Op: op, Net: "tcp", Err: os.NewSyscallError(op, errno)}
	}
	// Sự kiện lỗi giữa stream (openai toProviderErrFromStreamError): tiêu đề "stream error".
	midStream := func(transient bool) *fantasy.ProviderError {
		return &fantasy.ProviderError{Title: "stream error", Message: "invalid tool schema", TransientError: transient}
	}
	toolErr := func(err error) error {
		return &fantasy.ToolExecutionError{ToolName: "fetch", ToolCallID: "c1", Err: err}
	}
	usage := &fantasy.ProviderError{Title: "too many requests", StatusCode: 429,
		Message: "Usage limit reached for 5 hour. Your limit will reset at 2026-10-20 12:00:00"}
	quota429 := &fantasy.ProviderError{Title: "too many requests", StatusCode: 429,
		Message: "You exceeded your current quota, please check your plan and billing details."}
	rate429 := &fantasy.ProviderError{Title: "too many requests", StatusCode: 429,
		Message: "Rate limit reached for requests per min. Please try again in 20s."}
	geminiMinute := &fantasy.ProviderError{Title: "too many requests", StatusCode: 429,
		Message: "Quota exceeded for metric: generativelanguage.googleapis.com/generate_content_free_tier_requests, limit: 15, per minute"}
	geminiCurrent := &fantasy.ProviderError{Title: "too many requests", StatusCode: 429,
		Message: "You exceeded your current quota, please check your plan and billing details. Quota exceeded for metric: generate_content_free_tier_requests, limit: 15 per minute"}
	openaiFree := &fantasy.ProviderError{Title: "too many requests", StatusCode: 429,
		Message: "Rate limit reached for gpt-4o-mini in organization org-x on requests per min (RPM): Limit 3, Used 3, Requested 1. Please try again in 20s. You can increase your rate limit by adding a payment method to your account at https://platform.openai.com/account/billing."}
	overflow500 := &fantasy.ProviderError{Title: "internal server error", StatusCode: 500,
		Message: "This model's maximum context length is 8192 tokens. However, you requested 9000 tokens"}

	for name, tc := range map[string]struct {
		err    error
		class  string
		status int
	}{
		"idle stream timeout":                  {local(idle), loop.ClassStreamStall, 0},
		"idle stream timeout over server":      {remote(idle), loop.ClassStreamStall, 0},
		"total request timeout":                {local(total), "timeout", 0},
		"total request timeout over server":    {remote(total), "timeout", 0},
		"524 after fantasy retries":            {local(retried(pe("provider request failed", 524))), "server_error", 524},
		"524 over server with cloudflare body": {remote(retried(&fantasy.ProviderError{Title: "provider request failed", Message: "error code: 524", StatusCode: 524})), "server_error", 524},
		"503":                                  {local(retried(pe("service unavailable", 503))), "server_error", 503},
		"503 over server":                      {remote(retried(pe("service unavailable", 503))), "server_error", 503},
		"429":                                  {local(pe("too many requests", 429)), "rate_limit", 429},
		"429 over server":                      {remote(pe("too many requests", 429)), "rate_limit", 429},
		"mid-stream overloaded event":          {local(midStream(true)), "server_error", 0},
		"connection reset":                     {local(post(netErr("read", syscall.ECONNRESET))), "network", 0},
		"connection reset over server":         {remote(post(netErr("read", syscall.ECONNRESET))), "network", 0},
		"connection refused":                   {local(post(netErr("dial", syscall.ECONNREFUSED))), "network", 0},
		"connection refused over server":       {remote(post(netErr("dial", syscall.ECONNREFUSED))), "network", 0},
		"EOF":                                  {local(post(io.EOF)), "network", 0},
		"EOF over server":                      {remote(post(io.EOF)), "network", 0},
		"stream cut short":                     {local(fantasy.NewIncompleteStreamError()), "network", 0},
		"stream cut short over server":         {remote(fantasy.NewIncompleteStreamError()), "network", 0},

		"401":                     {local(pe("unauthorized", 401)), "", 0},
		"401 over server":         {remote(pe("unauthorized", 401)), "", 0},
		"403":                     {local(pe("forbidden", 403)), "", 0},
		"404 model not found":     {local(pe("not found", 404)), "", 0},
		"404 over server":         {remote(pe("not found", 404)), "", 0},
		"499 non-standard":        {local(pe("provider request failed", 499)), "", 0},
		"auth refresh error":      {local(&fantasy.ProviderError{Title: "auth", Message: "login again", AuthError: true}), "", 0},
		"context window exceeded": {local(&fantasy.ProviderError{Title: "bad request", StatusCode: 400, ContextTooLargeErr: true}), "", 0},
		"context window as 500":   {local(&fantasy.ProviderError{Title: "internal server error", StatusCode: 500, ContextMaxTokens: 8192}), "", 0},
		"model not allowed":       {errors.New("MODEL_NOT_ALLOWED: model qwen is not in DEW_ALLOWED_MODELS"), "", 0},
		"user cancel":             {local(context.Canceled), "", 0},
		"loop run timeout":        {errors.New("agent turn timed out after 45m0s"), "", 0},
		"unknown":                 {errors.New("boom"), "", 0},
		"nil":                     {nil, "", 0},

		// Soát độc lập 21/10: các ca từng bị coi nhầm là tạm thời.
		"mid-stream error, not transient":                      {local(midStream(false)), "", 0},
		"mid-stream error, not transient over server":          {remote(midStream(false)), "", 0},
		"stream decode error":                                  {local(&fantasy.Error{Title: "stream error", Message: "error unmarshalling delta"}), "", 0},
		"tool error wrapping connection refused":               {local(toolErr(fmt.Errorf("failed to fetch URL: %w", post(netErr("dial", syscall.ECONNREFUSED))))), "", 0},
		"tool error wrapping a sub-agent 503":                  {local(toolErr(retried(pe("service unavailable", 503)))), "", 0},
		"tool error over server":                               {remote(toolErr(fmt.Errorf("failed to fetch URL: %w", post(netErr("dial", syscall.ECONNREFUSED))))), "", 0},
		"context overflow as 500 over server":                  {remote(overflow500), "", 0},
		"4xx mentioning gateway timeout over server":           {remote(&fantasy.ProviderError{Title: "bad request", StatusCode: 400, Message: "upstream gateway timeout: retry later"}), "", 0},
		"4xx ending in connection reset over server":           {remote(&fantasy.ProviderError{Title: "bad request", StatusCode: 400, Message: "proxy said: connection reset by peer"}), "", 0},
		"4xx ending in EOF over server":                        {remote(&fantasy.ProviderError{Title: "bad request", StatusCode: 400, Message: "unexpected token: EOF"}), "", 0},
		"non-standard status without 5xx evidence over server": {remote(&fantasy.ProviderError{Title: "provider request failed", StatusCode: 499, Message: "<html>client closed</html>"}), "", 0},
		"local untyped error that reads like a 5xx":            {errors.New("mcp: service unavailable: server busy"), "", 0},

		// F4: hết hạn mức/tiền không phải lỗi tạm thời: không chạy lại, dừng QUOTA.
		"402":                                  {local(pe("payment required", 402)), loop.ClassQuota, 402},
		"402 over server":                      {remote(pe("payment required", 402)), loop.ClassQuota, 402},
		"429 usage limit (z.ai) after retries": {local(retried(usage)), loop.ClassQuota, 429},
		"429 usage limit over server":          {remote(retried(usage)), loop.ClassQuota, 429},
		"429 insufficient_quota (openai)":      {local(quota429), loop.ClassQuota, 429},
		"429 insufficient_quota over server":   {remote(quota429), loop.ClassQuota, 429},
		"402 insufficient balance":             {local(&fantasy.ProviderError{Title: "payment required", StatusCode: 402, Message: "Insufficient Balance"}), loop.ClassQuota, 402},
		"403 insufficient balance":             {local(&fantasy.ProviderError{Title: "forbidden", StatusCode: 403, Message: "Insufficient balance, please recharge"}), loop.ClassQuota, 403},
		"403 insufficient balance over server": {remote(&fantasy.ProviderError{Title: "forbidden", StatusCode: 403, Message: "Insufficient balance, please recharge"}), loop.ClassQuota, 403},
		// Soát độc lập: chữ "quota"/"billing" trần không đủ, nó có cả trong giới hạn theo phút.
		"403 bare quota wording":                  {local(&fantasy.ProviderError{Title: "forbidden", StatusCode: 403, Message: "Quota exceeded for quota metric 'Generate Content API requests'"}), "", 0},
		"403 billing disabled":                    {local(&fantasy.ProviderError{Title: "forbidden", StatusCode: 403, Message: "This API method requires billing to be enabled"}), "", 0},
		"429 gemini per-minute quota":             {local(geminiMinute), "rate_limit", 429},
		"429 gemini per-minute quota over server": {remote(geminiMinute), "rate_limit", 429},
		"429 gemini current quota per minute":     {local(geminiCurrent), "rate_limit", 429},
		"429 gemini current quota over server":    {remote(geminiCurrent), "rate_limit", 429},
		"429 openai free tier links billing":      {local(openaiFree), "rate_limit", 429},
		"429 openai free tier over server":        {remote(openaiFree), "rate_limit", 429},
		"mid-stream usage limit":                  {local(&fantasy.ProviderError{Title: "stream error", Message: "Usage limit reached for 5 hour"}), loop.ClassQuota, 0},
		"mid-stream usage limit over server":      {remote(&fantasy.ProviderError{Title: "stream error", Message: "Usage limit reached for 5 hour"}), loop.ClassQuota, 0},
		"429 rate limit without quota wording":    {local(rate429), "rate_limit", 429},
		"429 rate limit over server":              {remote(rate429), "rate_limit", 429},
		"400 mentioning quota":                    {local(&fantasy.ProviderError{Title: "bad request", StatusCode: 400, Message: "max_tokens is over the quota of this model"}), "", 0},
		"400 mentioning quota over server":        {remote(&fantasy.ProviderError{Title: "bad request", StatusCode: 400, Message: "max_tokens is over the quota of this model"}), "", 0},
		"401 mentioning billing":                  {local(&fantasy.ProviderError{Title: "unauthorized", StatusCode: 401, Message: "invalid key; see billing"}), "", 0},
		"local untyped usage limit text":          {errors.New("Usage limit reached for 5 hour"), "", 0},
		"tool error wrapping a usage limit":       {local(toolErr(retried(usage))), "", 0},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			class, status := TransientError(tc.err)
			require.Equal(t, tc.class, class)
			require.Equal(t, tc.status, status)
		})
	}
}
