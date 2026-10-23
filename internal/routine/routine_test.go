package routine

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const goodDef = `{
  "name": "nightly",
  "mode": "maintenance",
  "repo": "/home/u/proj",
  "goal": "Remove tests that can never fail",
  "verify": "go test ./...",
  "schedule": {"on_calendar": "Mon..Fri 09:00", "timezone": "Asia/Ho_Chi_Minh"}
}`

func TestParse_ReturnsStableHashAndDefaults(t *testing.T) {
	t.Parallel()
	d, sum, err := Parse([]byte(goodDef))
	require.NoError(t, err)
	require.Equal(t, "nightly", d.Name)
	_, sum2, _ := Parse([]byte(goodDef))
	require.Equal(t, sum, sum2)
	require.Len(t, sum, 64)
	_, sum3, _ := Parse([]byte(strings.Replace(goodDef, "09:00", "10:00", 1)))
	require.NotEqual(t, sum, sum3, "đổi một ký tự trong định nghĩa phải đổi dấu hash (cron ghim hash)")

	repairs, maxDur, runTO, verifyTO, err := d.Limits()
	require.NoError(t, err)
	require.Equal(t, 2, repairs, "mặc định hai lượt sửa")
	require.Zero(t, maxDur)
	require.Equal(t, 45*time.Minute, runTO)
	require.Equal(t, 30*time.Minute, verifyTO)
}

func TestParse_RejectsUnknownFieldsAndGarbage(t *testing.T) {
	t.Parallel()
	_, _, err := Parse([]byte(strings.Replace(goodDef, `"goal"`, `"gaol"`, 1)))
	require.Error(t, err, "gõ sai tên trường không được lặng lẽ bị bỏ qua")
	_, _, err = Parse([]byte(`not json`))
	require.Error(t, err)
	_, _, err = Parse([]byte(goodDef + goodDef))
	require.Error(t, err, "hai object liền nhau là tệp hỏng")
}

func TestValidate(t *testing.T) {
	t.Parallel()
	mod := func(f func(d *Definition)) Definition {
		d, _, err := Parse([]byte(goodDef))
		require.NoError(t, err)
		f(&d)
		return d
	}
	require.NoError(t, mod(func(*Definition) {}).Validate())

	for name, tc := range map[string]struct {
		def  Definition
		want string
	}{
		"no verify (cổng L0)":      {mod(func(d *Definition) { d.Verify = "  " }), "verify"},
		"no goal":                  {mod(func(d *Definition) { d.Goal = "" }), "goal"},
		"no mode (không lịch)":     {mod(func(d *Definition) { d.Mode = ""; d.Schedule = nil }), "mode must be"},
		"bad mode (không lịch)":    {mod(func(d *Definition) { d.Mode = "feature"; d.Schedule = nil }), "mode must be"},
		"relative repo":            {mod(func(d *Definition) { d.Repo = "proj" }), "repo"},
		"bad name":                 {mod(func(d *Definition) { d.Name = "../evil" }), "name"},
		"empty name":               {mod(func(d *Definition) { d.Name = "" }), "name"},
		"schedule on goal mode":    {mod(func(d *Definition) { d.Mode = ModeGoal }), "maintenance"},
		"schedule without when":    {mod(func(d *Definition) { d.Schedule = &Schedule{} }), "on_calendar"},
		"bad duration":             {mod(func(d *Definition) { d.MaxDuration = "soon" }), "max_duration"},
		"negative duration":        {mod(func(d *Definition) { d.RunTimeout = "-5m" }), "run_timeout"},
		"negative repairs":         {mod(func(d *Definition) { n := -1; d.MaxRepairs = &n }), "max_repairs"},
		"too many repairs":         {mod(func(d *Definition) { n := 6; d.MaxRepairs = &n }), "max_repairs"},
		"negative stream retries":  {mod(func(d *Definition) { n := -1; d.StreamRetries = &n }), "stream_retries"},
		"too many stream retries":  {mod(func(d *Definition) { n := 6; d.StreamRetries = &n }), "stream_retries"},
		"bad timezone":             {mod(func(d *Definition) { d.Schedule.Timezone = "Mars/Base" }), "timezone"},
		"cron wrong field count":   {mod(func(d *Definition) { d.Schedule = &Schedule{Cron: "0 9 * *"} }), "cron"},
		"unsafe char in on_cal":    {mod(func(d *Definition) { d.Schedule.OnCalendar = "daily\nExecStartPre=/bin/sh" }), "on_calendar"},
		"unsafe char in cron":      {mod(func(d *Definition) { d.Schedule = &Schedule{Cron: "0 9 * * *; rm -rf /"} }), "cron"},
		"unsafe char in timezone":  {mod(func(d *Definition) { d.Schedule.Timezone = "UTC\nX=1" }), "timezone"},
		"commit with newline name": {mod(func(d *Definition) { d.Name = "a\nb" }), "name"},
	} {
		err := tc.def.Validate()
		require.Errorf(t, err, "%s phải bị từ chối", name)
		require.Containsf(t, err.Error(), tc.want, "%s: lỗi phải nêu đúng trường", name)
	}

	// Vế ngược: goal-mode không lịch, và 5 lượt sửa, vẫn hợp lệ.
	require.NoError(t, mod(func(d *Definition) { d.Mode = ModeGoal; d.Schedule = nil }).Validate())
	require.NoError(t, mod(func(d *Definition) { n := 5; d.MaxRepairs = &n }).Validate())
	require.NoError(t, mod(func(d *Definition) { d.Schedule = &Schedule{Cron: "0 9 * * 1-5"} }).Validate())
}

