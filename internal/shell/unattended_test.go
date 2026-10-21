package shell

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func unattendedGuardForTest(allowCommit bool) ProcessGuard {
	return ProcessGuard{Deny: func(args []string) (string, bool) { return unattendedDenied(args, allowCommit) }}
}

// Chế độ không người trực: lệnh phá huỷ hoặc đẩy ra ngoài bị chặn CỨNG ở lớp
// thực thi (trước cả quyền đã cấp), kể cả bọc trong env/sh -c/vòng lặp.
func TestUnattendedDeniesDestructiveAndOutboundCommands(t *testing.T) {
	t.Parallel()
	g := unattendedGuardForTest(false)
	for _, args := range [][]string{
		// git đẩy/gộp/viết lại lịch sử/xoá
		{"git", "push", "origin", "main"},
		{"git", "push"},
		{"git", "-C", "/x", "push"},
		{"git", "-c", "a=b", "push", "--force"},
		{"/usr/bin/git", "push"},
		{"env", "GIT_SSH_COMMAND=true", "git", "push"},
		{"git", "pull"},
		{"git", "merge", "feature"},
		{"git", "rebase", "main"},
		{"git", "reset", "--hard", "HEAD~1"},
		{"git", "reset", "--merge"},
		{"git", "clean", "-fd"},
		{"git", "branch", "-D", "x"},
		{"git", "branch", "--delete", "x"},
		{"git", "branch", "-m", "a", "b"},
		{"git", "tag", "-d", "v1"},
		{"git", "worktree", "remove", "x"},
		{"git", "worktree", "prune"},
		{"git", "update-ref", "-d", "refs/heads/x"},
		{"git", "remote", "add", "o", "u"},
		{"git", "remote", "set-url", "origin", "u"},
		{"git", "commit", "-m", "x"},
		{"git", "filter-branch"},
		{"git", "reflog", "expire", "--all"},
		// ra mạng / máy khác
		{"rsync", "-a", "src", "host:dst"},
		// xoá đệ quy ra ngoài workspace
		{"rm", "-rf", "/tmp/x"},
		{"rm", "-rf", "~/x"},
		{"rm", "-rf", "../x"},
		{"rm", "-rf", "a/../../x"},
		{"rm", "-rf", "$HOME/x"},
		{"rm", "-r", "-f", "/x"},
		{"rm", "--recursive", "/x"},
		{"rm", "-fr", "/"},
		// lồng trong shell con / vòng lặp
		{"sh", "-c", "git push origin HEAD"},
		{"bash", "-c", "for i in 1 2; do git push; done"},
		{"bash", "-c", "true && (cd x && git reset --hard)"},
		{"bash", "-c", "echo hi; rm -rf /tmp/y"},
	} {
		reason, ok := g.Blocked(args)
		require.Truef(t, ok, "%q phải bị chặn", args)
		require.NotEmpty(t, reason)
	}
}

// Vế ngược: việc thường của một routine sửa code trong worktree của nó vẫn qua.
func TestUnattendedAllowsNormalWorkInTheWorktree(t *testing.T) {
	t.Parallel()
	g := unattendedGuardForTest(false)
	for _, args := range [][]string{
		{"git", "status"},
		{"git", "diff"},
		{"git", "log", "--oneline", "-5"},
		{"git", "add", "."},
		{"git", "branch", "-a"},
		{"git", "branch", "--list"},
		{"git", "reset", "HEAD", "file.go"},
		{"git", "reset", "--soft", "HEAD~1"},
		{"git", "tag", "-l"},
		{"git", "remote", "-v"},
		{"git", "worktree", "list"},
		{"git", "stash", "list"},
		{"git", "checkout", "-b", "feature"},
		{"git", "show", "HEAD"},
		{"git", "-C", "sub", "status"},
		{"rm", "-rf", "build"},
		{"rm", "-rf", "./dist"},
		{"rm", "-r", "node_modules"},
		{"rm", "file.txt"},
		{"go", "test", "./..."},
		{"ls", "-la"},
		{"bash", "-c", "go test ./... | tail -5"},
		{"sh", "-c", "for f in a b; do echo $f; done"},
	} {
		_, ok := g.Blocked(args)
		require.Falsef(t, ok, "%q là việc bình thường, không được chặn", args)
	}
}

