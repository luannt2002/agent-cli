package loop

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// memJournal giữ sự kiện trong bộ nhớ để test đọc lại; onAppend (nếu có) chạy sau mỗi sự kiện.
type memJournal struct {
	events   []Event
	onAppend func(Event)
}

func (j *memJournal) Append(e Event) error {
	j.events = append(j.events, e)
	if j.onAppend != nil {
		j.onAppend(e)
	}
	return nil
}

func (j *memJournal) kinds() []string {
	out := make([]string, len(j.events))
	for i, e := range j.events {
		out[i] = e.Kind
	}
	return out
}

// script dựng engine với agent/verify giả theo kịch bản: verifies[i] là kết quả
// verify lần thứ i (lần 0 là baseline trước mọi lượt agent), fingerprints[i] là
// dấu vân tay cây làm việc sau mỗi lượt agent, errs[i] là lỗi của lượt agent thứ i
// (nil = xong), resumes[i] là session mà Engine bảo lượt thứ i chạy tiếp.
type script struct {
	verifies     []VerifyResult
	fingerprints []string
	prompts      []string
	resumes      []string
	agentErr     error
	errs         []error
	onAgent      func()
	onVerify     func()
	vi, ai       int
	journal      *memJournal
	clock        time.Time
	step         time.Duration
}

func (s *script) engine(spec Spec) *Engine {
	s.journal = &memJournal{}
	s.clock = time.Unix(1_000_000, 0)
	return &Engine{
		Spec: spec,
		Agent: func(ctx context.Context, prompt string) (RunInfo, error) {
			s.prompts = append(s.prompts, prompt)
			s.resumes = append(s.resumes, ResumeSession(ctx))
			// Quy tắc "ghi trước khi làm": sự kiện bắt đầu lượt phải đã nằm trong journal.
			if got := s.journal.kinds(); len(got) == 0 || got[len(got)-1] != "attempt_begin" {
				return RunInfo{}, fmt.Errorf("journal chưa ghi attempt_begin trước khi chạy agent: %v", got)
			}
			s.ai++
			if s.onAgent != nil {
				s.onAgent()
			}
			if s.agentErr != nil {
				return RunInfo{}, s.agentErr
			}
			// Như driver thật: được bảo chạy tiếp session nào thì báo lại đúng session đó.
			id := ResumeSession(ctx)
			if id == "" {
				id = fmt.Sprintf("sess-%d", s.ai)
			}
			if i := s.ai - 1; i < len(s.errs) && s.errs[i] != nil {
				return RunInfo{SessionID: id}, s.errs[i]
			}
			s.clock = s.clock.Add(s.step)
			return RunInfo{SessionID: id, Text: "ok"}, nil
		},
		Verify: func(ctx context.Context, cmd string) (VerifyResult, error) {
			if s.vi >= len(s.verifies) {
				return VerifyResult{}, errors.New("hết kịch bản verify")
			}
			r := s.verifies[s.vi]
			s.vi++
			if s.onVerify != nil {
				s.onVerify()
			}
			return r, nil
		},
		Fingerprint: func(ctx context.Context) (string, error) {
			i := s.ai - 1
			if i < 0 || i >= len(s.fingerprints) {
				return "fp-base", nil
			}
			return s.fingerprints[i], nil
		},
		Journal:   s.journal,
		Now:       func() time.Time { return s.clock },
		Transient: fakeTransient,
		// Giữ đúng trần mặc định (số phần tử), chỉ bỏ thời gian chờ.
		retryWaits: make([]time.Duration, len(transientWaits)),
	}
}

func pass() VerifyResult { return VerifyResult{Status: StatusPass} }
func fail(out string) VerifyResult {
	return VerifyResult{Status: StatusFail, ExitCode: 1, Output: out}
}

func baseSpec() Spec {
	return Spec{Name: "t", Goal: "make the tests pass", Verify: "go test ./...", MaxRepairs: 2, StreamRetries: 2}
}

