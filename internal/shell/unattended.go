package shell

import (
	"regexp"
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
//
// Guard soi CHỮ của lệnh nên cũng không phủ: thư mục symlink (`ln -s /other r;
// git -C r ...`), `cd` ra ngoài trước lệnh, chuyển hướng/cp ghi ra ngoài
// (`> /other/f`), và tên biến git ghép lúc chạy (`"GIT_""DIR"`). Đây là danh sách
// đen, không phải sandbox.

// UnattendedGuard là BlockFunc chặn lệnh nguy hiểm khi enabled() trả true.
// allowCommit() mở các lệnh tạo commit (git commit, am, cherry-pick, revert) khi
// routine bật commit tường minh; không mở thứ nào khác. onBlock (có thể nil) nhận lý do để ghi log.
func UnattendedGuard(enabled, allowCommit func() bool, onBlock func(reason string, args []string)) BlockFunc {
	g := ProcessGuard{Deny: func(args []string) (string, bool) {
		return unattendedDenied(args, allowCommit())
	}}
	return func(args []string) bool {
		if !enabled() {
			return false
		}
		// Soi chữ thô trước khi bóc lớp bọc: script của sh -c, chuỗi của env -S,
		// eval, và tiền tố `env NAME=…` mà blockHandler thêm cho biến đã export.
		reason, blocked := gitEnvMentioned(args)
		if !blocked {
			reason, blocked = g.Blocked(args)
		}
		if blocked && onBlock != nil {
			onBlock(reason, args)
		}
		return blocked
	}
}

// outboundTools nói chuyện với dịch vụ ngoài hoặc mở kênh mạng thô.
var outboundTools = map[string]bool{"gh": true, "glab": true, "socat": true, "ncat": true}

// unattendedDenied quyết một lệnh đơn (đã bóc lớp bọc) có bị chặn không.
func unattendedDenied(args []string, allowCommit bool) (string, bool) {
	name := baseName(args[0])
	switch {
	case name == "git":
		return gitDenied(args[1:], allowCommit)
	case strings.HasPrefix(name, "git-"):
		return name + ": git helper commands run git plumbing directly and are not allowed while unattended", true
	case name == "rm":
		return rmDenied(args[1:])
	case name == "rsync" || name == "sftp":
		return name + " copies to or from other machines, not allowed while unattended", true
	case outboundTools[name]:
		return name + " talks to remote services, not allowed while unattended", true
	}
	return "", false
}

// gitEnvName khớp biến môi trường trỏ git sang repo/chỉ mục/kho đối tượng khác hoặc
// nạp cấu hình: việc của -C/--git-dir/-c, mà guard thấy qua argv còn biến thì không.
var gitEnvName = regexp.MustCompile(`\bGIT_(DIR|WORK_TREE|COMMON_DIR|INDEX_FILE|OBJECT_DIRECTORY|NAMESPACE|EXEC_PATH|CONFIG_(PARAMETERS|COUNT|GLOBAL|SYSTEM|KEY_\w+|VALUE_\w+))\b`)

// gitEnvMentioned chặn MỌI lệnh mà chữ của nó nhắc tới một biến trong gitEnvName,
// bất kể giá trị và bất kể lệnh: biến đi theo tiến trình con (make, find -exec,
// script), còn cách gán thì vô số (export trong ngoặc, gán trước khi gọi hàm, tên
// ghép từ biến), nên soi chữ thay vì phân tích phép gán. Nghiêng về chặn: chỉ nhắc
// tên trong một chuỗi cũng bị chặn khi không người trực.
func gitEnvMentioned(args []string) (string, bool) {
	for _, a := range args {
		if m := gitEnvName.FindString(a); m != "" {
			return "a command that sets or mentions " + m + " (it points git at another repository or injects config) is not allowed while unattended", true
		}
	}
	return "", false
}

// gitGlobalValueFlags là cờ toàn cục của git ăn thêm một giá trị ở arg kế. Cờ
// khác ăn giá trị tách rời mà ta không biết thì giá trị rơi vào chỗ lệnh con và
// bị danh sách trắng chặn (fail-closed).
var gitGlobalValueFlags = map[string]bool{"-C": true, "--git-dir": true, "--work-tree": true, "--namespace": true}

// gitReadOnly: lệnh con chỉ đọc, luôn qua.
var gitReadOnly = map[string]bool{
	"status": true, "diff": true, "log": true, "show": true, "blame": true, "grep": true,
	"ls-files": true, "ls-tree": true, "rev-parse": true, "rev-list": true, "describe": true,
	"shortlog": true, "cat-file": true, "merge-base": true, "name-rev": true, "show-ref": true,
	"for-each-ref": true, "check-ignore": true, "diff-tree": true, "diff-index": true,
	"diff-files": true, "count-objects": true, "version": true,
}

// gitWorktreeOnly: chỉ đổi cây làm việc/chỉ mục của worktree riêng của routine.
var gitWorktreeOnly = map[string]bool{"add": true, "rm": true, "mv": true, "apply": true}

// gitDenied là DANH SÁCH TRẮNG cho git khi không người trực. Danh sách đen theo
// tên lệnh con bị vòng ngay bằng alias (`git -c alias.p=push p`, hay `git config
// alias.p push` rồi `git p`), cờ toàn cục lạ, biến chưa mở rộng; nên lệnh con
// lạ, rỗng hay động đều bị chặn, `-c`/`--config-env` bị cấm, `git config` chỉ
// được đọc.
func gitDenied(args []string, allowCommit bool) (string, bool) {
	i := 0
	for i < len(args) && strings.HasPrefix(args[i], "-") {
		a := args[i]
		if a == "-c" || strings.HasPrefix(a, "--config-env") {
			return "git " + a + " can define aliases and hooks and is not allowed while unattended", true
		}
		// Trỏ ra ngoài workspace thì lệnh "chỉ đổi worktree riêng" chạy trên repo bất kỳ.
		if dir, ok := gitRepoFlagValue(args, i); ok && leavesWorkspace(dir) {
			return "git " + a + " " + dir + " points git outside the workspace and is not allowed while unattended", true
		}
		if gitGlobalValueFlags[a] {
			i++
		}
		i++
	}
	if i >= len(args) {
		return "git without a subcommand (its arguments come from elsewhere) is not allowed while unattended", true
	}
	sub, rest := args[i], args[i+1:]
	deny := func(why string) (string, bool) {
		return "git " + sub + ": " + why + " is not allowed while unattended", true
	}
	switch {
	case gitReadOnly[sub], gitWorktreeOnly[sub]:
		return "", false
	}
	switch sub {
	case "commit", "am", "cherry-pick", "revert":
		if allowCommit {
			return "", false
		}
		return deny("committing (set \"commit\": true in the routine definition)")
	case "reset":
		if hasAny(rest, "--hard", "--merge", "--keep") {
			return deny("a hard reset")
		}
		return "", false
	case "branch":
		if hasFlagLetters(rest, "dDmMfcC", "--delete", "--move", "--force", "--copy") {
			return deny("deleting, renaming or copying branches")
		}
		return "", false
	case "tag":
		if hasFlagLetters(rest, "df", "--delete", "--force") {
			return deny("deleting or moving tags")
		}
		return "", false
	case "checkout", "switch":
		// Chỉ tạo nhánh mới: checkout/switch khác có thể bỏ thay đổi hay đổi nhánh dùng chung.
		if hasAny(rest, "-b", "-c", "--create") && !hasAny(rest, "-B", "-C", "-f", "--force", "--force-create", "--") {
			return "", false
		}
		return deny("anything but creating a new branch")
	case "restore":
		if hasAny(rest, "--staged") && !hasAny(rest, "--worktree", "-W", "-S") {
			return "", false
		}
		return deny("restoring files in the worktree")
	case "stash":
		if len(rest) > 0 && (rest[0] == "list" || rest[0] == "show") {
			return "", false
		}
		return deny("changing the stash (shared with your checkout)")
	case "remote":
		if len(rest) == 0 || rest[0] == "-v" || rest[0] == "--verbose" || rest[0] == "show" || rest[0] == "get-url" {
			return "", false
		}
		return deny("changing or contacting remotes")
	case "worktree":
		if len(rest) > 0 && rest[0] == "list" {
			return "", false
		}
		return deny("managing worktrees")
	case "config":
		if hasAny(rest, "--get", "--get-all", "--get-regexp", "--list", "-l") &&
			!hasAny(rest, "--add", "--unset", "--unset-all", "--replace-all", "--rename-section", "--remove-section", "-e", "--edit", "-f", "--file") {
			return "", false
		}
		return deny("writing git config")
	}
	return deny("it is not on the allow-list (read-only commands, add/rm/mv/apply, new branches, local tags)")
}

// hasAny báo rest có một trong các cờ cho trước (nguyên văn).
func hasAny(rest []string, flags ...string) bool {
	for _, a := range rest {
		for _, f := range flags {
			if flagMatches(a, f) {
				return true
			}
		}
	}
	return false
}

// flagMatches: a là cờ f, kể cả khi a là tiền tố viết tắt của option dài f (git
// nhận tiền tố duy nhất: `--har` = `--hard`) hoặc có `=giá-trị`. Tiền tố quá ngắn
// (`--h`) cũng tính, nghiêng về chặn.
func flagMatches(a, f string) bool {
	if a == f {
		return true
	}
	if !strings.HasPrefix(f, "--") || !strings.HasPrefix(a, "--") || len(a) < 3 {
		return false
	}
	name, _, _ := strings.Cut(a, "=")
	return len(name) >= 3 && strings.HasPrefix(f, name)
}

// hasFlagLetters báo rest có cờ dài trong long, hoặc cụm cờ ngắn (gộp được, vd
// -dr; chỉ xét cụm ngắn toàn chữ, để giá trị như `-committerdate` của --sort không
// bị coi là cờ) chứa một chữ trong letters.
func hasFlagLetters(rest []string, letters string, long ...string) bool {
	for _, a := range rest {
		if strings.HasPrefix(a, "--") {
			for _, l := range long {
				if flagMatches(a, l) {
					return true
				}
			}
			continue
		}
		if len(a) > 1 && len(a) <= 4 && strings.HasPrefix(a, "-") && isLetters(a[1:]) && strings.ContainsAny(a[1:], letters) {
			return true
		}
	}
	return false
}

func isLetters(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

// leavesWorkspace: đường dẫn có thể ra ngoài thư mục hiện tại (tuyệt đối, ~, biến
// chưa mở rộng, thành phần ".."). Không xét `cd` trước đó.
func leavesWorkspace(p string) bool {
	return strings.HasPrefix(p, "/") || strings.HasPrefix(p, "~") || strings.Contains(p, "$") ||
		strings.Contains(p, dynamicWord) || p == ".." || strings.HasPrefix(p, "../") ||
		strings.HasSuffix(p, "/..") || strings.Contains(p, "/../")
}

// gitRepoFlagValue trả thư mục mà cờ toàn cục ở args[i] (`-C`, `--git-dir`,
// `--work-tree`, kể cả dạng `=`) chọn cho git. Thiếu giá trị thì trả "$" để bị chặn.
func gitRepoFlagValue(args []string, i int) (string, bool) {
	a := args[i]
	switch {
	case a == "-C" || a == "--git-dir" || a == "--work-tree":
		if i+1 < len(args) {
			return args[i+1], true
		}
		return "$", true
	case strings.HasPrefix(a, "--git-dir=") || strings.HasPrefix(a, "--work-tree="):
		return a[strings.IndexByte(a, '=')+1:], true
	}
	return "", false
}

// rmDenied chặn `rm` có đích ngoài workspace, đệ quy hay không (`rm -f
// ~/.ssh/id_rsa` cũng nguy hiểm): đường dẫn tuyệt đối, ~, có biến chưa mở rộng,
// hoặc có thành phần "..". Không có đích nào (tham số đến từ xargs/stdin) cũng bị
// chặn vì không kiểm được. Xoá trong workspace (rm -rf build, rm file) vẫn qua. KHÔNG
// xét thư mục hiện tại: `cd / && rm -rf home` không bị bắt ở đây (tool bash từ chối
// working_dir ngoài workspace khi không người trực, nhưng `cd` trong lệnh thì không).
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
	_ = recursive // mọi rm đều phải ở trong workspace, đệ quy hay không
	if len(targets) == 0 {
		return "rm without explicit targets (they come from xargs or stdin) is not allowed while unattended", true
	}
	for _, t := range targets {
		if leavesWorkspace(t) {
			return "rm with a target outside the workspace (" + t + ") is not allowed while unattended", true
		}
	}
	return "", false
}