// git commit bị chặn mặc định, mở được bằng lựa chọn tường minh của routine.
func TestUnattendedCommitIsOptIn(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"git", "commit", "-m", "x"},
		{"git", "commit", "--amend"},
		{"sh", "-c", "git add -A && git commit -m x"},
	} {
		_, blocked := unattendedGuardForTest(false).Blocked(args)
		require.Truef(t, blocked, "%q", args)
		_, blocked = unattendedGuardForTest(true).Blocked(args)
		require.Falsef(t, blocked, "%q mở được khi routine bật commit", args)
	}
	// Bật commit không mở cánh cửa nào khác.
	_, blocked := unattendedGuardForTest(true).Blocked([]string{"git", "push"})
	require.True(t, blocked)
}

// UnattendedGuard chỉ chặn khi chế độ đang bật.
func TestUnattendedGuardRespectsTheSwitch(t *testing.T) {
	t.Parallel()
	on := false
	guard := UnattendedGuard(func() bool { return on }, func() bool { return false }, nil)
	require.False(t, guard([]string{"git", "push"}), "tắt thì không chặn gì (dùng tương tác bình thường)")
	on = true
	require.True(t, guard([]string{"git", "push"}))
	require.False(t, guard([]string{"git", "status"}))
}

// Có onBlock thì nhận lý do để ghi log.
func TestUnattendedGuardReportsTheReason(t *testing.T) {
	t.Parallel()
	var gotReason string
	var gotArgs []string
	guard := UnattendedGuard(func() bool { return true }, func() bool { return false }, func(reason string, args []string) {
		gotReason, gotArgs = reason, args
	})
	require.True(t, guard([]string{"git", "push", "origin"}))
	require.Contains(t, gotReason, "push")
	require.Equal(t, []string{"git", "push", "origin"}, gotArgs)
}

// Lượt soát độc lập 16/10: đường vòng ĐÃ CHỨNG MINH với danh sách đen theo tên lệnh
// con. `git` trong chế độ không người trực nay là DANH SÁCH TRẮNG: lệnh con lạ,
// rỗng hoặc động thì chặn (fail-closed).
func TestUnattendedGitIsAnAllowListNotADenyList(t *testing.T) {
	t.Parallel()
	g := unattendedGuardForTest(false)
	for _, args := range [][]string{
		// alias và cấu hình
		{"git", "-c", "alias.p=push", "p", "origin"},
		{"git", "--config-env", "alias.p=V", "p"},
		{"git", "config", "alias.p", "push"},
		{"git", "config", "core.hooksPath", "/tmp/h"},
		{"git", "config", "diff.external", "/tmp/x"},
		{"git", "p"}, // alias do chính agent vừa ghi
		// cờ toàn cục lạ ăn một giá trị tách rời
		{"git", "--attr-source", "HEAD", "push"},
		{"git", "--exec-path", "/x", "push"},
		// lệnh đẩy/viết lại không có trong danh sách trắng
		{"git", "send-pack", "origin"},
		{"git", "http-push", "u"},
		{"git", "imap-send"},
		{"git", "fetch"},
		{"git", "gc", "--prune=now"},
		{"git", "prune"},
		{"git", "stash", "clear"},
		{"git", "stash", "drop"},
		{"git", "stash", "pop"},
		{"git", "stash"},
		{"git", "checkout", "-f"},
		{"git", "checkout", "--", "."},
		{"git", "checkout", "-B", "other"},
		{"git", "restore", "."},
		{"git", "switch", "-C", "other"},
		{"git", "submodule", "update"},
		{"git", "commit-tree", "HEAD^{tree}"},
		// chạy thẳng trình giúp việc của git
		{"/usr/lib/git-core/git-push"},
		{"git-push", "origin"},
		{"git-receive-pack", "x"},
		// biến chưa biết giá trị lúc phân tích, và lệnh không có lệnh con
		{"bash", "-c", `git "$1"`, "_", "push"},
		{"bash", "-c", "git $0", "push"},
		{"git"},
		// tham số đến từ nơi khác
		{"xargs", "git"},
		{"xargs", "-r", "rm", "-rf"},
		{"rm", "-rf"},
		// lớp bọc chưa bóc được
		{"timeout", "-s", "KILL", "5", "git", "push"},
		{"timeout", "--signal=KILL", "5", "git", "push"},
		{"taskset", "-c", "0,2", "git", "push"},
		{"taskset", "0x3", "git", "push"},
		{"flock", "lockfile", "git", "push"},
		{"strace", "-f", "git", "push"},
		{"systemd-run", "--user", "git", "push"},
		{"nice", "-n", "5", "git", "push"},
		{"chrt", "-i", "0", "git", "push"},
		// ra mạng / lên dịch vụ git ngoài
		{"gh", "pr", "merge"},
		{"gh", "api", "-X", "DELETE", "repos/x/y"},
		{"glab", "mr", "merge"},
		{"socat", "-", "TCP:h:1"},
		{"ncat", "h", "1"},
	} {
		reason, ok := g.Blocked(args)
		require.Truef(t, ok, "%q phải bị chặn", args)
		require.NotEmpty(t, reason)
	}
}