func TestEngine_GreenOnFirstAttempt(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("red"), pass()}, fingerprints: []string{"fp1"}}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopGreen, res.Reason)
	require.Equal(t, 1, s.ai)
	require.Contains(t, s.prompts[0], "make the tests pass")
	require.Contains(t, s.prompts[0], "go test ./...", "đề bài phải nói rõ 'xong' là lệnh nào")
	require.Contains(t, s.prompts[0], "Do not modify", "cấm sửa test/cổng để qua")
}

func TestEngine_RepairsWithVerifyOutputThenGreen(t *testing.T) {
	t.Parallel()
	s := &script{
		verifies:     []VerifyResult{fail("base"), fail("FAIL TestFoo\nexpected 3 got 4"), pass()},
		fingerprints: []string{"fp1", "fp2"},
	}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopGreen, res.Reason)
	require.Equal(t, 2, s.ai)
	require.Contains(t, s.prompts[1], "FAIL TestFoo", "lượt sửa nhận đuôi output của verify")
	require.Contains(t, s.prompts[1], "exit code 1")
	require.Contains(t, s.prompts[1], "Do not modify")
}

func TestEngine_StopsAfterMaxRepairsExactly(t *testing.T) {
	t.Parallel()
	// Mỗi lần lỗi khác nhau và cây đổi: không dính STALLED, chỉ dính trần số lần sửa.
	s := &script{
		verifies:     []VerifyResult{fail("b"), fail("err A"), fail("err B"), fail("err C"), fail("err D")},
		fingerprints: []string{"fp1", "fp2", "fp3", "fp4"},
	}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopMaxRepairs, res.Reason)
	require.Equal(t, 3, s.ai, "1 lượt đầu + MaxRepairs=2 lượt sửa, không có lần vá thứ ba")
}

func TestEngine_MaxRepairsZeroRunsOnce(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("b"), fail("still red")}, fingerprints: []string{"fp1"}}
	spec := baseSpec()
	spec.MaxRepairs = 0
	res, err := s.engine(spec).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopMaxRepairs, res.Reason)
	require.Equal(t, 1, s.ai)
}

func TestEngine_StalledWhenSameFailureAndTreeUnchanged(t *testing.T) {
	t.Parallel()
	// Cùng chữ ký lỗi (chỉ khác số đếm/thời gian) và cùng dấu vân tay cây: không tiến triển.
	s := &script{
		verifies:     []VerifyResult{fail("b"), fail("FAIL TestFoo 12ms"), fail("FAIL TestFoo 15ms")},
		fingerprints: []string{"same", "same"},
	}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopStalled, res.Reason)
	require.Equal(t, 2, s.ai, "dừng ngay lần thứ hai, không đốt nốt lượt sửa cuối")
}

func TestEngine_SameFailureButTreeChangedIsNotStalled(t *testing.T) {
	t.Parallel()
	// Vế ngược của test trên: cùng lỗi nhưng cây đã đổi thì vẫn được sửa tiếp.
	s := &script{
		verifies:     []VerifyResult{fail("b"), fail("FAIL TestFoo"), fail("FAIL TestFoo"), pass()},
		fingerprints: []string{"fp1", "fp2", "fp3"},
	}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopGreen, res.Reason)
	require.Equal(t, 3, s.ai)
}

func TestEngine_UnmeasuredVerifyStopsWithoutRepair(t *testing.T) {
	t.Parallel()
	// Không chạy được verify là CHƯA ĐO: không được tính xanh, cũng không đốt lượt sửa.
	s := &script{verifies: []VerifyResult{fail("b"), {Status: StatusUnmeasured, ExitCode: 127, Output: "command not found"}}, fingerprints: []string{"fp1"}}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopUnmeasured, res.Reason)
	require.Equal(t, 1, s.ai)
}

