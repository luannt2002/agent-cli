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

// Hai biến là biến tin cậy: ChildEnv phải mang chúng sang server con theo ảnh
// chụp, không theo env đã bị repo đổi.
func TestUnattendedKeysAreTrusted(t *testing.T) {
	t.Parallel()
	require.True(t, slices.Contains(trustedKeys, UnattendedEnv))
	require.True(t, slices.Contains(trustedKeys, UnattendedCommitEnv))
}
