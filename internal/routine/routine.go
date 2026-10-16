// Package routine là định nghĩa một routine: quy trình lặp lại, chạy bằng engine
// Loop (internal/loop) và có thể được lên lịch.
//
// Theo từ vựng của chủ dự án (learn_claudecode): Routine không có đích, theo thời
// gian; Loop có đích, dừng khi điều kiện đúng. Ở đây một engine phục vụ hai kiểu
// qua trường Mode: "maintenance" (cơ học, kiểm được, vô tận: được phép lên lịch)
// và "goal" (có đích như làm tính năng: chỉ chạy tay). Lịch cho việc làm tính
// năng bị từ chối ngay từ lúc kiểm định nghĩa.
package routine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	ModeMaintenance = "maintenance"
	ModeGoal        = "goal"
)

// Giá trị mặc định khi định nghĩa không khai.
const (
	DefaultMaxRepairs    = 2
	DefaultRunTimeout    = 45 * time.Minute
	DefaultVerifyTimeout = 30 * time.Minute
	// maxRepairsCap: hai vòng là hết; cho phép tới 5 nhưng không hơn.
	maxRepairsCap = 5
)

// Schedule là lịch của routine maintenance. Khai on_calendar (systemd) và/hoặc
// cron (5 trường). agent-cli chỉ IN văn bản lịch, không tự cài.
type Schedule struct {
	OnCalendar string `json:"on_calendar,omitempty"`
	Cron       string `json:"cron,omitempty"`
	Timezone   string `json:"timezone,omitempty"`
}

// Definition là một routine. Tệp định nghĩa nằm ở thư mục cấu hình của NGƯỜI DÙNG
// (không phải trong repo): repo lạ không tự thêm được việc chạy theo lịch.
type Definition struct {
	Name          string    `json:"name"`
	Mode          string    `json:"mode"`
	Repo          string    `json:"repo"`
	Goal          string    `json:"goal"`
	Verify        string    `json:"verify"`
	MaxRepairs    *int      `json:"max_repairs,omitempty"`
	MaxDuration   string    `json:"max_duration,omitempty"`
	RunTimeout    string    `json:"run_timeout,omitempty"`
	VerifyTimeout string    `json:"verify_timeout,omitempty"`
	Model         string    `json:"model,omitempty"`
	Commit        bool      `json:"commit,omitempty"` // cho phép git commit trên nhánh routine/*; không bao giờ push
	Schedule      *Schedule `json:"schedule,omitempty"`
}

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// safeText: văn bản đi vào tệp unit/crontab không được có ký tự điều khiển hay
// dấu nối lệnh (chống chèn dòng).
var safeText = regexp.MustCompile(`^[A-Za-z0-9 .,:*/\-_+@=]+$`)

// Parse đọc một định nghĩa, từ chối trường lạ (gõ sai tên trường không bị bỏ qua
// lặng lẽ) và nội dung thừa; trả về cả sha256 của nguyên văn tệp để ghim.
func Parse(data []byte) (Definition, string, error) {
	var d Definition
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return Definition{}, "", fmt.Errorf("invalid routine definition: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return Definition{}, "", errors.New("invalid routine definition: unexpected content after the object")
	}
	sum := sha256.Sum256(data)
	return d, hex.EncodeToString(sum[:]), nil
}

// Validate là cổng L0: không nhận routine chưa phát biểu được "xong" (verify),
// và không cho lên lịch việc không phải bảo trì.
func (d Definition) Validate() error {
	switch {
	case !nameRe.MatchString(d.Name):
		return fmt.Errorf("name %q must match %s", d.Name, nameRe)
	case d.Mode != ModeMaintenance && d.Mode != ModeGoal:
		return fmt.Errorf("mode must be %q or %q (got %q)", ModeMaintenance, ModeGoal, d.Mode)
	case !filepath.IsAbs(d.Repo):
		return fmt.Errorf("repo must be an absolute path (got %q)", d.Repo)
	case strings.TrimSpace(d.Goal) == "":
		return errors.New("goal is empty")
	case strings.TrimSpace(d.Verify) == "":
		return errors.New("verify is empty: a routine needs the command that defines \"done\" (exit 0)")
	}
	if _, _, _, _, err := d.Limits(); err != nil {
		return err
	}
	if d.Schedule == nil {
		return nil
	}
	s := d.Schedule
	switch {
	case d.Mode != ModeMaintenance:
		return fmt.Errorf("a schedule is only allowed for mode %q: scheduled runs are unattended and must be mechanical, verifiable and endless (mode %q runs by hand)", ModeMaintenance, d.Mode)
	case s.OnCalendar == "" && s.Cron == "":
		return errors.New("schedule needs on_calendar (systemd) or cron")
	case s.OnCalendar != "" && !safeText.MatchString(s.OnCalendar):
		return fmt.Errorf("schedule.on_calendar has unsupported characters: %q", s.OnCalendar)
	case s.Cron != "" && !safeText.MatchString(s.Cron):
		return fmt.Errorf("schedule.cron has unsupported characters: %q", s.Cron)
	case s.Cron != "" && len(strings.Fields(s.Cron)) != 5:
		return fmt.Errorf("schedule.cron must have 5 fields (got %q)", s.Cron)
	case s.Timezone != "" && (!safeText.MatchString(s.Timezone) || strings.Contains(s.Timezone, " ")):
		return fmt.Errorf("schedule.timezone has unsupported characters: %q", s.Timezone)
	}
	if s.Timezone != "" {
		if _, err := time.LoadLocation(s.Timezone); err != nil {
			return fmt.Errorf("schedule.timezone %q is not a known time zone", s.Timezone)
		}
	}
	return nil
}