func TestEngine_BaselineGreenIsFlaggedButRunContinues(t *testing.T) {
	t.Parallel()
	// Verify đã xanh trước mọi thay đổi: cổng có thể không đo đúng mục tiêu. Ghi
	// cảnh báo vào journal, vẫn chạy lượt agent, rồi xanh thì GREEN.
	s := &script{verifies: []VerifyResult{pass(), pass()}, fingerprints: []string{"fp1"}}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopGreen, res.Reason)
	require.Equal(t, 1, s.ai)
	var warned bool
	for _, e := range s.journal.events {
		if e.Kind == "baseline" && strings.Contains(e.Note, "already passes") {
			warned = true
		}
	}
	require.True(t, warned, "baseline xanh phải để lại cảnh báo: %v", s.journal.events)
}

func TestEngine_TimeoutBeforeNextAgentTurn(t *testing.T) {
	t.Parallel()
	s := &script{
		verifies:     []VerifyResult{fail("b"), fail("err A"), fail("err B")},
		fingerprints: []string{"fp1", "fp2"},
		step:         40 * time.Minute,
	}
	spec := baseSpec()
	spec.MaxDuration = time.Hour
	res, err := s.engine(spec).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopTimeout, res.Reason)
	require.Equal(t, 2, s.ai, "lượt thứ ba không được phóng khi đã quá hạn")
}

func TestEngine_CancelledContext(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("b")}}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	res, err := s.engine(baseSpec()).Run(ctx)
	require.NoError(t, err)
	require.Equal(t, StopCancelled, res.Reason)
	require.Zero(t, s.ai)
}

// Huỷ giữa chừng (Ctrl+C, SIGTERM): lượt đang chạy xong rồi dừng, KHÔNG phóng lượt kế.
func TestEngine_CancelDuringRunStopsBeforeNextTurn(t *testing.T) {
	t.Parallel()
	s := &script{
		verifies:     []VerifyResult{fail("b"), fail("err A"), pass()},
		fingerprints: []string{"fp1", "fp2"},
	}
	ctx, cancel := context.WithCancel(t.Context())
	s.onAgent = cancel
	res, err := s.engine(baseSpec()).Run(ctx)
	require.NoError(t, err)
	require.Equal(t, StopCancelled, res.Reason)
	require.Equal(t, 1, s.ai)
}

// Huỷ rơi đúng lúc lệnh verify đang chạy: CANCELLED, không phải UNMEASURED.
func TestEngine_CancelDuringVerifyIsCancelledNotUnmeasured(t *testing.T) {
	t.Parallel()
	killed := VerifyResult{Status: StatusUnmeasured, Output: "killed"}
	for name, tc := range map[string]struct {
		verifies []VerifyResult
		atCall   int
	}{
		"baseline":         {[]VerifyResult{killed}, 1},
		"after an attempt": {[]VerifyResult{fail("b"), killed}, 2},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := &script{verifies: tc.verifies, fingerprints: []string{"fp1"}}
			ctx, cancel := context.WithCancel(t.Context())
			calls := 0
			s.onVerify = func() {
				if calls++; calls == tc.atCall {
					cancel()
				}
			}
			res, err := s.engine(baseSpec()).Run(ctx)
			require.NoError(t, err)
			require.Equal(t, StopCancelled, res.Reason)
		})
	}
}

func TestEngine_AgentErrorStopsWithRunError(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("b")}, agentErr: errors.New("provider down")}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopRunError, res.Reason)
	require.Contains(t, res.Detail, "provider down")
}

// tempErr là lỗi tạm thời giả cho bộ phân loại giả fakeTransient; bộ phân loại
// thật (agent.TransientError) có test bảng riêng ở internal/agent.
type tempErr struct {
	class  string
	status int
}

func (e tempErr) Error() string { return fmt.Sprintf("temporary %s (HTTP %d)", e.class, e.status) }

func fakeTransient(err error) (string, int) {
	var t tempErr
	if errors.As(err, &t) {
		return t.class, t.status
	}
	return "", 0
}

// transientEvents là các sự kiện journal ghi một lỗi tạm thời.
func transientEvents(j *memJournal) []Event {
	var out []Event
	for _, e := range j.events {
		if e.ErrorClass != "" {
			out = append(out, e)
		}
	}
	return out
}

