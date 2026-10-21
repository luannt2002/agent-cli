package routine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// VerifyPassNotGreen: verify có chạy và xanh nhưng run không đạt (bị cắt, lỗi, hết hạn mức…): cần người soát.
const VerifyPassNotGreen = "PASS_NOT_GREEN"

// RunSummary là lượt chạy MỚI NHẤT của một routine, đọc từ report.json cho màn
// Home. Chỉ đọc: không chạm khoá của routine (thử khoá có thể làm một lượt đang
// khởi động thoát 75).
type RunSummary struct {
	Routine  string
	Stamp    string // Tên thư mục của lượt (dấu thời gian).
	Reason   string // Lý do trong báo cáo; RUNNING được đổi thành ABANDONED (chết hẳn) hoặc UNKNOWN (không kiểm được).
	Verify   string // "PASS" (run đạt), "FAIL", VerifyPassNotGreen, hoặc rỗng (verify chưa chạy).
	Worktree string
	// WorktreeExists: thư mục worktree còn trên đĩa (đo lúc đọc, để Home khỏi stat).
	WorktreeExists bool
	Branch         string
	Started        time.Time
	Duration       time.Duration // Đang chạy: now - started; xong: finished - started; 0 = không rõ.
	Running        bool          // Báo cáo RUNNING và tiến trình thật sự còn sống.
	// Unknown: báo cáo RUNNING mà không kiểm được sống hay chết (không có pid,
	// hoặc hệ điều hành không cho kiểm). Coi như có thể đang chạy: không mở worktree.
	Unknown bool
	// Unreadable: báo cáo mất, hỏng hoặc dở dang. Các trường còn lại trống.
	Unreadable bool
	Attempts   int
	Changed    int
	PID        int
}

type runReport struct {
	Started      time.Time `json:"started"`
	Finished     time.Time `json:"finished"`
	Reason       string    `json:"reason"`
	VerifyStatus string    `json:"verify_status"`
	Attempts     int       `json:"attempts"`
	ChangedFiles []string  `json:"changed_files"`
	Branch       string    `json:"branch"`
	Worktree     string    `json:"worktree"`
	PID          int       `json:"pid"`
}

// LatestRuns trả lượt mới nhất của mỗi routine dưới dataDir (dataDir/<tên>/runs/
// <dấu thời gian>/report.json): lượt đang chạy trước, rồi lượt bắt đầu gần nhất.
// Mỗi routine tốn một lần liệt kê thư mục và một lần đọc báo cáo. Thư mục không
// phải routine (không có runs/) hoặc routine chưa có lượt nào bị bỏ qua; báo
// cáo không đọc được cho một dòng Unreadable, không ảnh hưởng routine khác.
func LatestRuns(dataDir string, now time.Time) []RunSummary {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil
	}
	var out []RunSummary
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		stamps, err := os.ReadDir(filepath.Join(dataDir, e.Name(), "runs"))
		if err != nil {
			continue
		}
		stamp := ""
		for i := len(stamps) - 1; i >= 0 && stamp == ""; i-- { // ReadDir sắp theo tên.
			if stamps[i].IsDir() {
				stamp = stamps[i].Name()
			}
		}
		if stamp == "" {
			continue
		}
		out = append(out, readRun(filepath.Join(dataDir, e.Name(), "runs", stamp), e.Name(), stamp, now))
	}
	slices.SortStableFunc(out, func(a, b RunSummary) int {
		if a.Running != b.Running {
			if a.Running {
				return -1
			}
			return 1
		}
		return sortTime(b).Compare(sortTime(a))
	})
	return out
}

// sortTime là lúc bắt đầu, hoặc (báo cáo không đọc được) lúc ghi trong tên thư mục
// lượt (UTC, cùng dạng với cmd/routine.go), để dòng lỗi không luôn đứng cuối.
func sortTime(r RunSummary) time.Time {
	if !r.Started.IsZero() {
		return r.Started
	}
	t, _ := time.Parse("20060102-150405", r.Stamp)
	return t
}

func readRun(dir, name, stamp string, now time.Time) RunSummary {
	s := RunSummary{Routine: name, Stamp: stamp}
	var rep runReport
	raw, err := os.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil || json.Unmarshal(raw, &rep) != nil || rep.Started.IsZero() {
		s.Unreadable = true
		return s
	}
	s.Reason, s.Worktree, s.Branch, s.PID = rep.Reason, rep.Worktree, rep.Branch, rep.PID
	s.Started, s.Attempts, s.Changed = rep.Started, rep.Attempts, len(rep.ChangedFiles)
	if rep.Reason == "RUNNING" {
		switch {
		case rep.PID <= 0 || runtime.GOOS == "windows":
			// Chỉ "có pid và chắc chắn đã chết" mới là ABANDONED; còn lại không dám nói.
			s.Reason, s.Unknown = "UNKNOWN", true
		case RunAlive(rep.PID, name):
			s.Running = true
			s.Duration = max(0, now.Sub(rep.Started))
		default:
			s.Reason = "ABANDONED"
		}
	} else if !rep.Finished.IsZero() && rep.Finished.After(rep.Started) {
		s.Duration = rep.Finished.Sub(rep.Started)
	}
	if s.Worktree != "" {
		fi, err := os.Stat(s.Worktree)
		s.WorktreeExists = err == nil && fi.IsDir()
	}
	// Verify xanh chỉ là đạt khi run GREEN/NO_CHANGE; run kết thúc khác (bị cắt, lỗi, hết hạn mức…) mà verify xanh
	// thì ghi đúng như vậy, không giấu thành "chưa chạy" và không hiện như đạt.
	switch v := rep.VerifyStatus; {
	case v == "FAIL":
		s.Verify = v
	case v == "PASS" && (rep.Reason == "GREEN" || rep.Reason == "NO_CHANGE"):
		s.Verify = v
	case v == "PASS":
		s.Verify = VerifyPassNotGreen
	}
	return s
}

// RunAlive báo tiến trình pid còn sống VÀ đang là `routine run <name>`: pid bị
// tái dùng cho tiến trình khác không tính. Nơi có /proc thì đọc cmdline; nơi
// không có (macOS) chỉ còn Signal(0). Windows: coi như không sống.
// internal/cmd có một phép kiểm tương đương; điều phối sẽ gộp về đây sau.
// shortcut: macOS không kiểm cmdline, pid tái dùng có thể báo nhầm "đang chạy".
func RunAlive(pid int, name string) bool {
	if pid <= 0 || runtime.GOOS == "windows" {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil || p.Signal(syscall.Signal(0)) != nil {
		return false
	}
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cmdline")
	if err != nil {
		_, noProc := os.Stat("/proc/self") // lỗi = không có /proc
		return noProc != nil               // Không có /proc: tin Signal(0); có mà đọc lỗi: coi là chết.
	}
	args := strings.Split(strings.TrimRight(string(raw), "\x00"), "\x00")
	return slices.Contains(args, "routine") && slices.Contains(args, "run") &&
		(slices.Contains(args, name) || slices.ContainsFunc(args, func(a string) bool { return filepath.Base(a) == name+".json" }))
}