// Limits trả các giới hạn đã giải (mặc định khi không khai).
func (d Definition) Limits() (maxRepairs int, maxDuration, runTimeout, verifyTimeout time.Duration, err error) {
	maxRepairs = DefaultMaxRepairs
	if d.MaxRepairs != nil {
		if *d.MaxRepairs < 0 || *d.MaxRepairs > maxRepairsCap {
			return 0, 0, 0, 0, fmt.Errorf("max_repairs must be between 0 and %d (got %d)", maxRepairsCap, *d.MaxRepairs)
		}
		maxRepairs = *d.MaxRepairs
	}
	parse := func(field, v string, def time.Duration) (time.Duration, error) {
		if v == "" {
			return def, nil
		}
		dur, err := time.ParseDuration(v)
		if err != nil || dur < 0 {
			return 0, fmt.Errorf("%s must be a non-negative duration like 90m or 2h (got %q)", field, v)
		}
		return dur, nil
	}
	if maxDuration, err = parse("max_duration", d.MaxDuration, 0); err != nil {
		return 0, 0, 0, 0, err
	}
	if runTimeout, err = parse("run_timeout", d.RunTimeout, DefaultRunTimeout); err != nil {
		return 0, 0, 0, 0, err
	}
	if verifyTimeout, err = parse("verify_timeout", d.VerifyTimeout, DefaultVerifyTimeout); err != nil {
		return 0, 0, 0, 0, err
	}
	return maxRepairs, maxDuration, runTimeout, verifyTimeout, nil
}

// ScheduleText sinh văn bản lịch để NGƯỜI DÙNG tự cài (agent-cli không bao giờ
// tự sửa crontab hay timer: agent bị cấm lệnh đó). bin là đường dẫn agent-cli,
// sum là sha256 của định nghĩa (lượt chạy từ chối khi tệp đã đổi), path là PATH
// tường minh (cron/systemd có env gần như rỗng).
func (d Definition) ScheduleText(bin, sum, path string) (string, error) {
	if d.Mode != ModeMaintenance {
		return "", fmt.Errorf("only mode %q routines can be scheduled", ModeMaintenance)
	}
	if d.Schedule == nil {
		return "", errors.New("the definition has no schedule")
	}
	if err := d.Validate(); err != nil {
		return "", err
	}
	s := d.Schedule
	_, maxDur, _, _, _ := d.Limits()
	timeout := 6 * time.Hour
	if maxDur > 0 {
		timeout = maxDur + 30*time.Minute
	}
	cmdline := fmt.Sprintf("%s routine run %s --scheduled --expect-sha %s", quoteExec(bin), d.Name, sum)

	var b strings.Builder
	fmt.Fprintf(&b, "# Routine %q (mode %s). agent-cli only PRINTS this; install it yourself.\n", d.Name, d.Mode)
	fmt.Fprintf(&b, "# The run refuses to start if the definition file changed (sha256 %s).\n", sum)
	fmt.Fprintf(&b, "# It works in its own git worktree on a routine/* branch, never pushes or merges,\n")
	fmt.Fprintf(&b, "# and takes a lock, so a second run while one is active exits 75 (EX_TEMPFAIL).\n\n")
	if s.OnCalendar != "" {
		fmt.Fprintf(&b, "# --- systemd user units: ~/.config/systemd/user/agent-cli-routine-%s.service\n", d.Name)
		fmt.Fprintf(&b, "[Unit]\nDescription=agent-cli routine %s\n\n[Service]\nType=oneshot\n", d.Name)
		fmt.Fprintf(&b, "Environment=HOME=%%h\nEnvironment=PATH=%s\n", path)
		if s.Timezone != "" {
			fmt.Fprintf(&b, "Environment=TZ=%s\n", s.Timezone)
		}
		fmt.Fprintf(&b, "ExecStart=%s\nTimeoutStartSec=%s\n\n", cmdline, strconv.Itoa(int(timeout.Seconds())))
		fmt.Fprintf(&b, "# --- ~/.config/systemd/user/agent-cli-routine-%s.timer\n", d.Name)
		fmt.Fprintf(&b, "[Unit]\nDescription=agent-cli routine %s schedule\n\n[Timer]\nOnCalendar=%s\nPersistent=true\n\n[Install]\nWantedBy=timers.target\n\n", d.Name, s.OnCalendar)
		fmt.Fprintf(&b, "# --- install (run these yourself)\n")
		fmt.Fprintf(&b, "#   systemd-analyze --user verify agent-cli-routine-%s.service agent-cli-routine-%s.timer\n", d.Name, d.Name)
		fmt.Fprintf(&b, "#   systemctl --user daemon-reload && systemctl --user enable --now agent-cli-routine-%s.timer\n", d.Name)
		fmt.Fprintf(&b, "#   loginctl enable-linger $USER   # needed for the timer to fire while you are logged out\n")
		if s.Cron != "" {
			b.WriteString("\n")
		}
	}
	if s.Cron != "" {
		fmt.Fprintf(&b, "# --- crontab line (crontab -e). cron has an almost empty environment: HOME/PATH/TZ are explicit.\n")
		env := ""
		if s.Timezone != "" {
			env += " TZ=" + s.Timezone
		}
		fmt.Fprintf(&b, "%s%s PATH=%s %s\n", s.Cron, env, path, cmdline)
	}
	return b.String(), nil
}

func quoteExec(p string) string {
	if strings.ContainsAny(p, " \t") {
		return `"` + p + `"`
	}
	return p
}