// 524 sau khi fantasy đã tự thử 3 lần (ca 20/10) không giết cả lần chạy: lượt đó
// chạy lại trong cùng session, prompt mở đầu bằng lời dặn kiểm trạng thái, journal
// ghi lớp lỗi và mã HTTP, và không tốn lượt sửa (MaxRepairs=1 vẫn còn lượt sửa).
func TestEngine_TransientErrorRetriesTheSameSessionThenGreen(t *testing.T) {
	t.Parallel()
	s := &script{
		verifies:     []VerifyResult{fail("b"), fail("err A"), pass()},
		fingerprints: []string{"fp1", "fp2", "fp3"},
		errs:         []error{fmt.Errorf("agent processing failed: %w", tempErr{"server_error", 524})},
	}
	spec := baseSpec()
	spec.MaxRepairs = 1
	res, err := s.engine(spec).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopGreen, res.Reason)
	require.Equal(t, 3, s.ai, "lượt đầu, lượt chạy lại, rồi vẫn còn đúng 1 lượt sửa")
	require.Equal(t, 3, res.Attempts, "Attempts đếm mọi lượt agent đã chạy, kể cả lượt chạy lại")
	require.Equal(t, []string{"", "sess-1", ""}, s.resumes, "lượt chạy lại tiếp đúng session bị cắt; lượt sửa vẫn mở session mới")
	require.Equal(t, []string{"sess-1", "sess-3"}, res.SessionIDs)
	require.True(t, strings.HasPrefix(s.prompts[1], interruptedNote+"\n\n"), s.prompts[1])
	require.True(t, strings.HasSuffix(s.prompts[1], s.prompts[0]), "phần sau là đúng đề bài của lượt bị cắt")
	require.Contains(t, s.prompts[2], "Repair 1 of 1.")
	ev := transientEvents(s.journal)
	require.Len(t, ev, 1)
	require.Equal(t, "agent_end", ev[0].Kind)
	require.Equal(t, 1, ev[0].Attempt)
	require.Equal(t, "server_error", ev[0].ErrorClass)
	require.Equal(t, 524, ev[0].HTTPStatus)
}

// Lỗi tạm thời liên tiếp có trần (mặc định 2 lần chạy lại): lần thứ ba dừng RUN_ERROR.
func TestEngine_TransientErrorStopsWithRunErrorAfterTheCap(t *testing.T) {
	t.Parallel()
	s := &script{
		verifies: []VerifyResult{fail("b")},
		errs: []error{
			tempErr{"server_error", 524},
			tempErr{"timeout", 0},
			tempErr{"rate_limit", 429},
		},
	}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopRunError, res.Reason)
	require.Equal(t, 3, s.ai, "1 lượt + 2 lần chạy lại, không có lần thứ ba")
	require.Contains(t, res.Detail, "after 2 retries")
	require.Contains(t, res.Detail, "temporary rate_limit (HTTP 429)")
	ev := transientEvents(s.journal)
	require.Len(t, ev, 3)
	require.Equal(t, []string{"server_error", "timeout", "rate_limit"}, []string{ev[0].ErrorClass, ev[1].ErrorClass, ev[2].ErrorClass})
	require.Equal(t, 429, ev[2].HTTPStatus)
}

// Lỗi mà bộ phân loại không coi là tạm thời (vd 401) dừng ngay như trước: không
// chạy lại, không ghi lớp lỗi, chi tiết y nguyên.
func TestEngine_NonTransientErrorStopsImmediately(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("b")}, errs: []error{errors.New("unauthorized: upstream said no")}}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopRunError, res.Reason)
	require.Equal(t, 1, s.ai)
	require.Equal(t, "unauthorized: upstream said no", res.Detail, "chi tiết y như trước khi có chạy lại")
	require.Empty(t, transientEvents(s.journal))
}

