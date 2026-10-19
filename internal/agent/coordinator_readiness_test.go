package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/agent/prompt"
	"github.com/charmbracelet/crush/internal/agent/tools/mcp"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/stretchr/testify/require"
)

// TestBuildAgentReadinessSurvivesCallerCancellation is a regression test for
// the CRUSH_CLIENT_SERVER=1 "new session hangs" bug.
//
// buildAgent starts readiness goroutines that build the system prompt and the
// initial tool list. Several server entry points build an agent from a
// short-lived HTTP request context — the InitAgent/UpdateAgent handlers, and
// the sub-agent build reached through UpdateModels -> buildTools -> agentTool.
// When that request context was canceled the moment the handler returned, the
// readyWg errgroup recorded context.Canceled and every later coordinator.run
// failed at readyWg.Wait() before emitting anything — the session hung with
// no visible LLM response. (This was made worse while the tool-list goroutine
// also blocked in mcp.WaitForInit, which kept it parked long enough to
// observe the cancellation; the readiness work no longer waits on MCP init —
// see coordinator.run — but the cancellation detachment still matters.)
//
// The fix detaches the readiness work from the caller context via
// context.WithoutCancel, so canceling the context that triggered the build no
// longer poisons readyWg. Here we build an agent with a cancelable context,
// cancel it, and require that readyWg still completes cleanly.
func TestBuildAgentReadinessSurvivesCallerCancellation(t *testing.T) {
	env := testEnv(t)

	// Minimal hermetic config: one openai-typed provider with selected large
	// and small models so buildAgentModels and the system-prompt build both
	// succeed. No MCP servers are configured, so initialization would complete
	// instantly if we let it — we arm the gate anyway to prove the readiness
	// goroutines no longer block on it.
	crushJSON := `{
  "options": {"disable_default_providers": true, "disable_provider_auto_update": true},
  "providers": {"mock": {"id": "mock", "name": "Mock", "type": "openai",
    "base_url": "http://127.0.0.1:9/v1", "api_key": "test-key",
    "models": [{"id": "mock-model", "name": "Mock", "context_window": 8192, "default_max_tokens": 128}]}},
  "models": {"large": {"provider": "mock", "model": "mock-model"},
             "small": {"provider": "mock", "model": "mock-model"}}
}`
	require.NoError(t, os.WriteFile(filepath.Join(env.workingDir, "crush.json"), []byte(crushJSON), 0o644))

	cfg, err := config.Init(env.workingDir, "", false)
	require.NoError(t, err)
	cfg.SetupAgents()

	coord := &coordinator{
		cfg:         cfg,
		sessions:    env.sessions,
		messages:    env.messages,
		permissions: env.permissions,
		history:     env.history,
		filetracker: *env.filetracker,
	}

	// Arm the MCP init gate. We never complete init; the readiness goroutines
	// must not care, since they build the tool list from the registry as it
	// stands rather than waiting for initialization to finish.
	mcp.ArmInit()
	t.Cleanup(mcp.DisarmInit)

	p, err := coderPrompt(prompt.WithWorkingDir(env.workingDir))
	require.NoError(t, err)
	agentCfg := cfg.Config().Agents[config.AgentCoder]

	ctx, cancel := context.WithCancel(context.Background())
	_, err = coord.buildAgent(ctx, p, agentCfg, false)
	require.NoError(t, err)

	// The caller goes away, mirroring an HTTP handler returning and canceling
	// its request context.
	cancel()

	done := make(chan error, 1)
	go func() { done <- coord.readyWg.Wait() }()

	select {
	case err := <-done:
		// context.Canceled is the regression: the caller's cancellation
		// leaked into the readiness work and poisoned the errgroup.
		require.NotErrorIs(t, err, context.Canceled,
			"readyWg was poisoned by caller cancellation (client/server new-session hang regression)")
		require.NoError(t, err, "unexpected buildAgent readiness error")
	case <-time.After(2 * time.Second):
		t.Fatal("readyWg did not complete; the readiness goroutines must not block on MCP init")
	}
}

