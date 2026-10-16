package shell

import (
	"strings"
)

// Chế độ không người trực (routine chạy theo lịch hay lúc không ai nhìn): `run`
// tự duyệt mọi yêu cầu quyền, nên chỉ còn lớp chặn ở trình thực thi này. Đây là
// DANH SÁCH ĐEN các lệnh phá huỷ hoặc đẩy ra ngoài, không phải sandbox: tiến
// trình vẫn có quyền của người dùng. Routine vì vậy chạy trong worktree riêng,
// và lớp này chặn thứ không đảo ngược được: đẩy/gộp/viết lại lịch sử git, xoá
// đệ quy ra ngoài workspace, ghi cấu hình git toàn cục, đồng bộ sang máy khác.
//
// Cố ý KHÔNG phủ: ngôn ngữ khác (python/node tự xoá hay mở socket), `find
// -delete`, script nạp từ tệp hoặc stdin. Xem docs trong CHANGELOG.

// UnattendedGuard là BlockFunc chặn lệnh nguy hiểm khi enabled() trả true.
// allowCommit() mở `git commit` (routine bật commit tường minh); không mở thứ
// nào khác. onBlock (có thể nil) nhận lý do để ghi log.
func UnattendedGuard(enabled, allowCommit func() bool, onBlock func(reason string, args []string)) BlockFunc {
	g := ProcessGuard{Deny: func(args []string) (string, bool) {
		return unattendedDenied(args, allowCommit())
	}}
	return func(args []string) bool {
		if !enabled() {
			return false
		}
		reason, blocked := g.Blocked(args)
		if blocked && onBlock != nil {
			onBlock(reason, args)
		}
		return blocked
	}
}

// unattendedDenied quyết một lệnh đơn (đã bóc lớp bọc) có bị chặn không.
func unattendedDenied(args []string, allowCommit bool) (string, bool) {
	switch baseName(args[0]) {
	case "git":
		return gitDenied(args[1:], allowCommit)
	case "rm":
		return rmDenied(args[1:])
	case "rsync", "sftp":
		return baseName(args[0]) + " copies to or from other machines, not allowed while unattended", true
	}
	return "", false
}

// gitGlobalValueFlags là cờ toàn cục của git ăn thêm một giá trị ở arg kế.
var gitGlobalValueFlags = map[string]bool{"-C": true, "-c": true, "--git-dir": true, "--work-tree": true, "--namespace": true}

func gitDenied(args []string, allowCommit bool) (string, bool) {
	i := 0
	for i < len(args) && strings.HasPrefix(args[i], "-") {
		if gitGlobalValueFlags[args[i]] {
			i++
		}
		i++
	}
	if i >= len(args) {
		return "", false
	}
	sub, rest := args[i], args[i+1:]
	deny := func(why string) (string, bool) {
		return "git " + sub + ": " + why + " is not allowed while unattended", true
	}
	switch sub {
	case "push", "pull", "merge", "rebase", "filter-branch", "update-ref", "clean":
		return deny("it publishes, merges or rewrites history or deletes untracked files")
	case "commit", "am", "cherry-pick", "revert":
		if !allowCommit {
			return deny("committing (enable commit in the routine definition)")
		}
	case "reset":
		if hasAny(rest, "--hard", "--merge", "--keep") {
			return deny("a hard reset")
		}
	case "branch":
		if hasFlagLetters(rest, "dDmMfcC", "--delete", "--move", "--force", "--copy") {
			return deny("deleting, renaming or copying branches")
		}
	case "tag":
		if hasFlagLetters(rest, "df", "--delete", "--force") {
			return deny("deleting or moving tags")
		}
	case "worktree":
		if len(rest) > 0 && (rest[0] == "remove" || rest[0] == "prune" || rest[0] == "move") {
			return deny("removing or moving worktrees")
		}
	case "remote":
		if len(rest) > 0 {
			switch rest[0] {
			case "add", "remove", "rm", "rename", "set-url", "set-head", "set-branches", "prune", "update":
				return deny("changing or contacting remotes")
			}
		}
	case "reflog":
		if len(rest) > 0 && (rest[0] == "expire" || rest[0] == "delete") {
			return deny("deleting reflog entries")
		}
	case "config":
		if hasAny(rest, "--global", "--system", "--file", "-f") {
			return deny("writing git config outside the repository")
		}
	}
	return "", false
}

// hasAny báo rest có một trong các cờ cho trước (nguyên văn).
func hasAny(rest []string, flags ...string) bool {
	for _, a := range rest {
		for _, f := range flags {
			if a == f {
				return true
			}
		}
	}
	return false
}

// hasFlagLetters báo rest có cờ dài trong long, hoặc cờ ngắn (gộp được, vd -dr)
// chứa một chữ trong letters.
func hasFlagLetters(rest []string, letters string, long ...string) bool {
	for _, a := range rest {
		if strings.HasPrefix(a, "--") {
			for _, l := range long {
				if a == l {
					return true
				}
			}
			continue
		}
		if strings.HasPrefix(a, "-") && strings.ContainsAny(a[1:], letters) {
			return true
		}
	}
	return false
}

// rmDenied chặn `rm` đệ quy có đích ra ngoài workspace: đường dẫn tuyệt đối, ~,
// có biến chưa mở rộng, hoặc có thành phần "..". Xoá đệ quy trong workspace
// (rm -rf build) và xoá một tệp (kể cả tuyệt đối, không đệ quy) vẫn qua.
func rmDenied(args []string) (string, bool) {
	recursive := false
	var targets []string
	afterDashes := false
	for _, a := range args {
		switch {
		case afterDashes:
			targets = append(targets, a)
		case a == "--":
			afterDashes = true
		case a == "--recursive":
			recursive = true
		case strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--"):
			if strings.ContainsAny(a[1:], "rR") {
				recursive = true
			}
		case strings.HasPrefix(a, "--"):
		default:
			targets = append(targets, a)
		}
	}
	if !recursive {
		return "", false
	}
	for _, t := range targets {
		if strings.HasPrefix(t, "/") || strings.HasPrefix(t, "~") || strings.Contains(t, "$") ||
			strings.Contains(t, dynamicWord) || t == ".." || strings.HasPrefix(t, "../") ||
			strings.HasSuffix(t, "/..") || strings.Contains(t, "/../") {
			return "recursive rm with a target outside the workspace (" + t + ") is not allowed while unattended", true
		}
	}
	return "", false
}