// Ctrl+C rơi vào lúc đang chờ để chạy lại: dừng CANCELLED ngay, không ngủ hết giờ chờ.
func TestEngine_CancelDuringRetryWaitIsCancelledPromptly(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("b")}, errs: []error{tempErr{"server_error", 503}}}
	e := s.engine(baseSpec())
	e.retryWaits = []time.Duration{time.Hour, time.Hour}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s.journal.onAppend = func(ev Event) {
		if ev.ErrorClass != "" {
			time.AfterFunc(20*time.Millisecond, cancel)
		}
	}
	start := time.Now()
	res, err := e.Run(ctx)
	require.NoError(t, err)
	require.Equal(t, StopCancelled, res.Reason)
	require.Contains(t, res.Detail, "waiting to retry")
	require.Equal(t, 1, s.ai)
	require.Less(t, time.Since(start), 10*time.Second)
}

func TestEngine_JournalOrderAndStopEvent(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("b"), fail("err A"), pass()}, fingerprints: []string{"fp1", "fp2"}}
	_, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{
		"start", "baseline",
		"attempt_begin", "agent_end", "verify",
		"attempt_begin", "agent_end", "verify",
		"stop",
	}, s.journal.kinds())
	last := s.journal.events[len(s.journal.events)-1]
	require.Equal(t, string(StopGreen), last.Reason)
}

func TestSignature_IgnoresNumbersTimesAndAddresses(t *testing.T) {
	t.Parallel()
	a := signature(1, "FAIL TestFoo (12ms)\n  at 0xdeadbeef line 42\nexpected 3 got 4")
	b := signature(1, "FAIL TestFoo (907ms)\n  at 0xcafef00d line 99\nexpected 7 got 8")
	require.Equal(t, a, b)
	require.NotEqual(t, a, signature(1, "FAIL TestBar (12ms)"), "tên test khác là lỗi khác")
	require.NotEqual(t, a, signature(2, "FAIL TestFoo (12ms)\n  at 0xdeadbeef line 42\nexpected 3 got 4"), "mã thoát khác là lỗi khác")
}

func TestTail_KeepsLastLines(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	for i := range 100 {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	got := tail(b.String(), 5)
	require.Equal(t, "line 95\nline 96\nline 97\nline 98\nline 99", got)
	require.Equal(t, "x", tail("x", 5))
}

// Verify chạy ở chính working dir của agent (VerifyDir rỗng) thì prompt giữ
// nguyên từng byte như trước khi có VerifyDir; khác thì nêu thư mục.
func TestPromptsNameTheVerifyDirOnlyWhenItDiffers(t *testing.T) {
	t.Parallel()

	s := Spec{Goal: " g ", Verify: "go test ./...", MaxRepairs: 2}
	v := VerifyResult{ExitCode: 1, Output: "FAIL"}
	require.Equal(t, "g\n\nDefinition of done: running `go test ./...` exits with code 0.\n"+noEditRule, goalPrompt(s))
	require.Equal(t, "g\n\nThe previous attempt did not pass verification.\nVerify command: `go test ./...`\nResult: exit code 1\n"+
		"Output (last 60 lines):\nFAIL\n\nFind and fix the cause in the code. "+noEditRule+"\nRepair 1 of 2.", repairPrompt(s, v, 1))

	s.VerifyDir = "sub/"
	require.Contains(t, goalPrompt(s), "running `go test ./...` in `sub/` exits with code 0.")
	require.Contains(t, repairPrompt(s, v, 1), "Verify command: `go test ./...` (run in `sub/`)\nResult: exit code 1")
}

// Trần là số lỗi tạm thời LIÊN TIẾP: một lượt chạy xong thì đếm lại từ đầu, nên
// lượt sửa sau đó vẫn được chạy lại đủ 2 lần.
func TestEngine_TransientRetryCounterResetsAfterASuccessfulTurn(t *testing.T) {
	t.Parallel()
	temp := tempErr{"server_error", 503}
	s := &script{
		verifies:     []VerifyResult{fail("b"), fail("err A"), pass()},
		fingerprints: []string{"fp1", "fp2", "fp3", "fp4", "fp5", "fp6"},
		errs:         []error{temp, temp, nil, temp, temp, nil},
	}
	spec := baseSpec()
	spec.MaxRepairs = 1
	res, err := s.engine(spec).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopGreen, res.Reason, res.Detail)
	require.Equal(t, 6, s.ai)
	require.Len(t, transientEvents(s.journal), 4)
}

