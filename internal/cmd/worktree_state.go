package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// worktreeState là câu trả lời DUY NHẤT cho "agent đã làm ra gì trong cây" (docs/routine-invariants.md,
// Q3). Mọi mục của `git status --ignored=matching -uall`, cộng mục bị giấu khỏi status và commit ngoài
// base, nằm đúng một hạng. Dấu vân tay (gitFingerprint), changed_files (changedFiles), bộ dọn
// (keepReason) và export (buildPatch) đều đọc nó, không tự hỏi git theo quy tắc riêng.
// Đường dẫn tính từ Root (gốc repo); thư mục bị ignore trọn hiện một lần, kết thúc bằng "/".
type worktreeState struct {
	Root        string
	Changed     []string // tệp theo dõi bị đổi so với HEAD (đã stage hay chưa, kể cả xoá): việc
	Untracked   []string // tệp chưa theo dõi, không bị ignore: việc
	IgnoredWork []string // bị ignore, ngoài .crush/, không phải thư mục dựng lại được: việc
	Regenerable []string // thư mục bị ignore dựng lại được (regenerableDirs): không phải việc
	DewData     []string // chưa theo dõi hoặc bị ignore dưới .crush/: dữ liệu phiên của dew, không phải việc
	Hidden      []string // assume-unchanged, hoặc skip-worktree mà tệp có trên đĩa: việc mà status không thấy
	Commits     int      // commit ngoài base trên HEAD hoặc nhánh của lượt (chỉ khi có base): việc
	Committed   []string // tệp khác nhau giữa base và HEAD hoặc nhánh của lượt (chỉ khi Commits > 0)
}

// workFiles là tập "việc" (mọi hạng là việc), sắp xếp, không trùng.
func (s worktreeState) workFiles() []string {
	f := slices.Concat(s.Changed, s.Committed, s.Untracked, s.IgnoredWork, s.Hidden)
	slices.Sort(f)
	return slices.Compact(f)
}

// classifyWorktree xếp cây làm việc chứa dir vào các hạng của worktreeState. base rỗng thì không
// đếm commit (dấu vân tay băm HEAD thay vào đó); branch là nhánh của lượt, đếm cùng HEAD (commit
// trên HEAD tách rời không nằm trên nhánh nào). Không chạy fsmonitor hay hook do repo cấu hình
// (agent ghi được cấu hình đó), không lấy khoá index (lượt có thể đang chạy git trong cây này).
// shortcut: clean filter do repo khai (.gitattributes + filter.<x>.clean) vẫn chạy khi status so
// nội dung; agent vốn chạy được shell trong chính cây này nên đây không phải leo quyền. Chặn bằng
// --attr-source khi phải phân loại một cây mà agent không chạy được lệnh trong đó.
func classifyWorktree(ctx context.Context, dir, base, branch string) (worktreeState, error) {
	// Có nhánh của lượt mà không có base thì không đếm được commit: lỗi, không phải "0 commit" (bộ
	// dọn sẽ gỡ worktree có commit trên HEAD tách rời).
	if branch != "" && base == "" {
		return worktreeState{}, errors.New("cannot count the run's commits without its base commit")
	}
	git := func(args ...string) (string, error) {
		// status.showUntrackedFiles=all: cấu hình repo không giấu được tệp mới.
		c := exec.CommandContext(ctx, "git", append([]string{"--no-optional-locks", "-C", dir,
			"-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null", "-c", "status.showUntrackedFiles=all"}, args...)...)
		var stderr bytes.Buffer
		c.Stderr = &stderr
		out, err := c.Output()
		if err != nil {
			return "", fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(stderr.String()))
		}
		return string(out), nil
	}
	var st worktreeState
	top, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		return st, err
	}
	// Đường dẫn của status --porcelain luôn tính từ gốc repo: các lệnh sau (git đọc dir) chạy ở đó.
	st.Root = strings.TrimSuffix(top, "\n")
	dir = st.Root
	// -uall: từng tệp mới. --ignored=matching: thư mục bị ignore trọn hiện MỘT lần (node_modules), mục
	// sâu hơn (build/x.env) chứng tỏ thư mục đó không bị ignore trọn. --no-renames: một đường dẫn mỗi mục.
	out, err := git("status", "--porcelain", "-z", "--no-renames", "--untracked-files=all", "--ignored=matching")
	if err != nil {
		return st, err
	}
	for _, rec := range strings.Split(out, "\x00") {
		if len(rec) < 4 {
			continue
		}
		code, p := rec[:2], rec[3:]
		switch {
		case code != "??" && code != "!!":
			st.Changed = append(st.Changed, p)
		case strings.HasPrefix(p, ".crush/"):
			st.DewData = append(st.DewData, p)
		case code == "??":
			st.Untracked = append(st.Untracked, p)
		case regenerable(p):
			st.Regenerable = append(st.Regenerable, p)
		default:
			st.IgnoredWork = append(st.IgnoredWork, p)
		}
	}
	// assume-unchanged (thẻ chữ thường) và skip-worktree (S/s) giấu sửa đổi khỏi status, và
	// `worktree remove` xoá luôn nó. skip-worktree mà tệp không có trên đĩa (sparse-checkout) thì
	// không có gì để mất.
	files, err := git("ls-files", "-v", "-z")
	if err != nil {
		return st, err
	}
	for _, rec := range strings.Split(files, "\x00") {
		if len(rec) < 3 {
			continue
		}
		switch tag := rec[0]; {
		case tag == 'S' || tag == 's':
			if _, err := os.Lstat(filepath.Join(dir, rec[2:])); errors.Is(err, fs.ErrNotExist) {
				continue
			}
		case tag < 'a' || tag > 'z':
			continue
		}
		st.Hidden = append(st.Hidden, rec[2:])
	}
	if base == "" {
		return st, nil
	}
	tips := []string{"HEAD"}
	if branch != "" {
		tips = append(tips, "refs/heads/"+branch)
	}
	// "^"+base: base (đọc từ report.json) không bao giờ thành một cờ của git.
	n, err := git(slices.Concat([]string{"rev-list", "--count"}, tips, []string{"^" + base})...)
	if err != nil {
		return st, err
	}
	if st.Commits, err = strconv.Atoi(strings.TrimSpace(n)); err != nil || st.Commits == 0 {
		return st, err
	}
	// Tệp của commit trên HEAD và trên nhánh (HEAD có thể đã tách khỏi nhánh, về lại base).
	for _, tip := range tips {
		names, err := git("diff", "--name-only", "-z", "--no-renames", "--no-ext-diff", "--end-of-options", base, tip)
		if err != nil {
			return st, err
		}
		for _, p := range strings.Split(names, "\x00") {
			if p != "" {
				st.Committed = append(st.Committed, p)
			}
		}
	}
	slices.Sort(st.Committed)
	st.Committed = slices.Compact(st.Committed)
	return st, nil
}