// Vế ngược của danh sách trắng: việc git thường của một routine sửa code vẫn qua.
func TestUnattendedGitAllowListPassesNormalWork(t *testing.T) {
	t.Parallel()
	g := unattendedGuardForTest(false)
	for _, args := range [][]string{
		{"git", "status", "--short"},
		{"git", "diff", "--stat", "HEAD~1"},
		{"git", "log", "--oneline", "-5"},
		{"git", "show", "HEAD:file.go"},
		{"git", "blame", "file.go"},
		{"git", "grep", "-n", "foo"},
		{"git", "ls-files"},
		{"git", "ls-tree", "HEAD"},
		{"git", "rev-parse", "--show-toplevel"},
		{"git", "rev-list", "--count", "HEAD"},
		{"git", "describe", "--tags"},
		{"git", "cat-file", "-p", "HEAD"},
		{"git", "merge-base", "a", "b"},
		{"git", "for-each-ref"},
		{"git", "add", "-A"},
		{"git", "rm", "old.go"},
		{"git", "mv", "a.go", "b.go"},
		{"git", "apply", "x.patch"},
		{"git", "reset", "HEAD", "f.go"},
		{"git", "restore", "--staged", "f.go"},
		{"git", "checkout", "-b", "feature/x"},
		{"git", "switch", "-c", "feature/y"},
		{"git", "branch", "--show-current"},
		{"git", "branch", "-a", "-v"},
		{"git", "tag", "-l"},
		{"git", "tag", "v1"},
		{"git", "remote", "-v"},
		{"git", "remote", "show", "origin"},
		{"git", "remote", "get-url", "origin"},
		{"git", "worktree", "list"},
		{"git", "stash", "list"},
		{"git", "stash", "show"},
		{"git", "config", "--get", "user.name"},
		{"git", "config", "--list"},
		{"git", "-C", "sub", "diff"},
		{"git", "--no-pager", "log"},
		{"env", "FOO=1", "git", "status"},
		{"timeout", "5s", "git", "status"},
		{"timeout", "-s", "KILL", "5", "git", "status"},
		{"taskset", "-c", "0,2", "go", "test", "./..."},
		{"flock", "lockfile", "go", "test", "./..."},
		{"strace", "-f", "ls"},
		{"rm", "-rf", "build"},
		{"bash", "-c", "git status && git diff"},
		{"bash", "-c", `for f in a b; do git add "$f"; done`},
	} {
		reason, ok := g.Blocked(args)
		require.Falsef(t, ok, "%q là việc bình thường, không được chặn (%s)", args, reason)
	}
}