// Hạn max_duration đã qua (hoặc sẽ qua trước lần chạy lại) thì dừng TIMEOUT ngay,
// không ngủ hết thời gian chờ rồi mới dừng.
func TestEngine_DeadlineIsCheckedBeforeTheRetryWait(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("b")}, errs: []error{tempErr{"server_error", 503}}}
	spec := baseSpec()
	spec.MaxDuration = time.Hour
	e := s.engine(spec)
	e.retryWaits = []time.Duration{2 * time.Second, 2 * time.Second}
	s.onAgent = func() { s.clock = s.clock.Add(59*time.Minute + 59*time.Second) } // còn 1s, chờ 2s
	start := time.Now()
	res, err := e.Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopTimeout, res.Reason)
	require.Equal(t, 1, s.ai)
	require.Less(t, time.Since(start), time.Second, "không được ngủ thời gian chờ khi lần chạy lại chắc chắn quá hạn")
}

// verifyEvents là các sự kiện verify trong journal (baseline không tính).
func verifyEvents(j *memJournal) []Event {
	var out []Event
	for _, e := range j.events {
		if e.Kind == "verify" {
			out = append(out, e)
		}
	}
	return out
}

// F2: lượt bị chính hạn chờ mỗi lượt cắt (run_timeout) dừng TIMEOUT, nhưng vẫn đo verify
// MỘT lần trên cây agent bỏ lại; verify xanh cũng KHÔNG thành GREEN.
func TestEngine_TurnTimeoutVerifiesOnceAndIsNeverGreen(t *testing.T) {
	t.Parallel()
	s := &script{
		verifies: []VerifyResult{fail("b"), pass()},
		errs:     []error{fmt.Errorf("%w after 45m0s", ErrTurnTimeout)},
	}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopTimeout, res.Reason, res.Detail)
	require.Equal(t, 1, s.ai, "bị cắt vì hết hạn lượt thì không chạy lại")
	require.Equal(t, 2, s.vi, "baseline + đúng một lần verify sau khi bị cắt")
	require.Equal(t, StatusPass, res.Last.Status)
	require.Contains(t, res.Detail, "agent turn timed out after 45m0s")
	require.Contains(t, res.Detail, "verify PASS after the agent was cut off — review before trusting")
	require.Len(t, verifyEvents(s.journal), 1)
}

// Verify đỏ sau khi bị cắt: vẫn TIMEOUT, chi tiết nêu mã thoát thật.
func TestEngine_TurnTimeoutWithFailingVerify(t *testing.T) {
	t.Parallel()
	s := &script{
		verifies: []VerifyResult{fail("b"), {Status: StatusFail, ExitCode: 3, Output: "x"}},
		errs:     []error{fmt.Errorf("%w after 1s", ErrTurnTimeout)},
	}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopTimeout, res.Reason)
	require.Equal(t, 3, res.Last.ExitCode)
	require.Contains(t, res.Detail, "verify FAIL (exit 3) after the agent was cut off")
}

// Stream im (lớp stream_stall) hết lần chạy lại: STREAM_STALL, verify chạy một lần.
func TestEngine_StreamStallAfterTheRetriesVerifiesOnce(t *testing.T) {
	t.Parallel()
	stall := tempErr{ClassStreamStall, 0}
	s := &script{verifies: []VerifyResult{fail("b"), fail("c")}, errs: []error{stall, stall, stall}}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopStreamStall, res.Reason, res.Detail)
	require.Equal(t, 3, s.ai, "1 lượt + 2 lần chạy lại")
	require.Equal(t, 2, s.vi, "baseline + một lần verify sau khi bị cắt")
	require.Equal(t, StatusFail, res.Last.Status)
	require.Contains(t, res.Detail, "after 2 retries")
	require.Contains(t, res.Detail, "verify FAIL (exit 1) after the agent was cut off")
}

