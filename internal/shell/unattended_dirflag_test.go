package shell

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Cờ chọn repo khác (`-C`, `--git-dir`, `--work-tree`) biến lệnh "chỉ đổi worktree
// riêng" thành lệnh trên repo bất kỳ: audit g3 chạy thật `git -C <repo của người
// dùng> rm -rfq .` và `reset --soft` lọt qua. Khi không người trực thì cấm hẳn.
func TestUnattendedGitDeniesFlagsThatPointAtAnotherRepo(t *testing.T) {
	t.Parallel()
	g := unattendedGuardForTest(false)
	for _, args := range [][]string{
		{"git", "-C", "/home/u/repo", "rm", "-rfq", "."},
		{"git", "-C", "../main", "reset", "--soft", "HEAD~1"},
		{"git", "-C", "/home/u/repo", "status"},
		{"git", "--git-dir", "/home/u/repo/.git", "add", "."},
		{"git", "--git-dir=/home/u/repo/.git", "add", "."},
		{"git", "--work-tree", "/home/u/repo", "checkout", "-b", "x"},
		{"git", "--work-tree=/home/u/repo", "add", "."},
	} {
		reason, ok := g.Blocked(args)
		require.Truef(t, ok, "%q must be blocked", args)
		require.NotEmpty(t, reason)
	}
	for _, args := range [][]string{{"git", "status"}, {"git", "add", "."}} {
		_, ok := g.Blocked(args)
		require.Falsef(t, ok, "%q must pass", args)
	}
}