// git commit và các lệnh tạo commit chỉ mở khi routine bật commit; lời mô tả phải
// nói đúng là cả bốn lệnh được mở, và không mở thêm gì.
func TestUnattendedCommitOptInCoversExactlyTheCommitMakingCommands(t *testing.T) {
	t.Parallel()
	for _, sub := range []string{"commit", "am", "cherry-pick", "revert"} {
		_, blocked := unattendedGuardForTest(false).Blocked([]string{"git", sub})
		require.Truef(t, blocked, "git %s bị chặn mặc định", sub)
		_, blocked = unattendedGuardForTest(true).Blocked([]string{"git", sub})
		require.Falsef(t, blocked, "git %s mở được khi bật commit", sub)
	}
	for _, args := range [][]string{{"git", "push"}, {"git", "stash", "clear"}, {"git", "config", "alias.p", "push"}, {"git", "-c", "alias.p=push", "p"}} {
		_, blocked := unattendedGuardForTest(true).Blocked(args)
		require.Truef(t, blocked, "%q: mở commit không mở gì khác", args)
	}
}

// rm ra ngoài workspace bị chặn dù KHÔNG đệ quy (`rm -f ~/.ssh/id_rsa`), và rm không có
// đích (tham số đến từ xargs/stdin) cũng bị chặn; xoá trong workspace vẫn qua.
func TestUnattendedRmTargetsMustStayInTheWorkspaceEvenWithoutRecursion(t *testing.T) {
	t.Parallel()
	g := unattendedGuardForTest(false)
	for _, args := range [][]string{
		{"rm", "-f", "/tmp/one-file"},
		{"rm", "/etc/hosts"},
		{"rm", "-f", "~/.ssh/id_rsa"},
		{"rm", "-f", "$HOME/x"},
		{"rm", "../outside.txt"},
		{"rm", "-f", "--", "/x"},
		{"xargs", "rm"},
		{"xargs", "-r", "rm", "-f"},
		{"rm", "-f"},
	} {
		reason, ok := g.Blocked(args)
		require.Truef(t, ok, "%q phải bị chặn", args)
		require.NotEmpty(t, reason)
	}
	for _, args := range [][]string{
		{"rm", "file.txt"},
		{"rm", "-f", "build/out.o"},
		{"rm", "-rf", "build"},
		{"rm", "./a", "b/c"},
	} {
		_, ok := g.Blocked(args)
		require.Falsef(t, ok, "%q ở trong workspace, không được chặn", args)
	}
}

// Thứ mà build offline hay dữ liệu cần, không dựng lại được: module cache của Go và volume
// của Docker. Cache build (`go clean -cache`) và lệnh đọc của docker vẫn qua.
func TestUnattendedDeniesDeletingWhatCannotBeRebuilt(t *testing.T) {
	t.Parallel()
	g := unattendedGuardForTest(false)
	for _, args := range [][]string{
		{"go", "clean", "-modcache"},
		{"go", "clean", "--modcache"},
		{"go", "clean", "-modcache=true"},
		{"/usr/local/go/bin/go", "clean", "-x", "-modcache"},
		{"go", "-C", "sub", "clean", "-modcache"},
		{"docker", "volume", "rm", "pgdata"},
		{"docker", "volume", "remove", "pgdata"},
		{"docker", "volume", "prune", "-f"},
		{"docker", "--context", "x", "volume", "prune"},
		{"docker", "system", "prune"},
		{"docker", "system", "prune", "-a", "--volumes"},
		{"docker", "compose", "down", "-v"},
		{"docker", "compose", "-f", "dev.yml", "down", "--volumes"},
		{"docker-compose", "down", "-v"},
		{"docker-compose", "down", "--volumes", "--remove-orphans"},
		{"sh", "-c", "docker compose down -v && docker compose up -d"},
	} {
		reason, ok := g.Blocked(args)
		require.Truef(t, ok, "%q phải bị chặn", args)
		require.Contains(t, reason, "cannot be rebuilt", args)
	}
	for _, args := range [][]string{
		{"go", "clean", "-cache"},
		{"go", "clean", "-testcache"},
		{"go", "build", "./..."},
		{"docker", "ps"},
		{"docker", "volume", "ls"},
		{"docker", "system", "df"},
		{"docker", "compose", "down"},
		{"docker", "compose", "up", "-d"},
		{"docker-compose", "down", "--remove-orphans"},
		{"docker", "image", "prune", "-f"},
	} {
		reason, ok := g.Blocked(args)
		require.Falsef(t, ok, "%q không được chặn (%s)", args, reason)
	}
}
