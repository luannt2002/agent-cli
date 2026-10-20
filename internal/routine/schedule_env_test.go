package routine

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Lịch in ra phải mang các biến định vị cấu hình/dữ liệu của launcher: thiếu
// CRUSH_GLOBAL_CONFIG thì lần chạy theo lịch tìm định nghĩa ở ~/.config/crush và
// thoát 2 mỗi lần (audit g3, chạy thật với env của cron).
func TestScheduleText_CarriesLauncherEnv(t *testing.T) {
	t.Parallel()
	d := Definition{Name: "nightly", Mode: ModeMaintenance, Repo: "/r", Goal: "g", Verify: "true",
		Schedule: &Schedule{OnCalendar: "*-*-* 03:00:00", Cron: "0 3 * * *"}}
	sum := strings.Repeat("a", 64)
	env := []string{"CRUSH_GLOBAL_CONFIG=/home/u/.config/agent-cli", "CRUSH_GLOBAL_DATA=/home/u/.local/share/agent-cli"}
	out, err := d.ScheduleText("/usr/local/bin/agent-cli", sum, "/usr/bin:/bin", env)
	require.NoError(t, err)
	require.Contains(t, out, "Environment=CRUSH_GLOBAL_CONFIG=/home/u/.config/agent-cli\n")
	require.Contains(t, out, "Environment=CRUSH_GLOBAL_DATA=/home/u/.local/share/agent-cli\n")
	var cronLine string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "0 3 * * *") {
			cronLine = l
		}
	}
	require.Contains(t, cronLine, " CRUSH_GLOBAL_CONFIG=/home/u/.config/agent-cli ")
	require.Contains(t, cronLine, " CRUSH_GLOBAL_DATA=/home/u/.local/share/agent-cli ")
}

// Giá trị có ký tự shell/khoảng trắng không được ghi vào unit hay crontab.
func TestScheduleText_RefusesUnsafeEnvValues(t *testing.T) {
	t.Parallel()
	d := Definition{Name: "nightly", Mode: ModeMaintenance, Repo: "/r", Goal: "g", Verify: "true",
		Schedule: &Schedule{Cron: "0 3 * * *"}}
	_, err := d.ScheduleText("/bin/agent-cli", strings.Repeat("a", 64), "/bin", []string{"CRUSH_GLOBAL_CONFIG=/x; rm -rf ~"})
	require.Error(t, err)
}

// Danh sách model được phép có dấu phẩy và hai chấm: phải qua được. Ký tự shell,
// khoảng trắng, nháy, $, %, xuống dòng, gạch ngược vẫn bị từ chối.
func TestScheduleText_EnvAllowsModelAllowlist(t *testing.T) {
	t.Parallel()
	d := Definition{Name: "nightly", Mode: ModeMaintenance, Repo: "/r", Goal: "g", Verify: "true",
		Schedule: &Schedule{Cron: "0 3 * * *"}}
	sum := strings.Repeat("a", 64)
	out, err := d.ScheduleText("/bin/agent-cli", sum, "/bin", []string{"DEW_ALLOWED_MODELS=openai:gpt-5,anthropic:claude-x"})
	require.NoError(t, err)
	require.Contains(t, out, " DEW_ALLOWED_MODELS=openai:gpt-5,anthropic:claude-x ")
	for _, bad := range []string{"a b", `a"b`, "a'b", "a$b", "a%b", "a\nb", `a\b`, "a;b", "a`b"} {
		_, err := d.ScheduleText("/bin/agent-cli", sum, "/bin", []string{"DEW_ALLOWED_MODELS=" + bad})
		require.Error(t, err, "%q", bad)
	}
}
