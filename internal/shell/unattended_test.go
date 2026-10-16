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
		{"rm", "-f", "/tmp/one-file"}, // không đệ quy
		{"go", "test", "./..."},
		{"ls", "-la"},
		{"bash", "-c", "go test ./... | tail -5"},
		{"bash", "-c", `git commit-tree --help >/dev/null`},
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
