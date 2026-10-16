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

// memJournal giữ sự kiện trong bộ nhớ để test đọc lại.
type memJournal struct{ events []Event }

func (j *memJournal) Append(e Event) error { j.events = append(j.events, e); return nil }

func (j *memJournal) kinds() []string {
	out := make([]string, len(j.events))
	for i, e := range j.events {
		out[i] = e.Kind
	}
	return out
}

// script dựng engine với agent/verify giả theo kịch bản: verifies[i] là kết quả
// verify lần thứ i (lần 0 là baseline trước mọi lượt agent), fingerprints[i] là
// dấu vân tay cây làm việc sau mỗi lượt agent.
type script struct {
	verifies     []VerifyResult
	fingerprints []string
	prompts      []string
	agentErr     error
	onAgent      func()
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
			s.clock = s.clock.Add(s.step)
			return RunInfo{SessionID: fmt.Sprintf("sess-%d", s.ai), Text: "ok"}, nil
		},
		Verify: func(ctx context.Context, cmd string) (VerifyResult, error) {
			if s.vi >= len(s.verifies) {
				return VerifyResult{}, errors.New("hết kịch bản verify")
			}
			r := s.verifies[s.vi]
			s.vi++
			return r, nil
		},
		Fingerprint: func(ctx context.Context) (string, error) {
			i := s.ai - 1
			if i < 0 || i >= len(s.fingerprints) {
				return "fp-base", nil
			}
			return s.fingerprints[i], nil
		},
		Journal: s.journal,
		Now:     func() time.Time { return s.clock },
	}
}

func pass() VerifyResult { return VerifyResult{Status: StatusPass} }
func fail(out string) VerifyResult {
	return VerifyResult{Status: StatusFail, ExitCode: 1, Output: out}
}

func baseSpec() Spec {
	return Spec{Name: "t", Goal: "make the tests pass", Verify: "go test ./...", MaxRepairs: 2}
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

func TestEngine_AgentErrorStopsWithRunError(t *testing.T) {
	t.Parallel()
	s := &script{verifies: []VerifyResult{fail("b")}, agentErr: errors.New("provider down")}
	res, err := s.engine(baseSpec()).Run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StopRunError, res.Reason)
	require.Contains(t, res.Detail, "provider down")
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