func TestScheduleText_SystemdAndCron(t *testing.T) {
	t.Parallel()
	d, sum, err := Parse([]byte(goodDef))
	require.NoError(t, err)
	out, err := d.ScheduleText("/usr/local/bin/dew", sum, "/usr/bin:/bin", nil)
	require.NoError(t, err)
	for _, want := range []string{
		"dew-routine-nightly.service",
		"dew-routine-nightly.timer",
		"OnCalendar=Mon..Fri 09:00",
		"Persistent=true",
		"Environment=TZ=Asia/Ho_Chi_Minh",
		"Environment=PATH=/usr/bin:/bin",
		"ExecStart=/usr/local/bin/dew routine run nightly --scheduled --expect-sha " + sum,
		"TimeoutStartSec=",
		"systemctl --user daemon-reload",
		"loginctl enable-linger",
	} {
		require.Contains(t, out, want)
	}

	d.Schedule = &Schedule{Cron: "0 9 * * 1-5", Timezone: "Asia/Ho_Chi_Minh"}
	out, err = d.ScheduleText("/usr/local/bin/dew", sum, "/usr/bin:/bin", nil)
	require.NoError(t, err)
	require.Contains(t, out, "0 9 * * 1-5 TZ=Asia/Ho_Chi_Minh PATH=/usr/bin:/bin /usr/local/bin/dew routine run nightly --scheduled --expect-sha "+sum)
	require.NotContains(t, out, "OnCalendar=", "chỉ khai cron thì không sinh timer systemd")
}

func TestScheduleText_RefusesWhatCannotBeScheduled(t *testing.T) {
	t.Parallel()
	d, sum, _ := Parse([]byte(goodDef))
	d.Mode = ModeGoal
	_, err := d.ScheduleText("/bin/dew", sum, "/bin", nil)
	require.ErrorContains(t, err, "maintenance")

	d, _, _ = Parse([]byte(goodDef))
	d.Schedule = nil
	_, err = d.ScheduleText("/bin/dew", sum, "/bin", nil)
	require.ErrorContains(t, err, "schedule")

	d, _, _ = Parse([]byte(goodDef))
	_, err = d.ScheduleText("/bin/agent cli", sum, "/bin", nil)
	require.NoError(t, err)
	out, _ := d.ScheduleText("/bin/agent cli", sum, "/bin", nil)
	require.Contains(t, out, `ExecStart="/bin/agent cli" routine run`, "đường dẫn có khoảng trắng phải được bọc nháy")
}

// F1: stream_retries (0–5, mặc định 2; 0 = không chạy lại) đọc từ tệp định nghĩa.
func TestStreamRetries(t *testing.T) {
	t.Parallel()
	d, _, err := Parse([]byte(goodDef))
	require.NoError(t, err)
	n, err := d.StreamRetryLimit()
	require.NoError(t, err)
	require.Equal(t, 2, n, "mặc định")
	for _, v := range []int{0, 5} {
		d, _, err := Parse([]byte(strings.Replace(goodDef, `"verify"`, fmt.Sprintf(`"stream_retries": %d, "verify"`, v), 1)))
		require.NoError(t, err)
		require.NoError(t, d.Validate())
		n, err := d.StreamRetryLimit()
		require.NoError(t, err)
		require.Equal(t, v, n)
	}
}

// Văn bản lịch không hứa checkout của người dùng không bị đụng: lớp chặn là guard-rail, không phải
// sandbox (audit 22/10: agent đổi được nhánh, tệp và .git dùng chung của checkout thật).
func TestScheduleText_SaysGuardRailNotSandbox(t *testing.T) {
	t.Parallel()
	d, sum, err := Parse([]byte(goodDef))
	require.NoError(t, err)
	out, err := d.ScheduleText("/usr/local/bin/dew", sum, "/usr/bin:/bin", nil)
	require.NoError(t, err)
	require.Contains(t, out, "guard-rail, not a sandbox")
	require.Contains(t, out, "shared .git")
	require.NotContains(t, out, "never touched")
}
