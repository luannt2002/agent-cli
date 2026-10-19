package execpolicy

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// Công tắc "không người trực" đọc từ env GỐC (đã chụp đầu main) nên `.env` của repo
// không bật/tắt được; trong tiến trình thì bật bằng SetUnattended. Test này dùng
// biến toàn cục nên KHÔNG chạy song song.
func TestUnattendedSwitch(t *testing.T) {
	t.Cleanup(func() { SetUnattended(false, false) })
	require.False(t, Unattended())
	require.False(t, UnattendedAllowsCommit())

	for _, v := range []string{"1", "true", "on", "ON"} {
		t.Setenv(UnattendedEnv, v)
		require.Truef(t, Unattended(), "%q phải bật", v)
	}
	for _, v := range []string{"", "0", "off", "false", "no"} {
		t.Setenv(UnattendedEnv, v)
		require.Falsef(t, Unattended(), "%q không được bật", v)
	}

	t.Setenv(UnattendedEnv, "")
	SetUnattended(true, false)
	require.True(t, Unattended())
	require.False(t, UnattendedAllowsCommit(), "bật chế độ không tự mở commit")
	SetUnattended(true, true)
	require.True(t, UnattendedAllowsCommit())

	SetUnattended(false, false)
	t.Setenv(UnattendedCommitEnv, "1")
	require.True(t, UnattendedAllowsCommit())
	require.False(t, Unattended(), "mở commit không bật chế độ")
}

// Hai biến là biến tin cậy: tiến trình con nhận chúng theo ảnh chụp (hoặc theo trạng
// thái không người trực hiện hành), không theo env đã bị repo đổi.
func TestUnattendedKeysAreTrusted(t *testing.T) {
	t.Parallel()
	require.True(t, slices.Contains(trustedKeys, UnattendedEnv))
	require.True(t, slices.Contains(trustedKeys, UnattendedCommitEnv))
}

// `agent-cli loop` và `agent-cli routine run` chạy agent; từ bash của một agent đang
// chạy không người trực chúng bị chặn như run/send/server (agent lồng).
func TestSelfAgentSubcommandsIncludeLoopAndRoutine(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"agent-cli", "loop", "--verify", "true", "x"},
		{"crush", "routine", "run", "x"},
		{"/usr/local/bin/agent-cli", "loop"},
		{"env", "A=b", "agent-cli", "routine", "run", "x"},
	} {
		name, ok := ExternalAgentCommand(args)
		require.Truef(t, ok, "%q phải bị chặn", args)
		require.NotEmpty(t, name)
	}
	// Vế ngược: lệnh con không chạy agent thì không bị chặn.
	for _, args := range [][]string{
		{"agent-cli", "ps"},
		{"agent-cli", "logs"},
		{"agent-cli", "jobs"},
	} {
		_, ok := ExternalAgentCommand(args)
		require.Falsef(t, ok, "%q không chạy agent, không được chặn", args)
	}
}

// Tiến trình con dựng qua ChildEnv (verify của routine/loop...) mang trạng thái không người
// trực HIỆN HÀNH của tiến trình (SetUnattended, đúng như exportUnattendedEnv xuất: commit
// "1" hoặc "0"), không theo env: `.env` của repo đặt giá trị nào cũng không bật được.
// Không parallel: đổi ảnh chụp và công tắc cấp gói.
func TestChildEnvCarriesTheCurrentUnattendedState(t *testing.T) {
	prev := snapshot
	t.Cleanup(func() { snapshot = prev; SetUnattended(false, false) })
	snapshot = map[string]string{}
	dotenv := []string{UnattendedEnv + "=1", UnattendedCommitEnv + "=1"}
	require.Empty(t, ChildEnv(dotenv), "not unattended: a .env value does not turn it on")
	SetUnattended(true, false)
	require.ElementsMatch(t, []string{UnattendedEnv + "=1", UnattendedCommitEnv + "=0"}, ChildEnv(dotenv))
	SetUnattended(true, true)
	require.ElementsMatch(t, []string{UnattendedEnv + "=1", UnattendedCommitEnv + "=1"}, ChildEnv(nil))
}