// TestRunWaitIsSafeAgainstConcurrentAgentBuild là test hồi quy cho panic
// "sync: WaitGroup is reused before previous Wait has returned" ở
// coordinator.run khi ~10 `agent-cli run` cùng vào một server.
//
// run chờ việc dựng agent (system prompt, bộ tool) xong rồi mới chạy lượt.
// Việc dựng do buildAgent khởi, mà buildAgent chạy cả từ UpdateModels (mỗi
// `run` gọi UpdateAgent) lẫn từ updateAgentModels của lượt khác, qua
// buildTools -> agentTool. Với errgroup.Group dùng chung, một Go bắt đầu từ
// bộ đếm 0 mà không happens-before một Wait đang chờ là dùng sai
// sync.WaitGroup; trúng cửa sổ hẹp thì panic và kéo chết cả server.
//
// Cửa sổ panic quá hẹp để dựng lại chắc chắn, nên test dựng đúng thứ tự xấu
// rồi để -race bắt vi phạm một cách tất định (task test và CI đều chạy
// -race): agentTool khởi việc dựng từ 0, rồi một lượt run ở goroutine không
// đồng bộ gì với nó phải chờ việc dựng còn dở. Goroutine của lượt chỉ dò
// runtime.Stack để biết lúc vào, không chạm khoá hay kênh chung, nên không
// tạo happens-before che mắt -race. Sau Go, goroutine test chỉ làm việc nhẹ
// trước khi lượt chờ: -race chỉ báo khi còn khôi phục được stack của lần
// truy cập trước, việc nặng xen giữa (cả buildTools) đẩy nó khỏi lịch sử.
//
// Test cũng giữ ngữ nghĩa cũ: lượt không chạy khi việc dựng còn dở, kể cả
// việc dựng lồng được khởi từ bên trong một việc dựng khác (agentTool trong
// lúc dựng bộ tool), và không kẹt khi việc dựng lồng như vậy.
func TestRunWaitIsSafeAgainstConcurrentAgentBuild(t *testing.T) {
	env := testEnv(t)
	crushJSON := `{
  "options": {"disable_default_providers": true, "disable_provider_auto_update": true},
  "providers": {"mock": {"id": "mock", "name": "Mock", "type": "openai",
    "base_url": "http://127.0.0.1:9/v1", "api_key": "test-key",
    "models": [{"id": "mock-model", "name": "Mock", "context_window": 8192, "default_max_tokens": 128}]}},
  "models": {"large": {"provider": "mock", "model": "mock-model"},
             "small": {"provider": "mock", "model": "mock-model"}}
}`
	require.NoError(t, os.WriteFile(filepath.Join(env.workingDir, "crush.json"), []byte(crushJSON), 0o644))
	cfg, err := config.Init(env.workingDir, "", false)
	require.NoError(t, err)
	cfg.SetupAgents()

	coord := &coordinator{
		cfg:         cfg,
		sessions:    env.sessions,
		messages:    env.messages,
		permissions: env.permissions,
		history:     env.history,
		filetracker: *env.filetracker,
		mainAgent: newMockAgent("mock", 128, func(context.Context, SessionAgentCall) (*fantasy.AgentResult, error) {
			return agentResultWithText("ok"), nil
		}),
		mainAgentName: config.AgentCoder,
	}
	sess, err := env.sessions.Create(t.Context(), "readiness")
	require.NoError(t, err)

	ctx := t.Context()
	release, releaseNested := make(chan struct{}), make(chan struct{})
	runDone := make(chan error, 1)
	go func() {
		// Không gọi t.* trước khi vào run: khoá trong testing.T tạo
		// happens-before với goroutine test và che vi phạm khỏi -race.
		if !waitParked("agent.holdReadiness(") {
			runDone <- errors.New("readiness gate never parked")
			return
		}
		_, err := coord.Run(ctx, sess.ID, "hi")
		runDone <- err
	}()

	// Dựng sub-agent như buildTools làm: Go từ bộ đếm 0.
	_, err = coord.agentTool(ctx)
	// Giữ việc dựng còn dở (như dựng system prompt chậm) để lượt phải chờ;
	// khi được thả, nó khởi thêm một việc dựng lồng rồi mới xong.
	coord.readyWg.Go(func() error {
		holdReadiness(release)
		coord.readyWg.Go(func() error {
			holdReadiness(releaseNested)
			return nil
		})
		return nil
	})
	require.NoError(t, err)

	notYet := func(stage string) {
		require.True(t, waitParked("(*coordinator).run(", ").Wait("), "run is not waiting for readiness (%s)", stage)
		select {
		case err := <-runDone:
			t.Fatalf("run went ahead before the agent was ready (%s): %v", stage, err)
		default:
		}
	}
	notYet("build pending")
	close(release)
	// Việc dựng ngoài không thể đứng lại trên kênh đã đóng, nên goroutine
	// còn đứng trong holdReadiness chỉ có thể là việc dựng lồng.
	require.True(t, waitParked("agent.holdReadiness("), "nested build never started")
	notYet("nested build pending")

	close(releaseNested)
	select {
	case err := <-runDone:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("run stayed blocked after readiness completed")
	}
	require.NoError(t, coord.readyWg.Wait())
}

//go:noinline
func holdReadiness(release <-chan struct{}) { <-release }

// waitParked chờ tới khi có goroutine đang đứng (không chạy) mà stack chứa
// mọi chuỗi cho trước. Chỉ đọc runtime.Stack nên không tạo happens-before.
func waitParked(subs ...string) bool {
	buf := make([]byte, 8<<20)
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		for g := range strings.SplitSeq(string(buf[:runtime.Stack(buf, true)]), "\n\n") {
			header, _, _ := strings.Cut(g, "\n")
			if strings.Contains(header, "[running]") || strings.Contains(header, "[runnable]") {
				continue
			}
			found := true
			for _, s := range subs {
				found = found && strings.Contains(g, s)
			}
			if found {
				return true
			}
		}
	}
	return false
}