// 5xx hết lần chạy lại vẫn là RUN_ERROR (không phải STREAM_STALL), nêu mã HTTP, và cũng
// đo verify một lần; xanh vẫn không thành GREEN.
func TestEngine_ServerErrorAfterTheRetriesIsRunErrorAndVerifiesOnce(t *testing.T) {
	t.Parallel()
	e503 := tempErr{"server_error", 503}
	s := &script{verifies: []VerifyResult{fail("b"), pass()}, errs: []error{tempErr{ClassStreamStall, 0}, e503, e503}}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopRunError, res.Reason, "lỗi làm cạn trần quyết định lý do")
	require.Equal(t, 2, s.vi)
	require.Contains(t, res.Detail, "server_error error (HTTP 503) persists after 2 retries")
	require.Contains(t, res.Detail, "review before trusting")
}

// F1: stream_retries là trần cấu hình được; 0 = không chạy lại.
func TestEngine_StreamRetriesIsTheCap(t *testing.T) {
	t.Parallel()
	stall := tempErr{ClassStreamStall, 0}
	for _, n := range []int{0, 1, 4} {
		errs := make([]error, n+1)
		for i := range errs {
			errs[i] = stall
		}
		s := &script{verifies: []VerifyResult{fail("b"), fail("c")}, errs: errs}
		spec := baseSpec()
		spec.StreamRetries = n
		res, err := s.engine(spec).Run(t.Context())
		require.NoError(t, err)
		require.Equal(t, StopStreamStall, res.Reason, "n=%d", n)
		require.Equal(t, n+1, s.ai, "n=%d: 1 lượt + n lần chạy lại", n)
		require.Len(t, transientEvents(s.journal), n+1)
	}
}

// Chờ trước lần chạy lại thứ i: 30s, 60s, rồi giữ 60s cho các lần sau.
func TestRetryWait(t *testing.T) {
	t.Parallel()
	require.Equal(t, []time.Duration{30 * time.Second, time.Minute, time.Minute, time.Minute, time.Minute},
		[]time.Duration{retryWait(nil, 0), retryWait(nil, 1), retryWait(nil, 2), retryWait(nil, 3), retryWait(nil, 4)})
}

// F4: hết hạn mức (lớp quota) KHÔNG chạy lại, dừng QUOTA ngay với lời của provider.
func TestEngine_QuotaStopsImmediatelyWithoutRetry(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("b")}, errs: []error{fmt.Errorf("too many requests: Usage limit reached for 5 hour: %w", tempErr{ClassQuota, 429})}}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopQuota, res.Reason)
	require.Equal(t, 1, s.ai, "không chạy lại")
	require.Equal(t, 1, s.vi, "không đo lại verify")
	require.Contains(t, res.Detail, "Usage limit reached for 5 hour")
	ev := transientEvents(s.journal)
	require.Len(t, ev, 1)
	require.Equal(t, ClassQuota, ev[0].ErrorClass)
	require.Equal(t, 429, ev[0].HTTPStatus)
}

// Huỷ trong lúc verify sau khi bị cắt: CANCELLED, không phải TIMEOUT kèm kết quả dở.
func TestEngine_CancelDuringTheVerifyAfterACut(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s := &script{verifies: []VerifyResult{fail("b"), pass()}, errs: []error{fmt.Errorf("%w after 1s", ErrTurnTimeout)}}
	s.onVerify = func() {
		if s.vi == 2 {
			cancel()
		}
	}
	res, err := s.engine(baseSpec()).Run(ctx)
	require.NoError(t, err)
	require.Equal(t, StopCancelled, res.Reason)
}
